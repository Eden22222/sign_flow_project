package handler

import (
	"sign_flow_project/internal/service/workflow_service"
	"sign_flow_project/pkg/response"
	"strconv"

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

func (h *workflowHandlerImpl) GetDetail(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	result, err := service.WorkflowService.GetDetail(workflowID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) GetTasks(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	result, err := service.WorkflowService.GetTasks(workflowID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) GetSigners(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	result, err := service.WorkflowService.GetSigners(workflowID)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(result, c)
}

func parseWorkflowID(c *gin.Context) (uint, bool) {
	workflowIDStr := c.Param("workflowId")
	workflowID64, err := strconv.ParseUint(workflowIDStr, 10, 64)
	if err != nil || workflowID64 == 0 {
		response.FailWithMessage("invalid workflowId", c)
		return 0, false
	}
	return uint(workflowID64), true
}
