package repository

import (
	"student-achievement-backend/app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository adalah kontrak/interface.
// Gunanya: Agar nanti Service tidak peduli database apa yang dipakai,
// asalkan punya fungsi Create, FindByEmail, dan FindByID.
type UserRepository interface {
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByID(id uuid.UUID) (*model.User, error)
}

// userRepository adalah implementasi aslinya yang memegang koneksi database GORM.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository adalah fungsi pembuat (Constructor).
// Service akan memanggil ini untuk mendapatkan akses ke repository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create bertugas menyimpan data user baru ke tabel 'users'.
func (r *userRepository) Create(user *model.User) error {
	// db.Create adalah fungsi bawaan GORM untuk melakukan INSERT SQL.
	return r.db.Create(user).Error
}
// FindByEmail sangat PENTING untuk fitur LOGIN
// Kita mencari user berdasarkan email untuk mengecek password nanti.
func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	
	// Chain Method GORM:
	// 1. Preload("Role.Permissions"): "Tolong ambilkan juga data Role dan Permission-nya sekaligus".
	//    Ini penting agar nanti kita tahu user ini boleh ngapain aja (RBAC).
	// 2. Where("email = ?", email): "Cari yang emailnya sama".
	// 3. First(&user): "Ambil data pertama yang ketemu dan masukkan ke variabel user".
	err := r.db.Preload("Role.Permissions").
		Where("email = ?", email).
		First(&user).Error
	
	if err != nil {
		return nil, err // Balikkan error jika user tidak ditemukan
	}
	return &user, nil // Balikkan data user jika ketemu
}

// FindByID mencari user berdasarkan ID uniknya (UUID).
// Biasa dipakai untuk mengambil profil user yang sedang login.
func (r *userRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	// Kita Preload "Role" supaya tahu jabatannya apa (Mahasiswa/Admin/Dosen)
	err := r.db.Preload("Role").Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}