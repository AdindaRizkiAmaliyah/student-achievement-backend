package service

import (
	"context"
	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"
	"time"
	"errors"

	"github.com/google/uuid" // Jangan lupa import ini untuk parsing UUID
)

// AchievementService interface
type AchievementService interface {
	SubmitAchievement(ctx context.Context, studentID string, pgData *model.AchievementReference, mongoData *model.Achievement) error
	SubmitForVerification(ctx context.Context, achievementID string, userID string) error
	DeleteAchievement(ctx context.Context, achievementID string, userID string) error
	// [UPDATE BARU] Fungsi Get List
	// Returnnya interface{} agar kita bisa bikin struktur custom (gabungan)
	GetAchievementsByStudent(ctx context.Context, userID string) ([]map[string]interface{}, error)
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

// Implementasi Logika Submit Verification (FR-004)
func (s *achievementService) SubmitForVerification(ctx context.Context, achievementID string, userID string) error {
	// 1. Cari data prestasi berdasarkan ID
	achievement, err := s.achievementRepo.FindByID(achievementID)
	if err != nil {
		return errors.New("prestasi tidak ditemukan")
	}

	// 2. Cek Kepemilikan (Authorization Check)
	// Pastikan yang mensubmit adalah mahasiswa pemilik prestasi itu sendiri
	// Kita bandingkan UserID yang login dengan StudentID di data prestasi
	// Note: Di database kita simpan studentID sebagai UUID, jadi harus konversi dulu untuk membandingkan
	if achievement.StudentID.String() != userID {
		// Pengecekan kasar: karena di create kita simpan user.ID ke studentID.
		// Idealnya user.ID -> cari student -> match student.ID.
		// Tapi di tahap create sebelumnya kita direct mapping UserID -> StudentID.
		// Jadi perbandingan ini valid untuk struktur saat ini.
		return errors.New("anda tidak berhak mengubah prestasi ini")
	}

	// 3. Cek Status Awal (Precondition)
	// Prestasi hanya boleh disubmit jika statusnya masih 'draft'
	if achievement.Status != "draft" {
		return errors.New("prestasi hanya bisa disubmit jika statusnya draft")
	}

	// 4. Update status menjadi 'submitted' [cite: 195]
	err = s.achievementRepo.UpdateStatus(achievementID, "submitted")
	if err != nil {
		return err
	}

	return nil
}

// [UPDATE BARU] Implementasi DeleteAchievement (FR-005)
func (s *achievementService) DeleteAchievement(ctx context.Context, achievementID string, userID string) error {
    // 1. Cari data prestasi dulu
    achievement, err := s.achievementRepo.FindByID(achievementID)
    if err != nil {
        return errors.New("prestasi tidak ditemukan")
    }

    // 2. Validasi Kepemilikan (Authorization)
    // Cek apakah yang mau menghapus adalah pemilik datanya
    if achievement.StudentID.String() != userID {
        return errors.New("anda tidak berhak menghapus prestasi ini")
    }

    // 3. Validasi Status (Precondition)
    // Sesuai SRS FR-005: Precondition Status 'draft' [cite: 201]
    if achievement.Status != "draft" {
        return errors.New("prestasi tidak bisa dihapus karena sudah disubmit atau diverifikasi")
    }

    // 4. Panggil Repository untuk hapus permanen
    return s.achievementRepo.Delete(ctx, achievementID)
}

// [UPDATE BARU] Implementasi Get List
func (s *achievementService) GetAchievementsByStudent(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	// 1. Ambil daftar referensi dari PostgreSQL (Status, ID, Tanggal)
	pgDataList, err := s.achievementRepo.FindByStudentID(userID)
	if err != nil {
		return nil, err
	}

	// 2. Siapkan wadah untuk hasil gabungan
	var result []map[string]interface{}

	// 3. Looping setiap data dari Postgres
	for _, pgItem := range pgDataList {
		// Ambil detail (Judul) dari MongoDB berdasarkan ID yang tersimpan di Postgres
		mongoDetail, err := s.achievementRepo.FindDetailByMongoID(ctx, pgItem.MongoAchievementID)
		
		title := "Data Corrupt/Missing"
		achievementType := "Unknown"
		
		// Jika data di Mongo ketemu, ambil Judul aslinya
		if err == nil {
			title = mongoDetail.Title
			achievementType = mongoDetail.AchievementType
		}

		// 4. Rakit data gabungan untuk dikirim ke Frontend
		combinedData := map[string]interface{}{
			"id":          pgItem.ID,         // ID untuk Edit/Delete
			"title":       title,             // Dari Mongo
			"type":        achievementType,   // Dari Mongo
			"status":      pgItem.Status,     // Dari Postgres
			"points":      mongoDetail.Points,// Dari Mongo
			"created_at":  pgItem.CreatedAt,
			"submitted_at": pgItem.SubmittedAt,
		}

		result = append(result, combinedData)
	}

	return result, nil
}