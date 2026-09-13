package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/blueship581/codelearn/internal/model"
)

// 错题仓储哨兵错误。
var (
	ErrMistakeNotFound = errors.New("mistake not found")
	ErrMistakeExists   = errors.New("mistake already collected")
)

// MistakeRepository 错题本仓储。
type MistakeRepository struct {
	coll *mongo.Collection
}

// NewMistakeRepository 构造错题仓储。
func NewMistakeRepository(db *mongo.Database) *MistakeRepository {
	return &MistakeRepository{coll: db.Collection("mistakes")}
}

// EnsureIndexes 创建索引：用户+题目部分唯一索引（自动收录防重复），
// 以及复习日期/掌握状态/知识点的查询索引。
func (r *MistakeRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "problem_id", Value: 1}},
			Options: options.Index().SetUnique(true).
				SetPartialFilterExpression(bson.M{"problem_id": bson.M{"$exists": true}}),
		},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "next_review_at", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "mastery", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "knowledge_points", Value: 1}}},
	})
	return err
}

// Create 收录一条错题（user_id + problem_id 唯一索引保证同一题不重复收录）。
func (r *MistakeRepository) Create(ctx context.Context, m *model.Mistake) error {
	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now()
	m.UpdatedAt = m.CreatedAt
	_, err := r.coll.InsertOne(ctx, m)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("create mistake: %w", ErrMistakeExists)
		}
		return fmt.Errorf("create mistake: %w", err)
	}
	return nil
}

// FindByID 按 ID 查找本人错题记录。
func (r *MistakeRepository) FindByID(ctx context.Context, id, userID primitive.ObjectID) (*model.Mistake, error) {
	var m model.Mistake
	err := r.coll.FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("find mistake by id: %w", ErrMistakeNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("find mistake by id: %w", err)
	}
	return &m, nil
}

// FindByUserAndProblem 按用户+题目查找错题记录（自动收录幂等判断）。
func (r *MistakeRepository) FindByUserAndProblem(ctx context.Context, userID, problemID primitive.ObjectID) (*model.Mistake, error) {
	var m model.Mistake
	err := r.coll.FindOne(ctx, bson.M{"user_id": userID, "problem_id": problemID}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, fmt.Errorf("find mistake by problem: %w", ErrMistakeNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("find mistake by problem: %w", err)
	}
	return &m, nil
}

// List 按条件分页查询错题（按下次复习日期升序，到期的排在前面）。
func (r *MistakeRepository) List(ctx context.Context, filter bson.M, skip, limit int64) ([]*model.Mistake, int64, error) {
	total, err := r.coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count mistakes: %w", err)
	}
	opts := options.Find().SetSkip(skip).SetLimit(limit).
		SetSort(bson.D{{Key: "next_review_at", Value: 1}, {Key: "created_at", Value: -1}})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list mistakes: %w", err)
	}
	defer cur.Close(ctx)
	out := make([]*model.Mistake, 0)
	for cur.Next(ctx) {
		var m model.Mistake
		if err := cur.Decode(&m); err != nil {
			return nil, 0, fmt.Errorf("decode mistake: %w", err)
		}
		out = append(out, &m)
	}
	return out, total, nil
}

// UpdateFields 部分更新错题字段（仅 $set 传入的字段，未传字段保持原值）。
func (r *MistakeRepository) UpdateFields(ctx context.Context, id, userID primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()
	res, err := r.coll.UpdateOne(ctx, bson.M{"_id": id, "user_id": userID}, bson.M{"$set": fields})
	if err != nil {
		return fmt.Errorf("update mistake: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("update mistake: %w", ErrMistakeNotFound)
	}
	return nil
}

// MarkReviewed 完成一次复习：更新传入字段并原子累加复习次数。
func (r *MistakeRepository) MarkReviewed(ctx context.Context, id, userID primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()
	res, err := r.coll.UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": fields, "$inc": bson.M{"review_count": 1}},
	)
	if err != nil {
		return fmt.Errorf("review mistake: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("review mistake: %w", ErrMistakeNotFound)
	}
	return nil
}

// Delete 移除本人一条错题记录（不影响其他记录）。
func (r *MistakeRepository) Delete(ctx context.Context, id, userID primitive.ObjectID) error {
	res, err := r.coll.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return fmt.Errorf("delete mistake: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("delete mistake: %w", ErrMistakeNotFound)
	}
	return nil
}

// DistinctKnowledgePoints 查询当前用户错题本中已使用的知识点（去重）。
func (r *MistakeRepository) DistinctKnowledgePoints(ctx context.Context, userID primitive.ObjectID) ([]string, error) {
	values, err := r.coll.Distinct(ctx, "knowledge_points", bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("distinct knowledge points: %w", err)
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}
