package main

import (
	"log"
	"student-achievement-backend/database" // Pastikan ini sesuai nama module di go.mod
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load file .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 2. Inisialisasi Database (Postgres & Mongo)
	// Fungsi ini otomatis menjalankan AutoMigrate untuk Postgres
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Gagal menginisialisasi database: %v", err)
	}

	// Jika sampai sini, berarti sukses!
	log.Println("--------------------------------------------")
	log.Println("SUKSES! Database terhubung & Tabel berhasil dibuat.")
	log.Println("Koneksi Postgres:", db.Postgres)
	log.Println("Koneksi Mongo:", db.Mongo)
	log.Println("--------------------------------------------")
}