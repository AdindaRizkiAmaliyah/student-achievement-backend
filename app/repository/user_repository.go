package repository

import (
	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository mendefinisikan kontrak operasi database untuk entity User.
type UserRepository interface {
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	FindByID(id uuid.UUID) (*model.User, error)
	FindStudentByUserID(userID uuid.UUID) (*model.Student, error)
}

// userRepository adalah implementasi konkret UserRepository berbasis GORM.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat instance baru userRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db}
}

// Create menyimpan data user baru ke database.
func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// FindByEmail mencari user berdasarkan email (digunakan saat login dengan email).
func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername mencari user berdasarkan username (digunakan saat login dengan username).
func (r *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID mengambil user berdasarkan ID (dipakai misalnya untuk endpoint profile).
func (r *userRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindStudentByUserID mencari data mahasiswa yang terhubung ke user tertentu.
func (r *userRepository) FindStudentByUserID(userID uuid.UUID) (*model.Student, error) {
	var s model.Student
	err := r.db.Where("user_id = ?", userID).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}
