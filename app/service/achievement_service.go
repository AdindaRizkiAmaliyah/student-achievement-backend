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

// AchievementService mendefinisikan behavior fitur prestasi (FR-003 s.d. FR-007).
type AchievementService interface {
	// CreateAchievement digunakan mahasiswa untuk membuat draft prestasi (FR-003).
	CreateAchievement(ctx *gin.Context)
	// SubmitForVerification digunakan mahasiswa untuk submit prestasi ke dosen wali (FR-004).
	SubmitForVerification(ctx *gin.Context)
	// DeleteAchievement digunakan mahasiswa untuk menghapus prestasi draft (FR-005).
	DeleteAchievement(ctx *gin.Context)
	// GetAchievementsByStudent:
	//   - mahasiswa: list prestasi sendiri
	//   - dosen_wali: list prestasi mahasiswa bimbingan (FR-006)
	GetAchievementsByStudent(ctx *gin.Context)
	// VerifyAchievement digunakan dosen wali untuk memverifikasi prestasi mahasiswa (FR-007).
	VerifyAchievement(ctx *gin.Context)
}

// achievementService adalah implementasi konkret AchievementService.
type achievementService struct {
	repo         repository.AchievementRepository
	lecturerRepo repository.LecturerRepository
}

// NewAchievementService membuat instance baru achievementService.
func NewAchievementService(
	repo repository.AchievementRepository,
	lecturerRepo repository.LecturerRepository,
) AchievementService {
	return &achievementService{
		repo:         repo,
		lecturerRepo: lecturerRepo,
	}
}

// customError dipakai untuk pesan error internal sederhana.
type customError struct{ msg string }

// Error mengembalikan pesan error untuk customError.
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
// - Hanya mahasiswa (studentID wajib ada di token).
// - Simpan dokumen ke MongoDB + referensi ke PostgreSQL (status draft).
// - Sesuai FR-003 di SRS.
func (s *achievementService) CreateAchievement(ctx *gin.Context) {
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
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
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	now := time.Now()

	// Data referensi di PostgreSQL (tabel achievement_references).
	pg := model.AchievementReference{
		StudentID:          studentID,
		MongoAchievementID: "",
		Status:             "draft",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Data detail di MongoDB (collection achievements).
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
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menyimpan prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusCreated,
		utils.BuildResponseSuccess("Prestasi berhasil disimpan sebagai draft", map[string]any{
			"id":                 pg.ID,
			"mongoAchievementId": pg.MongoAchievementID,
			"status":             pg.Status,
		}))
}

// SubmitForVerification:
// - Hanya mahasiswa pemilik prestasi.
// - Hanya bisa dari status draft → submitted.
// - Sesuai FR-004 di SRS.
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

	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Anda tidak berhak submit prestasi ini", "forbidden", nil))
		return
	}

	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Prestasi hanya bisa disubmit jika status draft", "invalid_status", nil))
		return
	}

	if err := s.repo.UpdateStatus(id, "submitted", repository.UpdateStatusOptions{}); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal submit prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil disubmit", nil))
}

// DeleteAchievement:
// - Hanya mahasiswa pemilik prestasi.
// - Hanya untuk status draft.
// - Status di Postgres → deleted, di Mongo → flag deleted (handle di repo).
// - Sesuai FR-005 di SRS (+ enum tambahan deleted sesuai revisi kamu).
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

	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Anda tidak berhak menghapus prestasi ini", "forbidden", nil))
		return
	}

	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Hanya prestasi draft yang dapat dihapus", "invalid_status", nil))
		return
	}

	if err := s.repo.UpdateStatus(id, "deleted", repository.UpdateStatusOptions{}); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menghapus prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil dihapus", nil))
}

// GetAchievementsByStudent:
// - Mahasiswa: melihat daftar prestasi miliknya sendiri.
// - Dosen wali: melihat prestasi semua mahasiswa bimbingan (advisor_id = lecturer.id).
// - Sesuai FR-006 di SRS, endpoint tetap: GET /api/v1/achievements.
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
		// Flow mahasiswa: hanya prestasi dirinya sendiri.
		studentID, err := getStudentIDFromContext(ctx)
		if err != nil || studentID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
			return
		}
		refs, err = s.repo.FindByStudentID(studentID.String())

	case "dosen_wali":
		// Flow dosen wali: seluruh prestasi mahasiswa bimbingan.
		if userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
			return
		}

		lec, errLec := s.lecturerRepo.FindByUserID(userID)
		if errLec != nil {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Data dosen wali tidak ditemukan", errLec.Error(), nil))
			return
		}

		refs, err = s.repo.FindByAdvisorID(lec.ID)

	default:
		// Role lain (misal admin) belum di-handle di endpoint ini.
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Role tidak diizinkan mengakses daftar prestasi", "role_not_supported", nil))
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil prestasi", err.Error(), nil))
		return
	}

	// Bentuk response: ringkasan + detail dari Mongo (jika ada).
	var list []map[string]any
	for _, r := range refs {
		item := map[string]any{
			"id":          r.ID,
			"status":      r.Status,
			"createdAt":   r.CreatedAt,
			"submittedAt": r.SubmittedAt,
			"studentId":   r.StudentID,
		}

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

// VerifyAchievement:
// - Hanya untuk role = dosen_wali.
// - Precondition: status prestasi = 'submitted'.
// - Hanya boleh memverifikasi prestasi mahasiswa bimbingannya (advisor_id = dosen wali).
// - Update status → 'verified', set verified_by & verified_at di repo.
// - Sesuai FR-007 di SRS.
func (s *achievementService) VerifyAchievement(ctx *gin.Context) {
	// Pastikan role = dosen_wali.
	roleVal, _ := ctx.Get("role")
	role, _ := roleVal.(string)
	if role != "dosen_wali" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya dosen wali yang dapat memverifikasi prestasi", "role_not_allowed", nil))
		return
	}

	// Ambil userID dosen dari token.
	userIDVal, _ := ctx.Get("userID")
	userID, _ := userIDVal.(uuid.UUID)
	if userID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
		return
	}

	// Ambil data dosen wali dari tabel lecturers (berdasarkan user_id).
	lecturer, err := s.lecturerRepo.FindByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
		return
	}

	// Ambil ID prestasi dari path parameter.
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	// Ambil referensi prestasi dari PostgreSQL.
	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	// Precondition: status harus 'submitted' (SRS FR-007).
	if ref.Status != "submitted" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Prestasi hanya bisa diverifikasi jika status 'submitted'", "invalid_status", nil))
		return
	}

	// Pastikan mahasiswa pemilik prestasi adalah bimbingan dosen ini.
	isAdvisee, err := s.lecturerRepo.IsAdvisorOf(lecturer.ID, ref.StudentID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal memeriksa relasi dosen wali", err.Error(), nil))
		return
	}
	if !isAdvisee {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Prestasi ini bukan milik mahasiswa bimbingan Anda", "not_advisee", nil))
		return
	}

	// Set verifierID = userID dosen wali (kolom verified_by di Postgres).
	verifierIDStr := userID.String()
	opts := repository.UpdateStatusOptions{
		VerifierID: &verifierIDStr,
	}

	// Update status → verified (verified_at & verified_by dihandle di repository.UpdateStatus).
	if err := s.repo.UpdateStatus(id, "verified", opts); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal memverifikasi prestasi", err.Error(), nil))
		return
	}

	// Response ringkas sesuai SRS: return updated status.
	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil diverifikasi", map[string]any{
			"id":     ref.ID,
			"status": "verified",
		}))
}
