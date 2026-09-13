package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mistake 错题本记录：收录做错的题目，记录错误原因与复盘结论，
// 按掌握状态（mastery 枚举）排期下次复习日期，到期在列表中提醒。
type Mistake struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID          primitive.ObjectID  `bson:"user_id" json:"user_id"`
	ProblemID       *primitive.ObjectID `bson:"problem_id,omitempty" json:"problem_id,omitempty"`
	Title           string              `bson:"title" json:"title"`
	KnowledgePoints []string            `bson:"knowledge_points" json:"knowledge_points"`
	ErrorReason     string              `bson:"error_reason" json:"error_reason"`
	ReviewNote      string              `bson:"review_note" json:"review_note"`
	Mastery         string              `bson:"mastery" json:"mastery"`
	NextReviewAt    time.Time           `bson:"next_review_at" json:"next_review_at"`
	LastReviewedAt  *time.Time          `bson:"last_reviewed_at,omitempty" json:"last_reviewed_at,omitempty"`
	ReviewCount     int                 `bson:"review_count" json:"review_count"`
	CreatedAt       time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time           `bson:"updated_at" json:"updated_at"`
}
