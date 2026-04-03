package middleware

import (
	"net/http"
	"strings"

	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	CtxCurrentUserID    = "currentUserID"
	CtxCurrentUserCode  = "currentUserCode"
	CtxCurrentUserEmail = "currentUserEmail"
)

// JWTAuth 解析 Authorization: Bearer <token>，将当前用户写入 Gin Context。
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if raw == "" {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "missing authorization", c)
			c.Abort()
			return
		}
		const prefix = "Bearer "
		if !strings.HasPrefix(raw, prefix) {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid authorization scheme", c)
			c.Abort()
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
		if tokenStr == "" {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "missing token", c)
			c.Abort()
			return
		}

		uid, code, email, err := usersvc.UserLoginService.ParseAccessToken(tokenStr)
		if err != nil {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid or expired token", c)
			c.Abort()
			return
		}

		c.Set(CtxCurrentUserID, uid)
		c.Set(CtxCurrentUserCode, code)
		c.Set(CtxCurrentUserEmail, email)
		c.Next()
	}
}
