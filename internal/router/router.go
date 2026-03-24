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

	api.GET("/workflows/:workflowId", handler.WorkflowHandler.GetDetail)
	api.GET("/workflows/:workflowId/tasks", handler.WorkflowHandler.GetTasks)
	api.GET("/workflows/:workflowId/signers", handler.WorkflowHandler.GetSigners)
	api.POST("/workflows/:workflowId/submit", handler.SigningHandler.Submit)
}
