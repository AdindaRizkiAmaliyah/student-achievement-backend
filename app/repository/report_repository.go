package repository

import (
	"context"

	// "student-achievement-backend/app/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ReportFilter menentukan scope data statistik:
// - StudentIDs kosong  => semua mahasiswa
// - StudentIDs diisi   => hanya prestasi milik studentId tersebut (string UUID)
type ReportFilter struct {
	StudentIDs []string
}

// StudentScore menyimpan agregat per mahasiswa (untuk top students).
// StudentID dikirim sebagai string UUID (sesuai representasi di Mongo & JSON).
type StudentScore struct {
	StudentID         string `json:"studentId"`
	TotalPoints       int64  `json:"totalPoints"`
	TotalAchievements int64  `json:"totalAchievements"`
}

// ReportResult adalah struktur hasil agregasi statistik prestasi.
type ReportResult struct {
	TotalAchievements    int64            `json:"totalAchievements"`
	TotalByType          map[string]int64 `json:"totalByType"`
	TotalByPeriod        map[string]int64 `json:"totalByPeriod"` // key: "YYYY-MM"
	CompetitionLevelDist map[string]int64 `json:"competitionLevelDistribution"`
	TopStudents          []StudentScore   `json:"topStudents"`
}

// ReportRepository menangani query statistik (FR-011) ke MongoDB.
type ReportRepository interface {
	// GetStatistics menjalankan agregasi statistik berdasarkan filter studentIds.
	GetStatistics(ctx context.Context, filter ReportFilter) (*ReportResult, error)
}

// reportRepository implementasi konkrit ReportRepository.
type reportRepository struct {
	mongo *mongo.Database
}

// NewReportRepository membuat instance baru reportRepository.
func NewReportRepository(mongoDB *mongo.Database) ReportRepository {
	return &reportRepository{mongo: mongoDB}
}

// buildMatchFilter membentuk filter dasar untuk query Mongo (deleted=false + optional studentIds).
func buildMatchFilter(filter ReportFilter) bson.M {
	match := bson.M{
		"deleted": bson.M{"$ne": true}, // exclude dokumen yang sudah soft-deleted
	}

	if len(filter.StudentIDs) > 0 {
		// filter berdasarkan studentId (string UUID)
		match["studentId"] = bson.M{"$in": filter.StudentIDs}
	}

	return match
}

// GetStatistics menjalankan beberapa agregasi di MongoDB:
// - totalAchievements
// - totalByType
// - totalByPeriod (YYYY-MM dari createdAt)
// - competitionLevelDistribution
// - topStudents (berdasarkan totalPoints & jumlah prestasi)
func (r *reportRepository) GetStatistics(ctx context.Context, filter ReportFilter) (*ReportResult, error) {
	coll := r.mongo.Collection("achievements")

	match := buildMatchFilter(filter)

	result := &ReportResult{
		TotalByType:          make(map[string]int64),
		TotalByPeriod:        make(map[string]int64),
		CompetitionLevelDist: make(map[string]int64),
		TopStudents:          []StudentScore{},
	}

	// =========================
	// 1) Total achievements
	// =========================
	total, err := coll.CountDocuments(ctx, match)
	if err != nil {
		return nil, err
	}
	result.TotalAchievements = total

	// =========================
	// 2) Total by achievementType
	// =========================
	typePipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$achievementType",
			"count": bson.M{"$sum": 1},
		}}},
	}
	cur, err := coll.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.ID == "" {
			row.ID = "unknown"
		}
		result.TotalByType[row.ID] = row.Count
	}
	_ = cur.Close(ctx)

	// =========================
	// 3) Total by period (YYYY-MM dari createdAt)
	// =========================
	periodPipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"$dateToString": bson.M{
					"format": "%Y-%m",
					"date":   "$createdAt",
				},
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id": 1}}},
	}
	cur, err = coll.Aggregate(ctx, periodPipeline)
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.ID == "" {
			row.ID = "unknown"
		}
		result.TotalByPeriod[row.ID] = row.Count
	}
	_ = cur.Close(ctx)

	// =========================
	// 4) Distribusi tingkat kompetisi (details.competitionLevel)
	// =========================
	levelPipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$details.competitionLevel",
			"count": bson.M{"$sum": 1},
		}}},
	}
	cur, err = coll.Aggregate(ctx, levelPipeline)
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.ID == "" {
			row.ID = "unknown"
		}
		result.CompetitionLevelDist[row.ID] = row.Count
	}
	_ = cur.Close(ctx)

	// =========================
	// 5) Top Students (berdasarkan total points & jumlah prestasi)
	// =========================
	topPipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.M{
			"_id":              "$studentId",      // string UUID
			"totalPoints":      bson.M{"$sum": "$points"},
			"achievementCount": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{
			"totalPoints":      -1,
			"achievementCount": -1,
		}}},
		{{Key: "$limit", Value: 10}},
	}

	cur, err = coll.Aggregate(ctx, topPipeline)
	if err != nil {
		return nil, err
	}
	for cur.Next(ctx) {
		// _id adalah string (studentId)
		var row struct {
			ID               string `bson:"_id"`
			TotalPoints      int64  `bson:"totalPoints"`
			AchievementCount int64  `bson:"achievementCount"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		if row.ID == "" {
			continue // safety: skip jika studentId kosong
		}

		result.TopStudents = append(result.TopStudents, StudentScore{
			StudentID:         row.ID,
			TotalPoints:       row.TotalPoints,
			TotalAchievements: row.AchievementCount,
		})
	}
	_ = cur.Close(ctx)

	return result, nil
}
