package workflow

import (
	"net/http"
	"sign_flow_project/internal/middleware"
	workflowsvc "sign_flow_project/internal/service/workflow_service"
	"sign_flow_project/pkg/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type workflowHandlerImpl struct{}

var WorkflowHandler = new(workflowHandlerImpl)

// CreateWorkflow POST /api/v1/workflows，创建签署草稿（唯一入口）。
func (h *workflowHandlerImpl) CreateWorkflow(c *gin.Context) {
	var req workflowsvc.CreateWorkflowDraftRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	userID, ok := currentUserIDFromContext(c)
	if !ok {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid current user", c)
		return
	}
	req.InitiatorID = userID

	result, err := workflowsvc.DraftWorkflowService.CreateWorkflowDraft(req)
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

	result, err := workflowsvc.WorkflowQueryService.GetDetail(workflowID)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) GetSigningDetail(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	result, err := workflowsvc.WorkflowQueryService.GetSigningDetail(workflowID)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) GetSignFields(c *gin.Context) {
	workflowID, ok := parseWorkflowID(c)
	if !ok {
		return
	}

	result, err := workflowsvc.WorkflowQueryService.GetSignFields(workflowID)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func (h *workflowHandlerImpl) List(c *gin.Context) {
	page := 1
	pageSize := 10

	if pageStr := strings.TrimSpace(c.Query("page")); pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err != nil {
			response.BadRequestWithMessage("invalid page", c)
			return
		}
		page = parsedPage
	}

	if pageSizeStr := strings.TrimSpace(c.Query("pageSize")); pageSizeStr != "" {
		parsedPageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil {
			response.BadRequestWithMessage("invalid pageSize", c)
			return
		}
		pageSize = parsedPageSize
	}

	result, err := workflowsvc.WorkflowQueryService.List(page, pageSize)
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

	result, err := workflowsvc.WorkflowQueryService.GetTasks(workflowID)
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

	result, err := workflowsvc.WorkflowQueryService.GetSigners(workflowID)
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

	var req workflowsvc.SaveWorkflowFieldsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	userID, ok := currentUserIDFromContext(c)
	if !ok {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid current user", c)
		return
	}

	result, err := workflowsvc.DraftWorkflowService.SaveWorkflowFields(workflowID, userID, req)
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

	userID, ok := currentUserIDFromContext(c)
	if !ok {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid current user", c)
		return
	}

	result, err := workflowsvc.DraftWorkflowService.ActivateWorkflow(workflowID, userID)
	if err != nil {
		respondWorkflowError(c, err)
		return
	}

	response.OkWithData(result, c)
}

// currentUserIDFromContext 读取 JWT 中间件写入的当前用户 ID（供 workflow / signing handler 共用）。
func currentUserIDFromContext(c *gin.Context) (uint, bool) {
	v, ok := c.Get(middleware.CtxCurrentUserID)
	if !ok {
		return 0, false
	}
	uid, ok := v.(uint)
	if !ok {
		return 0, false
	}
	if uid == 0 {
		return 0, false
	}
	return uid, true
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
	if strings.Contains(errMsg, "only initiator can") {
		response.ForbiddenWithMessage(errMsg, c)
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
		strings.Contains(errMsg, "initiator") ||
		strings.Contains(errMsg, "field") ||
		strings.Contains(errMsg, "stored file") {
		response.BadRequestWithMessage(errMsg, c)
		return
	}
	response.InternalErrorWithMessage(errMsg, c)
}
