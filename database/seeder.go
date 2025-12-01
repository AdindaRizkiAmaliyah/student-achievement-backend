package database

import (
	"log"
	"time"

	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RunSeeders menjalankan seluruh seeder yang dibutuhkan.
// Panggil ini sekali di main.go setelah InitDB berhasil.
func RunSeeders(db *gorm.DB) {
	SeedRoles(db)
	SeedPermissions(db)
	SeedRolePermissions(db)
	SeedUsers(db)
	SeedLecturerAndStudent(db)
}

// ===============================
//  SEED ROLES
// ===============================

// SeedRoles menambahkan 3 role utama sesuai SRS:
// admin, dosen_wali, mahasiswa.
func SeedRoles(db *gorm.DB) {
	var count int64
	db.Model(&model.Role{}).Count(&count)
	if count > 0 {
		log.Println("[SEEDER] Role sudah ada, skip seeding roles.")
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
//  SEED PERMISSIONS
// ===============================

// SeedPermissions menambahkan daftar permission dasar
// untuk modul prestasi dan user management (sesuai konsep RBAC SRS).
func SeedPermissions(db *gorm.DB) {
	var count int64
	db.Model(&model.Permission{}).Count(&count)
	if count > 0 {
		log.Println("[SEEDER] Permission sudah ada, skip seeding.")
		return
	}

	perms := []model.Permission{
		// Modul achievement
		{Name: "achievement:create", Resource: "achievement", Action: "create"},
		{Name: "achievement:read", Resource: "achievement", Action: "read"},
		{Name: "achievement:update", Resource: "achievement", Action: "update"},
		{Name: "achievement:delete", Resource: "achievement", Action: "delete"},
		{Name: "achievement:verify", Resource: "achievement", Action: "verify"},

		// Modul user management
		{Name: "user:read", Resource: "user", Action: "read"},
		{Name: "user:create", Resource: "user", Action: "create"},
		{Name: "user:update", Resource: "user", Action: "update"},
		{Name: "user:delete", Resource: "user", Action: "delete"},
	}

	if err := db.Create(&perms).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal seed permissions: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed permissions")
}

// ===============================
//  SEED ROLE-PERMISSIONS
// ===============================

// SeedRolePermissions mengaitkan role dengan permission
// (mengisi tabel many2many role_permissions).
func SeedRolePermissions(db *gorm.DB) {
	// Cek apakah sudah ada role_permissions (approx dengan cek association)
	var adminRole model.Role
	if err := db.Preload("Permissions").Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		log.Println("[SEEDER] Role admin belum ada, skip role_permissions (pastikan SeedRoles & SeedPermissions jalan dulu).")
		return
	}
	if len(adminRole.Permissions) > 0 {
		log.Println("[SEEDER] role_permissions sudah terisi, skip.")
		return
	}

	// Ambil semua permissions
	var perms []model.Permission
	if err := db.Find(&perms).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal mengambil permissions: %v", err)
	}

	// Ambil roles
	var admin, doswal, mhs model.Role
	db.Where("name = ?", "admin").First(&admin)
	db.Where("name = ?", "dosen_wali").First(&doswal)
	db.Where("name = ?", "mahasiswa").First(&mhs)

	// Admin: semua permission
	if err := db.Model(&admin).Association("Permissions").Append(&perms); err != nil {
		log.Fatalf("[SEEDER] Gagal assign permission ke admin: %v", err)
	}

	// Dosen wali: dapat read + verify achievement
	var doswalPerms []model.Permission
	for _, p := range perms {
		if p.Name == "achievement:read" || p.Name == "achievement:verify" {
			doswalPerms = append(doswalPerms, p)
		}
	}
	if err := db.Model(&doswal).Association("Permissions").Append(&doswalPerms); err != nil {
		log.Fatalf("[SEEDER] Gagal assign permission ke dosen_wali: %v", err)
	}

	// Mahasiswa: create/read/update/delete achievement milik sendiri
	var mhsPerms []model.Permission
	for _, p := range perms {
		switch p.Name {
		case "achievement:create", "achievement:read", "achievement:update", "achievement:delete":
			mhsPerms = append(mhsPerms, p)
		}
	}
	if err := db.Model(&mhs).Association("Permissions").Append(&mhsPerms); err != nil {
		log.Fatalf("[SEEDER] Gagal assign permission ke mahasiswa: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed role_permissions")
}

// ===============================
//  SEED USERS
// ===============================

// SeedUsers menambahkan 3 user awal:
// - admin
// - doswal (dosen wali)
// - mahasiswa1
func SeedUsers(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		log.Println("[SEEDER] User sudah ada, skip seeding.")
		return
	}

	// Ambil role ID
	var adminRole, doswalRole, mhsRole model.Role

	db.Where("name = ?", "admin").First(&adminRole)
	db.Where("name = ?", "dosen_wali").First(&doswalRole)
	db.Where("name = ?", "mahasiswa").First(&mhsRole)

	password := "123123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	now := time.Now()

	users := []model.User{
		{
			ID:           uuid.New(),
			Username:     "admin",
			Email:        "admin@kampus.ac.id",
			PasswordHash: string(hash),
			FullName:     "Admin Sistem",
			RoleID:       adminRole.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New(),
			Username:     "doswal",
			Email:        "doswal@kampus.ac.id",
			PasswordHash: string(hash),
			FullName:     "Dosen Wali",
			RoleID:       doswalRole.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           uuid.New(),
			Username:     "mahasiswa1",
			Email:        "mahasiswa1@kampus.ac.id",
			PasswordHash: string(hash),
			FullName:     "Mahasiswa Satu",
			RoleID:       mhsRole.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}

	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal seed users: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed 3 user (admin, doswal, mahasiswa), password: 123123")
}

// ===============================
//  SEED LECTURER & STUDENT
// ===============================

// SeedLecturerAndStudent membuat 1 entri lecturer dan 1 entri student
// yang terhubung dengan user 'doswal' dan 'mahasiswa1',
// sehingga fitur FR-03 s.d. FR-05 bisa langsung digunakan.
func SeedLecturerAndStudent(db *gorm.DB) {
	// Cek kalau sudah ada student, skip
	var studentCount int64
	db.Model(&model.Student{}).Count(&studentCount)
	if studentCount > 0 {
		log.Println("[SEEDER] Student sudah ada, skip seeding mahasiswa & dosen wali.")
		return
	}

	// Ambil user doswal & mahasiswa1
	var doswalUser, mhsUser model.User
	if err := db.Where("username = ?", "doswal").First(&doswalUser).Error; err != nil {
		log.Println("[SEEDER] User 'doswal' tidak ditemukan, skip seeding lecturer.")
		return
	}
	if err := db.Where("username = ?", "mahasiswa1").First(&mhsUser).Error; err != nil {
		log.Println("[SEEDER] User 'mahasiswa1' tidak ditemukan, skip seeding student.")
		return
	}

	now := time.Now()

	// Buat lecturer untuk doswal
	lect := model.Lecturer{
		ID:         uuid.New(),
		UserID:     doswalUser.ID,
		LecturerID: "L001",
		Department: "Teknik Informatika",
		CreatedAt:  now,
	}
	if err := db.Create(&lect).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal membuat lecturer: %v", err)
	}

	// Buat student untuk mahasiswa1, dengan doswal sebagai advisor
	stu := model.Student{
		ID:           uuid.New(),
		UserID:       mhsUser.ID,
		StudentID:    "230001", // NIM contoh
		ProgramStudy: "Teknik Informatika",
		AcademicYear: "2023/2024",
		AdvisorID:    &lect.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := db.Create(&stu).Error; err != nil {
		log.Fatalf("[SEEDER] Gagal membuat student: %v", err)
	}

	log.Println("[SEEDER] Berhasil seed lecturer & student awal")
}
