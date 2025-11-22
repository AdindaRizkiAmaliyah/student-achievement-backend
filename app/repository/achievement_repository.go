package repository

import (
	"context"
	"student-achievement-backend/app/model"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// Interface kontrak untuk Achievement
type AchievementRepository interface {
	// Create menerima dua jenis data:
	// 1. pgData: Data ringkas untuk tabel referensi di PostgreSQL (status, tanggal, id mahasiswa)
	// 2. mongoData: Data detail prestasi yang dinamis (judul, lomba, ranking, dll)
	Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error
}

// Struct ini menyimpan dua koneksi database sekaligus!
type achievementRepository struct {
	pgDB    *gorm.DB        // Koneksi ke Postgres
	mongoDB *mongo.Database // Koneksi ke Mongo
}

// Constructor
func NewAchievementRepository(pgDB *gorm.DB, mongoDB *mongo.Database) AchievementRepository {
	return &achievementRepository{
		pgDB:    pgDB,
		mongoDB: mongoDB,
	}
}

// Create menjalankan logika "Hybrid Database" sesuai SRS
func (r *achievementRepository) Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error {
	
	// LANGKAH 1: Mulai Transaksi PostgreSQL.
	// "Transaction" artinya: Kalau nanti di tengah jalan ada error, semua perubahan dibatalkan.
	tx := r.pgDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// LANGKAH 2: Simpan data detail ke MongoDB dulu.
	// Kita pilih collection "achievements".
	collection := r.mongoDB.Collection("achievements")
	
	// InsertOne perintah untuk menyimpan dokumen JSON ke Mongo.
	insertResult, err := collection.InsertOne(ctx, mongoData)
	if err != nil {
		// Jika gagal simpan ke Mongo, batalkan transaksi Postgres (Rollback).
		tx.Rollback()
		return err
	}

	// Update ID di data postgres dengan ID asli dari MongoDB yang baru dibuat.
	// Ini kunci penghubung antara Postgres dan Mongo!
	// (Asumsi: mongoData._id sudah di-set atau kita ambil dari insertResult)
	// Di sini kita biarkan sesuai input, tapi pastikan service mengirim ID yang sinkron.

	// LANGKAH 3: Simpan data referensi ke PostgreSQL.
	// tx.Create artinya kita pakai koneksi transaksi, bukan koneksi biasa.
	if err := tx.Create(pgData).Error; err != nil {
		// BAHAYA: Jika simpan ke Postgres gagal, data di Mongo sudah terlanjur masuk!
		// Kita harus menghapusnya manual (Kompensasi) agar data bersih.
		collection.DeleteOne(ctx, map[string]interface{}{"_id": insertResult.InsertedID})
		
		// Dan jangan lupa batalkan transaksi Postgres.
		tx.Rollback()
		return err
	}

	// LANGKAH 4: Jika semua sukses, "Commit" transaksi.
	// Data baru benar-benar permanen tersimpan di Postgres setelah baris ini.
	return tx.Commit().Error
}