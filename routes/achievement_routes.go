package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// AchievementRoutes mendaftarkan semua endpoint prestasi (FR-003 s.d. FR-010)
func AchievementRoutes(r *gin.Engine, s service.AchievementService) {

	// Semua endpoint di bawah ini butuh JWT
	g := r.Group("/api/v1/achievements")
	g.Use(middleware.AuthMiddleware())

	{
		// -----------------------------------------------------------
		// FR-003: Mahasiswa membuat prestasi (status draft)
		// POST /api/v1/achievements
		// -----------------------------------------------------------
		g.POST("/", s.CreateAchievement)

		// -----------------------------------------------------------
		// FR-004: Mahasiswa submit prestasi draft
		// POST /api/v1/achievements/:id/submit
		// -----------------------------------------------------------
		g.POST("/:id/submit", s.SubmitForVerification)

		// -----------------------------------------------------------
		// FR-005: Mahasiswa menghapus draft prestasi
		// DELETE /api/v1/achievements/:id
		// -----------------------------------------------------------
		g.DELETE("/:id", s.DeleteAchievement)

		// -----------------------------------------------------------
		// FR-006, FR-007, FR-008, FR-010:
		// GET /api/v1/achievements
		//
		// Behavior:
		// - Mahasiswa → list prestasi miliknya
		// - Dosen wali → list prestasi semua mahasiswa bimbingan
		// - Admin      → list semua prestasi (with status filter + pagination)
		// -----------------------------------------------------------
		g.GET("/", s.GetAchievements)

		// -----------------------------------------------------------
		// FR-007: Dosen wali memverifikasi prestasi mahasiswa
		// POST /api/v1/achievements/:id/verify
		// -----------------------------------------------------------
		g.POST("/:id/verify", s.VerifyAchievement)

		// -----------------------------------------------------------
		// FR-008: Dosen wali menolak prestasi mahasiswa
		// POST /api/v1/achievements/:id/reject
		// -----------------------------------------------------------
		g.POST("/:id/reject", s.RejectAchievement)
	}
}
