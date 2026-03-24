package service

import (
	"errors"
	"fmt"
	"strings"

	"sign_flow_project/internal/dao"
	infradb "sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type workflowServiceImpl struct{}

var WorkflowService = new(workflowServiceImpl)

type CreateWorkflowRequest struct {
	Title   string   `json:"title"`
	Signers []string `json:"signers"`
}

type CreateWorkflowResult struct {
	DocumentID  uint   `json:"documentId"`
	WorkflowID  uint   `json:"workflowId"`
	FirstSigner string `json:"firstSigner"`
}

// create workflow
func (s *workflowServiceImpl) CreateWorkflow(req CreateWorkflowRequest) (*CreateWorkflowResult, error) {
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
			FirstSigner: strings.TrimSpace(req.Signers[0]),
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
