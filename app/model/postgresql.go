package model

import (
	"time"

	"github.com/google/uuid"
)

// 3.1.1 Tabel users
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username     string    `gorm:"type:varchar(50);unique;not null" json:"username"`
	Email        string    `gorm:"type:varchar(100);unique;not null" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"` // Password tidak di-return di JSON
	FullName     string    `gorm:"type:varchar(100);not null" json:"fullName"`
	RoleID       uuid.UUID `gorm:"type:uuid;not null" json:"roleId"`
	Role         Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	IsActive     bool      `gorm:"default:true" json:"isActive"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// 3.1.2 Tabel roles
type Role struct {
	ID          uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string           `gorm:"type:varchar(50);unique;not null" json:"name"`
	Description string           `gorm:"type:text" json:"description"`
	CreatedAt   time.Time        `gorm:"autoCreateTime" json:"createdAt"`
	Permissions []Permission     `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// 3.1.3 Tabel permissions
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(100);unique;not null" json:"name"`
	Resource    string    `gorm:"type:varchar(50);not null" json:"resource"`
	Action      string    `gorm:"type:varchar(50);not null" json:"action"`
	Description string    `gorm:"type:text" json:"description"`
}

// 3.1.4 Tabel role_permissions (Junction Table)
// Note: GORM menangani ini via many2many, tapi jika butuh explicit struct:
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"primaryKey" json:"roleId"`
	PermissionID uuid.UUID `gorm:"primaryKey" json:"permissionId"`
}

// 3.1.5 Tabel students 
type Student struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	User         User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	StudentID    string    `gorm:"type:varchar(20);unique;not null" json:"studentId"` // NIM
	ProgramStudy string    `gorm:"type:varchar(100)" json:"programStudy"`
	AcademicYear string    `gorm:"type:varchar(10)" json:"academicYear"`
	AdvisorID    uuid.UUID `gorm:"type:uuid" json:"advisorId"`
	Advisor      Lecturer  `gorm:"foreignKey:AdvisorID" json:"advisor,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// 3.1.6 Tabel lecturers
type Lecturer struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null" json:"userId"`
	User       User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	LecturerID string    `gorm:"type:varchar(20);unique;not null" json:"lecturerId"` // NIP/NIDN
	Department string    `gorm:"type:varchar(100)" json:"department"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// 3.1.7 Tabel achievement_references
type AchievementReference struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	StudentID          uuid.UUID  `gorm:"type:uuid;not null" json:"studentId"`
	Student            Student    `gorm:"foreignKey:StudentID" json:"student,omitempty"`
	MongoAchievementID string     `gorm:"type:varchar(24);not null" json:"mongoAchievementId"`
	Status             string     `gorm:"type:varchar(20);default:'draft'" json:"status"` // Enum: draft, submitted, verified, rejected
	SubmittedAt        *time.Time `json:"submittedAt"`
	VerifiedAt         *time.Time `json:"verifiedAt"`
	VerifiedBy         *uuid.UUID `gorm:"type:uuid" json:"verifiedBy"`
	Verifier           *User      `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
	RejectionNote      string     `gorm:"type:text" json:"rejectionNote"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}
