package middleware

import (
	"net/http"
	"strings"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware adalah "Satpam" yang mengecek Token JWT di setiap request.
// Sesuai FR-002
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 1. Ambil Header "Authorization" dari request
		// Format yang benar: "Bearer <token_panjang_disini>"
		authHeader := ctx.GetHeader("Authorization")
		
		// 2. Cek apakah header ada
		if authHeader == "" {
			resp := utils.BuildResponseFailed("Akses ditolak", "Header Authorization tidak ditemukan", nil)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, resp)
			return
		}

		// 3. Pecah string untuk mengambil tokennya saja (hapus kata "Bearer ")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			resp := utils.BuildResponseFailed("Akses ditolak", "Format token salah (harus Bearer <token>)", nil)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, resp)
			return
		}

		tokenString := parts[1]

		// 4. Validasi Token menggunakan Utils yang sudah kita buat
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			// Jika token expired atau palsu
			resp := utils.BuildResponseFailed("Token tidak valid", err.Error(), nil)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, resp)
			return
		}

		// 5. Simpan data user ke dalam Context (Penyimpanan sementara Gin)
		// Agar nanti di Controller (Handler), kita bisa panggil: ctx.MustGet("userID")
		ctx.Set("userID", claims.UserID)
		ctx.Set("role", claims.Role)
		ctx.Set("permissions", claims.Permissions)

		// 6. Lanjut ke handler berikutnya (Controller)
		ctx.Next()
	}
}

// PermissionMiddleware (Opsional/Lanjutan) untuk mengecek hak akses spesifik RBAC
// Contoh: RequirePermission("achievement:verify") untuk route dosen
func PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Ambil permission user dari context (yang diset oleh AuthMiddleware di atas)
		permissions, exists := ctx.Get("permissions")
		if !exists {
			resp := utils.BuildResponseFailed("Forbidden", "User permissions not found", nil)
			ctx.AbortWithStatusJSON(http.StatusForbidden, resp)
			return
		}

		// Cek apakah user punya izin yang diminta
		userPerms := permissions.([]string)
		hasPermission := false
		for _, p := range userPerms {
			if p == requiredPermission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			resp := utils.BuildResponseFailed("Forbidden", "Anda tidak memiliki izin akses ini", nil)
			ctx.AbortWithStatusJSON(http.StatusForbidden, resp)
			return
		}

		ctx.Next()
	}
}