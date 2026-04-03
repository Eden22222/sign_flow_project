package handler

import (
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

type userHandlerImpl struct{}

var UserHandler = new(userHandlerImpl)

func (h *userHandlerImpl) CreateUser(c *gin.Context) {
	var req usersvc.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}
	result, err := usersvc.UserService.CreateUser(req)
	if err != nil {
		respondUserError(c, err)
		return
	}
	response.OkWithData(result, c)
}

func (h *userHandlerImpl) GetByUserCode(c *gin.Context) {
	userCode := strings.TrimSpace(c.Param("userCode"))
	if userCode == "" {
		response.BadRequestWithMessage("userCode is required", c)
		return
	}
	result, err := usersvc.UserService.GetByUserCode(userCode)
	if err != nil {
		respondUserError(c, err)
		return
	}
	response.OkWithData(result, c)
}

func respondUserError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	low := strings.ToLower(errMsg)
	if strings.Contains(errMsg, "not found") {
		response.NotFoundWithMessage(errMsg, c)
		return
	}
	if strings.Contains(low, "duplicate key") || strings.Contains(low, "unique constraint") {
		response.BadRequestWithMessage("userCode already exists", c)
		return
	}
	if strings.Contains(errMsg, "required") ||
		strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "empty") ||
		strings.Contains(errMsg, "already exists") {
		response.BadRequestWithMessage(errMsg, c)
		return
	}
	response.InternalErrorWithMessage(errMsg, c)
}
