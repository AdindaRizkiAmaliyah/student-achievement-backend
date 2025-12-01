package repository

import (
	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LecturerRepository mendefinisikan operasi ke tabel lecturers.
type LecturerRepository interface {
	// FindByUserID mencari dosen berdasarkan user_id (user yang login).
	FindByUserID(userID uuid.UUID) (*model.Lecturer, error)
}

// lecturerRepository adalah implementasi konkret LecturerRepository.
type lecturerRepository struct {
	db *gorm.DB
}

// NewLecturerRepository membuat instance baru LecturerRepository.
func NewLecturerRepository(db *gorm.DB) LecturerRepository {
	return &lecturerRepository{db: db}
}

// FindByUserID mencari baris Lecturer yang terkait dengan user_id tertentu.
func (r *lecturerRepository) FindByUserID(userID uuid.UUID) (*model.Lecturer, error) {
	var lec model.Lecturer
	if err := r.db.
		Preload("User"). // preload user dosen (optional, tapi berguna kalau nanti perlu)
		Where("user_id = ?", userID).
		First(&lec).Error; err != nil {
		return nil, err
	}
	return &lec, nil
}
