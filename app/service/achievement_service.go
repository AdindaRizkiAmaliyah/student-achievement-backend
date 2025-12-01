package service

import (
	"net/http"
	"time"

	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AchievementService mendefinisikan behavior fitur prestasi (FR-03 s.d. FR-05).
type AchievementService interface {
	CreateAchievement(ctx *gin.Context)       // FR-03
	SubmitForVerification(ctx *gin.Context)   // FR-04
	DeleteAchievement(ctx *gin.Context)       // FR-05
	GetAchievementsByStudent(ctx *gin.Context) // FR-06 (list, pendukung)
}

// achievementService adalah implementasi konkret AchievementService.
type achievementService struct {
	repo repository.AchievementRepository
}

// NewAchievementService membuat instance achievementService.
func NewAchievementService(repo repository.AchievementRepository) AchievementService {
	return &achievementService{repo: repo}
}

// customError tipe sederhana untuk error internal service.
type customError struct{ msg string }

func (e *customError) Error() string { return e.msg }

var ErrNoStudentIDInContext = &customError{msg: "studentID not found in context (ensure middleware sets studentID)"}

// getStudentIDFromContext mengambil studentID dari context (diset oleh AuthMiddleware khusus mahasiswa).
// Tidak ada fallback ke userID supaya endpoint ini hanya bisa diakses role mahasiswa.
func getStudentIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
	if v, ok := ctx.Get("studentID"); ok {
		if sid, ok2 := v.(uuid.UUID); ok2 {
			return sid, nil
		}
	}
	return uuid.Nil, ErrNoStudentIDInContext
}

// CreateAchievement – FR-03
// Endpoint: POST /api/v1/achievements
// Fungsinya: Mahasiswa membuat prestasi baru dengan status awal 'draft'.
func (s *achievementService) CreateAchievement(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	// Payload mengikuti SRS: achievementType, title, description, details, tags, points, attachments.
	var input struct {
		AchievementType string                   `json:"achievementType" binding:"required"`
		Title           string                   `json:"title" binding:"required"`
		Description     string                   `json:"description"`
		Details         model.AchievementDetails `json:"details"`
		Tags            []string                 `json:"tags"`
		Points          float64                  `json:"points"`
		Attachments     []model.AchievementFile  `json:"attachments"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	now := time.Now()

	// Data referensi di PostgreSQL (achievement_references)
	pg := model.AchievementReference{
		StudentID:          studentID,
		MongoAchievementID: "",
		Status:             "draft",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Data lengkap prestasi di MongoDB (achievements)
	mongo := model.Achievement{
		StudentID:       studentID.String(),   // simpan sebagai string UUID
		AchievementType: input.AchievementType,
		Title:           input.Title,
		Description:     input.Description,
		Details:         input.Details,
		Attachments:     input.Attachments,
		Tags:            input.Tags,
		Points:          input.Points,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx.Request.Context(), &pg, &mongo); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menyimpan prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusCreated,
		utils.BuildResponseSuccess("Prestasi berhasil disimpan sebagai draft", map[string]interface{}{
			"id":                 pg.ID,
			"mongoAchievementId": pg.MongoAchievementID,
			"status":             pg.Status,
		}))
}

// SubmitForVerification – FR-04
// Endpoint: POST /api/v1/achievements/:id/submit
// Fungsinya: Mahasiswa mengajukan prestasi draft ke dosen wali untuk diverifikasi.
func (s *achievementService) SubmitForVerification(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	// Ambil reference di Postgres
	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	// Pastikan pemilik prestasi adalah mahasiswa yang login
	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Anda tidak berhak submit prestasi ini", "forbidden", nil))
		return
	}

	// Hanya boleh submit jika masih draft
	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Prestasi hanya bisa disubmit jika status draft", "invalid_status", nil))
		return
	}

	// Update status menjadi submitted + isi submitted_at
	if err := s.repo.UpdateStatus(id, "submitted", repository.UpdateStatusOptions{}); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal submit prestasi", err.Error(), nil))
		return
	}

	// Pembuatan notifikasi ke dosen wali (jika ada) bisa ditambahkan kemudian.

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil disubmit", nil))
}

// DeleteAchievement – FR-05
// Endpoint: DELETE /api/v1/achievements/:id
// Fungsinya: Mahasiswa menghapus prestasi dengan status draft (soft delete).
func (s *achievementService) DeleteAchievement(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	// Ambil reference di Postgres
	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	// Pastikan pemilik prestasi adalah mahasiswa yang login
	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Anda tidak berhak menghapus prestasi ini", "forbidden", nil))
		return
	}

	// Hanya prestasi draft yang dapat dihapus
	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Hanya prestasi draft yang dapat dihapus", "invalid_status", nil))
		return
	}

	// Update status menjadi deleted + soft-delete di Mongo (melalui repository.UpdateStatus)
	if err := s.repo.UpdateStatus(id, "deleted", repository.UpdateStatusOptions{}); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menghapus prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil dihapus", nil))
}

// GetAchievementsByStudent – FR-06 (list prestasi mahasiswa)
// Endpoint: GET /api/v1/achievements
// Fungsinya: Mengambil daftar prestasi milik mahasiswa yang login (tanpa status deleted).
func (s *achievementService) GetAchievementsByStudent(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	// Ambil daftar reference (Postgres)
	refs, err := s.repo.FindByStudentID(studentID.String())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil prestasi", err.Error(), nil))
		return
	}

	var list []map[string]interface{}
	for _, r := range refs {
		item := map[string]interface{}{
			"id":          r.ID,
			"status":      r.Status,
			"createdAt":   r.CreatedAt,
			"submittedAt": r.SubmittedAt,
		}

		// Ambil detail dari Mongo, jika berhasil
		if md, err := s.repo.FindDetailByMongoID(ctx.Request.Context(), r.MongoAchievementID); err == nil && md != nil {
			item["title"] = md.Title
			item["type"] = md.AchievementType
			item["points"] = md.Points
		}

		list = append(list, item)
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil daftar prestasi", list))
}
