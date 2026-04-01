package handler

import (
	"strconv"

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
		response.FailWithMessage("invalid workflowId", c)
		return
	}

	var req workflowsvc.SubmitSigningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("invalid request body", c)
		return
	}

	result, err := workflowsvc.SigningService.Submit(uint(workflowID64), req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithData(result, c)
}
