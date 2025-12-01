package database

import (
	"log"
	"time"

	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ===============================
//  SEED ROLES (admin, dosen_wali, mahasiswa)
// ===============================
func SeedRoles(db *gorm.DB) {
	var count int64
	db.Model(&model.Role{}).Count(&count)
	if count > 0 {
		log.Println("[SEEDER] Role sudah ada, skip seeding.")
		return
	}

	roles := []model.Role{
		{ID: uuid.New(), Name: "admin"},
		{ID: uuid.New(), Name: "dosen_wali"},
		{ID: uuid.New(), Name: "mahasiswa"},
	}

	if err := db.Create(&roles).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal seed roles: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed role: admin, dosen_wali, mahasiswa")
}

// ===============================
//  SEED USERS AWAL (admin, doswal, 1 mahasiswa)
//   - Hanya jalan kalau tabel users masih kosong
// ===============================
func SeedUsers(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		log.Println("[SEEDER] User sudah ada, skip seeding awal.")
		return
	}

	// Ambil role ID
	var adminRole, doswalRole, mhsRole model.Role
	db.Where("name = ?", "admin").First(&adminRole)
	db.Where("name = ?", "dosen_wali").First(&doswalRole)
	db.Where("name = ?", "mahasiswa").First(&mhsRole)

	password := "123123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	users := []model.User{
		{
			ID:           uuid.New(),
			Username:     "admin",
			Email:        "admin@kampus.ac.id",
			PasswordHash: string(hash),
			FullName:     "Admin Sistem",
			RoleID:       adminRole.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           uuid.New(),
			Username:     "doswal",
			Email:        "doswal@kampus.ac.id",
			PasswordHash: string(hash),
			FullName:     "Dosen Wali",
			RoleID:       doswalRole.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           uuid.New(),
			Username:     "mahasiswa1",
			Email:        "mahasiswa1@kampus.ac.id",
			PasswordHash: string(hash),
			FullName:     "Mahasiswa Satu",
			RoleID:       mhsRole.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal seed users: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed 3 user (admin, doswal, mahasiswa1), password: 123123")
}

// ===============================
//  SEED MAHASISWA KEDUA
//  - Tambah user "mahasiswa2"
//  - Tambah record di tabel students untuk user tsb
//  - Boleh dipanggil berulang, tidak akan duplikasi
// ===============================
func SeedMahasiswaKedua(db *gorm.DB) {
	// Cek apakah user dengan username "mahasiswa2" sudah ada
	var existingUser model.User
	if err := db.Where("username = ?", "mahasiswa2").First(&existingUser).Error; err == nil {
		log.Println("[SEEDER] mahasiswa2 sudah ada, skip.")
		return
	}

	// Ambil role mahasiswa
	var mhsRole model.Role
	if err := db.Where("name = ?", "mahasiswa").First(&mhsRole).Error; err != nil {
		log.Printf("[SEEDER] Role 'mahasiswa' tidak ditemukan: %v", err)
		return
	}

	// Hash password (pakai password yang sama: 123123)
	password := "123123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	// Buat user baru untuk mahasiswa2
	newUser := model.User{
		ID:           uuid.New(),
		Username:     "mahasiswa2",
		Email:        "mahasiswa2@kampus.ac.id",
		PasswordHash: string(hash),
		FullName:     "Mahasiswa Dua",
		RoleID:       mhsRole.ID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&newUser).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal membuat user mahasiswa2: %v", err)
	}

	// Cari 1 lecturer (dosen wali) sebagai advisor, jika ada
	var lecturer model.Lecturer
	var advisorID *uuid.UUID
	if err := db.First(&lecturer).Error; err == nil {
		advisorID = &lecturer.ID
	}

	// Buat record student untuk mahasiswa2
	newStudent := model.Student{
		ID:           uuid.New(),
		UserID:       newUser.ID,
		StudentID:    "24010002",       // NIM untuk mahasiswa2 (silakan sesuaikan)
		ProgramStudy: "Informatika",    // contoh prodi
		AcademicYear: "2024",           // contoh tahun akademik
		AdvisorID:    advisorID,        // bisa nil kalau belum ada lecturer
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := db.Create(&newStudent).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal membuat student untuk mahasiswa2: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed mahasiswa kedua (mahasiswa2), password: 123123, NIM: 24010002")
}

// ===============================
//  RUN ALL SEEDERS
//  - Panggil ini dari main.go
// ===============================
func RunSeeders(db *gorm.DB) {
	SeedRoles(db)
	SeedUsers(db)
	SeedMahasiswaKedua(db)
}
