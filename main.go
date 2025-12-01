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
	// ==============================================================
	//  LOAD ENV
	//  - Membaca konfigurasi dari file .env (APP_PORT, DB_HOST, JWT_SECRET, dll.)
	// ==============================================================
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env tidak ditemukan, menggunakan environment default")
	}

	// ==============================================================
	//  INIT DATABASE (PostgreSQL + MongoDB)
	//  - Koneksi ke Postgres
	//  - Migrasi tabel (Role, Permission, User, Student, Lecturer, AchievementReference)
	//  - Koneksi ke MongoDB (database achievements)
	// ==============================================================
	dbConn, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Gagal koneksi database: %v", err)
	}

	// ==============================================================
	//  SEED DATA AWAL (ROLES, PERMISSIONS, USERS, LECTURER, STUDENT)
	//  - Hanya dijalankan jika tabel masih kosong
	//  - Berguna karena tidak ada fitur registrasi di SRS (user awal di-seed)
	// ==============================================================
	database.RunSeeders(dbConn.Postgres)

	// ==============================================================
	//  REPOSITORY LAYER
	//  - Menghubungkan service dengan database (Postgres & Mongo)
	// ==============================================================
	userRepo := repository.NewUserRepository(dbConn.Postgres)
	achievementRepo := repository.NewAchievementRepository(dbConn.Postgres, dbConn.Mongo)

	// ==============================================================
	//  SERVICE LAYER
	//  - Mewakili bisnis logic sesuai SRS
	//    * AuthService: FR-001 (Login)
	//    * AchievementService: FR-003, FR-004, FR-005 (+ FR-006 list)
	// ==============================================================
	authService := service.NewAuthService(userRepo)
	achievementService := service.NewAchievementService(achievementRepo)

	// ==============================================================
	//  HTTP ROUTER (GIN)
	//  - Mendefinisikan endpoint REST API sesuai SRS
	// ==============================================================
	r := gin.Default()

	// Auth (FR-001: Login)
	routes.AuthRoutes(r, authService)

	// Achievements (FR-003: create, FR-004: submit, FR-005: delete, FR-006: list)
	routes.AchievementRoutes(r, achievementService)

	// Root endpoint (opsional, untuk health check sederhana)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Student Achievement API RUNNING",
			"version": "1.0.0",
		})
	})

	// ==============================================================
	//  START SERVER
	//  - Menggunakan APP_PORT dari .env (default: 8080)
	// ==============================================================
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running at http://localhost:" + port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("❌ Gagal menjalankan server: %v", err)
	}
}
