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

type workflowServiceImpl struct{}

var WorkflowService = new(workflowServiceImpl)

type CreateWorkflowRequest struct {
	Title   string   `json:"title"`
	Signers []string `json:"signers"`
}

// CreateWorkflowResult 为历史接口保留（旧的 /createWorkflow）。
// 新流程以 CreateWorkflowDraftResult 为准。
type CreateWorkflowResult struct {
	DocumentID  uint   `json:"documentId"`
	WorkflowID  uint   `json:"workflowId"`
	FirstSigner string `json:"firstSigner"`
}

type CreateWorkflowDraftRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	DueDate     string   `json:"dueDate"`
	Priority    string   `json:"priority"`
	FileKey     string   `json:"fileKey"`
	FileName    string   `json:"fileName"`
	FileType    string   `json:"fileType"`
	FileSize    int64    `json:"fileSize"`
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

type SaveWorkflowFieldsRequest struct {
	Fields []SaveWorkflowFieldItem `json:"fields"`
}

type SaveWorkflowFieldItem struct {
	SignerID   string  `json:"signerId"`
	FieldType  string  `json:"fieldType"`
	PageNumber int     `json:"pageNumber"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Required   bool    `json:"required"`
}

type SaveWorkflowFieldsResult struct {
	WorkflowID uint `json:"workflowId"`
	DocumentID uint `json:"documentId"`
	FieldCount int  `json:"fieldCount"`
}

type ActivateWorkflowResult struct {
	WorkflowID      uint   `json:"workflowId"`
	DocumentID      uint   `json:"documentId"`
	CurrentStep     int    `json:"currentStep"`
	WorkflowStatus  string `json:"workflowStatus"`
	DocumentStatus  string `json:"documentStatus"`
	CurrentSignerID string `json:"currentSignerId"`
}

// create workflow
func (s *workflowServiceImpl) CreateWorkflow(req CreateWorkflowRequest) (*CreateWorkflowResult, error) {
	// 兼容旧接口：将旧 request 映射为 draft request（但旧接口无法提供 fileKey，会在校验阶段报错）
	draftReq := CreateWorkflowDraftRequest{
		Title:   req.Title,
		Signers: req.Signers,
	}
	draftRes, err := s.CreateWorkflowDraft(draftReq)
	if err != nil {
		return nil, err
	}
	firstSigner := ""
	if len(req.Signers) > 0 {
		firstSigner = strings.TrimSpace(req.Signers[0])
	}
	return &CreateWorkflowResult{
		DocumentID:  draftRes.DocumentID,
		WorkflowID:  draftRes.WorkflowID,
		FirstSigner: firstSigner,
	}, nil
}

// CreateWorkflowLegacy 供旧测试/管理端使用：创建 pending workflow + 第一条 task（不依赖 fileKey）。
func (s *workflowServiceImpl) CreateWorkflowLegacy(req CreateWorkflowRequest) (*CreateWorkflowResult, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(req.Signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}
	for i, signer := range req.Signers {
		if strings.TrimSpace(signer) == "" {
			return nil, fmt.Errorf("signer at index %d is empty", i)
		}
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var result *CreateWorkflowResult
	err := db.Transaction(func(tx *gorm.DB) error {
		document := &model.DocumentModel{
			Title:          strings.TrimSpace(req.Title),
			CurrentVersion: 1,
			Status:         model.DocumentStatusDraft,
		}
		if err := dao.DocumentDao.CreateTx(tx, document); err != nil {
			return err
		}

		workflow := &model.WorkflowModel{
			DocumentID:  document.ID,
			CurrentStep: 1,
			Status:      model.WorkflowStatusPending,
		}
		if err := dao.WorkflowDao.CreateTx(tx, workflow); err != nil {
			return err
		}

		workflowSigners := make([]*model.WorkflowSignerModel, 0, len(req.Signers))
		for i, signerID := range req.Signers {
			workflowSigners = append(workflowSigners, &model.WorkflowSignerModel{
				WorkflowID: workflow.ID,
				SignerID:   strings.TrimSpace(signerID),
				StepIndex:  i + 1,
			})
		}
		if err := dao.WorkflowSignerDao.CreateTx(tx, workflowSigners); err != nil {
			return err
		}

		firstTask := &model.TaskModel{
			WorkflowID: workflow.ID,
			SignerID:   strings.TrimSpace(req.Signers[0]),
			StepIndex:  1,
			Status:     model.TaskStatusPending,
		}
		if err := dao.TaskDao.CreateTx(tx, firstTask); err != nil {
			return err
		}

		result = &CreateWorkflowResult{
			DocumentID:  document.ID,
			WorkflowID:  workflow.ID,
			FirstSigner: firstTask.SignerID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *workflowServiceImpl) CreateWorkflowDraft(req CreateWorkflowDraftRequest) (*CreateWorkflowDraftResult, error) {
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
		return nil, err
	}
	return result, nil
}

func (s *workflowServiceImpl) SaveWorkflowFields(workflowID uint, req SaveWorkflowFieldsRequest) (*SaveWorkflowFieldsResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	if len(req.Fields) == 0 {
		return nil, fmt.Errorf("at least one field is required")
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var result *SaveWorkflowFieldsResult
	err := db.Transaction(func(tx *gorm.DB) error {
		workflow, err := dao.WorkflowDao.SelectByIDTx(tx, workflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("workflow not found")
			}
			return err
		}
		if workflow.Status != model.WorkflowStatusDraft {
			return fmt.Errorf("workflow is not editable")
		}

		signers, err := dao.WorkflowSignerDao.SelectByWorkflowIDTx(tx, workflowID)
		if err != nil {
			return err
		}
		signerSet := make(map[string]struct{}, len(signers))
		for _, s := range signers {
			signerSet[s.SignerID] = struct{}{}
		}

		fields := make([]*model.DocumentFieldModel, 0, len(req.Fields))
		for i, f := range req.Fields {
			sid := strings.TrimSpace(f.SignerID)
			if sid == "" {
				return fmt.Errorf("signer at index %d is empty", i)
			}
			if _, ok := signerSet[sid]; !ok {
				return fmt.Errorf("signer %s is not in current workflow", sid)
			}

			ft := strings.TrimSpace(strings.ToLower(f.FieldType))
			if ft != "signature" && ft != "date" {
				return fmt.Errorf("invalid fieldType at index %d", i)
			}
			if f.PageNumber <= 0 {
				return fmt.Errorf("invalid pageNumber at index %d", i)
			}
			if f.X < 0 || f.Y < 0 {
				return fmt.Errorf("invalid position at index %d", i)
			}
			if f.Width <= 0 || f.Height <= 0 {
				return fmt.Errorf("invalid size at index %d", i)
			}

			fields = append(fields, &model.DocumentFieldModel{
				DocumentID: workflow.DocumentID,
				WorkflowID: workflow.ID,
				SignerID:   sid,
				FieldType:  ft,
				PageNumber: f.PageNumber,
				X:          f.X,
				Y:          f.Y,
				Width:      f.Width,
				Height:     f.Height,
				Required:   f.Required,
				Status:     string(model.DocumentFieldStatusPending),
				Value:      "",
			})
		}

		if err := dao.DocumentFieldDao.DeleteByWorkflowIDTx(tx, workflow.ID); err != nil {
			return err
		}
		if err := dao.DocumentFieldDao.BatchCreateTx(tx, fields); err != nil {
			return err
		}

		result = &SaveWorkflowFieldsResult{
			WorkflowID: workflow.ID,
			DocumentID: workflow.DocumentID,
			FieldCount: len(fields),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *workflowServiceImpl) ActivateWorkflow(workflowID uint) (*ActivateWorkflowResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var result *ActivateWorkflowResult
	err := db.Transaction(func(tx *gorm.DB) error {
		workflow, err := dao.WorkflowDao.SelectByIDTx(tx, workflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("workflow not found")
			}
			return err
		}
		if workflow.Status != model.WorkflowStatusDraft {
			return fmt.Errorf("only draft workflow can be activated")
		}

		document, err := dao.DocumentDao.SelectByIDTx(tx, workflow.DocumentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("document not found")
			}
			return err
		}

		signers, err := dao.WorkflowSignerDao.SelectByWorkflowIDTx(tx, workflowID)
		if err != nil {
			return err
		}
		if len(signers) == 0 {
			return fmt.Errorf("at least one signer is required")
		}

		var firstSigner *model.WorkflowSignerModel
		for i := range signers {
			if signers[i].StepIndex == 1 {
				firstSigner = &signers[i]
				break
			}
		}
		if firstSigner == nil || strings.TrimSpace(firstSigner.SignerID) == "" {
			return fmt.Errorf("stepIndex=1 signer not found")
		}

		fields, err := dao.DocumentFieldDao.SelectByWorkflowIDTx(tx, workflowID)
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			return fmt.Errorf("at least one field is required before activation")
		}

		signerSet := make(map[string]struct{}, len(signers))
		for _, s := range signers {
			signerSet[s.SignerID] = struct{}{}
		}
		for _, f := range fields {
			if _, ok := signerSet[f.SignerID]; !ok {
				return fmt.Errorf("signer %s is not in current workflow", f.SignerID)
			}
		}

		workflow.Status = model.WorkflowStatusPending
		workflow.CurrentStep = 1
		if err := dao.WorkflowDao.UpdateTx(tx, workflow); err != nil {
			return err
		}

		document.Status = model.DocumentStatusReady
		if err := dao.DocumentDao.UpdateTx(tx, document); err != nil {
			return err
		}

		firstTask := &model.TaskModel{
			WorkflowID: workflow.ID,
			SignerID:   strings.TrimSpace(firstSigner.SignerID),
			StepIndex:  1,
			Status:     model.TaskStatusPending,
		}
		if err := dao.TaskDao.CreateTx(tx, firstTask); err != nil {
			return err
		}

		result = &ActivateWorkflowResult{
			WorkflowID:      workflow.ID,
			DocumentID:      document.ID,
			CurrentStep:     workflow.CurrentStep,
			WorkflowStatus:  string(workflow.Status),
			DocumentStatus:  string(document.Status),
			CurrentSignerID: firstTask.SignerID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// detail result

type WorkflowDetailResult struct {
	WorkflowID      uint                 `json:"workflowId"`
	DocumentID      uint                 `json:"documentId"`
	Title           string               `json:"title"`
	CurrentStep     int                  `json:"currentStep"`
	WorkflowStatus  model.WorkflowStatus `json:"workflowStatus"`
	DocumentStatus  model.DocumentStatus `json:"documentStatus"`
	DocumentVersion int                  `json:"documentVersion"`
	CurrentSignerID string               `json:"currentSignerId"`
}

type WorkflowTaskItem struct {
	TaskID     uint             `json:"taskId"`
	WorkflowID uint             `json:"workflowId"`
	SignerID   string           `json:"signerId"`
	StepIndex  int              `json:"stepIndex"`
	Status     model.TaskStatus `json:"status"`
}

type WorkflowTaskListResult struct {
	WorkflowID uint               `json:"workflowId"`
	Tasks      []WorkflowTaskItem `json:"tasks"`
}

type WorkflowSignerItem struct {
	SignerID  string `json:"signerId"`
	StepIndex int    `json:"stepIndex"`
}

type WorkflowSignerListResult struct {
	WorkflowID uint                 `json:"workflowId"`
	Signers    []WorkflowSignerItem `json:"signers"`
}

func (s *workflowServiceImpl) GetDetail(workflowID uint) (*WorkflowDetailResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}

	workflow, err := dao.WorkflowDao.SelectByID(workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, err
	}

	document, err := dao.DocumentDao.SelectByID(workflow.DocumentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document not found")
		}
		return nil, err
	}

	currentSignerID := ""
	currentTask, err := dao.TaskDao.SelectCurrentPendingByWorkflowID(workflowID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		currentSignerID = currentTask.SignerID
	}

	return &WorkflowDetailResult{
		WorkflowID:      workflow.ID,
		DocumentID:      document.ID,
		Title:           document.Title,
		CurrentStep:     workflow.CurrentStep,
		WorkflowStatus:  workflow.Status,
		DocumentStatus:  document.Status,
		DocumentVersion: document.CurrentVersion,
		CurrentSignerID: currentSignerID,
	}, nil
}

func (s *workflowServiceImpl) GetTasks(workflowID uint) (*WorkflowTaskListResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}

	_, err := dao.WorkflowDao.SelectByID(workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, err
	}

	tasks, err := dao.TaskDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}

	items := make([]WorkflowTaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, WorkflowTaskItem{
			TaskID:     task.ID,
			WorkflowID: task.WorkflowID,
			SignerID:   task.SignerID,
			StepIndex:  task.StepIndex,
			Status:     task.Status,
		})
	}

	return &WorkflowTaskListResult{
		WorkflowID: workflowID,
		Tasks:      items,
	}, nil
}

func (s *workflowServiceImpl) GetSigners(workflowID uint) (*WorkflowSignerListResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}

	_, err := dao.WorkflowDao.SelectByID(workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, err
	}

	signers, err := dao.WorkflowSignerDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}

	items := make([]WorkflowSignerItem, 0, len(signers))
	for _, signer := range signers {
		items = append(items, WorkflowSignerItem{
			SignerID:  signer.SignerID,
			StepIndex: signer.StepIndex,
		})
	}

	return &WorkflowSignerListResult{
		WorkflowID: workflowID,
		Signers:    items,
	}, nil
}
