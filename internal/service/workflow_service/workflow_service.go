package service

import (
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
