package user

import (
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/response"
	"strconv"
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

func (h *userHandlerImpl) GetByID(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		response.BadRequestWithMessage("id is required", c)
		return
	}
	result, err := usersvc.UserService.GetByID(uint(id64))
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
		response.BadRequestWithMessage("email already exists", c)
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
