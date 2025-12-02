package service

import (
	"context"
	"net/http"

	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReportService mendefinisikan behavior untuk fitur FR-011 (statistik prestasi).
type ReportService interface {
	// GetGlobalStatistics:
	// - Admin: semua prestasi
	// - Dosen Wali: prestasi semua mahasiswa bimbingan
	// - Mahasiswa: hanya prestasi sendiri
	GetGlobalStatistics(ctx *gin.Context)

	// GetStudentStatistics:
	// - Admin: boleh lihat statistik student manapun
	// - Dosen Wali: hanya student bimbingan
	// - Mahasiswa: hanya dirinya sendiri (id harus = claim.studentId)
	GetStudentStatistics(ctx *gin.Context)
}

// reportService implementasi konkrit ReportService.
type reportService struct {
	reportRepo   repository.ReportRepository
	lecturerRepo repository.LecturerRepository
}

// NewReportService membuat instance baru reportService.
func NewReportService(reportRepo repository.ReportRepository, lecturerRepo repository.LecturerRepository) ReportService {
	return &reportService{
		reportRepo:   reportRepo,
		lecturerRepo: lecturerRepo,
	}
}

// getUUIDFromContext membantu mengambil uuid.UUID dari gin.Context key tertentu.
func getUUIDFromContext(ctx *gin.Context, key string) (uuid.UUID, bool) {
	if v, ok := ctx.Get(key); ok {
		if id, ok2 := v.(uuid.UUID); ok2 {
			return id, true
		}
	}
	return uuid.Nil, false
}

// GetGlobalStatistics mengembalikan statistik prestasi sesuai role pemanggil.
// - Admin      → semua mahasiswa
// - Dosen Wali → hanya mahasiswa bimbingan
// - Mahasiswa  → hanya prestasi dirinya
func (s *reportService) GetGlobalStatistics(ctx *gin.Context) {
	role := ctx.GetString("role")

	filter := repository.ReportFilter{}

	switch role {
	case "admin":
		// admin: filter kosong → semua data (tidak perlu isi StudentIDs)

	case "dosen_wali":
		// dosen wali: statistik hanya untuk mahasiswa bimbingan
		userID, ok := getUUIDFromContext(ctx, "userID")
		if !ok || userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi dosen wali tidak valid", "no_user_id", nil))
			return
		}

		lecturer, err := s.lecturerRepo.FindByUserID(userID)
		if err != nil {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
			return
		}

		adviseeIDs, err := s.lecturerRepo.GetAdviseeStudentIDs(lecturer.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				utils.BuildResponseFailed("Gagal mengambil daftar mahasiswa bimbingan", err.Error(), nil))
			return
		}

		// Konversi []uuid.UUID → []string (UUID string)
		ids := make([]string, 0, len(adviseeIDs))
		for _, id := range adviseeIDs {
			ids = append(ids, id.String())
		}
		filter.StudentIDs = ids

	case "mahasiswa":
		// mahasiswa: statistik hanya miliknya sendiri
		studentID, ok := getUUIDFromContext(ctx, "studentID")
		if !ok || studentID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi mahasiswa tidak valid", "no_student_id", nil))
			return
		}
		filter.StudentIDs = []string{studentID.String()}

	default:
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Role tidak diizinkan mengakses statistik global", "forbidden_role", nil))
		return
	}

	stats, err := s.reportRepo.GetStatistics(context.Background(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menghitung statistik prestasi", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil statistik prestasi", stats))
}

// GetStudentStatistics mengembalikan statistik untuk 1 mahasiswa tertentu.
// - Admin: bebas student manapun
// - Dosen Wali: hanya advisee-nya
// - Mahasiswa: hanya dirinya sendiri (id = claim.studentId)
func (s *reportService) GetStudentStatistics(ctx *gin.Context) {
	role := ctx.GetString("role")
	idParam := ctx.Param("id")

	studentID, err := uuid.Parse(idParam)
	if err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("ID mahasiswa tidak valid", err.Error(), nil))
		return
	}

	// Role-based access control
	switch role {
	case "admin":
		// admin boleh lihat siapa saja

	case "dosen_wali":
		// pastikan student ini adalah advisee dosen wali tsb
		userID, ok := getUUIDFromContext(ctx, "userID")
		if !ok || userID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi dosen wali tidak valid", "no_user_id", nil))
			return
		}
		lecturer, err := s.lecturerRepo.FindByUserID(userID)
		if err != nil {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Data dosen wali tidak ditemukan", err.Error(), nil))
			return
		}

		isAdvisor, err := s.lecturerRepo.IsAdvisorOf(lecturer.ID, studentID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				utils.BuildResponseFailed("Gagal memeriksa relasi dosen wali", err.Error(), nil))
			return
		}
		if !isAdvisor {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Anda bukan dosen wali mahasiswa ini", "forbidden", nil))
			return
		}

	case "mahasiswa":
		// mahasiswa hanya boleh akses statistik dirinya sendiri
		claimStudentID, ok := getUUIDFromContext(ctx, "studentID")
		if !ok || claimStudentID == uuid.Nil {
			ctx.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Autentikasi mahasiswa tidak valid", "no_student_id", nil))
			return
		}
		if claimStudentID != studentID {
			ctx.JSON(http.StatusForbidden,
				utils.BuildResponseFailed("Anda tidak boleh melihat statistik mahasiswa lain", "forbidden", nil))
			return
		}

	default:
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Role tidak diizinkan mengakses statistik mahasiswa", "forbidden_role", nil))
		return
	}

	// Query statistik untuk 1 studentId
	filter := repository.ReportFilter{
		StudentIDs: []string{studentID.String()},
	}
	stats, err := s.reportRepo.GetStatistics(context.Background(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menghitung statistik prestasi mahasiswa", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil statistik prestasi mahasiswa", stats))
}
