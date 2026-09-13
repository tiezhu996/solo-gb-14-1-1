package router

import (
	"github.com/gin-gonic/gin"

	"github.com/blueship581/codelearn/internal/handler"
)

// registerMistakeRoutes 注册错题本路由（写操作记录审计日志）。
func registerMistakeRoutes(api *gin.RouterGroup, h *handler.MistakeHandler, auth, audit gin.HandlerFunc) {
	api.GET("/mistakes", auth, h.List)
	api.GET("/mistakes/due", auth, h.ListDue)
	api.GET("/mistakes/knowledge-points", auth, h.KnowledgePoints)
	api.GET("/mistakes/:id", auth, h.Get)
	api.POST("/mistakes", auth, audit, h.Create)
	api.PUT("/mistakes/:id", auth, audit, h.Update)
	api.DELETE("/mistakes/:id", auth, audit, h.Delete)
	api.POST("/mistakes/:id/review", auth, audit, h.Review)
}
