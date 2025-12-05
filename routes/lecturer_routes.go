package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// LecturerRoutes mendaftarkan endpoint SRS 5.5 Lecturers:
// GET /api/v1/lecturers
// GET /api/v1/lecturers/:id/advisees
func LecturerRoutes(r *gin.Engine, s service.LecturerService) {
	g := r.Group("/api/v1/lecturers")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("/", s.GetLecturers)
		g.GET("/:id/advisees", s.GetLecturerAdvisees)
	}
}
