package document

import (
	"strconv"
	"strings"

	docversvc "sign_flow_project/internal/service/document_version_service"
	"sign_flow_project/pkg/response"

	"github.com/gin-gonic/gin"
)

type documentVersionHandlerImpl struct{}

var DocumentVersionHandler = new(documentVersionHandlerImpl)

// ListByDocument GET /api/v1/documents/:documentId/versions
func (h *documentVersionHandlerImpl) ListByDocument(c *gin.Context) {
	documentIDStr := strings.TrimSpace(c.Param("documentId"))
	documentID64, err := strconv.ParseUint(documentIDStr, 10, 64)
	if err != nil || documentID64 == 0 {
		response.BadRequestWithMessage("invalid documentId", c)
		return
	}

	result, err := docversvc.DocumentVersionService.ListByDocumentID(uint(documentID64))
	if err != nil {
		respondDocumentVersionError(c, err)
		return
	}
	response.OkWithData(result, c)
}

// PreviewVersion GET /api/v1/document-versions/:versionId/preview
func (h *documentVersionHandlerImpl) PreviewVersion(c *gin.Context) {
	versionIDStr := strings.TrimSpace(c.Param("versionId"))
	versionID64, err := strconv.ParseUint(versionIDStr, 10, 64)
	if err != nil || versionID64 == 0 {
		response.BadRequestWithMessage("invalid versionId", c)
		return
	}

	absPath, err := docversvc.DocumentVersionService.PreviewDocumentVersion(uint(versionID64))
	if err != nil {
		respondDocumentVersionError(c, err)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.File(absPath)
}

func respondDocumentVersionError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	lower := strings.ToLower(errMsg)
	if strings.Contains(lower, "not found") {
		response.NotFoundWithMessage(errMsg, c)
		return
	}
	if strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "required") ||
		strings.Contains(lower, "empty") {
		response.BadRequestWithMessage(errMsg, c)
		return
	}
	response.InternalErrorWithMessage(errMsg, c)
}
