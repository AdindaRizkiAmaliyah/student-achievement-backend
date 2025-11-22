package repository

import (
	"context"
	"student-achievement-backend/app/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// AchievementRepository adalah kontrak interface
type AchievementRepository interface {
	Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error
}

// achievementRepository struct implementasi
type achievementRepository struct {
	pgDB    *gorm.DB
	mongoDB *mongo.Database
}

// NewAchievementRepository constructor
func NewAchievementRepository(pgDB *gorm.DB, mongoDB *mongo.Database) AchievementRepository {
	return &achievementRepository{
		pgDB:    pgDB,
		mongoDB: mongoDB,
	}
}

// Create menyimpan data ke MongoDB dan PostgreSQL dalam satu transaksi logis
func (r *achievementRepository) Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error {
	
	// 1. Mulai Transaksi PostgreSQL (untuk keamanan data)
	tx := r.pgDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 2. Simpan Detail ke MongoDB
	collection := r.mongoDB.Collection("achievements")
	insertResult, err := collection.InsertOne(ctx, mongoData)
	if err != nil {
		tx.Rollback() // Batalkan transaksi Postgres jika Mongo gagal
		return err
	}

	// --- [PERBAIKAN UTAMA] ---
	// Ambil ID unik yang baru saja dibuat oleh MongoDB (ObjectID)
	// Lalu konversi jadi string dan masukkan ke struct Postgres
	if oid, ok := insertResult.InsertedID.(primitive.ObjectID); ok {
		pgData.MongoAchievementID = oid.Hex()
	}
	// -------------------------

	// 3. Simpan Referensi ke PostgreSQL
	if err := tx.Create(pgData).Error; err != nil {
		// Jika simpan ke Postgres gagal, kita harus hapus data yang terlanjur masuk ke Mongo
		collection.DeleteOne(ctx, map[string]interface{}{"_id": insertResult.InsertedID})
		
		tx.Rollback() // Batalkan transaksi
		return err
	}

	// 4. Commit (Simpan Permanen)
	return tx.Commit().Error
}