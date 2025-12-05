package service

import (
	"net/http"
	"strings"

	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService mendefinisikan behavior untuk proses autentikasi (login, refresh, dll).
type AuthService interface {
	Login(ctx *gin.Context)         // POST /api/v1/auth/login
	RefreshToken(ctx *gin.Context)  // POST /api/v1/auth/refresh
	Logout(ctx *gin.Context)        // POST /api/v1/auth/logout
	GetProfile(ctx *gin.Context)    // GET  /api/v1/auth/profile
}

// authService adalah implementasi konkret AuthService.
type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService membuat instance baru authService dengan dependency UserRepository.
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo}
}

// ===============================================================
//      LOGIN — FR-001 (SRS)
//      Endpoint: POST /api/v1/auth/login
//      Deskripsi:
//        - Terima username/email + password
//        - Validasi kredensial & status aktif
//        - Generate JWT berisi userID, studentID (jika mahasiswa), role & permissions
//        - Return token, refreshToken, dan user profile
// ===============================================================
func (s *authService) Login(ctx *gin.Context) {
	// Struct input mengikuti SRS: field JSON "username" (bisa berisi username atau email).
	var input struct {
		Username string `json:"username" binding:"required"` // bisa username atau email
		Password string `json:"password" binding:"required"`
	}

	// Bind dan validasi body request.
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input login tidak valid", err.Error(), nil))
		return
	}

	// Cari user berdasarkan username atau email (sesuai FR-001).
	var (
		user *model.User
		err  error
	)

	// Jika mengandung "@", anggap sebagai email, selain itu username.
	if strings.Contains(input.Username, "@") {
		user, err = s.userRepo.FindByEmail(input.Username)
	} else {
		user, err = s.userRepo.FindByUsername(input.Username)
	}

	if err != nil {
		// Untuk keamanan, pesan tetap generic (tidak membocorkan mana yang salah).
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Username atau password salah", "invalid credentials", nil))
		return
	}

	// Cocokkan password plaintext dengan hash di database.
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Username atau password salah", "invalid credentials", nil))
		return
	}

	// Cek status aktif user (FR-001 step 3).
	if !user.IsActive {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Akun dinonaktifkan", "inactive account", nil))
		return
	}

	// Kumpulkan permission names dari role user (FR-001 step 4).
	var perms []string
	for _, p := range user.Role.Permissions {
		perms = append(perms, p.Name)
	}

	// Ambil StudentID jika role adalah mahasiswa, untuk disimpan di JWT.
	// Jika bukan mahasiswa, StudentID akan tetap uuid.Nil.
	var studentID uuid.UUID
	if user.Role.Name == "mahasiswa" {
		if stu, err := s.userRepo.FindStudentByUserID(user.ID); err == nil && stu != nil {
			studentID = stu.ID
		}
	}

	// Generate JWT access token (isi: userID, studentID, roleName, permissions).
	token, err := utils.GenerateToken(
		user.ID,       // userID
		studentID,     // studentID (uuid.Nil jika bukan mahasiswa)
		user.Role.Name, // roleName
		perms,         // permissions
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal membuat token", err.Error(), nil))
		return
	}

	// Untuk sementara, refreshToken disamakan dengan token.
	// TODO: jika nanti ada mekanisme refresh token terpisah, ubah ke generator khusus.
	refreshToken := token

	// Bentuk response sesuai contoh di SRS (token, refreshToken, user + permissions).
	data := map[string]any{
		"token":        token,
		"refreshToken": refreshToken,
		"user": map[string]any{
			"id":          user.ID,
			"username":    user.Username,
			"fullName":    user.FullName,
			"role":        user.Role.Name,
			"permissions": perms,
		},
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Login berhasil", data))
}
// RefreshToken memvalidasi refreshToken dan membuat access token baru.
func (s *authService) RefreshToken(ctx *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input refresh token tidak valid", err.Error(), nil))
		return
	}

	claims, err := utils.ValidateToken(input.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Refresh token tidak valid atau kedaluwarsa", err.Error(), nil))
		return
	}

	newAccessToken, err := utils.GenerateToken(
		claims.UserID,
		claims.StudentID,
		claims.Role,
		claims.Permissions,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal membuat token baru", err.Error(), nil))
		return
	}

	data := map[string]any{
		"token":        newAccessToken,
		"refreshToken": input.RefreshToken,
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Token berhasil diperbarui", data))
}

// Logout mengembalikan respon sukses (JWT tetap stateless).
func (s *authService) Logout(ctx *gin.Context) {
	// Implementasi stateless: client cukup menghapus token dari storage.
	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Logout berhasil, silakan hapus token di sisi client", nil))
}

// GetProfile mengembalikan profil user berdasarkan klaim JWT.
func (s *authService) GetProfile(ctx *gin.Context) {
	v, ok := ctx.Get("userID")
	if !ok {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("User belum terautentikasi", "no_user_id", nil))
		return
	}
	userID, ok := v.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		ctx.JSON(http.StatusUnauthorized,
			utils.BuildResponseFailed("Klaim token tidak valid", "invalid_user_id", nil))
		return
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("User tidak ditemukan", err.Error(), nil))
		return
	}

	var studentProfile any
	if user.Role.Name == "mahasiswa" {
		if sp, err := s.userRepo.FindStudentByUserID(user.ID); err == nil && sp != nil {
			studentProfile = map[string]any{
				"id":           sp.ID,
				"studentId":    sp.StudentID,
				"programStudy": sp.ProgramStudy,
				"academicYear": sp.AcademicYear,
			}
		}
	}

	var perms []string
	for _, p := range user.Role.Permissions {
		perms = append(perms, p.Name)
	}

	data := map[string]any{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"fullName":       user.FullName,
		"role":           user.Role.Name,
		"permissions":    perms,
		"studentProfile": studentProfile,
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil profil", data))
}
