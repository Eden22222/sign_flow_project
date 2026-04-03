package service

import (
	"errors"

	"sign_flow_project/internal/dao"
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/util"

	"gorm.io/gorm"
)

type workflowQueryServiceImpl struct{}

var WorkflowQueryService = new(workflowQueryServiceImpl)

type WorkflowListItem struct {
	WorkflowID     uint   `json:"workflowId"`
	Title          string `json:"title"`
	FileName       string `json:"fileName"`
	DocumentStatus string `json:"documentStatus"`
	Initiator      string `json:"initiator"`
	SignerCount    int    `json:"signerCount"`
	CurrentStep    int    `json:"currentStep"`
	TotalSteps     int    `json:"totalSteps"`
	WorkflowStatus string `json:"workflowStatus"`
	CreatedAt      string `json:"createdAt"`
}

type WorkflowListResult struct {
	List     []WorkflowListItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

func (s *workflowQueryServiceImpl) List(page int, pageSize int) (*WorkflowListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	workflows, total, err := dao.WorkflowDao.SelectPage(page, pageSize)
	if err != nil {
		return nil, err
	}

	initiatorIDs := make([]uint, 0, len(workflows))
	for _, wf := range workflows {
		if wf.InitiatorID != 0 {
			initiatorIDs = append(initiatorIDs, wf.InitiatorID)
		}
	}
	initiatorUserMap, err := usersvc.UserService.BatchGetMapByIDs(initiatorIDs)
	if err != nil {
		return nil, err
	}

	list := make([]WorkflowListItem, 0, len(workflows))
	for _, workflow := range workflows {
		document, err := dao.DocumentDao.SelectByID(workflow.DocumentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}

		signers, err := dao.WorkflowSignerDao.SelectByWorkflowID(workflow.ID)
		if err != nil {
			return nil, err
		}
		signerCount := len(signers)

		initiatorName := ""
		if u, ok := initiatorUserMap[workflow.InitiatorID]; ok {
			initiatorName = u.Name
		}

		list = append(list, WorkflowListItem{
			WorkflowID:     workflow.ID,
			Title:          document.Title,
			FileName:       document.FileName,
			DocumentStatus: document.Status,
			Initiator:      initiatorName,
			SignerCount:    signerCount,
			CurrentStep:    workflow.CurrentStep,
			TotalSteps:     signerCount,
			WorkflowStatus: string(workflow.Status),
			CreatedAt:      util.FormatWorkflowCreatedAt(workflow.CreatedAt),
		})
	}

	return &WorkflowListResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
