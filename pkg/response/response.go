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
)

func Result(code int, data interface{}, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:      code,
		Data:      data,
		Msg:       msg,
		Timestamp: time.Now().UnixMilli(),
	})
}

func ResultWithStatus(status int, code int, data interface{}, msg string, c *gin.Context) {
	c.JSON(status, Response{
		Code:      code,
		Data:      data,
		Msg:       msg,
		Timestamp: time.Now().UnixNano() / 1e6,
	})
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

func ForbiddenWithMessage(message string, c *gin.Context) {
	ResultWithStatus(http.StatusForbidden, http.StatusForbidden, nil, message, c)
}

func FailWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(ERROR, data, message, c)
}
