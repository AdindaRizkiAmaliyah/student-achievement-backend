package service

import (
	"context"
	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"time"
)

// Interface AchievementService
type AchievementService interface {
	SubmitAchievement(ctx context.Context, studentID string, pgData *model.AchievementReference, mongoData *model.Achievement) error
}

type achievementService struct {
	achievementRepo repository.AchievementRepository
}

func NewAchievementService(achievementRepo repository.AchievementRepository) AchievementService {
	return &achievementService{
		achievementRepo: achievementRepo,
	}
}

// SubmitAchievement: Logika mahasiswa melaporkan prestasi baru [cite: 178]
func (s *achievementService) SubmitAchievement(ctx context.Context, studentID string, pgData *model.AchievementReference, mongoData *model.Achievement) error {
	
	// 1. Set Default Values (Aturan Bisnis)
	
	// Status awal harus selalu 'draft' saat baru dibuat [cite: 185]
	pgData.Status = "draft" 
	
	// Isi waktu created_at otomatis jika belum ada
	now := time.Now()
	pgData.CreatedAt = now
	pgData.UpdatedAt = now
	mongoData.CreatedAt = now
	mongoData.UpdatedAt = now

	// Pastikan data Mongo punya StudentID yang sesuai (Sinkronisasi data)
	// (Di sini kita asumsikan ID Mahasiswa valid, validasi lengkap biasanya di controller)
	
	// 2. Panggil Repository untuk melakukan transaksi penyimpanan
	// Repository akan mengurus penyimpanan ke Mongo dulu, baru ke Postgres.
	err := s.achievementRepo.Create(ctx, pgData, mongoData)
	if err != nil {
		return err
	}

	return nil
}