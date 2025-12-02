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
	// LOAD ENV
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env tidak ditemukan, menggunakan environment default")
	}

	// INIT DATABASE (PostgreSQL + MongoDB)
	dbConn, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Gagal koneksi database: %v", err)
	}

	// SEED DATA AWAL (roles, users, students, dll.)
	database.RunSeeders(dbConn.Postgres)

	// REPOSITORY LAYER
	userRepo := repository.NewUserRepository(dbConn.Postgres)
	achievementRepo := repository.NewAchievementRepository(dbConn.Postgres, dbConn.Mongo)
	lecturerRepo := repository.NewLecturerRepository(dbConn.Postgres)
	userAdminRepo := repository.NewUserAdminRepository(dbConn.Postgres)

	// SERVICE LAYER
	authService := service.NewAuthService(userRepo)
	achievementService := service.NewAchievementService(achievementRepo, lecturerRepo)
	adminService := service.NewAdminService(userAdminRepo)

	// ROUTES
	r := gin.Default()

	routes.AuthRoutes(r, authService)
	routes.AchievementRoutes(r, achievementService)
	routes.AdminRoutes(r, adminService)

	// Root endpoint (opsional)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Student Achievement API RUNNING",
			"version": "1.0.0",
		})
	})

	// START SERVER
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running at http://localhost:" + port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("❌ Gagal menjalankan server: %v", err)
	}
}
