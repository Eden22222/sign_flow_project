package user

import (
	"errors"
	"net/http"
	"strings"

	"sign_flow_project/internal/middleware"
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
)

type userLoginHandlerImpl struct{}

var UserLoginHandler = new(userLoginHandlerImpl)

func (h *userLoginHandlerImpl) Register(c *gin.Context) {
	var req usersvc.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}
	result, err := usersvc.UserLoginService.Register(req)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	response.OkWithData(result, c)
}

func (h *userLoginHandlerImpl) Login(c *gin.Context) {
	var req usersvc.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequestWithMessage("invalid request body", c)
		return
	}
	result, err := usersvc.UserLoginService.Login(req)
	if err != nil {
		respondAuthError(c, err)
		return
	}
	response.OkWithData(result, c)
}

func (h *userLoginHandlerImpl) Logout(c *gin.Context) {
	v, ok := c.Get(middleware.CtxCurrentAccessToken)
	if !ok {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "missing token", c)
		return
	}
	tokenStr, ok := v.(string)
	if !ok || strings.TrimSpace(tokenStr) == "" {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "missing token", c)
		return
	}

	if err := usersvc.UserLoginService.Logout(tokenStr); err != nil {
		if errors.Is(err, usersvc.ErrInvalidToken) {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid or expired token", c)
			return
		}
		response.InternalErrorWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("logout success", c)
}

func (h *userLoginHandlerImpl) Me(c *gin.Context) {
	v, ok := c.Get(middleware.CtxCurrentUserID)
	if !ok {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "unauthorized", c)
		return
	}
	uid, ok := v.(uint)
	if !ok {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "unauthorized", c)
		return
	}
	result, err := usersvc.UserLoginService.GetMe(uid)
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "not found") {
			response.NotFoundWithMessage(msg, c)
			return
		}
		response.InternalErrorWithMessage(msg, c)
		return
	}
	response.OkWithData(result, c)
}

func respondAuthError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, usersvc.ErrInvalidCredentials) {
		response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, err.Error(), c)
		return
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(low, "required") ||
		strings.Contains(low, "at least") ||
		strings.Contains(low, "invalid") {
		response.BadRequestWithMessage(msg, c)
		return
	}
	if strings.Contains(low, "duplicate key") ||
		strings.Contains(low, "unique constraint") ||
		strings.Contains(low, "already registered") {
		response.BadRequestWithMessage("registration failed: email conflict, retry", c)
		return
	}
	response.InternalErrorWithMessage(msg, c)
}
