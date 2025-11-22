package service

import (
	"context"
	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"time"

	"github.com/google/uuid" // Jangan lupa import ini untuk parsing UUID
)

// AchievementService interface
type AchievementService interface {
	SubmitAchievement(ctx context.Context, studentID string, pgData *model.AchievementReference, mongoData *model.Achievement) error
}

// achievementService struct
type achievementService struct {
	achievementRepo repository.AchievementRepository
}

// NewAchievementService constructor
func NewAchievementService(achievementRepo repository.AchievementRepository) AchievementService {
	return &achievementService{
		achievementRepo: achievementRepo,
	}
}

// SubmitAchievement logika bisnis pelaporan prestasi
func (s *achievementService) SubmitAchievement(ctx context.Context, studentID string, pgData *model.AchievementReference, mongoData *model.Achievement) error {
	
	// --- [PERBAIKAN UTAMA] ---
	// Konversi studentID (string dari token) menjadi UUID (format database)
	uid, err := uuid.Parse(studentID)
	if err == nil {
		// Masukkan UUID yang valid ke struct data
		pgData.StudentID = uid
		mongoData.StudentID = uid
	}
	// -------------------------

	// 1. Set Default Values sesuai Aturan Bisnis SRS
	pgData.Status = "draft" // Status awal wajib draft
	
	// Isi timestamp otomatis
	now := time.Now()
	pgData.CreatedAt = now
	pgData.UpdatedAt = now
	mongoData.CreatedAt = now
	mongoData.UpdatedAt = now

	// 2. Panggil Repository
	// Data pgData sekarang sudah berisi StudentID yang benar dan MongoAchievementID akan diisi di repo
	return s.achievementRepo.Create(ctx, pgData, mongoData)
}