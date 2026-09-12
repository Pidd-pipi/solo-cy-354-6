package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lp/campus-market/internal/handler"
)

// RegisterReportRoutes registers user report endpoints and admin handling endpoints.
func RegisterReportRoutes(g *gin.RouterGroup, h *handler.ReportHandler, auth, requireAdmin, apiLimiter gin.HandlerFunc) {
	reports := g.Group("/reports", auth)
	{
		reports.POST("", apiLimiter, h.Create)
	}
	admin := g.Group("/admin", auth, requireAdmin)
	{
		admin.GET("/reports", apiLimiter, h.ListPending)
		admin.POST("/reports/:id/handle", apiLimiter, h.Handle)
	}
}
