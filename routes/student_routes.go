package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// StudentRoutes mendaftarkan endpoint SRS 5.5 Students:
// GET /api/v1/students
// GET /api/v1/students/:id
// GET /api/v1/students/:id/achievements
// PUT /api/v1/students/:id/advisor
func StudentRoutes(r *gin.Engine, s service.StudentService) {
	g := r.Group("/api/v1/students")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("/", s.GetStudents)
		g.GET("/:id", s.GetStudentDetail)
		g.GET("/:id/achievements", s.GetStudentAchievements)
		g.PUT("/:id/advisor", s.UpdateAdvisor)
	}
}
