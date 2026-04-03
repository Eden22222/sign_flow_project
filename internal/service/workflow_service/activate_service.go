package service

import (
	"errors"
	"fmt"

	"sign_flow_project/internal/dao"
	infradb "sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type ActivateWorkflowResult struct {
	WorkflowID      uint   `json:"workflowId"`
	DocumentID      uint   `json:"documentId"`
	CurrentStep     int    `json:"currentStep"`
	WorkflowStatus  string `json:"workflowStatus"`
	DocumentStatus  string `json:"documentStatus"`
	CurrentSignerID uint   `json:"currentSignerId"`
}

func (s *draftWorkflowServiceImpl) ActivateWorkflow(workflowID uint, currentUserID uint) (*ActivateWorkflowResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	if currentUserID == 0 {
		return nil, fmt.Errorf("current user is required")
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
		if workflow.InitiatorID != currentUserID {
			return fmt.Errorf("only initiator can activate workflow")
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
		if firstSigner == nil || firstSigner.SignerID == 0 {
			return fmt.Errorf("stepIndex=1 signer not found")
		}

		fields, err := dao.DocumentFieldDao.SelectByWorkflowIDTx(tx, workflowID)
		if err != nil {
			return err
		}
		if len(fields) == 0 {
			return fmt.Errorf("at least one field is required before activation")
		}

		signerSet := make(map[uint]struct{}, len(signers))
		for _, s := range signers {
			signerSet[s.SignerID] = struct{}{}
		}
		for _, f := range fields {
			if _, ok := signerSet[f.SignerID]; !ok {
				return fmt.Errorf("signer %d is not in current workflow", f.SignerID)
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
			SignerID:   firstSigner.SignerID,
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
