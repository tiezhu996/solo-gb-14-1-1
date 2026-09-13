package service

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/dto"
	"github.com/blueship581/codelearn/internal/model"
	"github.com/blueship581/codelearn/internal/repository"
	"github.com/blueship581/codelearn/internal/util"
)

// newMistakeTestService 构造带真实 Mongo 的错题服务（TEST_MONGO_URI 未配置时跳过）。
func newMistakeTestService(t *testing.T) (*MistakeService, *repository.MistakeRepository, primitive.ObjectID, context.Context) {
	t.Helper()
	db := testMongo(t)
	logger := util.NewLogger("error")
	mistakeRepo := repository.NewMistakeRepository(db)
	problemRepo := repository.NewProblemRepository(db)
	if err := mistakeRepo.EnsureIndexes(context.Background()); err != nil {
		t.Fatalf("ensure mistake indexes: %v", err)
	}
	svc := NewMistakeService(mistakeRepo, problemRepo, logger)
	ctx := context.Background()
	userID := primitive.NewObjectID()
	// 清理该测试用户的历史数据，保证用例独立
	_, _ = db.Collection("mistakes").DeleteMany(ctx, bson.M{"user_id": userID})
	return svc, mistakeRepo, userID, ctx
}

// TestMistakeServiceCreateAndList 收录错题并按题目/知识点/掌握状态/到期筛选。
func TestMistakeServiceCreateAndList(t *testing.T) {
	svc, _, userID, ctx := newMistakeTestService(t)

	// 默认掌握状态为未掌握，复习日期默认为明天
	resp, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{
		Title:           "两数之和",
		KnowledgePoints: []string{"哈希表", "数组"},
		ErrorReason:     "没想到用哈希表去重",
		ReviewNote:      "一遍哈希：边遍历边存下标",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if resp.Mastery != constants.MasteryUnmastered {
		t.Errorf("default mastery = %s, want unmastered", resp.Mastery)
	}
	if resp.NextReviewAt == "" {
		t.Error("expected default next_review_at")
	}

	// 一条已掌握的记录、一条其他知识点记录
	if _, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{
		Title:           "二叉树层序遍历",
		KnowledgePoints: []string{"二叉树", "BFS"},
		Mastery:         constants.MasteryMastered,
	}); err != nil {
		t.Fatalf("Create second error: %v", err)
	}

	// 按题目关键词查询
	list, total, err := svc.List(ctx, userID, dto.MistakeListQuery{Q: "两数", Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List by q: total=%d len=%d err=%v", total, len(list), err)
	}
	// 按知识点查询
	_, total, err = svc.List(ctx, userID, dto.MistakeListQuery{KnowledgePoint: "哈希表", Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("List by knowledge_point: total=%d err=%v", total, err)
	}
	// 按掌握状态查询
	_, total, err = svc.List(ctx, userID, dto.MistakeListQuery{Mastery: constants.MasteryMastered, Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("List by mastery: total=%d err=%v", total, err)
	}
	// 非法掌握状态应报错
	if _, _, err = svc.List(ctx, userID, dto.MistakeListQuery{Mastery: "unknown", Page: 1, PageSize: 10}); err == nil {
		t.Error("expected error on invalid mastery filter")
	}
	// 知识点清单去重
	points, err := svc.KnowledgePoints(ctx, userID)
	if err != nil || len(points) != 4 {
		t.Fatalf("KnowledgePoints: %v err=%v", points, err)
	}
}

// TestMistakeServiceDueReminder 到期复习提醒：过期记录出现在提醒列表，未来的不出现。
func TestMistakeServiceDueReminder(t *testing.T) {
	svc, _, userID, ctx := newMistakeTestService(t)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	due, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{Title: "已到期错题", NextReviewAt: yesterday})
	if err != nil {
		t.Fatalf("Create due error: %v", err)
	}
	if !due.Due {
		t.Error("expected due=true for yesterday's review date")
	}
	future := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	if _, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{Title: "未到期错题", NextReviewAt: future}); err != nil {
		t.Fatalf("Create future error: %v", err)
	}

	list, total, err := svc.ListDue(ctx, userID)
	if err != nil {
		t.Fatalf("ListDue error: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Title != "已到期错题" {
		t.Fatalf("ListDue: total=%d list=%+v", total, list)
	}
	// 主列表 due_only 筛选同样生效
	_, total, err = svc.List(ctx, userID, dto.MistakeListQuery{DueOnly: true, Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Fatalf("List due_only: total=%d err=%v", total, err)
	}
}

// TestMistakeServiceUpdatePreservesMastery 修改错题时未提供掌握状态则保留当前值；
// 移除一条记录不影响其他记录的掌握状态。
func TestMistakeServiceUpdatePreservesMastery(t *testing.T) {
	svc, _, userID, ctx := newMistakeTestService(t)

	a, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{Title: "记录A"})
	if err != nil {
		t.Fatalf("Create A error: %v", err)
	}
	b, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{Title: "记录B"})
	if err != nil {
		t.Fatalf("Create B error: %v", err)
	}
	// 复习 A：掌握状态变为巩固中
	aid, _ := primitive.ObjectIDFromHex(a.ID)
	if _, err := svc.Review(ctx, aid, userID, &dto.ReviewMistakeRequest{Mastery: constants.MasteryLearning}); err != nil {
		t.Fatalf("Review A error: %v", err)
	}

	// 只修改题目与复盘结论，掌握状态必须保留为巩固中
	newTitle := "记录A（已订正）"
	note := "补充复盘：注意边界条件"
	updated, err := svc.Update(ctx, aid, userID, &dto.UpdateMistakeRequest{Title: &newTitle, ReviewNote: &note})
	if err != nil {
		t.Fatalf("Update A error: %v", err)
	}
	if updated.Mastery != constants.MasteryLearning {
		t.Errorf("mastery after update = %s, want learning (应保留当前掌握状态)", updated.Mastery)
	}
	if updated.Title != newTitle || updated.ReviewNote != note {
		t.Errorf("update fields not applied: %+v", updated)
	}
	if updated.ReviewCount != 1 {
		t.Errorf("review_count = %d, want 1", updated.ReviewCount)
	}

	// 显式修改掌握状态才允许变更
	mastered := constants.MasteryMastered
	updated, err = svc.Update(ctx, aid, userID, &dto.UpdateMistakeRequest{Mastery: &mastered})
	if err != nil {
		t.Fatalf("Update mastery error: %v", err)
	}
	if updated.Mastery != constants.MasteryMastered {
		t.Errorf("mastery after explicit update = %s, want mastered", updated.Mastery)
	}

	// 移除 B 后，A 的掌握状态不受影响
	bid, _ := primitive.ObjectIDFromHex(b.ID)
	if err := svc.Delete(ctx, bid, userID); err != nil {
		t.Fatalf("Delete B error: %v", err)
	}
	got, err := svc.Get(ctx, aid, userID)
	if err != nil {
		t.Fatalf("Get A error: %v", err)
	}
	if got.Mastery != constants.MasteryMastered {
		t.Errorf("mastery after deleting another record = %s, want mastered", got.Mastery)
	}
	// 删除不存在的记录应报未找到
	if err := svc.Delete(ctx, bid, userID); err == nil {
		t.Error("expected not found error on second delete")
	}
}

// TestMistakeServiceReviewSchedule 复习排期：按掌握状态默认间隔 1/3/7 天，可显式指定日期。
func TestMistakeServiceReviewSchedule(t *testing.T) {
	svc, _, userID, ctx := newMistakeTestService(t)

	m, err := svc.Create(ctx, userID, &dto.CreateMistakeRequest{Title: "排期验证"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	id, _ := primitive.ObjectIDFromHex(m.ID)

	cases := []struct {
		mastery  string
		wantDays int
	}{
		{constants.MasteryUnmastered, 1},
		{constants.MasteryLearning, 3},
		{constants.MasteryMastered, 7},
	}
	for _, c := range cases {
		resp, err := svc.Review(ctx, id, userID, &dto.ReviewMistakeRequest{Mastery: c.mastery})
		if err != nil {
			t.Fatalf("Review %s error: %v", c.mastery, err)
		}
		got, err := time.Parse("2006-01-02", resp.NextReviewAt)
		if err != nil {
			t.Fatalf("parse next_review_at: %v", err)
		}
		want := time.Now().AddDate(0, 0, c.wantDays)
		if got.Before(want.Add(-24*time.Hour)) || got.After(want.Add(24*time.Hour)) {
			t.Errorf("mastery %s: next_review_at=%s, want about %d days later", c.mastery, resp.NextReviewAt, c.wantDays)
		}
	}
	// 复习次数累计
	got, _ := svc.Get(ctx, id, userID)
	if got.ReviewCount != 3 {
		t.Errorf("review_count = %d, want 3", got.ReviewCount)
	}
	if got.LastReviewedAt == "" {
		t.Error("expected last_reviewed_at set after review")
	}
	// 显式指定下次复习日期
	date := "2099-01-01"
	resp, err := svc.Review(ctx, id, userID, &dto.ReviewMistakeRequest{Mastery: constants.MasteryLearning, NextReviewAt: &date})
	if err != nil {
		t.Fatalf("Review with date error: %v", err)
	}
	if resp.NextReviewAt != date {
		t.Errorf("next_review_at = %s, want %s", resp.NextReviewAt, date)
	}
	// 非法掌握状态
	if _, err := svc.Review(ctx, id, userID, &dto.ReviewMistakeRequest{Mastery: "bad"}); err == nil {
		t.Error("expected error on invalid mastery")
	}
}

// TestMistakeServiceCollectFromSubmission 评测未通过自动收录：首次创建，
// 重复失败不覆盖已有掌握状态与复盘记录。
func TestMistakeServiceCollectFromSubmission(t *testing.T) {
	svc, _, userID, ctx := newMistakeTestService(t)
	db := testMongo(t)

	// 造一道题目
	problem := &model.Problem{
		Title:      "自动收录测试题",
		Difficulty: constants.DifficultyEasy,
		Tags:       []string{"动态规划"},
		Status:     constants.StatusPublished,
	}
	problem.ID = primitive.NewObjectID()
	if _, err := db.Collection("problems").InsertOne(ctx, problem); err != nil {
		t.Fatalf("insert problem: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Collection("problems").DeleteOne(ctx, bson.M{"_id": problem.ID}) })

	// 首次失败自动收录
	if err := svc.CollectFromSubmission(ctx, userID, problem.ID); err != nil {
		t.Fatalf("CollectFromSubmission error: %v", err)
	}
	found, _, err := svc.List(ctx, userID, dto.MistakeListQuery{Q: "自动收录", Page: 1, PageSize: 10})
	if err != nil || len(found) != 1 {
		t.Fatalf("after collect: len=%d err=%v", len(found), err)
	}
	if found[0].Mastery != constants.MasteryUnmastered || found[0].ProblemID != problem.ID.Hex() {
		t.Errorf("collected mistake: %+v", found[0])
	}
	if len(found[0].KnowledgePoints) != 1 || found[0].KnowledgePoints[0] != "动态规划" {
		t.Errorf("knowledge points from problem tags: %+v", found[0].KnowledgePoints)
	}

	// 用户复盘并标记为已掌握
	id, _ := primitive.ObjectIDFromHex(found[0].ID)
	note := "状态转移方程写错了"
	if _, err := svc.Update(ctx, id, userID, &dto.UpdateMistakeRequest{ReviewNote: &note}); err != nil {
		t.Fatalf("Update note error: %v", err)
	}
	if _, err := svc.Review(ctx, id, userID, &dto.ReviewMistakeRequest{Mastery: constants.MasteryMastered}); err != nil {
		t.Fatalf("Review error: %v", err)
	}

	// 再次失败：不重置掌握状态、不清空复盘结论
	if err := svc.CollectFromSubmission(ctx, userID, problem.ID); err != nil {
		t.Fatalf("CollectFromSubmission again error: %v", err)
	}
	got, err := svc.Get(ctx, id, userID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Mastery != constants.MasteryMastered {
		t.Errorf("mastery after re-collect = %s, want mastered (重复收录不得重置掌握状态)", got.Mastery)
	}
	if got.ReviewNote != note {
		t.Errorf("review note after re-collect = %q, want %q", got.ReviewNote, note)
	}
	// 仍然只有一条记录
	_, total, err := svc.List(ctx, userID, dto.MistakeListQuery{Page: 1, PageSize: 10})
	if err != nil || total != 1 {
		t.Errorf("total after re-collect = %d, want 1 (同一题不重复收录)", total)
	}
}
