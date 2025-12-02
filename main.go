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

	// =================================================================
	// LOAD ENV
	// =================================================================
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  .env tidak ditemukan, menggunakan environment default")
	}

	// =================================================================
	// INIT DB (POSTGRES + MONGODB)
	// =================================================================
	dbConn, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Gagal koneksi database: %v", err)
	}

	// =================================================================
	// SEED DATA (ROLES + USERS)
	// =================================================================
	database.SeedRoles(dbConn.Postgres)
	database.SeedUsers(dbConn.Postgres)

	// =================================================================
	// REPOSITORIES
	// =================================================================
	userRepo := repository.NewUserRepository(dbConn.Postgres)
	achievementRepo := repository.NewAchievementRepository(dbConn.Postgres, dbConn.Mongo)
	lecturerRepo := repository.NewLecturerRepository(dbConn.Postgres)
	adminRepo := repository.NewUserAdminRepository(dbConn.Postgres)
	reportRepo := repository.NewReportRepository(dbConn.Mongo)

	// =================================================================
	// SERVICES
	// =================================================================
	authService := service.NewAuthService(userRepo)
	adminService := service.NewAdminService(adminRepo)
	achievementService := service.NewAchievementService(
		achievementRepo,
		userRepo,
		lecturerRepo,
	)
	reportService := service.NewReportService(reportRepo, lecturerRepo)

	// =================================================================
	// ROUTER
	// =================================================================
	r := gin.Default()

	// Auth (FR-001)
	routes.AuthRoutes(r, authService)

	// Admin user management (FR-009)
	routes.AdminRoutes(r, adminService)

	// Achievements (FR-003 s.d. FR-010)
	routes.AchievementRoutes(r, achievementService)

	// Reports & Analytics (FR-011)
	routes.ReportRoutes(r, reportService)

	// Root endpoint (optional)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Student Achievement API RUNNING",
			"version": "1.0.0",
		})
	})

	// =================================================================
	// START SERVER
	// =================================================================
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server running at http://localhost:" + port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("❌ Gagal menjalankan server: %v", err)
	}
}
