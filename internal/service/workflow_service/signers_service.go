package service

import (
	"errors"
	"fmt"

	"sign_flow_project/internal/dao"

	"gorm.io/gorm"
)

type WorkflowSignerItem struct {
	SignerID  uint   `json:"signerId"`
	StepIndex int    `json:"stepIndex"`
}

type WorkflowSignerListResult struct {
	WorkflowID uint                 `json:"workflowId"`
	Signers    []WorkflowSignerItem `json:"signers"`
}

func (s *workflowQueryServiceImpl) GetSigners(workflowID uint) (*WorkflowSignerListResult, error) {
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
