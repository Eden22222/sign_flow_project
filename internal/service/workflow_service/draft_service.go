package service

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"sign_flow_project/internal/dao"
	infradb "sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/service/file_service"

	"gorm.io/gorm"
)

type draftWorkflowServiceImpl struct{}

var DraftWorkflowService = new(draftWorkflowServiceImpl)

type CreateWorkflowDraftRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"dueDate"`
	Priority    string `json:"priority"`
	FileKey     string `json:"fileKey"`
	FileName    string `json:"fileName"`
	FileType    string `json:"fileType"`
	FileSize    int64  `json:"fileSize"`
	InitiatorID uint   `json:"initiatorId"`
	Signers     []uint `json:"signers"`
}

type CreateWorkflowDraftResult struct {
	DocumentID      uint   `json:"documentId"`
	WorkflowID      uint   `json:"workflowId"`
	Title           string `json:"title"`
	CurrentStep     int    `json:"currentStep"`
	WorkflowStatus  string `json:"workflowStatus"`
	DocumentStatus  string `json:"documentStatus"`
	DocumentVersion int    `json:"documentVersion"`
}

func (s *draftWorkflowServiceImpl) CreateWorkflowDraft(req CreateWorkflowDraftRequest) (*CreateWorkflowDraftResult, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	fileKey := strings.TrimSpace(req.FileKey)
	if fileKey == "" {
		return nil, fmt.Errorf("fileKey is required")
	}
	if len(req.Signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}

	if req.InitiatorID == 0 {
		return nil, fmt.Errorf("initiatorId is required")
	}

	signers := make([]uint, 0, len(req.Signers))
	seen := make(map[uint]struct{}, len(req.Signers))
	for i, signer := range req.Signers {
		if signer == 0 {
			return nil, fmt.Errorf("signer at index %d is empty", i)
		}
		if _, ok := seen[signer]; ok {
			return nil, fmt.Errorf("duplicate signer: %d", signer)
		}
		seen[signer] = struct{}{}
		signers = append(signers, signer)
	}

	uniqueUserIDs := make([]uint, 0, 1+len(signers))
	uniqueUserIDs = append(uniqueUserIDs, req.InitiatorID)
	uniqueUserIDs = append(uniqueUserIDs, signers...)
	users, err := dao.UserDao.SelectByIDs(uniqueUserIDs)
	if err != nil {
		return nil, err
	}
	userExists := make(map[uint]struct{}, len(users))
	for _, u := range users {
		userExists[u.ID] = struct{}{}
	}
	if _, ok := userExists[req.InitiatorID]; !ok {
		return nil, fmt.Errorf("initiator not found")
	}
	for _, sid := range signers {
		if _, ok := userExists[sid]; !ok {
			return nil, fmt.Errorf("signer not found: %d", sid)
		}
	}

	if strings.ToLower(path.Ext(fileKey)) != ".pdf" {
		return nil, fmt.Errorf("only pdf file is supported")
	}

	absPath := file_service.FileService.AbsPathFromFileKey(fileKey)
	st, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("stored file not found")
		}
		return nil, err
	}
	if st.Size() <= 0 {
		return nil, fmt.Errorf("uploaded file is empty")
	}

	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		fileName = path.Base(filepath.ToSlash(fileKey))
	}

	fileType := strings.TrimSpace(req.FileType)
	if fileType == "" {
		fileType = "application/pdf"
	}
	fileSize := req.FileSize
	if fileSize <= 0 {
		fileSize = st.Size()
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var result *CreateWorkflowDraftResult

	err = db.Transaction(func(tx *gorm.DB) error {
		document := &model.DocumentModel{
			Title:          title,
			Description:    strings.TrimSpace(req.Description),
			FileName:       fileName,
			FilePath:       filepath.ToSlash(fileKey),
			FileSize:       fileSize,
			FileType:       fileType,
			CurrentVersion: 1,
			Status:         model.DocumentStatusDraft,
		}
		if err := dao.DocumentDao.CreateTx(tx, document); err != nil {
			return err
		}

		workflow := &model.WorkflowModel{
			DocumentID:  document.ID,
			InitiatorID: req.InitiatorID,
			CurrentStep: 1,
			Status:      model.WorkflowStatusDraft,
		}
		if err := dao.WorkflowDao.CreateTx(tx, workflow); err != nil {
			return err
		}

		workflowSigners := make([]*model.WorkflowSignerModel, 0, len(signers))
		for i, signerID := range signers {
			workflowSigners = append(workflowSigners, &model.WorkflowSignerModel{
				WorkflowID: workflow.ID,
				SignerID:   signerID,
				StepIndex:  i + 1,
			})
		}
		if err := dao.WorkflowSignerDao.CreateTx(tx, workflowSigners); err != nil {
			return err
		}

		result = &CreateWorkflowDraftResult{
			DocumentID:      document.ID,
			WorkflowID:      workflow.ID,
			Title:           document.Title,
			CurrentStep:     workflow.CurrentStep,
			WorkflowStatus:  string(workflow.Status),
			DocumentStatus:  string(document.Status),
			DocumentVersion: document.CurrentVersion,
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("duplicate signer")
		}
		return nil, err
	}
	return result, nil
}
