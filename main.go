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

// main adalah entrypoint aplikasi:
// - load .env
// - init PostgreSQL + MongoDB
// - seed roles & users default
// - inisialisasi repository, service, dan routes
// - menjalankan HTTP server Gin
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
	// REPOSITORIES (akses data ke DB)
	// =================================================================
	userRepo := repository.NewUserRepository(dbConn.Postgres)
	achievementRepo := repository.NewAchievementRepository(dbConn.Postgres, dbConn.Mongo)
	studentRepo := repository.NewStudentRepository(dbConn.Postgres)
	lecturerRepo := repository.NewLecturerRepository(dbConn.Postgres)
	adminRepo := repository.NewUserAdminRepository(dbConn.Postgres)
	reportRepo := repository.NewReportRepository(dbConn.Mongo)

	// =================================================================
	// SERVICES (logic & handler HTTP)
	// =================================================================
	authService := service.NewAuthService(userRepo)
	adminService := service.NewAdminService(adminRepo)
	achievementService := service.NewAchievementService(
		achievementRepo,
		userRepo,
		lecturerRepo,
	)
	reportService := service.NewReportService(reportRepo, lecturerRepo)
	// StudentService butuh studentRepo + achievementRepo
	studentService := service.NewStudentService(studentRepo, achievementRepo)
	// LecturerService versi kamu saat ini hanya butuh lecturerRepo
	lecturerService := service.NewLecturerService(lecturerRepo)

	// =================================================================
	// ROUTER (registrasi endpoint sesuai SRS)
	// =================================================================
	r := gin.Default()

	// 5.1 Authentication
	routes.AuthRoutes(r, authService)

	// 5.2 Users (Admin)
	routes.AdminRoutes(r, adminService)

	// 5.4 Achievements
	routes.AchievementRoutes(r, achievementService)

	// 5.8 Reports & Analytics
	routes.ReportRoutes(r, reportService)

	// 5.5 Students & Lecturers
	routes.StudentRoutes(r, studentService)
	routes.LecturerRoutes(r, lecturerService)

	// Root endpoint (optional health check)
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
