package service

import (
	"errors"
	"student-achievement-backend/app/model"
	"student-achievement-backend/app/repository"

	"golang.org/x/crypto/bcrypt"
)

// Interface AuthService mendefinisikan apa saja yang bisa dilakukan layanan ini.
type AuthService interface {
	Register(user *model.User) error
	Login(email, password string) (*model.User, error)
}

type authService struct {
	userRepo repository.UserRepository
}

// NewAuthService menghubungkan Service dengan Repository
func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{
		userRepo: userRepo,
	}
}

// Register: Mendaftarkan user baru (Admin/Mahasiswa/Dosen)
func (s *authService) Register(user *model.User) error {
	// 1. Hash Password (Keamanan)
	// Kita ubah password "rahasia123" menjadi text acak "$2a$10$..."
	// agar admin database pun tidak tahu password asli user.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	// 2. Simpan hasil hash kembali ke struct user
	user.PasswordHash = string(hashedPassword)

	// 3. Panggil Repository untuk simpan ke Database PostgreSQL
	return s.userRepo.Create(user)
}

// Login: Memeriksa apakah email dan password cocok (FR-001) [cite: 163-166]
func (s *authService) Login(email, password string) (*model.User, error) {
	// 1. Cari user berdasarkan email lewat Repository
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("email tidak ditemukan")
	}

	// 2. Cek Password
	// Bandingkan password inputan ("12345") dengan hash di database ("$2a$10$...")
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("password salah")
	}

	// 3. Cek apakah user aktif [cite: 166]
	if !user.IsActive {
		return nil, errors.New("akun anda dinonaktifkan")
	}

	// Jika sukses, kembalikan data user (nanti Controller yang akan bikin Token JWT)
	return user, nil
}