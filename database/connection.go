package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"student-achievement-backend/app/model"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/bson"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database menyimpan koneksi Postgres & Mongo dalam satu struct
// supaya mudah di-pass ke layer lain.
type Database struct {
	Postgres *gorm.DB
	Mongo    *mongo.Database
}

// InitDB menginisialisasi koneksi ke PostgreSQL & MongoDB,
// menjalankan migrasi GORM, dan mengembalikan wrapper Database.
func InitDB() (*Database, error) {

	// 1. KONFIGURASI POSTGRES
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	pgDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke postgres: %v", err)
	}

	// 2. ENABLE EXTENSION PGCRYPTO — diperlukan untuk gen_random_uuid()
	if err := pgDB.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`).Error; err != nil {
		return nil, fmt.Errorf("gagal enable pgcrypto: %v", err)
	}
	log.Println("pgcrypto extension aktif ✔")

	// 3. MIGRATION
	log.Println("⏳ Migrating PostgreSQL...")

	err = pgDB.AutoMigrate(
		&model.Role{},
		&model.Permission{},
		&model.User{},
		&model.Student{},
		&model.Lecturer{},
		&model.AchievementReference{},
	)
	if err != nil {
		log.Fatalf("❌ Migration error: %v", err)
	}

	log.Println("✅ Migration complete")

	// 4. KONEKSI MONGODB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke mongo: %v", err)
	}

	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("gagal ping mongo: %v", err)
	}

	mongoDB := mongoClient.Database(os.Getenv("MONGO_DB_NAME"))

	// 5. OPSIONAL: BUAT INDEX UNTUK COLLECTION achievements
	//    - studentId: untuk query list prestasi per mahasiswa
	//    - details.customFields.isDeleted: untuk filter soft-delete
	achievementsCol := mongoDB.Collection("achievements")
	indexView := achievementsCol.Indexes()
	_, err = indexView.CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "studentId", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "details.customFields.isDeleted", Value: 1}},
		},
	})
	if err != nil {
		log.Printf("[MONGO] Gagal membuat index achievements: %v", err)
	} else {
		log.Println("[MONGO] Index achievements siap ✔")
	}

	log.Println("Berhasil terhubung ke PostgreSQL & MongoDB! ✔")

	return &Database{
		Postgres: pgDB,
		Mongo:    mongoDB,
	}, nil
}
