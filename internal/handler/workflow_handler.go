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
	// 兼容旧路由：保持函数名不变，但逻辑切到草稿创建。
	h.CreateWorkflowDraft(c)
}

func (h *workflowHandlerImpl) CreateWorkflowAdmin(c *gin.Context) {
	var req service.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	result, err := service.WorkflowService.CreateWorkflowLegacy(req)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}
	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) CreateWorkflowDraft(c *gin.Context) {
	var req service.CreateWorkflowDraftRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	result, err := service.WorkflowService.CreateWorkflowDraft(req)
	if err != nil {
		respondWorkflowError(c, err)
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

func (h *workflowHandlerImpl) SaveFields(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	var req service.SaveWorkflowFieldsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	result, err := service.WorkflowService.SaveWorkflowFields(workflowID, req)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) Activate(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	result, err := service.WorkflowService.ActivateWorkflow(workflowID)
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
	if strings.Contains(errMsg, "required") ||
		strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "empty") ||
		strings.Contains(errMsg, "duplicate") ||
		strings.Contains(errMsg, "only pdf") ||
		strings.Contains(errMsg, "editable") ||
		strings.Contains(errMsg, "activated") ||
		strings.Contains(errMsg, "signer") ||
		strings.Contains(errMsg, "field") ||
		strings.Contains(errMsg, "stored file") {
		response.BadRequestWithMessage(errMsg, c)
		return
	}
	response.InternalErrorWithMessage(errMsg, c)
}
