package handler

import (
	"sign_flow_project/internal/service/workflow_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
)

type workflowHandlerImpl struct{}

var WorkflowHandler = new(workflowHandlerImpl)

func (h *workflowHandlerImpl) CreateWorkflow(c *gin.Context) {
	var req service.CreateWorkflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("invalid request body", c)
		return
	}

	result, err := service.WorkflowService.CreateWorkflow(req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(result, c)
}
