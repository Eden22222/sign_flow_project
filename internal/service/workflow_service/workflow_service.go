package service

import (
	"fmt"

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
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if len(req.Signers) == 0 {
		return nil, fmt.Errorf("at least one signer is required")
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var result *CreateWorkflowResult

	err := db.Transaction(func(tx *gorm.DB) error {
		document := &model.DocumentModel{
			Title:          req.Title,
			CurrentVersion: 1,
			Status:         "draft",
		}
		if err := dao.DocumentDao.CreateTx(tx, document); err != nil {
			return err
		}

		workflow := &model.WorkflowModel{
			DocumentID:  document.ID,
			CurrentStep: 1,
			Status:      "pending",
		}
		if err := dao.WorkflowDao.CreateTx(tx, workflow); err != nil {
			return err
		}

		firstTask := &model.TaskModel{
			WorkflowID: workflow.ID,
			SignerID:   req.Signers[0],
			StepIndex:  1,
			Status:     "pending",
		}
		if err := dao.TaskDao.CreateTx(tx, firstTask); err != nil {
			return err
		}

		result = &CreateWorkflowResult{
			DocumentID:  document.ID,
			WorkflowID:  workflow.ID,
			FirstSigner: req.Signers[0],
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
