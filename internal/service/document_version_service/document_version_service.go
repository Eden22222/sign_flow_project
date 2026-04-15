package document_version_service

import (
	"errors"
	"fmt"
	"strings"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/service/file_service"
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/util"

	"gorm.io/gorm"
)

type documentVersionQueryServiceImpl struct{}

var DocumentVersionService = new(documentVersionQueryServiceImpl)

type DocumentVersionListItem struct {
	VersionID       uint   `json:"versionId"`
	VersionNo       int    `json:"versionNo"`
	DisplayVersion  string `json:"displayVersion"`
	SignerID        uint   `json:"signerId"`
	SignerName      string `json:"signerName"`
	FileName        string `json:"fileName"`
	FileSize        int64  `json:"fileSize"`
	ActionType      string `json:"actionType"`
	Remark          string `json:"remark"`
	CreatedAt       string `json:"createdAt"`
}

type ListDocumentVersionsResult struct {
	DocumentID uint                      `json:"documentId"`
	Items      []DocumentVersionListItem `json:"items"`
}

func displayVersionLabel(versionNo int) string {
	return fmt.Sprintf("v%d.0", versionNo)
}

// ListByDocumentID 返回文档版本列表（version_no 降序）。
func (s *documentVersionQueryServiceImpl) ListByDocumentID(documentID uint) (*ListDocumentVersionsResult, error) {
	if documentID == 0 {
		return nil, fmt.Errorf("documentId is required")
	}
	if _, err := dao.DocumentDao.SelectByID(documentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document not found")
		}
		return nil, err
	}

	rows, err := dao.DocumentVersionDao.SelectByDocumentID(documentID)
	if err != nil {
		return nil, err
	}

	signerIDs := make([]uint, 0, len(rows))
	for i := range rows {
		if rows[i].SignerID != 0 {
			signerIDs = append(signerIDs, rows[i].SignerID)
		}
	}
	userMap, err := usersvc.UserService.BatchGetMapByIDs(signerIDs)
	if err != nil {
		return nil, err
	}

	items := make([]DocumentVersionListItem, 0, len(rows))
	for i := range rows {
		r := rows[i]
		name := ""
		if r.SignerID != 0 {
			if u, ok := userMap[r.SignerID]; ok {
				name = u.Name
			}
		}
		items = append(items, DocumentVersionListItem{
			VersionID:      r.ID,
			VersionNo:      r.VersionNo,
			DisplayVersion: displayVersionLabel(r.VersionNo),
			SignerID:       r.SignerID,
			SignerName:     name,
			FileName:       r.FileName,
			FileSize:       r.FileSize,
			ActionType:     strings.TrimSpace(r.ActionType),
			Remark:         r.Remark,
			CreatedAt:      util.FormatWorkflowCreatedAt(r.CreatedAt),
		})
	}

	return &ListDocumentVersionsResult{
		DocumentID: documentID,
		Items:      items,
	}, nil
}

// PreviewDocumentVersion 解析版本记录并返回磁盘绝对路径（供 handler 流式输出）。
func (s *documentVersionQueryServiceImpl) PreviewDocumentVersion(versionID uint) (string, error) {
	if versionID == 0 {
		return "", fmt.Errorf("versionId is required")
	}
	ver, err := dao.DocumentVersionDao.SelectByID(versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("document version not found")
		}
		return "", err
	}
	return file_service.FileService.OpenDocumentByFileKey(ver.FilePath)
}
