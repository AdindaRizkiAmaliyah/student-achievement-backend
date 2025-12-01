package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// AchievementRoutes mendaftarkan endpoint terkait prestasi mahasiswa.
// Semua endpoint di sini wajib melalui AuthMiddleware (hanya user terautentikasi).
func AchievementRoutes(r *gin.Engine, s service.AchievementService) {
	// Sesuai SRS: prefix /api/v1/achievements
	auth := r.Group("/api/v1/achievements")
	auth.Use(middleware.AuthMiddleware())
	{
		// FR-03: Create achievement (mahasiswa buat prestasi baru, status draft)
		auth.POST("/", s.CreateAchievement)

		// FR-04: Submit achievement for verification
		auth.POST("/:id/submit", s.SubmitForVerification)

		// FR-05: Delete achievement (soft delete, hanya status draft)
		auth.DELETE("/:id", s.DeleteAchievement)

		// FR-06: List achievements milik mahasiswa yang login
		auth.GET("/", s.GetAchievementsByStudent)
	}
}
