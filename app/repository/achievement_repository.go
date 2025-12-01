package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// AchievementRepository mendefinisikan operasi yang berhubungan dengan prestasi
// dan referensinya di Postgres + datanya di MongoDB.
type AchievementRepository interface {
	// Create menyimpan data prestasi:
	// - insert dokumen ke MongoDB
	// - insert referensi ke PostgreSQL (AchievementReference)
	Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error

	// FindByID mengambil 1 AchievementReference berdasarkan ID di PostgreSQL.
	FindByID(id string) (*model.AchievementReference, error)

	// UpdateStatus mengubah status prestasi (draft, submitted, verified, rejected, deleted)
	// sekaligus mengelola kolom terkait (submitted_at, verified_at, rejection_note, dll).
	UpdateStatus(id string, status string, opts UpdateStatusOptions) error

	// FindByStudentID mengambil seluruh AchievementReference milik mahasiswa tertentu (kecuali yang status deleted).
	FindByStudentID(studentID string) ([]model.AchievementReference, error)

	// FindDetailByMongoID mengambil dokumen prestasi dari MongoDB berdasarkan MongoAchievementID.
	FindDetailByMongoID(ctx context.Context, mongoID string) (*model.Achievement, error)
}

// UpdateStatusOptions menampung opsi ekstra ketika update status prestasi.
type UpdateStatusOptions struct {
	VerifierID    *string
	RejectionNote *string
}

type achievementRepository struct {
	pgDB    *gorm.DB
	mongoDB *mongo.Database
}

// NewAchievementRepository membuat instance repository prestasi.
func NewAchievementRepository(pgDB *gorm.DB, mongoDB *mongo.Database) AchievementRepository {
	return &achievementRepository{pgDB: pgDB, mongoDB: mongoDB}
}

// Daftar status yang diperbolehkan (sesuai SRS + revisi deleted).
var validStatuses = map[string]bool{
	"draft":     true,
	"submitted": true,
	"verified":  true,
	"rejected":  true,
	"deleted":   true,
}

// Create:
// 1. Insert dokumen ke MongoDB (collection: achievements).
// 2. Simpan ID Mongo ke MongoAchievementID di AchievementReference.
// 3. Insert AchievementReference ke PostgreSQL (dalam transaksi).
func (r *achievementRepository) Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error {
	if pgData == nil || pgData.StudentID == uuid.Nil {
		return errors.New("StudentID harus di-set sebelum Create()")
	}

	tx := r.pgDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Step 1: insert ke MongoDB terlebih dahulu
	insertRes, err := r.mongoDB.Collection("achievements").InsertOne(ctx, mongoData)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("mongo insert error: %w", err)
	}

	oid, ok := insertRes.InsertedID.(primitive.ObjectID)
	if !ok {
		tx.Rollback()
		return fmt.Errorf("mongo insert returned non-ObjectID")
	}

	pgData.MongoAchievementID = oid.Hex()

	now := time.Now()
	if pgData.CreatedAt.IsZero() {
		pgData.CreatedAt = now
	}
	pgData.UpdatedAt = now

	// Step 2: insert ke PostgreSQL
	if err := tx.Create(pgData).Error; err != nil {
		// rollback Mongo jika insert Postgres gagal
		_, _ = r.mongoDB.Collection("achievements").DeleteOne(ctx, bson.M{"_id": oid})
		tx.Rollback()
		return fmt.Errorf("postgres insert error: %w", err)
	}

	return tx.Commit().Error
}

// FindByID mengambil AchievementReference dari PostgreSQL berdasarkan ID.
// Catatan: tidak lagi memanggil Preload("Student") karena field Student di model di-ignore (gorm:"-")
// sehingga Preload akan menimbulkan error "unsupported relations".
func (r *achievementRepository) FindByID(id string) (*model.AchievementReference, error) {
	var ref model.AchievementReference
	err := r.pgDB.
		Preload("Verifier"). // preload user yang memverifikasi (jika ada)
		Where("id = ?", id).
		First(&ref).Error
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

// UpdateStatus mengubah status prestasi di Postgres,
// dan jika status = deleted, juga melakukan soft-delete di MongoDB.
func (r *achievementRepository) UpdateStatus(id string, status string, opts UpdateStatusOptions) error {
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	// Perlakuan khusus untuk status deleted:
	if status == "deleted" {
		// Cari dulu referensi di Postgres
		var ref model.AchievementReference
		if err := r.pgDB.Where("id = ?", id).First(&ref).Error; err != nil {
			return err
		}

		// Konversi hex ke ObjectID Mongo
		objID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
		if err != nil {
			return err
		}

		now := time.Now()
		// Tandai dokumen Mongo sebagai deleted
		res, err := r.mongoDB.Collection("achievements").
			UpdateOne(context.Background(),
				bson.M{"_id": objID},
				bson.M{"$set": bson.M{"deleted": true, "deletedAt": now}},
			)
		if err != nil {
			return fmt.Errorf("mongo soft-delete failed: %w", err)
		}
		if res.MatchedCount == 0 {
			return fmt.Errorf("mongo document not found for deletion")
		}

		// Update status di Postgres dalam transaksi
		tx := r.pgDB.Begin()
		if tx.Error != nil {
			// rollback flag di Mongo
			_, _ = r.mongoDB.Collection("achievements").
				UpdateOne(context.Background(),
					bson.M{"_id": objID},
					bson.M{"$unset": bson.M{"deleted": "", "deletedAt": ""}},
				)
			return tx.Error
		}

		updates := map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}

		if err := tx.Model(&model.AchievementReference{}).
			Where("id = ?", id).
			Updates(updates).Error; err != nil {

			tx.Rollback()
			// rollback Mongo
			_, _ = r.mongoDB.Collection("achievements").
				UpdateOne(context.Background(),
					bson.M{"_id": objID},
					bson.M{"$unset": bson.M{"deleted": "", "deletedAt": ""}},
				)
			return err
		}
		return tx.Commit().Error
	}

	// Flow untuk status selain deleted
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	now := time.Now()
	switch status {
	case "submitted":
		updates["submitted_at"] = now
	case "verified":
		updates["verified_at"] = now
		if opts.VerifierID != nil {
			updates["verified_by"] = *opts.VerifierID
		}
	case "rejected":
		updates["verified_at"] = now
		if opts.VerifierID != nil {
			updates["verified_by"] = *opts.VerifierID
		}
		if opts.RejectionNote != nil {
			updates["rejection_note"] = *opts.RejectionNote
		}
	}

	return r.pgDB.Model(&model.AchievementReference{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// FindByStudentID mengambil semua prestasi milik 1 mahasiswa (kecuali yang sudah berstatus deleted).
func (r *achievementRepository) FindByStudentID(studentID string) ([]model.AchievementReference, error) {
	var refs []model.AchievementReference
	err := r.pgDB.
		Where("student_id = ? AND status != 'deleted'", studentID).
		Order("created_at DESC").
		Find(&refs).Error
	return refs, err
}

// FindDetailByMongoID mengambil dokumen prestasi (detail) dari MongoDB.
// Hanya mengambil dokumen yang tidak memiliki flag deleted = true.
func (r *achievementRepository) FindDetailByMongoID(ctx context.Context, mongoID string) (*model.Achievement, error) {
	objID, err := primitive.ObjectIDFromHex(mongoID)
	if err != nil {
		return nil, err
	}
	var achievement model.Achievement
	err = r.mongoDB.Collection("achievements").
		FindOne(ctx, bson.M{
			"_id":     objID,
			"deleted": bson.M{"$ne": true},
		}).
		Decode(&achievement)
	return &achievement, err
}
