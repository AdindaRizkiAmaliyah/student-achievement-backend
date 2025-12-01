package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

/*
 JWTCustomClaims

 Sesuai kebutuhan sistem (dan SRS), token harus menyimpan:
 - UserID     (uuid)  : identitas user
 - StudentID  (uuid)  : identitas mahasiswa untuk fitur prestasi
                       (bisa uuid.Nil apabila user bukan mahasiswa)
 - Role       (string): nama role (admin / dosen_wali / mahasiswa)
 - Permissions([]string): daftar permission yang dimiliki user
*/
type JWTCustomClaims struct {
	UserID      uuid.UUID `json:"userId"`
	StudentID   uuid.UUID `json:"studentId"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

// getJWTSecret membaca JWT_SECRET dari environment setiap kali dipanggil.
// Ini menghindari masalah ketika .env baru di-load setelah package di-import.
func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is not configured")
	}
	return []byte(secret), nil
}

// GenerateToken membuat JWT access token yang menyimpan userID, studentID, role, dan permissions.
// Expired time saat ini diset 24 jam (access token).
func GenerateToken(userID uuid.UUID, studentID uuid.UUID, role string, permissions []string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	claims := JWTCustomClaims{
		UserID:      userID,
		StudentID:   studentID, // bisa uuid.Nil kalau bukan mahasiswa
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // masa berlaku token
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ValidateToken mem-validasi JWT dan mengembalikan *JWTCustomClaims jika valid.
// - Mengecek signing method (HMAC).
// - Menggunakan JWT_SECRET dari environment.
// - Mengecek expiration dan validitas klaim.
func ValidateToken(tokenString string) (*JWTCustomClaims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&JWTCustomClaims{},
		func(t *jwt.Token) (interface{}, error) {
			// verifikasi signing method HMAC
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTCustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
