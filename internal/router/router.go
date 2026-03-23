package router

import (
	"sign_flow_project/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	admin := api.Group("/admin")
	{
		admin.POST("/workflows", handler.WorkflowHandler.CreateWorkflow)
	}

	api.POST("/workflows/:workflowId/submit", handler.SigningHandler.Submit)
}
