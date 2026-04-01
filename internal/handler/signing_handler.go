package handler

import (
	"strconv"
	"strings"

	workflowsvc "sign_flow_project/internal/service/workflow_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
)

type signingHandlerImpl struct{}

var SigningHandler = new(signingHandlerImpl)

func (h *signingHandlerImpl) Submit(c *gin.Context) {
	workflowIDStr := c.Param("workflowId")
	workflowID64, err := strconv.ParseUint(workflowIDStr, 10, 64)
	if err != nil || workflowID64 == 0 {
		response.BadRequestWithMessage("invalid workflowId", c)
		return
	}

	var req workflowsvc.SubmitSigningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	result, err := workflowsvc.SigningService.Submit(uint(workflowID64), req)
	if err != nil {
		respondSigningError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func (h *signingHandlerImpl) FillSignField(c *gin.Context) {
	workflowIDStr := c.Param("workflowId")
	workflowID64, err := strconv.ParseUint(workflowIDStr, 10, 64)
	if err != nil || workflowID64 == 0 {
		response.BadRequestWithMessage("invalid workflowId", c)
		return
	}

	fieldIDStr := c.Param("fieldId")
	fieldID64, err := strconv.ParseUint(fieldIDStr, 10, 64)
	if err != nil || fieldID64 == 0 {
		response.BadRequestWithMessage("invalid fieldId", c)
		return
	}

	var req workflowsvc.FillSignFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}

	result, err := workflowsvc.SigningService.FillSignField(uint(workflowID64), uint(fieldID64), req)
	if err != nil {
		respondSigningError(c, err)
		return
	}

	response.OkWithData(result, c)
}

func respondSigningError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	lower := strings.ToLower(errMsg)
	if strings.Contains(lower, "not found") {
		response.NotFoundWithMessage(errMsg, c)
		return
	}
	if strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "required") ||
		strings.Contains(lower, "mismatch") ||
		strings.Contains(lower, "does not match") ||
		strings.Contains(lower, "must") ||
		strings.Contains(lower, "pending") ||
		strings.Contains(lower, "does not belong") ||
		strings.Contains(lower, "not completed") {
		response.BadRequestWithMessage(errMsg, c)
		return
	}
	response.InternalErrorWithMessage(errMsg, c)
}
