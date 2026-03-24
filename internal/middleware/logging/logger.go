package logging

import (
	"bytes"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		method := c.Request.Method
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		clientIP := c.ClientIP()

		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = writer

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		fullPath := path
		if rawQuery != "" {
			fullPath = path + "?" + rawQuery
		}

		entry := log.WithFields(log.Fields{
			"method":        method,
			"path":          fullPath,
			"latency_ms":    latency.Milliseconds(),
			"client_ip":     clientIP,
			"response_body": writer.body.String(),
		})

		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.String())
		}

		switch {
		case statusCode >= 500:
			entry.WithFields(log.Fields{
				"result": "server_error",
			}).Error("request failed")
		case statusCode >= 400:
			entry.WithFields(log.Fields{
				"result": "client_error",
			}).Warn("request rejected")
		default:
			entry.WithFields(log.Fields{
				"result": "success",
			}).Info("request completed")
		}
	}
}
