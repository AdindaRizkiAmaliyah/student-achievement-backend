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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Database struct {
	Postgres *gorm.DB
	Mongo    *mongo.Database
}

func InitDB() (*Database, error) {
	// 1. Setup PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	// UPDATE PENTING: Tambahkan config DisableForeignKeyConstraintWhenMigrating
	pgDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke postgres: %v", err)
	}

	// Auto Migrate
	log.Println("Menjalankan migrasi database PostgreSQL...")
	err = pgDB.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.Student{},
		&model.Lecturer{},
		&model.AchievementReference{},
	)
	if err != nil {
		return nil, fmt.Errorf("gagal migrasi database: %v", err)
	}

	// 2. Setup MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoURI)
	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("gagal koneksi ke mongo: %v", err)
	}

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal ping mongo: %v", err)
	}

	mongoDBName := os.Getenv("MONGO_DB_NAME")
	mongoDatabase := mongoClient.Database(mongoDBName)

	log.Println("Berhasil terhubung ke PostgreSQL dan MongoDB!")

	return &Database{
		Postgres: pgDB,
		Mongo:    mongoDatabase,
	}, nil
}