package routes

import (
	"student-achievement-backend/app/service"
	"student-achievement-backend/middleware"

	"github.com/gin-gonic/gin"
)

// AuthRoutes mendaftarkan seluruh endpoint /api/v1/auth sesuai SRS.
func AuthRoutes(r *gin.Engine, s service.AuthService) {
	g := r.Group("/api/v1/auth")

	// Endpoint yang tidak membutuhkan JWT.
	g.POST("/login", s.Login)
	g.POST("/refresh", s.RefreshToken)
	g.POST("/logout", s.Logout)

	// Endpoint yang membutuhkan JWT.
	g.GET("/profile", middleware.AuthMiddleware(), s.GetProfile)
}
