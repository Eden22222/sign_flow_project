package handler

import (
	"errors"
	"strconv"
	"strings"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/service/file_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

// PreviewDocument GET /api/v1/documents/:documentId/preview，返回 PDF 文件流。
func (h *fileHandlerImpl) PreviewDocument(c *gin.Context) {
	documentIDStr := strings.TrimSpace(c.Param("documentId"))
	documentID64, err := strconv.ParseUint(documentIDStr, 10, 64)
	if err != nil || documentID64 == 0 {
		response.BadRequestWithMessage("invalid documentId", c)
		return
	}
	documentID := uint(documentID64)

	doc, err := dao.DocumentDao.SelectByID(documentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFoundWithMessage("document not found", c)
			return
		}
		response.InternalErrorWithMessage(err.Error(), c)
		return
	}

	absPath, err := file_service.FileService.OpenDocumentByFileKey(doc.FilePath)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "empty") || strings.Contains(msg, "not found") {
			response.NotFoundWithMessage("stored file not found", c)
			return
		}
		response.InternalErrorWithMessage(msg, c)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.File(absPath)
}
