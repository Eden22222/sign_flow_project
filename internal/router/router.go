package router

import (
	"net/http"
	"sign_flow_project/internal/handler/file"
	"sign_flow_project/internal/handler/user"
	"sign_flow_project/internal/handler/workflow"
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
	api.POST("/files/upload", file.FileHandler.Upload)

	api.POST("/auth/register", user.UserLoginHandler.Register)
	api.POST("/auth/login", user.UserLoginHandler.Login)
	api.GET("/auth/me", middleware.JWTAuth(), user.UserLoginHandler.Me)

	api.POST("/users", user.UserHandler.CreateUser)
	api.GET("/users/:id", user.UserHandler.GetByID)

	api.POST("/workflows", middleware.JWTAuth(), workflow.WorkflowHandler.CreateWorkflow)
	api.PUT("/workflows/:workflowId/fields", middleware.JWTAuth(), workflow.WorkflowHandler.SaveFields)
	api.POST("/workflows/:workflowId/activate", middleware.JWTAuth(), workflow.WorkflowHandler.Activate)

	api.GET("/workflows", workflow.WorkflowHandler.List)
	api.GET("/workflows/:workflowId", workflow.WorkflowHandler.GetDetail)
	api.GET("/workflows/:workflowId/signing-detail", workflow.WorkflowHandler.GetSigningDetail)
	api.GET("/workflows/:workflowId/sign-fields", workflow.WorkflowHandler.GetSignFields)
	api.GET("/workflows/:workflowId/tasks", workflow.WorkflowHandler.GetTasks)
	api.GET("/workflows/:workflowId/signers", workflow.WorkflowHandler.GetSigners)
	api.GET("/documents/:documentId/preview", file.FileHandler.PreviewDocument)
	api.POST("/workflows/:workflowId/sign-fields/:fieldId/fill", middleware.JWTAuth(), workflow.SigningHandler.FillSignField)
	api.POST("/workflows/:workflowId/submit", middleware.JWTAuth(), workflow.SigningHandler.Submit)
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
