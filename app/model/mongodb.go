package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Achievement merepresentasikan 1 dokumen prestasi di MongoDB (collection: achievements)
// Struktur mengikuti definisi di SRS bagian 3.2.1 Collection achievements.
type Achievement struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty"`          // _id: ObjectId
	StudentID       string              `bson:"studentId"`              // studentId: UUID (disimpan sebagai string, refer ke PostgreSQL students.id)
	AchievementType string              `bson:"achievementType"`        // achievementType: 'academic', 'competition', dll.
	Title           string              `bson:"title"`                  // title: judul prestasi
	Description     string              `bson:"description"`            // description: deskripsi prestasi
	Details         AchievementDetails  `bson:"details"`                // details: field dinamis sesuai tipe prestasi
	Attachments     []AchievementFile   `bson:"attachments,omitempty"`  // attachments: array dokumen pendukung
	Tags            []string            `bson:"tags,omitempty"`         // tags: label bebas
	Points          float64             `bson:"points"`                 // points: poin prestasi untuk scoring
	CreatedAt       time.Time           `bson:"createdAt"`              // createdAt: waktu dibuat
	UpdatedAt       time.Time           `bson:"updatedAt"`              // updatedAt: waktu diupdate
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
	CertificationName   *string   `bson:"certificationName,omitempty"`   // certificationName
	IssuedBy            *string   `bson:"issuedBy,omitempty"`            // issuedBy
	CertificationNumber *string   `bson:"certificationNumber,omitempty"` // certificationNumber
	ValidUntil          *time.Time `bson:"validUntil,omitempty"`         // validUntil

	// Common fields
	EventDate *time.Time `bson:"eventDate,omitempty"` // eventDate
	Location  *string    `bson:"location,omitempty"`  // location
	Organizer *string    `bson:"organizer,omitempty"` // organizer
	Score     *float64   `bson:"score,omitempty"`     // score

	// CustomFields dipakai untuk field tambahan yang tidak terdefinisi di SRS.
	// Soft delete bisa diletakkan di sini, contoh: customFields["isDeleted"] = true
	CustomFields map[string]any `bson:"customFields,omitempty"` // customFields
}

// Period merepresentasikan periode (untuk organization) dengan start dan end date.
type Period struct {
	Start *time.Time `bson:"start,omitempty"` // start
	End   *time.Time `bson:"end,omitempty"`   // end
}

// AchievementFile merepresentasikan 1 lampiran (file bukti) prestasi.
type AchievementFile struct {
	FileName   string    `bson:"fileName"`   // fileName
	FileURL    string    `bson:"fileUrl"`    // fileUrl
	FileType   string    `bson:"fileType"`   // fileType (pdf/jpg/dll)
	UploadedAt time.Time `bson:"uploadedAt"` // uploadedAt
}
