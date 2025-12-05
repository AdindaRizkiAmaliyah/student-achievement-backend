package service

import (
	"context"
	"net/http"
	"strconv"
	"time"
	"os"
	"path/filepath"

	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AchievementService mendefinisikan handler untuk fitur prestasi FR-003 s/d FR-010.
type AchievementService interface {
	// FR-003: CreateAchievement — mahasiswa membuat prestasi (status draft).
	CreateAchievement(ctx *gin.Context)
	// FR-004: SubmitForVerification — mahasiswa submit draft untuk diverifikasi.
	SubmitForVerification(ctx *gin.Context)
	// FR-005: DeleteAchievement — mahasiswa menghapus prestasi draft (soft delete).
	DeleteAchievement(ctx *gin.Context)
	// FR-006, FR-007, FR-008, FR-010: GetAchievements — list prestasi tergantung role.
	GetAchievements(ctx *gin.Context)
	// FR-007: VerifyAchievement — dosen wali memverifikasi prestasi.
	VerifyAchievement(ctx *gin.Context)
	// FR-008: RejectAchievement — dosen wali menolak prestasi dengan catatan.
	RejectAchievement(ctx *gin.Context)

	// --- Tambahan sesuai SRS 5.4 ---
	// DetailAchievement — GET /api/v1/achievements/:id (detail gabungan Postgres + Mongo).
	DetailAchievement(ctx *gin.Context)
	// UpdateAchievement — PUT /api/v1/achievements/:id (update konten, mahasiswa pemilik).
	UpdateAchievement(ctx *gin.Context)
	// GetAchievementHistory — GET /api/v1/achievements/:id/history (status history).
	GetAchievementHistory(ctx *gin.Context)
	// UploadAttachment — Mahasiswa mengunggah bukti prestasi (file).
	UploadAttachment(ctx *gin.Context) // POST /api/v1/achievements/:id/attachments
}

// achievementService adalah implementasi konkret AchievementService.
type achievementService struct {
	repo         repository.AchievementRepository
	userRepo     repository.UserRepository
	lecturerRepo repository.LecturerRepository // dipakai untuk FR-006/007/008 (advisor)
}

// NewAchievementService membuat instance baru AchievementService.
func NewAchievementService(
	repo repository.AchievementRepository,
	userRepo repository.UserRepository,
	lecturerRepo repository.LecturerRepository,
) AchievementService {
	return &achievementService{
		repo:         repo,
		userRepo:     userRepo,
		lecturerRepo: lecturerRepo,
	}
}

// customError sederhana agar bisa dibedakan kalau studentID tidak ada di context.
type customError struct{ msg string }

func (e *customError) Error() string { return e.msg }

var ErrNoStudentIDInContext = &customError{msg: "studentID not found in context (ensure middleware sets studentID)"}

// getStudentIDFromContext mengambil studentID dari JWT (kalau role mahasiswa).
func getStudentIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
	if v, ok := ctx.Get("studentID"); ok {
		if sid, ok2 := v.(uuid.UUID); ok2 {
			return sid, nil
		}
	}
	return uuid.Nil, ErrNoStudentIDInContext
}

// getUserIDFromContext mengambil userID dari JWT.
func getUserIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
	if v, ok := ctx.Get("userID"); ok {
		if uid, ok2 := v.(uuid.UUID); ok2 {
			return uid, nil
		}
	}
	return uuid.Nil, &customError{msg: "userID not found in context"}
}

// getRoleFromContext membaca role dari JWT.
func getRoleFromContext(ctx *gin.Context) string {
	if v, ok := ctx.Get("role"); ok {
		if r, ok2 := v.(string); ok2 {
			return r
		}
	}
	return ""
}

// ===============================================================
//  FR-003: CreateAchievement (Mahasiswa)
//  Endpoint: POST /api/v1/achievements
// ===============================================================
func (s *achievementService) CreateAchievement(ctx *gin.Context) {
	role := getRoleFromContext(ctx)
	if role != "mahasiswa" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya mahasiswa yang dapat membuat prestasi", "forbidden", nil))
		return
	}

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

// ===============================================================
//  FR-004: SubmitForVerification (Mahasiswa)
//  Endpoint: POST /api/v1/achievements/:id/submit
// ===============================================================
func (s *achievementService) SubmitForVerification(ctx *gin.Context) {
	role := getRoleFromContext(ctx)
	if role != "mahasiswa" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya mahasiswa yang dapat submit prestasi", "forbidden", nil))
		return
	}

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

// ===============================================================
//  FR-005: DeleteAchievement (Mahasiswa, status draft)
//  Endpoint: DELETE /api/v1/achievements/:id
// ===============================================================
func (s *achievementService) DeleteAchievement(ctx *gin.Context) {
	role := getRoleFromContext(ctx)
	if role != "mahasiswa" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya mahasiswa yang dapat menghapus prestasi", "forbidden", nil))
		return
	}

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

// ===============================================================
//  Helper: buildAchievementListItem
//  Membantu membentuk 1 item response list prestasi (reference + detail).
// ===============================================================
func (s *achievementService) buildAchievementListItem(ctx *gin.Context, ref model.AchievementReference) map[string]any {
	item := map[string]any{
		"id":          ref.ID,
		"studentId":   ref.StudentID,
		"status":      ref.Status,
		"createdAt":   ref.CreatedAt,
		"submittedAt": ref.SubmittedAt,
		"verifiedAt":  ref.VerifiedAt,
	}

	if ref.VerifiedBy != nil {
		item["verifiedBy"] = ref.VerifiedBy
	}
	if ref.RejectionNote != nil {
		item["rejectionNote"] = ref.RejectionNote
	}

	// Ambil detail dari MongoDB
	if md, err := s.repo.FindDetailByMongoID(ctx, ref.MongoAchievementID); err == nil && md != nil {
		item["title"] = md.Title
		item["type"] = md.AchievementType
		item["points"] = md.Points
		item["tags"] = md.Tags
	}

	return item
}

// ===============================================================
//  FR-006 / FR-007 / FR-008 / FR-010: GetAchievements
//  Endpoint: GET /api/v1/achievements
//
//  Perilaku per role:
//    - Mahasiswa: daftar prestasi miliknya (FR-006 dari sisi mahasiswa)
//    - Dosen Wali: daftar prestasi mahasiswa bimbingan (FR-006)
//    - Admin: lihat semua prestasi (FR-010, dengan filter & pagination)
// ===============================================================
func (s *achievementService) GetAchievements(ctx *gin.Context) {
	role := getRoleFromContext(ctx)

	switch role {

	// ================= Mahasiswa =================
	case "mahasiswa":
		studentID, err := getStudentIDFromContext(ctx)
		if err != nil || studentID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
			return
		}

		refs, err := s.repo.FindByStudentID(studentID.String())
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				utils.BuildResponseFailed("Gagal mengambil prestasi", err.Error(), nil))
			return
		}

		var list []map[string]any
		for _, r := range refs {
			list = append(list, s.buildAchievementListItem(ctx, r))
		}

		ctx.JSON(http.StatusOK,
			utils.BuildResponseSuccess("Berhasil mengambil daftar prestasi mahasiswa", list))
		return

	// ================= Dosen Wali =================
	case "dosen_wali":
		userID, err := getUserIDFromContext(ctx)
		if err != nil || userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
			return
		}

		// Ambil lecturer berdasarkan userID
		lecturer, err := s.lecturerRepo.FindByUserID(userID)
		if err != nil {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
			return
		}

		// Ambil semua studentID bimbingan dosen wali ini
		studentIDs, err := s.lecturerRepo.GetAdviseeStudentIDs(lecturer.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				utils.BuildResponseFailed("Gagal mengambil daftar mahasiswa bimbingan", err.Error(), nil))
			return
		}

		// Ambil semua achievement_references untuk daftar studentID tersebut
		refs, err := s.lecturerRepo.FindAchievementsByStudentIDs(ctx, studentIDs)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				utils.BuildResponseFailed("Gagal mengambil prestasi mahasiswa bimbingan", err.Error(), nil))
			return
		}

		var list []map[string]any
		for _, r := range refs {
			list = append(list, s.buildAchievementListItem(ctx, r))
		}

		ctx.JSON(http.StatusOK,
			utils.BuildResponseSuccess("Berhasil mengambil daftar prestasi mahasiswa bimbingan", list))
		return

	// ================= Admin (FR-010) =================
	case "admin":
		// Query params: ?status=submitted&page=1&limit=10
		statusParam := ctx.Query("status")
		var status *string
		if statusParam != "" {
			status = &statusParam
		}

		page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

		refs, total, err := s.repo.FindAll(status, page, limit)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				utils.BuildResponseFailed("Gagal mengambil daftar semua prestasi", err.Error(), nil))
			return
		}

		var list []map[string]any
		for _, r := range refs {
			list = append(list, s.buildAchievementListItem(ctx, r))
		}

		meta := map[string]any{
			"page":      page,
			"limit":     limit,
			"totalData": total,
			"totalPage": (total + int64(limit) - 1) / int64(limit),
		}

		ctx.JSON(http.StatusOK,
			utils.BuildResponseSuccess("Berhasil mengambil semua prestasi (admin)", map[string]any{
				"items": list,
				"meta":  meta,
			}))
		return

	default:
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Role tidak dikenali untuk akses daftar prestasi", "forbidden", nil))
		return
	}
}

// ===============================================================
//  FR-007: VerifyAchievement (Dosen Wali)
//  Endpoint: POST /api/v1/achievements/:id/verify
// ===============================================================
func (s *achievementService) VerifyAchievement(ctx *gin.Context) {
	role := getRoleFromContext(ctx)
	if role != "dosen_wali" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya dosen wali yang dapat memverifikasi prestasi", "forbidden", nil))
		return
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil || userID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
		return
	}

	lecturer, err := s.lecturerRepo.FindByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
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

	// Cek apakah mahasiswa ini benar advisee doswal tersebut
	ok, err := s.lecturerRepo.IsAdvisorOf(lecturer.ID, ref.StudentID)
	if err != nil || !ok {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Prestasi bukan milik mahasiswa bimbingan Anda", "forbidden", nil))
		return
	}

	if ref.Status != "submitted" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Hanya prestasi berstatus 'submitted' yang dapat diverifikasi", "invalid_status", nil))
		return
	}

	verifierID := userID.String()
	if err := s.repo.UpdateStatus(id, "verified", repository.UpdateStatusOptions{
		VerifierID: &verifierID,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal memverifikasi prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil diverifikasi", nil))
}

// ===============================================================
//  FR-008: RejectAchievement (Dosen Wali)
//  Endpoint: POST /api/v1/achievements/:id/reject
// ===============================================================
func (s *achievementService) RejectAchievement(ctx *gin.Context) {
	role := getRoleFromContext(ctx)
	if role != "dosen_wali" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya dosen wali yang dapat menolak prestasi", "forbidden", nil))
		return
	}

	userID, err := getUserIDFromContext(ctx)
	if err != nil || userID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
		return
	}

	lecturer, err := s.lecturerRepo.FindByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
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

	ok, err := s.lecturerRepo.IsAdvisorOf(lecturer.ID, ref.StudentID)
	if err != nil || !ok {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Prestasi bukan milik mahasiswa bimbingan Anda", "forbidden", nil))
		return
	}

	if ref.Status != "submitted" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Hanya prestasi berstatus 'submitted' yang dapat ditolak", "invalid_status", nil))
		return
	}

	var input struct {
		RejectionNote string `json:"rejectionNote" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Catatan penolakan wajib diisi", err.Error(), nil))
		return
	}

	verifierID := userID.String()
	note := input.RejectionNote

	if err := s.repo.UpdateStatus(id, "rejected", repository.UpdateStatusOptions{
		VerifierID:    &verifierID,
		RejectionNote: &note,
	}); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menolak prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil ditolak", nil))
}

// ===============================================================
//  DETAIL — SRS 5.4
//  Endpoint: GET /api/v1/achievements/:id
//  - Mahasiswa: hanya boleh lihat miliknya
//  - Dosen wali: hanya prestasi mahasiswa bimbingan
//  - Admin: boleh semua
// ===============================================================
func (s *achievementService) DetailAchievement(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	role := getRoleFromContext(ctx)
	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	switch role {
	case "mahasiswa":
		studentID, _ := getStudentIDFromContext(ctx)
		if studentID == uuid.Nil || ref.StudentID != studentID {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Anda tidak berhak melihat prestasi ini", "forbidden", nil))
			return
		}
	case "dosen_wali":
		userID, _ := getUserIDFromContext(ctx)
		if userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
			return
		}
		lecturer, err := s.lecturerRepo.FindByUserID(userID)
		if err != nil {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
			return
		}
		ok, err := s.lecturerRepo.IsAdvisorOf(lecturer.ID, ref.StudentID)
		if err != nil || !ok {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Prestasi bukan milik mahasiswa bimbingan Anda", "forbidden", nil))
			return
		}
	case "admin":
		// admin bebas
	default:
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Role tidak berhak mengakses detail prestasi", "forbidden", nil))
		return
	}

	detail, err := s.repo.FindDetailByMongoID(ctx, ref.MongoAchievementID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil detail prestasi", err.Error(), nil))
		return
	}

	data := map[string]any{
		"id":            ref.ID,
		"studentId":     ref.StudentID,
		"status":        ref.Status,
		"submittedAt":   ref.SubmittedAt,
		"verifiedAt":    ref.VerifiedAt,
		"verifiedBy":    ref.VerifiedBy,
		"rejectionNote": ref.RejectionNote,
		"createdAt":     ref.CreatedAt,
		"updatedAt":     ref.UpdatedAt,
		"detail":        detail,
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil detail prestasi", data))
}

// ===============================================================
//  UPDATE — SRS 5.4
//  Endpoint: PUT /api/v1/achievements/:id
//  - Hanya mahasiswa pemilik
//  - Contoh aturan: hanya boleh edit saat status 'draft'
// ===============================================================
func (s *achievementService) UpdateAchievement(ctx *gin.Context) {
	role := getRoleFromContext(ctx)
	if role != "mahasiswa" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya mahasiswa yang dapat mengubah prestasi", "forbidden", nil))
		return
	}

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
			utils.BuildResponseFailed("Anda tidak berhak mengubah prestasi ini", "forbidden", nil))
		return
	}

	// Di sini kita batasi hanya bisa edit saat status draft (mengikuti pola delete).
	if ref.Status != "draft" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Prestasi hanya dapat diubah saat status 'draft'", "invalid_status", nil))
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
	mongoUpdate := model.Achievement{
		StudentID:       ref.StudentID,
		AchievementType: input.AchievementType,
		Title:           input.Title,
		Description:     input.Description,
		Details:         input.Details,
		Attachments:     input.Attachments,
		Tags:            input.Tags,
		Points:          input.Points,
		UpdatedAt:       now,
	}

	if err := s.repo.UpdateContent(ctx, id, &mongoUpdate); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal memperbarui prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Prestasi berhasil diperbarui", nil))
}

// ===============================================================
//  HISTORY — SRS 5.4
//  Endpoint: GET /api/v1/achievements/:id/history
//  - Mengembalikan timeline status berdasarkan kolom created/submitted/verified/dll.
// ===============================================================
func (s *achievementService) GetAchievementHistory(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	role := getRoleFromContext(ctx)
	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}

	// Reuse rules autorisasi sama seperti DetailAchievement
	switch role {
	case "mahasiswa":
		studentID, _ := getStudentIDFromContext(ctx)
		if studentID == uuid.Nil || ref.StudentID != studentID {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Anda tidak berhak melihat riwayat prestasi ini", "forbidden", nil))
			return
		}
	case "dosen_wali":
		userID, _ := getUserIDFromContext(ctx)
		if userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi dosen wali diperlukan", "no_user_id", nil))
			return
		}
		lecturer, err := s.lecturerRepo.FindByUserID(userID)
		if err != nil {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
			return
		}
		ok, err := s.lecturerRepo.IsAdvisorOf(lecturer.ID, ref.StudentID)
		if err != nil || !ok {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Prestasi bukan milik mahasiswa bimbingan Anda", "forbidden", nil))
			return
		}
	case "admin":
		// no restriction
	default:
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Role tidak berhak mengakses riwayat prestasi", "forbidden", nil))
		return
	}

	events := []map[string]any{
		{
			"status": "created",
			"at":     ref.CreatedAt,
		},
	}

	if ref.SubmittedAt != nil {
		events = append(events, map[string]any{
			"status": "submitted",
			"at":     ref.SubmittedAt,
		})
	}
	if ref.VerifiedAt != nil && ref.Status == "verified" {
		events = append(events, map[string]any{
			"status": "verified",
			"at":     ref.VerifiedAt,
		})
	}
	if ref.VerifiedAt != nil && ref.Status == "rejected" {
		events = append(events, map[string]any{
			"status": "rejected",
			"at":     ref.VerifiedAt,
			"note":   ref.RejectionNote,
		})
	}
	if ref.Status == "deleted" {
		events = append(events, map[string]any{
			"status": "deleted",
			"at":     ref.UpdatedAt, // kita pakai updatedAt sebagai indikasi delete
		})
	}

	data := map[string]any{
		"id":            ref.ID,
		"studentId":     ref.StudentID,
		"currentStatus": ref.Status,
		"events":        events,
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil riwayat status prestasi", data))
}

// UploadAttachment menangani upload bukti prestasi (file) oleh mahasiswa.
// Endpoint: POST /api/v1/achievements/:id/attachments
// - Body: multipart/form-data dengan key "file" (tipe File).
// - Optional field: "fileType" (string), "description" kalau nanti mau dipakai.
// - Hanya boleh diakses oleh pemilik prestasi (role: mahasiswa).
func (s *achievementService) UploadAttachment(ctx *gin.Context) {
	// Pastikan role adalah mahasiswa.
	role := getRoleFromContext(ctx)
	if role != "mahasiswa" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya mahasiswa yang dapat mengunggah lampiran", "forbidden", nil))
		return
	}

	// Ambil studentID dari token.
	studentID, err := getStudentIDFromContext(ctx)
	if err != nil || studentID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Autentikasi mahasiswa diperlukan", "no_student_id", nil))
		return
	}

	// Ambil ID achievement dari path param.
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID prestasi diperlukan", "missing_id", nil))
		return
	}

	// Pastikan achievement ada dan memang milik mahasiswa ini.
	ref, err := s.repo.FindByID(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Prestasi tidak ditemukan", err.Error(), nil))
		return
	}
	if ref.StudentID != studentID {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Anda tidak berhak menambahkan lampiran ke prestasi ini", "forbidden", nil))
		return
	}
	if ref.Status == "deleted" {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Prestasi yang sudah dihapus tidak dapat diberi lampiran", "invalid_status", nil))
		return
	}

	// Ambil file dari form-data (key: "file").
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("File lampiran wajib diunggah (field 'file')", err.Error(), nil))
		return
	}

	// Optional: tipe file (misalnya "certificate", "photo", dll).
	fileType := ctx.PostForm("fileType")
	if fileType == "" {
		// default: pakai ekstensi sebagai tipe kasar.
		fileType = filepath.Ext(fileHeader.Filename)
		if len(fileType) > 0 && fileType[0] == '.' {
			fileType = fileType[1:]
		}
	}

	// Tentukan direktori penyimpanan lokal.
	// Contoh: ./uploads/achievements/<achievementID>/
	baseDir := "uploads"
	destDir := filepath.Join(baseDir, "achievements", id)

	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menyiapkan direktori upload", err.Error(), nil))
		return
	}

	// Buat nama file unik agar tidak bentrok.
	now := time.Now()
	filename := strconv.FormatInt(now.UnixNano(), 10) + "_" + fileHeader.Filename
	fullPath := filepath.Join(destDir, filename)

	// Simpan file ke disk.
	if err := ctx.SaveUploadedFile(fileHeader, fullPath); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menyimpan file upload", err.Error(), nil))
		return
	}

	// URL/relative path yang disimpan di Mongo (nanti bisa diserve via static files kalau mau).
	fileURL := "/" + filepath.ToSlash(filepath.Join(baseDir, "achievements", id, filename))

	// Bentuk objek attachment sesuai SRS.
	attachment := model.Attachment{
		FileName:   fileHeader.Filename,
		FileURL:    fileURL,
		FileType:   fileType,
		UploadedAt: now,
	}

	// Simpan ke MongoDB (append ke array attachments).
	if err := s.repo.AddAttachment(context.Background(), id, attachment); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menyimpan lampiran ke database", err.Error(), nil))
		return
	}

	// Response sukses berisi data attachment yang baru dibuat.
	ctx.JSON(http.StatusCreated,
		utils.BuildResponseSuccess("Lampiran berhasil diunggah", attachment))
}
