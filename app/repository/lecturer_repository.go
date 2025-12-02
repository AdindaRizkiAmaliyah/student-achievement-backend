package repository

import (
	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LecturerRepository mendefinisikan operasi ke tabel lecturers & relasi bimbingan.
type LecturerRepository interface {
	// FindByUserID mencari dosen berdasarkan user_id (user yang login).
	FindByUserID(userID uuid.UUID) (*model.Lecturer, error)

	// IsAdvisorOf mengecek apakah lecturerID adalah dosen wali dari studentID tertentu.
	// Digunakan saat dosen wali memverifikasi prestasi mahasiswa (FR-007).
	IsAdvisorOf(lecturerID uuid.UUID, studentID uuid.UUID) (bool, error)
}

// lecturerRepository adalah implementasi konkret LecturerRepository.
type lecturerRepository struct {
	db *gorm.DB
}

// NewLecturerRepository membuat instance baru LecturerRepository.
func NewLecturerRepository(db *gorm.DB) LecturerRepository {
	return &lecturerRepository{db: db}
}

// FindByUserID mencari data Lecturer yang terkait dengan user_id tertentu.
// Digunakan untuk mengidentifikasi dosen wali dari token (userID) saat dosen login.
func (r *lecturerRepository) FindByUserID(userID uuid.UUID) (*model.Lecturer, error) {
	var lec model.Lecturer
	if err := r.db.
		Preload("User").     // preload data user dosen (opsional, tapi aman)
		Preload("Advisees"). // preload mahasiswa bimbingannya (opsional)
		Where("user_id = ?", userID).
		First(&lec).Error; err != nil {
		return nil, err
	}
	return &lec, nil
}

// IsAdvisorOf mengecek apakah lecturerID adalah dosen wali dari studentID tertentu.
// Implementasi: cek di tabel students, apakah ada baris dengan:
//   id = studentID dan advisor_id = lecturerID
func (r *lecturerRepository) IsAdvisorOf(lecturerID uuid.UUID, studentID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.
		Model(&model.Student{}).
		Where("id = ? AND advisor_id = ?", studentID, lecturerID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
