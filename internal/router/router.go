package router

import (
	"net/http"
	"sign_flow_project/internal/handler"
	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/middleware"
	"sign_flow_project/internal/middleware/logging"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(
		logging.Logger(),
		logging.Recovery(),

		//TODO: 这里需要优化，后续要使用nginx反代解决跨域问题
		middleware.CORS(),
	)
	r.GET("/health", healthCheck)

	RegisterRoutes(r)
	return r
}

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	api.POST("/files/upload", handler.FileHandler.Upload)

	api.POST("/users", handler.UserHandler.CreateUser)
	api.GET("/users/:userCode", handler.UserHandler.GetByUserCode)

	api.POST("/workflows", handler.WorkflowHandler.CreateWorkflow)
	api.PUT("/workflows/:workflowId/fields", handler.WorkflowHandler.SaveFields)
	api.POST("/workflows/:workflowId/activate", handler.WorkflowHandler.Activate)

	api.GET("/workflows", handler.WorkflowHandler.List)
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
