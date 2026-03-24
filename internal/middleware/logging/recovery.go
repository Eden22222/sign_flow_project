package logging

import (
	"fmt"
	"runtime/debug"

	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(log.Fields{
					"method":    c.Request.Method,
					"path":      c.Request.URL.Path,
					"client_ip": c.ClientIP(),
					"panic":     fmt.Sprint(r),
					"stack":     string(debug.Stack()),
				}).Error("panic recovered")

				response.InternalErrorWithMessage("Internal Server Error", c)
				c.Abort()
			}
		}()

		c.Next()
	}
}
