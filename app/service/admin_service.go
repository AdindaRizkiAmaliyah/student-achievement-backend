package service

import (
	"net/http"
	"time"

	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"student-achievement-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AdminService interface {
	CreateUser(ctx *gin.Context)
	UpdateUser(ctx *gin.Context)
	DeleteUser(ctx *gin.Context)
	GetAllUsers(ctx *gin.Context)
	GetUserDetail(ctx *gin.Context)
	UpdateUserRole(ctx *gin.Context)
	// ❌ SetStudentAdvisor dihapus — sekarang dihandle oleh StudentService (PUT /api/v1/students/:id/advisor)
}

type adminService struct {
	repo repository.UserAdminRepository
}

func NewAdminService(repo repository.UserAdminRepository) AdminService {
	return &adminService{repo}
}

// helper: cek admin
func ensureAdmin(ctx *gin.Context) bool {
	roleI, _ := ctx.Get("role")
	if role, _ := roleI.(string); role != "admin" {
		ctx.JSON(http.StatusForbidden,
			utils.BuildResponseFailed("Hanya admin yang dapat mengakses fitur ini", "forbidden", nil))
		return false
	}
	return true
}

// FR-009: Create User
func (s *adminService) CreateUser(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	var input struct {
		Username       string `json:"username" binding:"required"`
		Email          string `json:"email" binding:"required"`
		Password       string `json:"password" binding:"required"`
		FullName       string `json:"fullName" binding:"required"`
		RoleID         string `json:"roleId" binding:"required"`
		StudentProfile *struct {
			StudentID    string `json:"studentId"`
			ProgramStudy string `json:"programStudy"`
			AcademicYear string `json:"academicYear"`
		} `json:"studentProfile"`
		LecturerProfile *struct {
			LecturerID string `json:"lecturerId"`
			Department string `json:"department"`
		} `json:"lecturerProfile"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 10)

	user := model.User{
		ID:           uuid.New(),
		Username:     input.Username,
		Email:        input.Email,
		FullName:     input.FullName,
		PasswordHash: string(hash),
		RoleID:       uuid.MustParse(input.RoleID),
		IsActive:     true,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(&user); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal membuat user", err.Error(), nil))
		return
	}

	// Jika role mahasiswa → buat profile student
	if input.StudentProfile != nil {
		sp := model.Student{
			ID:           uuid.New(),
			UserID:       user.ID,
			StudentID:    input.StudentProfile.StudentID, // NIM
			ProgramStudy: input.StudentProfile.ProgramStudy,
			AcademicYear: input.StudentProfile.AcademicYear,
		}
		_ = s.repo.CreateStudentProfile(&sp)
	}

	// Jika role dosen wali → create profile lecturer
	if input.LecturerProfile != nil {
		lp := model.Lecturer{
			ID:         uuid.New(),
			UserID:     user.ID,
			LecturerID: input.LecturerProfile.LecturerID,
			Department: input.LecturerProfile.Department,
		}
		_ = s.repo.CreateLecturerProfile(&lp)
	}

	ctx.JSON(http.StatusCreated,
		utils.BuildResponseSuccess("User berhasil dibuat", user))
}

// FR-009: update user
func (s *adminService) UpdateUser(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	id := ctx.Param("id")
	uid := uuid.MustParse(id)

	user, err := s.repo.FindUserByID(uid)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("User tidak ditemukan", err.Error(), nil))
		return
	}

	var input struct {
		FullName string `json:"fullName"`
		Email    string `json:"email"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	if input.FullName != "" {
		user.FullName = input.FullName
	}
	if input.Email != "" {
		user.Email = input.Email
	}

	_ = s.repo.UpdateUser(user)

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("User berhasil diperbarui", user))
}

// FR-009: Soft delete
func (s *adminService) DeleteUser(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	id := ctx.Param("id")
	uid := uuid.MustParse(id)

	if err := s.repo.SoftDeleteUser(uid); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal menghapus user", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("User berhasil di-nonaktifkan", nil))
}

// FR-009: List users
func (s *adminService) GetAllUsers(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	users, err := s.repo.FindAllUsers()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal mengambil user", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil semua user", users))
}

// FR-009: Detail user
func (s *adminService) GetUserDetail(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	id := ctx.Param("id")
	userID := uuid.MustParse(id)

	u, err := s.repo.FindUserByID(userID)
	if err != nil {
		ctx.JSON(http.StatusNotFound,
			utils.BuildResponseFailed("User tidak ditemukan", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Berhasil mengambil detail user", u))
}

// FR-009: Update role user
func (s *adminService) UpdateUserRole(ctx *gin.Context) {

	if !ensureAdmin(ctx) {
		return
	}

	id := ctx.Param("id")
	uid := uuid.MustParse(id)

	var input struct {
		RoleID string `json:"roleId" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest,
			utils.BuildResponseFailed("Input tidak valid", err.Error(), nil))
		return
	}

	rid := uuid.MustParse(input.RoleID)

	if err := s.repo.UpdateUserRole(uid, rid); err != nil {
		ctx.JSON(http.StatusInternalServerError,
			utils.BuildResponseFailed("Gagal update role", err.Error(), nil))
		return
	}

	ctx.JSON(http.StatusOK,
		utils.BuildResponseSuccess("Role user berhasil diperbarui", nil))
}
