package service

import (
	"errors"
	"strings"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/util"
)

type workflowQueryServiceImpl struct{}

var WorkflowQueryService = new(workflowQueryServiceImpl)

// WorkflowListRequest handler 解析 query 后传入，由本层做校验与归一化。
type WorkflowListRequest struct {
	UserID   uint
	View     string
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

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

const (
	workflowListViewInitiated = "initiated"
	workflowListViewAssigned  = "assigned"
)

func (s *workflowQueryServiceImpl) List(req WorkflowListRequest) (*WorkflowListResult, error) {
	view := strings.TrimSpace(req.View)
	if view == "" {
		return nil, errors.New("view is required")
	}
	if view != workflowListViewInitiated && view != workflowListViewAssigned {
		return nil, errors.New("invalid view: must be initiated or assigned")
	}

	status := strings.TrimSpace(req.Status)
	if status != "" {
		switch model.WorkflowStatus(status) {
		case model.WorkflowStatusDraft,
			model.WorkflowStatusPending,
			model.WorkflowStatusCompleted,
			model.WorkflowStatusCancelled:
		default:
			return nil, errors.New("invalid status: must be draft, pending, completed, or cancelled")
		}
	}

	keyword := strings.TrimSpace(req.Keyword)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	workflows, total, err := dao.WorkflowDao.SelectPageByUserFilters(req.UserID, view, status, keyword, page, pageSize)
	if err != nil {
		return nil, err
	}

	if len(workflows) == 0 {
		return &WorkflowListResult{
			List:     []WorkflowListItem{},
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	initiatorIDs := make([]uint, 0, len(workflows))
	docIDs := make([]uint, 0, len(workflows))
	wfIDs := make([]uint, 0, len(workflows))
	for _, wf := range workflows {
		wfIDs = append(wfIDs, wf.ID)
		if wf.InitiatorID != 0 {
			initiatorIDs = append(initiatorIDs, wf.InitiatorID)
		}
		if wf.DocumentID != 0 {
			docIDs = append(docIDs, wf.DocumentID)
		}
	}

	initiatorUserMap, err := usersvc.UserService.BatchGetMapByIDs(initiatorIDs)
	if err != nil {
		return nil, err
	}

	docs, err := dao.DocumentDao.SelectByIDs(docIDs)
	if err != nil {
		return nil, err
	}
	docByID := make(map[uint]model.DocumentModel, len(docs))
	for _, d := range docs {
		docByID[d.ID] = d
	}

	signerCountMap, err := dao.WorkflowSignerDao.CountSignersByWorkflowIDs(wfIDs)
	if err != nil {
		return nil, err
	}

	list := make([]WorkflowListItem, 0, len(workflows))
	for _, workflow := range workflows {
		document, ok := docByID[workflow.DocumentID]
		if !ok {
			continue
		}

		signerCount := signerCountMap[workflow.ID]

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
