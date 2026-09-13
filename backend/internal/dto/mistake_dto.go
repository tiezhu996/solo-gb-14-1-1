package dto

import (
	"time"

	"github.com/blueship581/codelearn/internal/model"
)

// CreateMistakeRequest 收录错题请求。
type CreateMistakeRequest struct {
	Title           string   `json:"title" binding:"required,max=200"`
	ProblemID       string   `json:"problem_id" binding:"omitempty"`
	KnowledgePoints []string `json:"knowledge_points" binding:"omitempty,max=10,dive,max=50"`
	ErrorReason     string   `json:"error_reason" binding:"omitempty,max=2000"`
	ReviewNote      string   `json:"review_note" binding:"omitempty,max=2000"`
	Mastery         string   `json:"mastery" binding:"omitempty"`
	NextReviewAt    string   `json:"next_review_at" binding:"omitempty"`
}

// UpdateMistakeRequest 修改错题请求：指针/切片字段，未传（nil）表示不修改，
// 掌握状态 mastery 未提供时保留当前值。
type UpdateMistakeRequest struct {
	Title           *string  `json:"title" binding:"omitempty,max=200"`
	KnowledgePoints []string `json:"knowledge_points" binding:"omitempty,max=10,dive,max=50"`
	ErrorReason     *string  `json:"error_reason" binding:"omitempty,max=2000"`
	ReviewNote      *string  `json:"review_note" binding:"omitempty,max=2000"`
	Mastery         *string  `json:"mastery" binding:"omitempty"`
	NextReviewAt    *string  `json:"next_review_at" binding:"omitempty"`
}

// ReviewMistakeRequest 完成一次复习请求：更新掌握状态并排期下次复习。
type ReviewMistakeRequest struct {
	Mastery      string  `json:"mastery" binding:"required"`
	NextReviewAt *string `json:"next_review_at" binding:"omitempty"`
}

// MistakeListQuery 错题列表查询条件：按题目关键词、知识点、掌握状态、是否到期筛选。
type MistakeListQuery struct {
	Q              string
	KnowledgePoint string
	Mastery        string
	DueOnly        bool
	Page           int64
	PageSize       int64
}

// MistakeResponse 错题记录响应。
type MistakeResponse struct {
	ID              string   `json:"id"`
	ProblemID       string   `json:"problem_id,omitempty"`
	Title           string   `json:"title"`
	KnowledgePoints []string `json:"knowledge_points"`
	ErrorReason     string   `json:"error_reason"`
	ReviewNote      string   `json:"review_note"`
	Mastery         string   `json:"mastery"`
	NextReviewAt    string   `json:"next_review_at"`
	LastReviewedAt  string   `json:"last_reviewed_at,omitempty"`
	ReviewCount     int      `json:"review_count"`
	Due             bool     `json:"due"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

// ToMistakeResponse 将错题模型转换为响应（due 表示已到复习时间）。
// next_review_at 按 UTC 零点日历日期格式化，与输入日期原样一致，不受服务器时区影响。
func ToMistakeResponse(m *model.Mistake) MistakeResponse {
	resp := MistakeResponse{
		ID:              m.ID.Hex(),
		Title:           m.Title,
		KnowledgePoints: m.KnowledgePoints,
		ErrorReason:     m.ErrorReason,
		ReviewNote:      m.ReviewNote,
		Mastery:         m.Mastery,
		NextReviewAt:    m.NextReviewAt.UTC().Format("2006-01-02"),
		ReviewCount:     m.ReviewCount,
		Due:             !m.NextReviewAt.After(time.Now()),
		CreatedAt:       m.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       m.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if m.ProblemID != nil {
		resp.ProblemID = m.ProblemID.Hex()
	}
	if resp.KnowledgePoints == nil {
		resp.KnowledgePoints = []string{}
	}
	if m.LastReviewedAt != nil {
		resp.LastReviewedAt = m.LastReviewedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}
