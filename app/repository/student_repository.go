package repository

import (
	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StudentRepository menangani operasi basis data untuk entity Student
// Digunakan di SRS 5.5 Students & Lecturers.
type StudentRepository interface {
	FindAll() ([]model.Student, error)                 // GET /students
	FindByID(id uuid.UUID) (*model.Student, error)     // GET /students/:id
	UpdateAdvisor(studentID, advisorID uuid.UUID) error // PUT /students/:id/advisor
}

type studentRepository struct {
	db *gorm.DB
}

func NewStudentRepository(db *gorm.DB) StudentRepository {
	return &studentRepository{db}
}

// FindAll mengembalikan semua mahasiswa.
func (r *studentRepository) FindAll() ([]model.Student, error) {
	var students []model.Student
	err := r.db.Find(&students).Error
	return students, err
}

// FindByID mengembalikan satu mahasiswa berdasarkan ID UUID.
func (r *studentRepository) FindByID(id uuid.UUID) (*model.Student, error) {
	var st model.Student
	err := r.db.
		Preload("Advisor"). // kalau di model Student ada relasi Advisor *Lecturer
		First(&st, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// UpdateAdvisor mengganti dosen wali mahasiswa.
func (r *studentRepository) UpdateAdvisor(studentID, advisorID uuid.UUID) error {
	return r.db.Model(&model.Student{}).
		Where("id = ?", studentID).
		Update("advisor_id", advisorID).Error
}
