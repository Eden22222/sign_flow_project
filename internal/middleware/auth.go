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
	CtxCurrentUserEmail = "currentUserEmail"
	CtxCurrentAccessToken = "currentAccessToken"
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
		tokenStr, ok := extractBearerToken(raw)
		if !ok {
			if strings.TrimSpace(raw) == "" {
				response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "missing authorization", c)
				c.Abort()
				return
			}
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid authorization scheme", c)
			c.Abort()
			return
		}
		if tokenStr == "" {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "missing token", c)
			c.Abort()
			return
		}

		uid, email, err := usersvc.UserLoginService.ParseAccessToken(tokenStr)
		if err != nil {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "invalid or expired token", c)
			c.Abort()
			return
		}

		if usersvc.UserLoginService.IsAccessTokenRevoked(tokenStr) {
			response.ResultWithStatus(http.StatusUnauthorized, http.StatusUnauthorized, nil, "token has been revoked", c)
			c.Abort()
			return
		}

		c.Set(CtxCurrentUserID, uid)
		c.Set(CtxCurrentUserEmail, email)
		c.Set(CtxCurrentAccessToken, tokenStr)
		c.Next()
	}
}

// extractBearerToken 解析 Authorization 头，要求严格两段：scheme + token。
// scheme 按 RFC 不区分大小写，接受 Bearer/bearer 等写法。
func extractBearerToken(raw string) (string, bool) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
