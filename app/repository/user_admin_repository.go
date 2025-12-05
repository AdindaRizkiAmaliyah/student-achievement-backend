package repository

import (
	"student-achievement-backend/app/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserAdminRepository: khusus untuk fitur admin (FR-009)
type UserAdminRepository interface {
	CreateUser(user *model.User) error
	UpdateUser(user *model.User) error
	FindAllUsers() ([]model.User, error)
	FindUserByID(id uuid.UUID) (*model.User, error)
	SoftDeleteUser(id uuid.UUID) error
	UpdateUserRole(id uuid.UUID, roleID uuid.UUID) error

	CreateStudentProfile(s *model.Student) error
	CreateLecturerProfile(l *model.Lecturer) error

	// ❌ SetStudentAdvisor dihapus karena sekarang ada di StudentService + StudentRepository
}

type userAdminRepository struct {
	db *gorm.DB
}

func NewUserAdminRepository(db *gorm.DB) UserAdminRepository {
	return &userAdminRepository{db}
}

// CreateUser → FR-009: admin membuat user baru
func (r *userAdminRepository) CreateUser(user *model.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return r.db.Create(user).Error
}

// UpdateUser → FR-009: admin edit data user
func (r *userAdminRepository) UpdateUser(user *model.User) error {
	user.UpdatedAt = time.Now()
	return r.db.Save(user).Error
}

// FindAllUsers → list semua user
func (r *userAdminRepository) FindAllUsers() ([]model.User, error) {
	var users []model.User
	err := r.db.Preload("Role").Find(&users).Error
	return users, err
}

// FindUserByID → ambil detail user
func (r *userAdminRepository) FindUserByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.Preload("Role").First(&user, "id = ?", id).Error
	return &user, err
}

// SoftDeleteUser → nonaktifkan user (IsActive = false)
func (r *userAdminRepository) SoftDeleteUser(id uuid.UUID) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		Update("is_active", false).Error
}

// UpdateUserRole → ganti role user
func (r *userAdminRepository) UpdateUserRole(id uuid.UUID, roleID uuid.UUID) error {
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		Update("role_id", roleID).Error
}

// CreateStudentProfile → buat profil mahasiswa (NIM, Prodi, dst)
func (r *userAdminRepository) CreateStudentProfile(s *model.Student) error {
	return r.db.Create(s).Error
}

// CreateLecturerProfile → buat profil dosen wali
func (r *userAdminRepository) CreateLecturerProfile(l *model.Lecturer) error {
	return r.db.Create(l).Error
}
