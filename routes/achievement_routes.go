package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// AchievementRoutes mendaftarkan semua endpoint terkait prestasi (FR-003 s.d. FR-007).
func AchievementRoutes(r *gin.Engine, s service.AchievementService) {
	// Group dengan AuthMiddleware → semua endpoint butuh JWT.
	auth := r.Group("/api/v1/achievements")
	auth.Use(middleware.AuthMiddleware())
	{
		// FR-003: Create achievement (Mahasiswa)
		auth.POST("/", s.CreateAchievement)

		// FR-004: Submit draft for verification (Mahasiswa)
		auth.POST("/:id/submit", s.SubmitForVerification)

		// FR-005: Delete draft (Mahasiswa)
		auth.DELETE("/:id", s.DeleteAchievement)

		// FR-006: List achievements
		// - Mahasiswa: prestasi sendiri
		// - Dosen wali: prestasi mahasiswa bimbingan
		auth.GET("/", s.GetAchievementsByStudent)

		// FR-007: Verify achievement (Dosen Wali)
		// Sesuai SRS: POST /api/v1/achievements/:id/verify
		auth.POST("/:id/verify", s.VerifyAchievement)
	}
}
