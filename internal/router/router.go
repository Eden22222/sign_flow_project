package router

import (
	"sign_flow_project/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	admin := r.Group("/admin")
	{
		admin.POST("/workflows", handler.WorkflowHandler.CreateWorkflow)
	}
}
