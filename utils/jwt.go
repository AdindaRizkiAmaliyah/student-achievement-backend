package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTSecret adalah "kunci rahasia" dapur.
// Hanya server yang tahu kunci ini untuk tanda tangan token.
// Kita ambil nilainya dari file .env agar aman.
var JWTSecret = []byte(os.Getenv("JWT_SECRET"))

// JWTCustomClaims mendefinisikan isi "daging" data di dalam token.
// Sesuai SRS, kita butuh Role dan Permission untuk pengecekan hak akses nanti.
type JWTCustomClaims struct {
	UserID      uuid.UUID `json:"user_id"`      // ID User pelapor
	Role        string    `json:"role"`         // Admin / Mahasiswa / Dosen Wali
	Permissions []string  `json:"permissions"`  // Daftar izin (misal: achievement:create)
	jwt.RegisteredClaims                        // Standar bawaan JWT (expired, issuer, dll)
}

// GenerateToken berfungsi membuat token baru saat user berhasil login.
// Token ini berlaku selama 24 jam.
func GenerateToken(userID uuid.UUID, role string, permissions []string) (string, error) {
	// 1. Siapkan isi "karcis" (Claims)
	claims := JWTCustomClaims{
		UserID:      userID,
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			// Token akan kadaluarsa dalam 24 jam dari sekarang
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			// Waktu token dibuat
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			// Identitas penerbit token (nama aplikasi kita)
			Issuer:    "student-achievement-backend",
		},
	}

	// 2. Pilih metode enkripsi (Signing Method)
	// HS256 adalah standar industri yang umum, cepat, dan aman.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 3. Tanda tangani token dengan kunci rahasia (JWT_SECRET)
	// Hasilnya adalah string panjang (eyJhbGciOiJIUzI1NiIs...)
	t, err := token.SignedString(JWTSecret)
	if err != nil {
		return "", err // Gagal bikin token
	}

	return t, nil // Berhasil, kembalikan string token
}

// ValidateToken akan dipakai oleh Middleware nanti untuk mengecek:
// "Apakah token ini asli buatan server kita atau palsu?"
func ValidateToken(tokenString string) (*JWTCustomClaims, error) {
	// 1. Parse (bongkar) token string kembali menjadi objek token
	token, err := jwt.ParseWithClaims(tokenString, &JWTCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validasi keamanan: Pastikan metode enkripsinya adalah HMAC (HS256).
		// Kalau metodenya beda (misal 'none'), tolak karena itu trik hacker.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		// Jika aman, kembalikan kunci rahasia untuk verifikasi tanda tangan
		return JWTSecret, nil
	})

	// 2. Cek apakah ada error saat parsing (misal token sudah kadaluarsa/expired)
	if err != nil {
		return nil, err
	}

	// 3. Jika token valid dan isinya bisa dibaca sebagai JWTCustomClaims...
	if claims, ok := token.Claims.(*JWTCustomClaims); ok && token.Valid {
		return claims, nil // Kembalikan data user (ID, Role, dll)
	}

	// 4. Jika token strukturnya aneh atau tidak valid
	return nil, errors.New("token invalid atau tidak dikenali")
}