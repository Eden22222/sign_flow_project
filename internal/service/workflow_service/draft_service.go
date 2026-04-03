package service

import (
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
	// InitiatorID 须由已挂 JWTAuth 的 handler 从 token 解析后注入；body 中的 initiatorId 仅兼容旧客户端，service 以注入值为准。
	InitiatorID string   `json:"initiatorId"`
	Signers     []string `json:"signers"`
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

// resolveUserCodes 将发起人与签署人标识统一解析为 userCode，兼容传入 userCode 或 email。
func (s *draftWorkflowServiceImpl) resolveUserCodes(initiatorID string, signers []string) (string, []string, error) {
	initiatorID = strings.TrimSpace(initiatorID)
	if initiatorID == "" {
		return "", nil, fmt.Errorf("initiatorId is required")
	}

	uniqSeen := make(map[string]struct{}, 1+len(signers))
	uniq := make([]string, 0, 1+len(signers))
	add := func(id string) {
		if _, ok := uniqSeen[id]; ok {
			return
		}
		uniqSeen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	add(initiatorID)
	for _, sid := range signers {
		add(sid)
	}

	userCodes := make([]string, 0, len(uniq))
	emails := make([]string, 0, len(uniq))
	for _, id := range uniq {
		if strings.Contains(id, "@") {
			emails = append(emails, id)
			continue
		}
		userCodes = append(userCodes, id)
	}

	resolved := make(map[string]string, len(uniq))
	if len(userCodes) > 0 {
		users, err := dao.UserDao.SelectByUserCodes(userCodes)
		if err != nil {
			return "", nil, err
		}
		for _, u := range users {
			resolved[u.UserCode] = u.UserCode
		}
	}
	if len(emails) > 0 {
		users, err := dao.UserDao.SelectByEmails(emails)
		if err != nil {
			return "", nil, err
		}
		for _, u := range users {
			resolved[u.Email] = u.UserCode
		}
	}

	resolvedInitiator, ok := resolved[initiatorID]
	if !ok {
		return "", nil, fmt.Errorf("initiator not found")
	}

	resolvedSigners := make([]string, 0, len(signers))
	for _, sid := range signers {
		code, exists := resolved[sid]
		if !exists {
			return "", nil, fmt.Errorf("signer not found: %s", sid)
		}
		resolvedSigners = append(resolvedSigners, code)
	}
	return resolvedInitiator, resolvedSigners, nil
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

	signers := make([]string, 0, len(req.Signers))
	seen := make(map[string]struct{}, len(req.Signers))
	for i, signer := range req.Signers {
		sid := strings.TrimSpace(signer)
		if sid == "" {
			return nil, fmt.Errorf("signer at index %d is empty", i)
		}
		if _, ok := seen[sid]; ok {
			return nil, fmt.Errorf("duplicate signer: %s", sid)
		}
		seen[sid] = struct{}{}
		signers = append(signers, sid)
	}

	initiatorID := strings.TrimSpace(req.InitiatorID)
	resolvedInitiatorID, resolvedSigners, err := s.resolveUserCodes(initiatorID, signers)
	if err != nil {
		return nil, err
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
			InitiatorID: resolvedInitiatorID,
			CurrentStep: 1,
			Status:      model.WorkflowStatusDraft,
		}
		if err := dao.WorkflowDao.CreateTx(tx, workflow); err != nil {
			return err
		}

		workflowSigners := make([]*model.WorkflowSignerModel, 0, len(resolvedSigners))
		for i, signerID := range resolvedSigners {
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
		return nil, err
	}
	return result, nil
}
