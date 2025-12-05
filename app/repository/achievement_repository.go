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

// AchievementRepository mendefinisikan operasi data prestasi
// yang menyentuh 2 database: PostgreSQL (reference) & MongoDB (detail).
type AchievementRepository interface {
	// Create: simpan prestasi baru ke MongoDB lalu buat reference di PostgreSQL.
	Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error
	// FindByID: ambil 1 reference prestasi berdasarkan ID UUID (Postgres).
	FindByID(id string) (*model.AchievementReference, error)
	// UpdateStatus: update status + field terkait (submitted_at, verified_at, dsb).
	UpdateStatus(id string, status string, opts UpdateStatusOptions) error
	// FindByStudentID: ambil semua reference prestasi milik 1 mahasiswa (kecuali deleted).
	FindByStudentID(studentID string) ([]model.AchievementReference, error)
	// FindDetailByMongoID: ambil detail prestasi dari MongoDB berdasarkan ObjectID (hex).
	FindDetailByMongoID(ctx context.Context, mongoID string) (*model.Achievement, error)
	// FindAll: FR-010 — ambil semua prestasi (opsional filter status + pagination).
	FindAll(status *string, page, limit int) ([]model.AchievementReference, int64, error)

	// UpdateContent: UPDATE isi prestasi di MongoDB (title, description, details, dll) + updated_at di Postgres.
	UpdateContent(ctx context.Context, id string, mongoData *model.Achievement) error
	// AddAttachment: menambahkan satu attachment ke dokumen achievement di MongoDB.
	AddAttachment(ctx context.Context, achievementID string, attachment model.Attachment) error
}

// UpdateStatusOptions menyimpan opsi tambahan ketika update status prestasi.
type UpdateStatusOptions struct {
	VerifierID    *string
	RejectionNote *string
}

// achievementRepository adalah implementasi konkret AchievementRepository.
type achievementRepository struct {
	pgDB    *gorm.DB
	mongoDB *mongo.Database
}

// NewAchievementRepository membuat instance repository baru.
func NewAchievementRepository(pgDB *gorm.DB, mongoDB *mongo.Database) AchievementRepository {
	return &achievementRepository{pgDB, mongoDB}
}

// validStatuses: daftar status yang diizinkan (sesuai SRS + tambahan 'deleted').
var validStatuses = map[string]bool{
	"draft":     true,
	"submitted": true,
	"verified":  true,
	"rejected":  true,
	"deleted":   true, // tambahan enum baru
}

// Create menyimpan prestasi baru ke MongoDB lalu membuat reference di PostgreSQL.
func (r *achievementRepository) Create(ctx context.Context, pgData *model.AchievementReference, mongoData *model.Achievement) error {
	if pgData == nil || pgData.StudentID == uuid.Nil {
		return errors.New("StudentID harus di-set sebelum Create()")
	}

	tx := r.pgDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 1. Insert ke MongoDB terlebih dahulu
	insertRes, err := r.mongoDB.Collection("achievements").InsertOne(ctx, mongoData)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("mongo insert error: %w", err)
	}

	// 2. Dapatkan ObjectID lalu simpan ke kolom mongo_achievement_id di Postgres
	oid := insertRes.InsertedID.(primitive.ObjectID)
	pgData.MongoAchievementID = oid.Hex()

	now := time.Now()
	if pgData.CreatedAt.IsZero() {
		pgData.CreatedAt = now
	}
	pgData.UpdatedAt = now

	// 3. Insert ke PostgreSQL
	if err := tx.Create(pgData).Error; err != nil {
		// Jika gagal, hapus dokumen Mongo yang baru dibuat
		_, _ = r.mongoDB.Collection("achievements").DeleteOne(ctx, bson.M{"_id": oid})
		tx.Rollback()
		return fmt.Errorf("postgres insert error: %w", err)
	}

	return tx.Commit().Error
}

// FindByID mengambil 1 reference prestasi berdasarkan id UUID (Postgres).
func (r *achievementRepository) FindByID(id string) (*model.AchievementReference, error) {
	var ref model.AchievementReference

	// Kita hanya preload Verifier (User yang memverifikasi),
	// karena relasi Student belum kita definisikan dengan benar di model
	// dan di seluruh flow kita hanya butuh StudentID, bukan objek Student-nya.
	err := r.pgDB.
		Preload("Verifier").
		Where("id = ?", id).
		First(&ref).Error

	if err != nil {
		return nil, err
	}

	return &ref, nil
}


// UpdateStatus mengubah status prestasi dan field-field terkait.
func (r *achievementRepository) UpdateStatus(id string, status string, opts UpdateStatusOptions) error {
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	// === Perlakuan khusus untuk status 'deleted' ===
	if status == "deleted" {
		// 1. Ambil reference terlebih dahulu
		var ref model.AchievementReference
		if err := r.pgDB.Where("id = ?", id).First(&ref).Error; err != nil {
			return err
		}

		// 2. Convert mongoAchievementID (hex) -> ObjectID
		objID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
		if err != nil {
			return err
		}

		// 3. Soft delete di Mongo: set field deleted=true, deletedAt=now
		now := time.Now()
		res, err := r.mongoDB.Collection("achievements").
			UpdateOne(
				context.Background(),
				bson.M{"_id": objID},
				bson.M{"$set": bson.M{"deleted": true, "deletedAt": now}},
			)
		if err != nil {
			return fmt.Errorf("mongo soft-delete failed: %w", err)
		}
		if res.MatchedCount == 0 {
			return fmt.Errorf("mongo document not found for deletion")
		}

		// 4. Update status di Postgres dalam transaksi
		tx := r.pgDB.Begin()
		if tx.Error != nil {
			// rollback perubahan di Mongo
			_, _ = r.mongoDB.Collection("achievements").
				UpdateOne(
					context.Background(),
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
				UpdateOne(
					context.Background(),
					bson.M{"_id": objID},
					bson.M{"$unset": bson.M{"deleted": "", "deletedAt": ""}},
				)
			return err
		}
		return tx.Commit().Error
	}

	// === Flow umum untuk status selain 'deleted' ===
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

	return r.pgDB.
		Model(&model.AchievementReference{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// FindByStudentID mengambil semua prestasi milik seorang mahasiswa (kecuali yang status 'deleted').
func (r *achievementRepository) FindByStudentID(studentID string) ([]model.AchievementReference, error) {
	var refs []model.AchievementReference
	err := r.pgDB.
		Where("student_id = ? AND status != 'deleted'", studentID).
		Order("created_at DESC").
		Find(&refs).Error
	return refs, err
}

// FindDetailByMongoID mengambil detail prestasi dari MongoDB berdasarkan _id ObjectID hex.
func (r *achievementRepository) FindDetailByMongoID(ctx context.Context, mongoID string) (*model.Achievement, error) {
	objID, err := primitive.ObjectIDFromHex(mongoID)
	if err != nil {
		return nil, err
	}
	var achievement model.Achievement
	err = r.mongoDB.Collection("achievements").
		FindOne(ctx, bson.M{"_id": objID, "deleted": bson.M{"$ne": true}}).
		Decode(&achievement)
	return &achievement, err
}

// FindAll mengembalikan daftar prestasi untuk admin (FR-010).
// Mendukung:
//   - filter status (?status=submitted)
//   - pagination basic (?page=1&limit=10)
func (r *achievementRepository) FindAll(status *string, page, limit int) ([]model.AchievementReference, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	db := r.pgDB.Model(&model.AchievementReference{})

	if status != nil && *status != "" {
		db = db.Where("status = ?", *status)
	}

	// Hitung total untuk pagination
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var refs []model.AchievementReference
	err := db.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&refs).Error

	return refs, total, err
}

// UpdateContent melakukan UPDATE konten prestasi di MongoDB lalu update updated_at di Postgres.
func (r *achievementRepository) UpdateContent(ctx context.Context, id string, mongoData *model.Achievement) error {
	// Ambil reference untuk mendapatkan mongo_achievement_id
	var ref model.AchievementReference
	if err := r.pgDB.Where("id = ?", id).First(&ref).Error; err != nil {
		return err
	}

	objID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil {
		return err
	}

	now := time.Now()

	// Siapkan dokumen update Mongo (field yang boleh diubah mahasiswa)
	updateDoc := bson.M{
		"achievementType": mongoData.AchievementType,
		"title":           mongoData.Title,
		"description":     mongoData.Description,
		"details":         mongoData.Details,
		"attachments":     mongoData.Attachments,
		"tags":            mongoData.Tags,
		"points":          mongoData.Points,
		"updatedAt":       now,
	}

	if _, err := r.mongoDB.Collection("achievements").
		UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateDoc}); err != nil {
		return fmt.Errorf("mongo update error: %w", err)
	}

	// Update updated_at di Postgres
	return r.pgDB.Model(&model.AchievementReference{}).
		Where("id = ?", id).
		Update("updated_at", now).Error
}

// AddAttachment menambahkan satu attachment ke dokumen achievement di MongoDB
// berdasarkan ID achievement di PostgreSQL (achievement_references.id).
func (r *achievementRepository) AddAttachment(
	ctx context.Context,
	achievementID string,
	attachment model.Attachment,
) error {
	// 1. Ambil reference di Postgres untuk mendapatkan mongoAchievementID.
	var ref model.AchievementReference
	if err := r.pgDB.Where("id = ?", achievementID).First(&ref).Error; err != nil {
		return err // achievement tidak ditemukan di Postgres
	}

	// 2. Konversi mongoAchievementID (hex string) ke ObjectID.
	objID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil {
		return err
	}

	// 3. Push attachment baru ke array attachments di dokumen Mongo.
	_, err = r.mongoDB.Collection("achievements").UpdateOne(
		ctx,
		bson.M{"_id": objID, "deleted": bson.M{"$ne": true}},
		bson.M{"$push": bson.M{"attachments": attachment}},
	)

	return err
}
