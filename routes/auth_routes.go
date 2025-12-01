package routes

import (
	"student-achievement-backend/app/service"

	"github.com/gin-gonic/gin"
)

// AuthRoutes mendaftarkan semua endpoint autentikasi ke router utama.
func AuthRoutes(r *gin.Engine, s service.AuthService) {
	g := r.Group("/api/v1/auth")

	// FR-001: Login
	// Endpoint harus sama dengan SRS: POST /api/v1/auth/login
	g.POST("/login", s.Login)

	// Catatan:
	// - Endpoint lain di SRS: /refresh, /logout, /profile
	//   bisa ditambahkan nanti ketika fitur tersebut diimplementasikan.
}
