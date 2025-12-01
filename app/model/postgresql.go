package model

import (
	"time"

	"github.com/google/uuid"
)

// User merepresentasikan data pengguna sistem (admin, mahasiswa, dosen wali)
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string    `gorm:"unique;not null"`
	Email        string    `gorm:"unique;not null"`
	PasswordHash string    `gorm:"not null"`
	FullName     string    `gorm:"not null"`
	RoleID       uuid.UUID `gorm:"type:uuid;not null"`
	Role         Role      `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	IsActive     bool      `gorm:"default:true"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// Role menyimpan peran pengguna (admin, mahasiswa, dosen_wali)
type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string       `gorm:"unique;not null"`
	Description string
	Permissions []Permission `gorm:"many2many:role_permissions;"`
	Users       []User       `gorm:"foreignKey:RoleID"`
	CreatedAt   time.Time    `gorm:"autoCreateTime"`
}

// Permission menyimpan hak akses granular untuk setiap resource & action
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `gorm:"unique;not null"`
	Resource    string
	Action      string
	Description string
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// Student merepresentasikan data mahasiswa
// Kolom mengikuti SRS: id, user_id, student_id, program_study, academic_year, advisor_id, created_at, updated_at
type Student struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null"`
	User         User       `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	StudentID    string     `gorm:"type:varchar(20);not null;column:student_id"` // NIM
	ProgramStudy string     `gorm:"type:varchar(100)"`
	AcademicYear string     `gorm:"type:varchar(10)"`
	AdvisorID    *uuid.UUID `gorm:"type:uuid"` // FK ke lecturers.id
	Advisor      *Lecturer  `gorm:"foreignKey:AdvisorID"`                        // dosen wali
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}


// Lecturer merepresentasikan data dosen (termasuk dosen wali)
type Lecturer struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null"`
	User       User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	LecturerID string    `gorm:"unique;not null"` // kode/nip dosen
	Department string
	Advisees   []Student `gorm:"foreignKey:AdvisorID"` // daftar mahasiswa bimbingan
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

// AchievementReference menyimpan referensi prestasi di Postgres yang terhubung ke dokumen di Mongo
type AchievementReference struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`

	// Simpan FK ke mahasiswa (students.id), TANPA bikin relasi otomatis dua arah
	StudentID uuid.UUID `gorm:"type:uuid;not null"`

	// Kalau kamu pengin tetap punya field Student di struct utk dipakai di kode,
	// tapi TANPA ikut migrasi/foreign key, pakai gorm:"-"
	Student Student `gorm:"-"` // diabaikan saat migrasi, tapi masih bisa dipakai manual di kode

	MongoAchievementID string `gorm:"not null"` // _id dokumen di MongoDB (hex string)

	// Status mengikuti SRS + revisi: draft, submitted, verified, rejected, deleted
	Status        string     `gorm:"type:varchar(20);not null;check:status IN ('draft','submitted','verified','rejected','deleted')"`
	SubmittedAt   *time.Time // waktu mahasiswa submit prestasi
	VerifiedAt    *time.Time // waktu dosen wali/verifier memverifikasi
	VerifiedBy    *uuid.UUID `gorm:"type:uuid"` // FK ke users.id (yang memverifikasi)
	Verifier      *User      `gorm:"foreignKey:VerifiedBy"`
	RejectionNote *string    // alasan penolakan jika status rejected
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}
