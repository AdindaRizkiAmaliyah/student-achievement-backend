package repository

import (
	"context"

	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LecturerRepository menangani operasi terkait dosen wali & relasinya dengan mahasiswa.
type LecturerRepository interface {
	// FindByUserID: cari dosen wali berdasarkan user_id (dipakai saat doswal login).
	FindByUserID(userID uuid.UUID) (*model.Lecturer, error)
	// IsAdvisorOf: cek apakah lecturerID adalah dosen wali dari studentID tertentu.
	IsAdvisorOf(lecturerID uuid.UUID, studentID uuid.UUID) (bool, error)
	// GetAdviseeStudentIDs: ambil semua student_id (UUID) mahasiswa bimbingan dosen wali.
	GetAdviseeStudentIDs(lecturerID uuid.UUID) ([]uuid.UUID, error)
	// FindAchievementsByStudentIDs: ambil semua achievement_references untuk kumpulan studentID.
	FindAchievementsByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]model.AchievementReference, error)
}

// lecturerRepository adalah implementasi konkret LecturerRepository.
type lecturerRepository struct {
	db *gorm.DB
}

// NewLecturerRepository membuat instance baru lecturerRepository.
func NewLecturerRepository(db *gorm.DB) LecturerRepository {
	return &lecturerRepository{db: db}
}

// FindByUserID mencari dosen wali berdasarkan user_id (FK ke tabel users).
func (r *lecturerRepository) FindByUserID(userID uuid.UUID) (*model.Lecturer, error) {
	var lec model.Lecturer
	if err := r.db.Where("user_id = ?", userID).First(&lec).Error; err != nil {
		return nil, err
	}
	return &lec, nil
}

// IsAdvisorOf mengecek apakah lecturerID adalah dosen wali dari studentID tertentu.
func (r *lecturerRepository) IsAdvisorOf(lecturerID uuid.UUID, studentID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Student{}).
		Where("id = ? AND advisor_id = ?", studentID, lecturerID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAdviseeStudentIDs mengembalikan semua ID mahasiswa bimbingan (students.id) untuk dosen wali tertentu.
func (r *lecturerRepository) GetAdviseeStudentIDs(lecturerID uuid.UUID) ([]uuid.UUID, error) {
	var students []model.Student
	if err := r.db.
		Where("advisor_id = ?", lecturerID).
		Find(&students).Error; err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, 0, len(students))
	for _, s := range students {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

// FindAchievementsByStudentIDs mengambil semua achievement_references
// untuk kumpulan studentID (dipakai dosen wali & admin saat melihat prestasi).
func (r *lecturerRepository) FindAchievementsByStudentIDs(
	_ context.Context,
	studentIDs []uuid.UUID,
) ([]model.AchievementReference, error) {

	if len(studentIDs) == 0 {
		return []model.AchievementReference{}, nil
	}

	var refs []model.AchievementReference
	if err := r.db.
		Where("student_id IN ? AND status != 'deleted'", studentIDs).
		Order("created_at DESC").
		Find(&refs).Error; err != nil {
		return nil, err
	}

	return refs, nil
}
