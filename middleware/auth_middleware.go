package middleware

import (
	"net/http"
	"strings"

	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware memvalidasi JWT dari header Authorization (Bearer token)
// dan menyimpan informasi user (userID, studentID, role, permissions) ke dalam context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ambil header Authorization
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Authorization token required", "missing_or_invalid_authorization_header", nil))
			c.Abort()
			return
		}

		// Ambil token string dan trim spasi sisa
		tokenString := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Authorization token required", "empty_token", nil))
			c.Abort()
			return
		}

		// Validasi token menggunakan utils (JWT parsing & verifikasi signature/expired)
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized,
				utils.BuildResponseFailed("Invalid or expired token", err.Error(), nil))
			c.Abort()
			return
		}

		// Inject nilai-nilai penting ke context untuk dipakai di handler/service
		c.Set("userID", claims.UserID)       // UUID user (tabel users)
		c.Set("studentID", claims.StudentID) // UUID student (tabel students) - bisa uuid.Nil jika bukan mahasiswa
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)

		// lanjut ke handler berikutnya
		c.Next()
	}
}
