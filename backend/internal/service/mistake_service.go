package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/dto"
	"github.com/blueship581/codelearn/internal/model"
	"github.com/blueship581/codelearn/internal/repository"
	"github.com/blueship581/codelearn/internal/util"
)

// MistakeService 错题本业务：收录错题、复盘记录、按掌握状态排期复习与到期提醒。
type MistakeService struct {
	mistakeRepo *repository.MistakeRepository
	problemRepo *repository.ProblemRepository
	logger      *slog.Logger
}

// NewMistakeService 构造错题本服务。
func NewMistakeService(mistakeRepo *repository.MistakeRepository, problemRepo *repository.ProblemRepository, logger *slog.Logger) *MistakeService {
	return &MistakeService{mistakeRepo: mistakeRepo, problemRepo: problemRepo, logger: logger}
}

// parseReviewDate 解析复习日期，支持 yyyy-MM-dd 与 RFC3339。
func parseReviewDate(s string) (time.Time, error) {
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid review date: %s", s)
}

// defaultNextReview 按掌握状态计算默认下次复习日期。
func defaultNextReview(mastery string) time.Time {
	days := constants.MasteryReviewIntervalDays[mastery]
	if days <= 0 {
		days = 1
	}
	return time.Now().AddDate(0, 0, days)
}

// Create 手动收录错题（未传掌握状态默认未掌握，未传复习日期按掌握状态自动排期）。
func (s *MistakeService) Create(ctx context.Context, userID primitive.ObjectID, req *dto.CreateMistakeRequest) (*dto.MistakeResponse, error) {
	mastery := req.Mastery
	if mastery == "" {
		mastery = constants.MasteryUnmastered
	}
	if !constants.ValidMastery(mastery) {
		return nil, util.WrapAppError(constants.CodeMistakeMastery, fmt.Sprintf(constants.MsgMistakeMastery, mastery), nil)
	}
	nextReview := defaultNextReview(mastery)
	if req.NextReviewAt != "" {
		t, err := parseReviewDate(req.NextReviewAt)
		if err != nil {
			return nil, util.WrapAppError(constants.CodeBadRequest, constants.MsgBadRequest, err)
		}
		nextReview = t
	}
	m := &model.Mistake{
		UserID:          userID,
		Title:           req.Title,
		KnowledgePoints: req.KnowledgePoints,
		ErrorReason:     req.ErrorReason,
		ReviewNote:      req.ReviewNote,
		Mastery:         mastery,
		NextReviewAt:    nextReview,
	}
	if req.ProblemID != "" {
		pid, err := primitive.ObjectIDFromHex(req.ProblemID)
		if err != nil {
			return nil, util.WrapAppError(constants.CodeBadRequest, fmt.Sprintf(constants.MsgInvalidID, "题目"), err)
		}
		m.ProblemID = &pid
	}
	if m.KnowledgePoints == nil {
		m.KnowledgePoints = []string{}
	}
	if err := s.mistakeRepo.Create(ctx, m); err != nil {
		if errors.Is(err, repository.ErrMistakeExists) {
			return nil, util.WrapAppError(constants.CodeConflict, "该题目已在错题本中", err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogMistakeCreated, "mistake_id", m.ID.Hex(), "user_id", userID.Hex(), "mastery", mastery)
	resp := dto.ToMistakeResponse(m)
	return &resp, nil
}

// buildFilter 组装列表查询条件（题目关键词模糊、知识点、掌握状态、到期筛选）。
func buildMistakeFilter(userID primitive.ObjectID, q dto.MistakeListQuery) bson.M {
	filter := bson.M{"user_id": userID}
	if q.Q != "" {
		filter["title"] = bson.M{"$regex": regexp.QuoteMeta(q.Q), "$options": "i"}
	}
	if q.KnowledgePoint != "" {
		filter["knowledge_points"] = q.KnowledgePoint
	}
	if q.Mastery != "" {
		filter["mastery"] = q.Mastery
	}
	if q.DueOnly {
		filter["next_review_at"] = bson.M{"$lte": time.Now()}
	}
	return filter
}

// List 分页查询错题列表（按题目、知识点、掌握状态、到期筛选）。
func (s *MistakeService) List(ctx context.Context, userID primitive.ObjectID, q dto.MistakeListQuery) ([]dto.MistakeResponse, int64, error) {
	if q.Mastery != "" && !constants.ValidMastery(q.Mastery) {
		return nil, 0, util.WrapAppError(constants.CodeMistakeMastery, fmt.Sprintf(constants.MsgMistakeMastery, q.Mastery), nil)
	}
	list, total, err := s.mistakeRepo.List(ctx, buildMistakeFilter(userID, q), (q.Page-1)*q.PageSize, q.PageSize)
	if err != nil {
		s.logger.Error(constants.LogMistakeListFailed, "user_id", userID.Hex(), "error", err.Error())
		return nil, 0, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	out := make([]dto.MistakeResponse, 0, len(list))
	for _, m := range list {
		out = append(out, dto.ToMistakeResponse(m))
	}
	return out, total, nil
}

// ListDue 到期待复习提醒列表（复用 List 的筛选逻辑，取最近 100 条到期记录）。
func (s *MistakeService) ListDue(ctx context.Context, userID primitive.ObjectID) ([]dto.MistakeResponse, int64, error) {
	return s.List(ctx, userID, dto.MistakeListQuery{DueOnly: true, Page: 1, PageSize: 100})
}

// KnowledgePoints 当前用户错题本中的知识点清单（供筛选下拉）。
func (s *MistakeService) KnowledgePoints(ctx context.Context, userID primitive.ObjectID) ([]string, error) {
	points, err := s.mistakeRepo.DistinctKnowledgePoints(ctx, userID)
	if err != nil {
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	return points, nil
}

// Get 查询本人一条错题记录。
func (s *MistakeService) Get(ctx context.Context, id, userID primitive.ObjectID) (*dto.MistakeResponse, error) {
	m, err := s.mistakeRepo.FindByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrMistakeNotFound) {
			return nil, util.WrapAppError(constants.CodeMistakeNotFound, constants.MsgMistakeNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	resp := dto.ToMistakeResponse(m)
	return &resp, nil
}

// Update 部分修改错题记录：仅更新请求中提供的字段，
// 掌握状态 mastery 未提供时保留当前值。
func (s *MistakeService) Update(ctx context.Context, id, userID primitive.ObjectID, req *dto.UpdateMistakeRequest) (*dto.MistakeResponse, error) {
	fields := bson.M{}
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.KnowledgePoints != nil {
		fields["knowledge_points"] = req.KnowledgePoints
	}
	if req.ErrorReason != nil {
		fields["error_reason"] = *req.ErrorReason
	}
	if req.ReviewNote != nil {
		fields["review_note"] = *req.ReviewNote
	}
	if req.Mastery != nil {
		if !constants.ValidMastery(*req.Mastery) {
			return nil, util.WrapAppError(constants.CodeMistakeMastery, fmt.Sprintf(constants.MsgMistakeMastery, *req.Mastery), nil)
		}
		fields["mastery"] = *req.Mastery
	}
	if req.NextReviewAt != nil {
		t, err := parseReviewDate(*req.NextReviewAt)
		if err != nil {
			return nil, util.WrapAppError(constants.CodeBadRequest, constants.MsgBadRequest, err)
		}
		fields["next_review_at"] = t
	}
	if err := s.mistakeRepo.UpdateFields(ctx, id, userID, fields); err != nil {
		if errors.Is(err, repository.ErrMistakeNotFound) {
			return nil, util.WrapAppError(constants.CodeMistakeNotFound, constants.MsgMistakeNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogMistakeUpdated, "mistake_id", id.Hex(), "user_id", userID.Hex())
	return s.Get(ctx, id, userID)
}

// Delete 移除本人一条错题记录（仅删除目标记录，其余记录的掌握状态不受影响）。
func (s *MistakeService) Delete(ctx context.Context, id, userID primitive.ObjectID) error {
	if err := s.mistakeRepo.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, repository.ErrMistakeNotFound) {
			return util.WrapAppError(constants.CodeMistakeNotFound, constants.MsgMistakeNotFound, err)
		}
		return util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogMistakeDeleted, "mistake_id", id.Hex(), "user_id", userID.Hex())
	return nil
}

// Review 完成一次复习：更新掌握状态，排期下次复习日期（缺省按掌握状态间隔），
// 复习次数 +1、记录最近复习时间。
func (s *MistakeService) Review(ctx context.Context, id, userID primitive.ObjectID, req *dto.ReviewMistakeRequest) (*dto.MistakeResponse, error) {
	if !constants.ValidMastery(req.Mastery) {
		return nil, util.WrapAppError(constants.CodeMistakeMastery, fmt.Sprintf(constants.MsgMistakeMastery, req.Mastery), nil)
	}
	nextReview := defaultNextReview(req.Mastery)
	if req.NextReviewAt != nil && *req.NextReviewAt != "" {
		t, err := parseReviewDate(*req.NextReviewAt)
		if err != nil {
			return nil, util.WrapAppError(constants.CodeBadRequest, constants.MsgBadRequest, err)
		}
		nextReview = t
	}
	now := time.Now()
	fields := bson.M{
		"mastery":          req.Mastery,
		"next_review_at":   nextReview,
		"last_reviewed_at": now,
	}
	if err := s.mistakeRepo.MarkReviewed(ctx, id, userID, fields); err != nil {
		if errors.Is(err, repository.ErrMistakeNotFound) {
			return nil, util.WrapAppError(constants.CodeMistakeNotFound, constants.MsgMistakeNotFound, err)
		}
		return nil, util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogMistakeReviewed, "mistake_id", id.Hex(), "user_id", userID.Hex(), "mastery", req.Mastery)
	return s.Get(ctx, id, userID)
}

// CollectFromSubmission 评测未通过时自动收录错题：题目已在错题本中则跳过，
// 不覆盖已有的掌握状态、错误原因与复盘结论。
func (s *MistakeService) CollectFromSubmission(ctx context.Context, userID, problemID primitive.ObjectID) error {
	if _, err := s.mistakeRepo.FindByUserAndProblem(ctx, userID, problemID); err == nil {
		return nil
	} else if !errors.Is(err, repository.ErrMistakeNotFound) {
		return util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	problem, err := s.problemRepo.FindByID(ctx, problemID)
	if err != nil {
		return util.WrapAppError(constants.CodeProblemNotFound, constants.MsgProblemNotFound, err)
	}
	tags := problem.Tags
	if tags == nil {
		tags = []string{}
	}
	m := &model.Mistake{
		UserID:          userID,
		ProblemID:       &problemID,
		Title:           problem.Title,
		KnowledgePoints: tags,
		Mastery:         constants.MasteryUnmastered,
		NextReviewAt:    defaultNextReview(constants.MasteryUnmastered),
	}
	if err := s.mistakeRepo.Create(ctx, m); err != nil {
		if errors.Is(err, repository.ErrMistakeExists) {
			return nil
		}
		return util.WrapAppError(constants.CodeInternal, constants.MsgInternalError, err)
	}
	s.logger.Info(constants.LogMistakeAutoCollected, "mistake_id", m.ID.Hex(), "user_id", userID.Hex(), "problem_id", problemID.Hex())
	return nil
}
