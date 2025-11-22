package routes

import (
	"net/http"
	"student-achievement-backend/app/model"
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"
	"student-achievement-backend/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AchievementHandler bertugas mengelola request terkait prestasi.
type AchievementHandler struct {
	achievementService service.AchievementService
}

// NewAchievementHandler membuat instance handler baru.
func NewAchievementHandler(achievementService service.AchievementService) *AchievementHandler {
	return &AchievementHandler{achievementService: achievementService}
}

// SetupAchievementRoutes mengatur URL endpoint untuk fitur prestasi.
func (h *AchievementHandler) SetupAchievementRoutes(r *gin.Engine) {
	// 1. Buat grup URL /api/v1/achievements
	achievementGroup := r.Group("/api/v1/achievements")

	// 2. PASANG SATPAM (Middleware) DI SINI!
	// Artinya: Semua URL di dalam grup ini WAJIB login dulu.
	// Middleware akan mengecek token JWT valid atau tidak.
	achievementGroup.Use(middleware.AuthMiddleware())

	{
		// FR-003: Endpoint untuk mahasiswa melapor prestasi baru
		// URL: POST /api/v1/achievements
		// Kita bisa tambah PermissionMiddleware jika ingin lebih spesifik, misal:
		// achievementGroup.POST("/", middleware.PermissionMiddleware("achievement:create"), h.Create)
		achievementGroup.POST("/", h.Create)
	}
}

// ==================================================================
// LOGIKA HANDLER (CONTROLLER)
// ==================================================================

// Create menangani submit prestasi dari Mahasiswa.
func (h *AchievementHandler) Create(ctx *gin.Context) {
	// 1. Ambil UserID dari Context (hasil kerja Middleware Auth tadi).
	// Middleware sudah menyimpan "userID" saat validasi token.
	userIDInterface, exists := ctx.Get("userID")
	if !exists {
		resp := utils.BuildResponseFailed("Unauthorized", "User ID tidak ditemukan", nil)
		ctx.JSON(http.StatusUnauthorized, resp)
		return
	}
	userID := userIDInterface.(uuid.UUID) // Konversi ke tipe UUID

	// 2. Siapkan DTO (Data Transfer Object) untuk menerima JSON.
	// Kita buat struct input yang mencakup semua kemungkinan field (Kompetisi, Organisasi, dll).
	var input struct {
		AchievementType string `json:"achievementType" binding:"required"` // academic, competition, dll
		Title           string `json:"title" binding:"required"`
		Description     string `json:"description"`
		
		// Field Detail Dinamis (Nested Object)
		Details struct {
			// Competition
			CompetitionName  string `json:"competitionName"`
			CompetitionLevel string `json:"competitionLevel"`
			Rank             int    `json:"rank"`
			MedalType        string `json:"medalType"`

			// Organization
			OrganizationName string    `json:"organizationName"`
			Position         string    `json:"position"`
			StartDate        time.Time `json:"startDate"`
			EndDate          time.Time `json:"endDate"`

			// Common
			EventDate *time.Time `json:"eventDate"`
			Location  string     `json:"location"`
			Organizer string     `json:"organizer"`
		} `json:"details"`

		// File bukti (nanti diupload terpisah atau berupa URL dulu)
		// Untuk sekarang kita terima URL dummy dulu sesuai SRS field attachments
	}

	// 3. Validasi Input JSON
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp := utils.BuildResponseFailed("Input prestasi tidak valid", err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	// 4. Mapping Data ke Model PostgreSQL (References)
	// Ingat, StudentID di tabel students adalah VARCHAR(NIM), tapi di tabel references
	// field student_id merujuk ke UUID tabel students.
	// *CATATAN*: Karena di Token JWT kita cuma punya UserID (bukan StudentID tabel students),
	// idealnya kita harus cari data Student berdasarkan UserID dulu.
	// TAPI, untuk simplifikasi tugas ini, kita akan kirim UserID sebagai referensi sementara
	// atau asumsikan frontend mengirim studentId (kurang aman).
	// *SOLUSI CLEAN*: Service harus mencari StudentID berdasarkan UserID.
	// Untuk tahap ini, kita kirim UserID ke service biar service yang urus.

	// Siapkan data PostgreSQL
	pgData := model.AchievementReference{
		// StudentID akan diurus di Service (karena butuh query ke tabel students)
		// Kita kosongkan dulu atau pakai UserID sementara jika struct mengizinkan
		Status: "draft", // Status awal selalu draft [cite: 185]
	}

	// 5. Mapping Data ke Model MongoDB (Details)
	mongoData := model.Achievement{
		AchievementType: input.AchievementType,
		Title:           input.Title,
		Description:     input.Description,
		Details: model.AchievementDetails{
			CompetitionName:  input.Details.CompetitionName,
			CompetitionLevel: input.Details.CompetitionLevel,
			Rank:             input.Details.Rank,
			MedalType:        input.Details.MedalType,
			OrganizationName: input.Details.OrganizationName,
			Position:         input.Details.Position,
			EventDate:        input.Details.EventDate,
			Location:         input.Details.Location,
			Organizer:        input.Details.Organizer,
		},
		// Attachments, Tags, dll bisa ditambahkan di sini
	}

	// 6. Panggil Service
	// Kita kirim UserID (string) agar Service bisa mencari data Student terkait.
	err := h.achievementService.SubmitAchievement(ctx, userID.String(), &pgData, &mongoData)
	if err != nil {
		resp := utils.BuildResponseFailed("Gagal menyimpan prestasi", err.Error(), nil)
		ctx.JSON(http.StatusInternalServerError, resp)
		return
	}

	// 7. Sukses
	resp := utils.BuildResponseSuccess("Prestasi berhasil disimpan sebagai draft", nil)
	ctx.JSON(http.StatusCreated, resp)
}