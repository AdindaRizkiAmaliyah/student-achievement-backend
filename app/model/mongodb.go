package model

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Achievement merepresentasikan 1 dokumen prestasi di MongoDB (collection: achievements)
// Struktur mengikuti definisi di SRS bagian 3.2.1 Collection achievements.
type Achievement struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`     // _id dokumen Mongo
	StudentID       uuid.UUID         `bson:"studentId"`         // ID mahasiswa (sama dengan students.id di Postgres)
	AchievementType string            `bson:"achievementType"`   // tipe prestasi: competition/publication/organization/certification
	Title           string            `bson:"title"`             // judul prestasi
	Description     string            `bson:"description"`       // deskripsi singkat
	Details         AchievementDetails `bson:"details"`          // detail spesifik tergantung tipe
	Attachments     []Attachment       `bson:"attachments"`      // daftar lampiran bukti
	Tags            []string           `bson:"tags"`             // tag/tagline pendukung
	Points          int                `bson:"points"`           // bobot poin prestasi
	CreatedAt       time.Time          `bson:"createdAt"`        // tanggal dibuat
	UpdatedAt       time.Time          `bson:"updatedAt"`        // tanggal terakhir diupdate
}

// AchievementDetails menyimpan field dinamis (competition/publication/organization/certification)
// Field-field ini langsung mengikuti nama di SRS, tanpa penambahan.
type AchievementDetails struct {
	// Competition fields
	CompetitionName  *string    `bson:"competitionName,omitempty"`  // competitionName
	CompetitionLevel *string    `bson:"competitionLevel,omitempty"` // competitionLevel: international/national/regional/local
	Rank             *int       `bson:"rank,omitempty"`             // rank
	MedalType        *string    `bson:"medalType,omitempty"`        // medalType

	// Publication fields
	PublicationType  *string   `bson:"publicationType,omitempty"`  // publicationType: journal/conference/book
	PublicationTitle *string   `bson:"publicationTitle,omitempty"` // publicationTitle
	Authors          []string  `bson:"authors,omitempty"`          // authors: array string
	Publisher        *string   `bson:"publisher,omitempty"`        // publisher
	ISSN             *string   `bson:"issn,omitempty"`             // issn

	// Organization fields
	OrganizationName *string   `bson:"organizationName,omitempty"` // organizationName
	Position         *string   `bson:"position,omitempty"`         // position
	Period           *Period   `bson:"period,omitempty"`           // period: { start, end }

	// Certification fields
	CertificationName   *string    `bson:"certificationName,omitempty"`   // certificationName
	IssuedBy            *string    `bson:"issuedBy,omitempty"`            // issuedBy
	CertificationNumber *string    `bson:"certificationNumber,omitempty"` // certificationNumber
	ValidUntil          *time.Time `bson:"validUntil,omitempty"`          // validUntil

	// Common fields
	EventDate *time.Time `bson:"eventDate,omitempty"` // eventDate
	Location  *string    `bson:"location,omitempty"`  // location
	Organizer *string    `bson:"organizer,omitempty"` // organizer
	Score     *float64   `bson:"score,omitempty"`     // score

	// CustomFields dipakai untuk field tambahan yang tidak terdefinisi di SRS.
	// Misal: customFields["isDeleted"] = true, dsb.
	CustomFields map[string]any `bson:"customFields,omitempty"` // customFields
}

// Period merepresentasikan periode (untuk organization) dengan start dan end date.
type Period struct {
	Start *time.Time `bson:"start,omitempty"` // start
	End   *time.Time `bson:"end,omitempty"`   // end
}

// Attachment merepresentasikan 1 lampiran (file bukti) prestasi.
// Nama type ini sengaja disamakan dengan yang dipakai di service ([]model.Attachment).
type Attachment struct {
	FileName   string    `bson:"fileName"`   // fileName
	FileURL    string    `bson:"fileUrl"`    // fileUrl
	FileType   string    `bson:"fileType"`   // fileType (pdf/jpg/dll)
	UploadedAt time.Time `bson:"uploadedAt"` // uploadedAt
}
