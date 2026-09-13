package handler

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/dto"
	"github.com/blueship581/codelearn/internal/service"
	"github.com/blueship581/codelearn/internal/util"
)

// MistakeHandler 错题本处理器。
type MistakeHandler struct {
	mistakeService *service.MistakeService
}

// NewMistakeHandler 构造错题本处理器。
func NewMistakeHandler(mistakeService *service.MistakeService) *MistakeHandler {
	return &MistakeHandler{mistakeService: mistakeService}
}

// parseMistakeID 解析路径中的错题 ID。
func parseMistakeID(c *gin.Context) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		_ = c.Error(util.NewAppError(constants.CodeBadRequest, "无效的错题 ID"))
		return primitive.NilObjectID, false
	}
	return id, true
}

// List 分页查询错题列表（题目关键词/知识点/掌握状态/到期筛选）。
func (h *MistakeHandler) List(c *gin.Context) {
	userID := getUserID(c)
	page, pageSize := parsePage(c.DefaultQuery("page", "1"), c.DefaultQuery("page_size", "10"))
	q := dto.MistakeListQuery{
		Q:              c.Query("q"),
		KnowledgePoint: c.Query("knowledge_point"),
		Mastery:        c.Query("mastery"),
		DueOnly:        c.Query("due_only") == "true",
		Page:           page,
		PageSize:       pageSize,
	}
	list, total, err := h.mistakeService.List(c.Request.Context(), userID, q)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.SuccessPage(c, list, total, page, pageSize)
}

// ListDue 到期待复习提醒列表。
func (h *MistakeHandler) ListDue(c *gin.Context) {
	userID := getUserID(c)
	list, total, err := h.mistakeService.ListDue(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.SuccessPage(c, list, total, 1, int64(len(list)))
}

// KnowledgePoints 当前用户错题本知识点清单。
func (h *MistakeHandler) KnowledgePoints(c *gin.Context) {
	userID := getUserID(c)
	points, err := h.mistakeService.KnowledgePoints(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, points)
}

// Get 查询错题详情。
func (h *MistakeHandler) Get(c *gin.Context) {
	userID := getUserID(c)
	id, ok := parseMistakeID(c)
	if !ok {
		return
	}
	resp, err := h.mistakeService.Get(c.Request.Context(), id, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, resp)
}

// Create 手动收录错题。
func (h *MistakeHandler) Create(c *gin.Context) {
	userID := getUserID(c)
	var req dto.CreateMistakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(util.NewAppError(util.ValidationCode(), util.ValidationMessage()))
		return
	}
	resp, err := h.mistakeService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, resp)
}

// Update 修改错题记录（未提供的字段保持原值，掌握状态保留）。
func (h *MistakeHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	id, ok := parseMistakeID(c)
	if !ok {
		return
	}
	var req dto.UpdateMistakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(util.NewAppError(util.ValidationCode(), util.ValidationMessage()))
		return
	}
	resp, err := h.mistakeService.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, resp)
}

// Delete 移除错题记录。
func (h *MistakeHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	id, ok := parseMistakeID(c)
	if !ok {
		return
	}
	if err := h.mistakeService.Delete(c.Request.Context(), id, userID); err != nil {
		_ = c.Error(err)
		return
	}
	util.SuccessMessage(c, constants.MsgMistakeDeleted)
}

// Review 完成一次复习（更新掌握状态并排期下次复习）。
func (h *MistakeHandler) Review(c *gin.Context) {
	userID := getUserID(c)
	id, ok := parseMistakeID(c)
	if !ok {
		return
	}
	var req dto.ReviewMistakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(util.NewAppError(util.ValidationCode(), util.ValidationMessage()))
		return
	}
	resp, err := h.mistakeService.Review(c.Request.Context(), id, userID, &req)
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, resp)
}
