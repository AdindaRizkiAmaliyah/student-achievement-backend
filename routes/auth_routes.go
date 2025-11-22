package routes

import (
	"net/http"
	"student-achievement-backend/app/model"
	"student-achievement-backend/app/service"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler adalah struct pengelola request untuk fitur Autentikasi.
// Struct ini menyimpan dependency ke AuthService.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler adalah constructor untuk membuat instance handler baru.
// Dipanggil di main.go nanti untuk menyambungkan Service ke Handler ini.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// SetupAuthRoutes mengatur Peta URL (Routing).
// Di sini kita tentukan endpoint mana lari ke fungsi mana.
func (h *AuthHandler) SetupAuthRoutes(r *gin.Engine) {
	// Kita buat grup URL supaya rapi, diawali /api/v1/auth
	authGroup := r.Group("/api/v1/auth")
	{
		// Jika ada request POST ke /register, jalankan fungsi h.Register
		authGroup.POST("/register", h.Register)
		// Jika ada request POST ke /login, jalankan fungsi h.Login (FR-001)
		authGroup.POST("/login", h.Login)
	}
}

// ==================================================================
// LOGIKA HANDLER (CONTROLLER) DIMULAI DI SINI
// ==================================================================

// Register menangani pendaftaran user baru.
func (h *AuthHandler) Register(ctx *gin.Context) {
	// 1. Siapkan DTO (Data Transfer Object).
	// Struct ini cuma wadah sementara untuk menampung JSON dari Frontend/Postman.
	var input struct {
		Username string `json:"username" binding:"required"`        // Wajib ada
		Email    string `json:"email" binding:"required,email"`     // Wajib format email
		Password string `json:"password" binding:"required,min=6"`  // Minimal 6 karakter
		FullName string `json:"fullName" binding:"required"`
		RoleID   string `json:"roleId" binding:"required"`          // ID Role dikirim sebagai string
	}

	// 2. Binding & Validasi Input.
	// Gin otomatis mengecek apakah JSON sesuai dengan aturan 'binding' di atas.
	if err := ctx.ShouldBindJSON(&input); err != nil {
		// Jika salah, kirim respon Error 400 (Bad Request).
		resp := utils.BuildResponseFailed("Input tidak valid", err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	// 3. Konversi RoleID dari String ke UUID.
	// Karena di database tipe datanya UUID, kita harus ubah dulu.
	roleUUID, err := uuid.Parse(input.RoleID)
	if err != nil {
		resp := utils.BuildResponseFailed("Format Role ID salah (harus UUID)", err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	// 4. Pindahkan data dari input (DTO) ke Model Asli (User).
	newUser := model.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: input.Password, // Password masih mentah, nanti di-hash di Service
		FullName:     input.FullName,
		RoleID:       roleUUID,
		IsActive:     true,           // Default user aktif
	}

	// 5. Panggil Service "Register" untuk menyimpan ke database.
	// Di sinilah logika hashing password terjadi.
	err = h.authService.Register(&newUser)
	if err != nil {
		// Jika gagal simpan (misal email duplikat), kirim Error 500.
		resp := utils.BuildResponseFailed("Gagal registrasi", err.Error(), nil)
		ctx.JSON(http.StatusInternalServerError, resp)
		return
	}

	// 6. Sukses! Kirim respon 201 (Created).
	resp := utils.BuildResponseSuccess("Registrasi berhasil", nil)
	ctx.JSON(http.StatusCreated, resp)
}

// Login menangani proses masuk user sesuai SRS FR-001 [cite: 161-168].
func (h *AuthHandler) Login(ctx *gin.Context) {
	// 1. Siapkan wadah input login.
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	// 2. Validasi input JSON.
	if err := ctx.ShouldBindJSON(&input); err != nil {
		resp := utils.BuildResponseFailed("Input login tidak valid", err.Error(), nil)
		ctx.JSON(http.StatusBadRequest, resp)
		return
	}

	// 3. Panggil Service Login.
	// Service akan mengecek apakah email ada dan password hash-nya cocok.
	user, err := h.authService.Login(input.Email, input.Password)
	if err != nil {
		// Jika gagal (Password salah / User gak ketemu), kirim Error 401 (Unauthorized).
		resp := utils.BuildResponseFailed("Login gagal", err.Error(), nil)
		ctx.JSON(http.StatusUnauthorized, resp)
		return
	}

	// 4. Siapkan Data Permission untuk Token.
	// Kita ambil list izin (misal: 'achievement:create') dari user yang login.
	var permissions []string
	for _, p := range user.Role.Permissions {
		permissions = append(permissions, p.Name)
	}

	// 5. Generate Token JWT (Karcis Masuk).
	// Kita panggil Utils yang sudah kita buat sebelumnya.
	token, err := utils.GenerateToken(user.ID, user.Role.Name, permissions)
	if err != nil {
		resp := utils.BuildResponseFailed("Gagal membuat token", err.Error(), nil)
		ctx.JSON(http.StatusInternalServerError, resp)
		return
	}

	// 6. Siapkan data respons agar Frontend senang.
	// Kirim Token + Info User dasar.
	data := map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"fullName": user.FullName,
			"role":     user.Role.Name,
		},
	}

	// 7. Kirim Respon Sukses 200 (OK).
	resp := utils.BuildResponseSuccess("Login berhasil", data)
	ctx.JSON(http.StatusOK, resp)
}