package service

import (
	"errors"
	"fmt"

	"sign_flow_project/internal/dao"
	infradb "sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type signingServiceImpl struct{}

var SigningService = new(signingServiceImpl)

type SubmitSigningRequest struct {
	SignerID string `json:"signerId"`
}

type SubmitSigningResult struct {
	WorkflowID      uint                 `json:"workflowId"`
	DocumentID      uint                 `json:"documentId"`
	SignedStep      int                  `json:"signedStep"`
	NextStep        int                  `json:"nextStep"`
	NextSignerID    string               `json:"nextSignerId"`
	WorkflowStatus  model.WorkflowStatus `json:"workflowStatus"`
	DocumentStatus  model.DocumentStatus `json:"documentStatus"`
	DocumentVersion int                  `json:"documentVersion"`
}

func (s *signingServiceImpl) Submit(workflowID uint, req SubmitSigningRequest) (*SubmitSigningResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	if req.SignerID == "" {
		return nil, fmt.Errorf("signerId is required")
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var result *SubmitSigningResult

	err := db.Transaction(func(tx *gorm.DB) error {
		workflow, err := dao.WorkflowDao.SelectByIDTx(tx, workflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("workflow not found")
			}
			return err
		}

		if workflow.Status != model.WorkflowStatusPending {
			return fmt.Errorf("workflow is not pending")
		}

		document, err := dao.DocumentDao.SelectByIDTx(tx, workflow.DocumentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("document not found")
			}
			return err
		}

		currentTask, err := dao.TaskDao.SelectCurrentPendingByWorkflowIDTx(tx, workflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("current pending task not found")
			}
			return err
		}

		if currentTask.StepIndex != workflow.CurrentStep {
			return fmt.Errorf("workflow current step does not match pending task")
		}

		if currentTask.SignerID != req.SignerID {
			return fmt.Errorf("current pending task does not belong to signer")
		}

		// 1. 当前任务改为已签
		currentTask.Status = model.TaskStatusSigned
		if err := dao.TaskDao.UpdateTx(tx, currentTask); err != nil {
			return err
		}

		// 2. 文档版本号 +1
		document.CurrentVersion += 1

		signedStep := workflow.CurrentStep
		nextStep := signedStep + 1

		// 3. 查下一步签署人（从 WorkflowSignerModel 取）
		nextWorkflowSigner, err := dao.WorkflowSignerDao.SelectByWorkflowIDAndStepIndexTx(tx, workflowID, nextStep)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 4. 没有下一步，说明流程完成
		if errors.Is(err, gorm.ErrRecordNotFound) {
			workflow.Status = model.WorkflowStatusCompleted
			document.Status = model.DocumentStatusCompleted

			if err := dao.WorkflowDao.UpdateTx(tx, workflow); err != nil {
				return err
			}
			if err := dao.DocumentDao.UpdateTx(tx, document); err != nil {
				return err
			}

			result = &SubmitSigningResult{
				WorkflowID:      workflow.ID,
				DocumentID:      document.ID,
				SignedStep:      signedStep,
				NextStep:        0,
				NextSignerID:    "",
				WorkflowStatus:  workflow.Status,
				DocumentStatus:  document.Status,
				DocumentVersion: document.CurrentVersion,
			}
			return nil
		}

		// 5. 有下一步，继续推进流程
		workflow.CurrentStep = nextStep
		workflow.Status = model.WorkflowStatusPending
		document.Status = model.DocumentStatusSigning

		if err := dao.WorkflowDao.UpdateTx(tx, workflow); err != nil {
			return err
		}
		if err := dao.DocumentDao.UpdateTx(tx, document); err != nil {
			return err
		}

		// 6. 创建下一步任务
		nextTask := &model.TaskModel{
			WorkflowID: workflow.ID,
			SignerID:   nextWorkflowSigner.SignerID,
			StepIndex:  nextWorkflowSigner.StepIndex,
			Status:     model.TaskStatusPending,
		}
		if err := dao.TaskDao.CreateTx(tx, nextTask); err != nil {
			return err
		}

		result = &SubmitSigningResult{
			WorkflowID:      workflow.ID,
			DocumentID:      document.ID,
			SignedStep:      signedStep,
			NextStep:        nextWorkflowSigner.StepIndex,
			NextSignerID:    nextWorkflowSigner.SignerID,
			WorkflowStatus:  workflow.Status,
			DocumentStatus:  document.Status,
			DocumentVersion: document.CurrentVersion,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
