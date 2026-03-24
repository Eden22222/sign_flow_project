package logging

import (
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const RequestIDKey = "request_id"

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		method := c.Request.Method
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		clientIP := c.ClientIP()

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		fullPath := path
		if rawQuery != "" {
			fullPath = path + "?" + rawQuery
		}

		requestID, _ := c.Get(RequestIDKey)
		requestIDStr, _ := requestID.(string)

		entry := log.WithFields(log.Fields{
			"request_id": requestIDStr,
			"method":     method,
			"path":       fullPath,
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  clientIP,
		})

		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.String())
		}

		switch {
		case statusCode >= 500:
			entry.Error("request completed")
		case statusCode >= 400:
			entry.Error("request completed")
		default:
			entry.Info("request completed")
		}
	}
}
