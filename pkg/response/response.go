package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      int    `json:"code"`
	Data      any    `json:"data"`
	Msg       string `json:"msg"`
	Timestamp int64  `json:"timestamp"`
}

const (
	ERROR       = http.StatusInternalServerError
	SUCCESS     = http.StatusOK
	BAD_REQUEST = http.StatusBadRequest
	NOT_FOUND   = http.StatusNotFound
	FORBIDDEN   = http.StatusForbidden
)

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func write(c *gin.Context, httpStatus int, code int, data any, msg string) {
	c.JSON(httpStatus, Response{
		Code:      code,
		Data:      data,
		Msg:       msg,
		Timestamp: nowMillis(),
	})
}

func Result(code int, data any, msg string, c *gin.Context) {
	write(c, http.StatusOK, code, data, msg)
}

func ResultWithStatus(status int, code int, data any, msg string, c *gin.Context) {
	write(c, status, code, data, msg)
}

func Ok(c *gin.Context) {
	Result(SUCCESS, nil, "Success", c)
}

func OkWithMessage(message string, c *gin.Context) {
	Result(SUCCESS, nil, message, c)
}

func OkWithData(data interface{}, c *gin.Context) {
	Result(SUCCESS, data, "Success", c)
}

func OkWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(SUCCESS, data, message, c)
}

func Fail(c *gin.Context) {
	Result(ERROR, nil, "Error", c)
}

func FailWithMessage(message string, c *gin.Context) {
	Result(ERROR, nil, message, c)
}

func BadRequestWithMessage(message string, c *gin.Context) {
	ResultWithStatus(BAD_REQUEST, BAD_REQUEST, nil, message, c)
}

func NotFoundWithMessage(message string, c *gin.Context) {
	ResultWithStatus(NOT_FOUND, NOT_FOUND, nil, message, c)
}

func InternalErrorWithMessage(message string, c *gin.Context) {
	ResultWithStatus(ERROR, ERROR, nil, message, c)
}

func ForbiddenWithMessage(message string, c *gin.Context) {
	ResultWithStatus(FORBIDDEN, FORBIDDEN, nil, message, c)
}

func FailWithDetailed(data any, message string, c *gin.Context) {
	Result(ERROR, data, message, c)
}
