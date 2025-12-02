package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

func AdminRoutes(r *gin.Engine, s service.AdminService) {

	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware()) // wajib JWT
	{
		admin.GET("/users", s.GetAllUsers)
		admin.GET("/users/:id", s.GetUserDetail)
		admin.POST("/users", s.CreateUser)
		admin.PUT("/users/:id", s.UpdateUser)
		admin.DELETE("/users/:id", s.DeleteUser)
		admin.PUT("/users/:id/role", s.UpdateUserRole)

		admin.PUT("/students/:id/advisor", s.SetStudentAdvisor)
	}
}
