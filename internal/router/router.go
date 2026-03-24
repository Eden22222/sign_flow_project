package router

import (
	"net/http"
	"sign_flow_project/internal/handler"
	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/middleware/logging"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(
		logging.Logger(),
		logging.Recovery(),
	)
	r.GET("/health", healthCheck)

	RegisterRoutes(r)
	return r
}

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	admin := api.Group("/admin")
	{
		admin.POST("/workflows", handler.WorkflowHandler.CreateWorkflow)
	}

	api.GET("/workflows/:workflowId", handler.WorkflowHandler.GetDetail)
	api.GET("/workflows/:workflowId/tasks", handler.WorkflowHandler.GetTasks)
	api.GET("/workflows/:workflowId/signers", handler.WorkflowHandler.GetSigners)
	api.POST("/workflows/:workflowId/submit", handler.SigningHandler.Submit)
}

func healthCheck(c *gin.Context) {
	if err := db.CheckDatabaseHealth(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "database unhealthy",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "ok",
	})
}
