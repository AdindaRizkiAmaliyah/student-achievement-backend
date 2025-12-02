package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// ReportRoutes mendaftarkan endpoint FR-011 (Reports & Analytics).
func ReportRoutes(r *gin.Engine, s service.ReportService) {

	g := r.Group("/api/v1/reports")
	g.Use(middleware.AuthMiddleware())

	{
		// FR-011 - Global statistics (scope tergantung role)
		// Admin      → semua prestasi
		// Dosen Wali → semua mahasiswa bimbingan
		// Mahasiswa  → prestasi sendiri
		// GET /api/v1/reports/statistics
		g.GET("/statistics", s.GetGlobalStatistics)

		// FR-011 - Student statistics (1 mahasiswa)
		// Admin      → boleh siapa saja
		// Dosen Wali → hanya advisee
		// Mahasiswa  → hanya dirinya sendiri
		// GET /api/v1/reports/student/:id
		g.GET("/student/:id", s.GetStudentStatistics)
	}
}
