package service

import (
	"net/http"

	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StudentService meng-handle endpoint SRS 5.5 untuk Students:
// - GET /api/v1/students
// - GET /api/v1/students/:id
// - GET /api/v1/students/:id/achievements
// - PUT /api/v1/students/:id/advisor
type StudentService interface {
	GetStudents(ctx *gin.Context)
	GetStudentDetail(ctx *gin.Context)
	GetStudentAchievements(ctx *gin.Context)
	UpdateAdvisor(ctx *gin.Context)
}

// studentService menyimpan dependency ke repository yang dibutuhkan.
type studentService struct {
	studentRepo     repository.StudentRepository
	achievementRepo repository.AchievementRepository
}

// NewStudentService membuat instance StudentService baru.
func NewStudentService(
	studentRepo repository.StudentRepository,
	achievementRepo repository.AchievementRepository,
) StudentService {
	return &studentService{
		studentRepo:     studentRepo,
		achievementRepo: achievementRepo,
	}
}

// =====================
// GET /api/v1/students
// Admin: melihat daftar semua mahasiswa
// =====================
func (s *studentService) GetStudents(ctx *gin.Context) {

	// gunakan helper ensureAdmin yang sudah ada di admin_service.go
	if !ensureAdmin(ctx) {
		return
	}

	students, err := s.studentRepo.FindAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil daftar mahasiswa", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil daftar mahasiswa", students))
}

// ========================
// GET /api/v1/students/:id
// Admin: melihat detail 1 mahasiswa
// ========================
func (s *studentService) GetStudentDetail(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	idStr := ctx.Param("id")
	studentID, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID mahasiswa tidak valid", err.Error(), nil))
		return
	}

	st, err := s.studentRepo.FindByID(studentID)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Mahasiswa tidak ditemukan", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil detail mahasiswa", st))
}

// ====================================
// GET /api/v1/students/:id/achievements
// Admin / Dosen Wali: melihat prestasi seorang mahasiswa
// ====================================
func (s *studentService) GetStudentAchievements(ctx *gin.Context) {

	// Admin & dosen wali boleh akses.
	roleI, _ := ctx.Get("role")
	role, _ := roleI.(string)
	if role != "admin" && role != "dosen_wali" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya admin atau dosen wali yang dapat melihat prestasi mahasiswa", "forbidden", nil))
		return
	}

	idStr := ctx.Param("id")
	studentID, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID mahasiswa tidak valid", err.Error(), nil))
		return
	}

	refs, err := s.achievementRepo.FindByStudentID(studentID.String())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil prestasi mahasiswa", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil prestasi mahasiswa", refs))
}

// ================================
// PUT /api/v1/students/:id/advisor
// Admin: mengubah dosen wali mahasiswa
// ================================
func (s *studentService) UpdateAdvisor(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	idStr := ctx.Param("id")
	studentID, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID mahasiswa tidak valid", err.Error(), nil))
		return
	}

	var body struct {
		AdvisorID string `json:"advisorId" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	advisorUUID, err := uuid.Parse(body.AdvisorID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID dosen wali tidak valid", err.Error(), nil))
		return
	}

	if err := s.studentRepo.UpdateAdvisor(studentID, advisorUUID); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal memperbarui dosen wali mahasiswa", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Dosen wali berhasil diperbarui", nil))
}
