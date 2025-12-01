package service

import (
	"context"
	"net/http"
	"time"

	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AchievementService mendefinisikan behavior fitur prestasi (FR-03 s.d. FR-06).
type AchievementService interface {
	// CreateAchievement digunakan mahasiswa untuk membuat draft prestasi (FR-03).
	CreateAchievement(ctx *gin.Context)
	// SubmitForVerification digunakan mahasiswa untuk submit prestasi ke dosen wali (FR-04).
	SubmitForVerification(ctx *gin.Context)
	// DeleteAchievement digunakan mahasiswa untuk menghapus prestasi draft (FR-05).
	DeleteAchievement(ctx *gin.Context)
	// GetAchievementsByStudent:
	//   - untuk role mahasiswa: list prestasi miliknya sendiri
	//   - untuk role dosen_wali: list prestasi mahasiswa bimbingan (FR-06)
	GetAchievementsByStudent(ctx *gin.Context)
}

// achievementService adalah implementasi konkret AchievementService.
type achievementService struct {
	repo         repository.AchievementRepository
	lecturerRepo repository.LecturerRepository
}

// NewAchievementService membuat instance baru achievementService.
func NewAchievementService(repo repository.AchievementRepository, lecturerRepo repository.LecturerRepository) AchievementService {
	return &achievementService{
		repo:         repo,
		lecturerRepo: lecturerRepo,
	}
}

// customError dipakai untuk pesan error internal sederhana.
type customError struct{ msg string }

func (e *customError) Error() string { return e.msg }

// ErrNoStudentIDInContext dipakai ketika context tidak memiliki studentID.
var ErrNoStudentIDInContext = &customError{msg: "studentID not found in context (ensure middleware sets studentID)"}

// getStudentIDFromContext mengambil studentID dari context (di-set oleh AuthMiddleware).
func getStudentIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
	if v, ok := ctx.Get("studentID"); ok {
		if sid, ok2 := v.(uuid.UUID); ok2 {
			return sid, nil
		}
	}
	if v, ok := ctx.Get("userID"); ok {
		if uid, ok2 := v.(uuid.UUID); ok2 {
			return uid, nil
		}
	}
	return uuid.Nil, ErrNoStudentIDInContext
}

// CreateAchievement:
// - Hanya boleh dipanggil oleh mahasiswa (token berisi studentID).
// - Membuat dokumen prestasi di MongoDB + referensi di PostgreSQL (status: draft).
func (s *achievementService) CreateAchievement(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized, utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	var input struct {
		AchievementType string                   `json:"achievementType" binding:"required"`
		Title           string                   `json:"title" binding:"required"`
		Description     string                   `json:"description"`
		Details         model.AchievementDetails `json:"details"`
		Tags            []string                 `json:"tags"`
		Points          int                      `json:"points"`
		Attachments     []model.Attachment       `json:"attachments"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	now := time.Now()

	pg := model.AchievementReference{
		StudentID:          studentID,
		MongoAchievementID: "",
		Status:             "draft",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	mongo := model.Achievement{
		StudentID:       studentID,
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

	if err := s.repo.Create(context.Background(), &pg, &mongo); err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.BuildResponseFailed("Gagal menyimpan prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusCreated, utils.BuildResponseSuccess("Prestasi berhasil disimpan sebagai draft", map[string]interface{}{
		"id":                 pg.ID,
		"mongoAchievementId": pg.MongoAchievementID,
		"status":             pg.Status,
	}))
}

// SubmitForVerification:
// - Hanya untuk mahasiswa pemilik prestasi.
// - Hanya bisa dari status draft → submitted.
func (s *achievementService) SubmitForVerification(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized, utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden, utils.BuildResponseFailed("Anda tidak berhak submit prestasi ini", "forbidden", nil))
		return
	}

	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("Prestasi hanya bisa disubmit jika status draft", "invalid_status", nil))
		return
	}

	if err := s.repo.UpdateStatus(id, "submitted", repository.UpdateStatusOptions{}); err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.BuildResponseFailed("Gagal submit prestasi", err.Error(), nil))
		return
	}

	// TODO: bisa ditambahkan notifikasi ke dosen wali jika diperlukan

	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Prestasi berhasil disubmit", nil))
}

// DeleteAchievement:
// - Hanya untuk mahasiswa pemilik prestasi.
// - Hanya bisa menghapus jika status = draft.
// - Implementasi: status di Postgres → deleted, di Mongo → flag deleted = true.
func (s *achievementService) DeleteAchievement(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized, utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden, utils.BuildResponseFailed("Anda tidak berhak menghapus prestasi ini", "forbidden", nil))
		return
	}

	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest, utils.BuildResponseFailed("Hanya prestasi draft yang dapat dihapus", "invalid_status", nil))
		return
	}

	if err := s.repo.UpdateStatus(id, "deleted", repository.UpdateStatusOptions{}); err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.BuildResponseFailed("Gagal menghapus prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Prestasi berhasil dihapus", nil))
}

// GetAchievementsByStudent:
// - Jika role = mahasiswa: menampilkan daftar prestasi milik dirinya sendiri.
// - Jika role = dosen_wali: menampilkan prestasi seluruh mahasiswa bimbingan (advisor).
func (s *achievementService) GetAchievementsByStudent(ctx *gin.Context) {
	roleVal, _ := ctx.Get("role")
	role, _ := roleVal.(string)

	userIDVal, _ := ctx.Get("userID")
	userID, _ := userIDVal.(uuid.UUID)

	var (
		refs []model.AchievementReference
		err  error
	)

	switch role {
	case "mahasiswa":
		// Flow lama: mahasiswa melihat prestasi milik sendiri.
		studentID, err := getStudentIDFromContext(ctx)
		if err != nil || studentID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized, utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
			return
		}
		refs, err = s.repo.FindByStudentID(studentID.String())

	case "dosen_wali":
		// Flow baru: dosen wali melihat prestasi mahasiswa bimbingan.
		if userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized, utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
			return
		}

		lec, errLec := s.lecturerRepo.FindByUserID(userID)
		if errLec != nil {
			ctx.JSON(http.StatusForbidden, utils.BuildResponseFailed("Data dosen wali tidak ditemukan", errLec.Error(), nil))
			return
		}

		refs, err = s.repo.FindByAdvisorID(lec.ID)

	default:
		// Role lain belum didukung untuk endpoint ini.
		ctx.JSON(http.StatusForbidden, utils.BuildResponseFailed("Role tidak diizinkan mengakses daftar prestasi", "role_not_supported", nil))
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.BuildResponseFailed("Gagal mengambil prestasi", err.Error(), nil))
		return
	}

	// Bentuk response ringkas per prestasi + detail dari Mongo (jika ada).
	var list []map[string]interface{}
	for _, r := range refs {
		item := map[string]interface{}{
			"id":          r.ID,
			"status":      r.Status,
			"createdAt":   r.CreatedAt,
			"submittedAt": r.SubmittedAt,
			"studentId":   r.StudentID, // berguna untuk dosen wali (tahu milik mahasiswa siapa)
		}

		if md, err := s.repo.FindDetailByMongoID(ctx.Request.Context(), r.MongoAchievementID); err == nil && md != nil {
			item["title"] = md.Title
			item["type"] = md.AchievementType
			item["points"] = md.Points
		}

		list = append(list, item)
	}

	ctx.JSON(http.StatusOK, utils.BuildResponseSuccess("Berhasil mengambil daftar prestasi", list))
}
