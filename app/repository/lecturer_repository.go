package repository

import (
	"context"

	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LecturerRepository menangani data dosen + relasinya ke mahasiswa & prestasi.
// Dipakai oleh:
// - SRS 5.5 (GET lecturers, GET advisees)
// - AchievementService (cek dosen wali, ambil prestasi bimbingan).
type LecturerRepository interface {
	// SRS 5.5
	FindAll() ([]model.Lecturer, error)                             // GET /lecturers
	FindByID(id uuid.UUID) (*model.Lecturer, error)                 // GET /lecturers/:id
	FindAdvisees(lecturerID uuid.UUID) ([]model.Student, error)     // GET /lecturers/:id/advisees

	// Untuk kebutuhan RBAC & achievement
	FindByUserID(userID uuid.UUID) (*model.Lecturer, error)
	GetAdviseeStudentIDs(lecturerID uuid.UUID) ([]uuid.UUID, error)
	IsAdvisorOf(lecturerID uuid.UUID, studentID uuid.UUID) (bool, error)
	FindAchievementsByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]model.AchievementReference, error)
}

type lecturerRepository struct {
	db *gorm.DB
}

func NewLecturerRepository(db *gorm.DB) LecturerRepository {
	return &lecturerRepository{db}
}

// ============ SRS 5.5 ============

// FindAll mengembalikan semua dosen.
func (r *lecturerRepository) FindAll() ([]model.Lecturer, error) {
	var lecturers []model.Lecturer
	err := r.db.Find(&lecturers).Error
	return lecturers, err
}

// FindByID mengambil satu dosen berdasarkan ID UUID.
func (r *lecturerRepository) FindByID(id uuid.UUID) (*model.Lecturer, error) {
	var lect model.Lecturer
	err := r.db.First(&lect, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &lect, nil
}

// FindAdvisees mengambil semua mahasiswa yang memiliki advisor_id = lecturerID.
func (r *lecturerRepository) FindAdvisees(lecturerID uuid.UUID) ([]model.Student, error) {
	var students []model.Student
	err := r.db.
		Where("advisor_id = ?", lecturerID).
		Find(&students).Error
	return students, err
}

// ============ Digunakan AchievementService ============

// FindByUserID mencari dosen berdasarkan user_id.
func (r *lecturerRepository) FindByUserID(userID uuid.UUID) (*model.Lecturer, error) {
	var lect model.Lecturer
	err := r.db.First(&lect, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &lect, nil
}

// GetAdviseeStudentIDs mengembalikan daftar ID mahasiswa bimbingan dosen wali.
func (r *lecturerRepository) GetAdviseeStudentIDs(lecturerID uuid.UUID) ([]uuid.UUID, error) {
	var students []model.Student
	err := r.db.
		Select("id").
		Where("advisor_id = ?", lecturerID).
		Find(&students).Error
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(students))
	for _, s := range students {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

// IsAdvisorOf mengecek apakah lecturerID adalah dosen wali studentID.
func (r *lecturerRepository) IsAdvisorOf(lecturerID uuid.UUID, studentID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.Student{}).
		Where("id = ? AND advisor_id = ?", studentID, lecturerID).
		Count(&count).Error
	return count > 0, err
}

// FindAchievementsByStudentIDs mengambil semua achievement_references
// untuk daftar mahasiswa tertentu (digunakan dosen wali untuk lihat prestasi bimbingan).
func (r *lecturerRepository) FindAchievementsByStudentIDs(
	_ context.Context,
	studentIDs []uuid.UUID,
) ([]model.AchievementReference, error) {

	if len(studentIDs) == 0 {
		return []model.AchievementReference{}, nil
	}

	var refs []model.AchievementReference
	err := r.db.
		Where("student_id IN ?", studentIDs).
		Where("status != ?", "deleted").
		Order("created_at DESC").
		Find(&refs).Error

	return refs, err
}
