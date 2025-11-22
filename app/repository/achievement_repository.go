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
	FindByID(id string) (*model.AchievementReference, error)
	UpdateStatus(id string, status string) error
	Delete(ctx context.Context, id string) error
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

// Implementasi FindByID
func (r *achievementRepository) FindByID(id string) (*model.AchievementReference, error) {
	var achievement model.AchievementReference
	// Cari data di tabel PostgreSQL berdasarkan Primary Key (ID)
	err := r.pgDB.Where("id = ?", id).First(&achievement).Error
	if err != nil {
		return nil, err
	}
	return &achievement, nil
}

// Implementasi UpdateStatus
func (r *achievementRepository) UpdateStatus(id string, status string) error {
	// Update kolom 'status' pada tabel achievement_references dimana id cocok
	return r.pgDB.Model(&model.AchievementReference{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// [UPDATE BARU] Implementasi Delete 
func (r *achievementRepository) Delete(ctx context.Context, id string) error {
    // 1. Cari dulu data referensinya di Postgres untuk dapat MongoAchievementID
    var achievement model.AchievementReference
    if err := r.pgDB.Where("id = ?", id).First(&achievement).Error; err != nil {
        return err
    }

    // 2. Mulai Transaksi
    tx := r.pgDB.Begin()
    if tx.Error != nil {
        return tx.Error
    }

    // 3. Hapus data di PostgreSQL
    if err := tx.Delete(&achievement).Error; err != nil {
        tx.Rollback()
        return err
    }

    // 4. Hapus data di MongoDB
    // Kita gunakan MongoAchievementID yang kita dapat dari Postgres tadi
    objID, _ := primitive.ObjectIDFromHex(achievement.MongoAchievementID)
    collection := r.mongoDB.Collection("achievements")
    _, err := collection.DeleteOne(ctx, map[string]interface{}{"_id": objID})
    
    if err != nil {
        tx.Rollback() // Batalkan hapus Postgres jika Mongo gagal
        return err
    }

    // 5. Commit Transaksi
    return tx.Commit().Error
}