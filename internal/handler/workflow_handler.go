package handler

import (
	"sign_flow_project/internal/service/workflow_service"
	"sign_flow_project/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type workflowHandlerImpl struct{}

var WorkflowHandler = new(workflowHandlerImpl)

func (h *workflowHandlerImpl) CreateWorkflow(c *gin.Context) {
	var req service.CreateWorkflowRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	result, err := service.WorkflowService.CreateWorkflow(req)
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "empty") {
			response.BadRequestWithMessage(err.Error(), c)
			return
		}
		response.InternalErrorWithMessage(err.Error(), c)
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
		respondWorkflowError(c, err)
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
		respondWorkflowError(c, err)
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
		respondWorkflowError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func parseWorkflowID(c *gin.Context) (uint, bool) {
	workflowIDStr := c.Param("workflowId")
	workflowID64, err := strconv.ParseUint(workflowIDStr, 10, 64)
	if err != nil || workflowID64 == 0 {
		response.BadRequestWithMessage("invalid workflowId", c)
		return 0, false
	}
	return uint(workflowID64), true
}

func respondWorkflowError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "not found") {
		response.NotFoundWithMessage(errMsg, c)
		return
	}
	if strings.Contains(errMsg, "required") || strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "empty") {
		response.BadRequestWithMessage(errMsg, c)
		return
	}
	response.InternalErrorWithMessage(errMsg, c)
}
