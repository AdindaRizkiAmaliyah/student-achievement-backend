package main

import (
	"log"
	"os"
	"student-achievement-backend/app/repository" // Import folder repository
	"student-achievement-backend/app/service"    // Import folder service
	"student-achievement-backend/database"       // Import folder database
	"student-achievement-backend/routes"         // Import folder routes

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Konfigurasi Environment (.env)
	// Ini wajib dilakukan paling awal agar aplikasi bisa baca password DB dll.
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 2. Inisialisasi Koneksi Database (PostgreSQL & MongoDB)
	// Fungsi InitDB ini kita ambil dari file database/connection.go
	dbConnection, err := database.InitDB()
	if err != nil {
		log.Fatalf("Gagal menginisialisasi database: %v", err)
	}

	// ==============================================================
	// DEPENDENCY INJECTION (PERAKITAN LAYER)
	// Urutan Rakit: Database -> Repository -> Service -> Handler
	// ==============================================================

	// TAHAP A: Setup Repository (Si Pengambil Data)
	// User Repository butuh koneksi Postgres
	userRepo := repository.NewUserRepository(dbConnection.Postgres)

	// TAHAP B: Setup Service (Si Otak Bisnis)
	// Auth Service butuh User Repository agar bisa cek data user
	authService := service.NewAuthService(userRepo)

	// TAHAP C: Setup Handler/Routes (Si Resepsionis)
	// Auth Handler butuh Auth Service untuk memproses logika
	authHandler := routes.NewAuthHandler(authService)

	// ==============================================================
	// SETUP WEB SERVER (GIN)
	// ==============================================================

	// 3. Buat Mesin Server Gin (Default sudah ada Logger & Recovery)
	r := gin.Default()

	// 4. Daftarkan Rute (Routes) yang sudah kita buat
	// Ini akan mengaktifkan URL seperti /api/v1/auth/login
	authHandler.SetupAuthRoutes(r)

	// 5. Endpoint Cek Kesehatan Server (Ping)
	// Gunanya cuma buat memastikan server nyala atau tidak di browser
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Student Achievement API Server is RUNNING!",
			"version": "1.0.0",
		})
	})

	// 6. Jalankan Server
	// Ambil port dari .env (APP_PORT=8080)
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080" // Default port kalau di .env kosong
	}

	log.Println("Server berjalan di http://localhost:" + appPort)
	
	// r.Run akan menahan program agar terus berjalan (Looping)
	if err := r.Run(":" + appPort); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}