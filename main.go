package main

import (
	"log"
	"os"
	"student-achievement-backend/app/repository"
	"student-achievement-backend/app/service"
	"student-achievement-backend/database"
	"student-achievement-backend/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// ... (Kode Load Env dan InitDB SAMA SEPERTI SEBELUMNYA) ...
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dbConnection, err := database.InitDB()
	if err != nil {
		log.Fatalf("Gagal menginisialisasi database: %v", err)
	}

	// ==============================================================
	// DEPENDENCY INJECTION
	// ==============================================================

	// 1. Repository
	userRepo := repository.NewUserRepository(dbConnection.Postgres)
	// UPDATE BARU: Tambahkan Achievement Repository
	achievementRepo := repository.NewAchievementRepository(dbConnection.Postgres, dbConnection.Mongo)

	// 2. Service
	authService := service.NewAuthService(userRepo)
	// UPDATE BARU: Tambahkan Achievement Service
	achievementService := service.NewAchievementService(achievementRepo)

	// 3. Handler
	authHandler := routes.NewAuthHandler(authService)
	// UPDATE BARU: Tambahkan Achievement Handler
	achievementHandler := routes.NewAchievementHandler(achievementService)

	// ==============================================================
	// SERVER SETUP
	// ==============================================================
	r := gin.Default()

	// Setup Routes
	authHandler.SetupAuthRoutes(r)
	// UPDATE BARU: Daftarkan rute prestasi
	achievementHandler.SetupAchievementRoutes(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Student Achievement API Server is RUNNING!",
			"version": "1.0.0",
		})
	})

	// ... (Kode Menjalankan Server SAMA SEPERTI SEBELUMNYA) ...
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}
	log.Println("Server berjalan di http://localhost:" + appPort)
	if err := r.Run(":" + appPort); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}