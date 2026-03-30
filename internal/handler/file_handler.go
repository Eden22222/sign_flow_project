package handler

import (
	"strings"

	"sign_flow_project/internal/service/file_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
)

type fileHandlerImpl struct{}

var FileHandler = new(fileHandlerImpl)

func (h *fileHandlerImpl) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader == nil {
		response.BadRequestWithMessage("file is required", c)
		return
	}

	result, err := file_service.FileService.UploadPDF(fileHeader)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "required") || strings.Contains(msg, "empty") || strings.Contains(msg, "only pdf") {
			response.BadRequestWithMessage(msg, c)
			return
		}
		response.InternalErrorWithMessage(msg, c)
		return
	}

	response.OkWithData(result, c)
}

