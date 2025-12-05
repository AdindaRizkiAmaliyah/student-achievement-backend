package service

import (
	"net/http"

	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LecturerService meng-handle endpoint SRS 5.5 untuk Lecturers:
// GET /lecturers
// GET /lecturers/:id/advisees
type LecturerService interface {
	GetLecturers(ctx *gin.Context)
	GetLecturerAdvisees(ctx *gin.Context)
}

type lecturerService struct {
	lecturerRepo repository.LecturerRepository
}

func NewLecturerService(lecturerRepo repository.LecturerRepository) LecturerService {
	return &lecturerService{lecturerRepo}
}

// =======================
// GET /api/v1/lecturers
// =======================
func (s *lecturerService) GetLecturers(ctx *gin.Context) {

	// Misal: hanya admin yang boleh melihat semua dosen.
	roleI, _ := ctx.Get("role")
	if role, _ := roleI.(string); role != "admin" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya admin yang dapat melihat daftar dosen", "forbidden", nil))
		return
	}

	lects, err := s.lecturerRepo.FindAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil daftar dosen", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil daftar dosen", lects))
}

// =================================
// GET /api/v1/lecturers/:id/advisees
// =================================
func (s *lecturerService) GetLecturerAdvisees(ctx *gin.Context) {

	roleI, _ := ctx.Get("role")
	role, _ := roleI.(string)
	if role != "admin" && role != "dosen_wali" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya admin atau dosen wali yang dapat melihat mahasiswa bimbingan", "forbidden", nil))
		return
	}

	idStr := ctx.Param("id")
	lectID, err := uuid.Parse(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID dosen tidak valid", err.Error(), nil))
		return
	}

	// Pastikan dosen ada (opsional, tapi bagus punya)
	if _, err := s.lecturerRepo.FindByID(lectID); err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("Dosen tidak ditemukan", err.Error(), nil))
		return
	}

	students, err := s.lecturerRepo.FindAdvisees(lectID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil mahasiswa bimbingan", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil daftar mahasiswa bimbingan", students))
}
