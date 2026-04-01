package service

import (
	"errors"
	"fmt"
	"strings"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"
	usersvc "sign_flow_project/internal/service/user_service"
	"sign_flow_project/pkg/util"

	"gorm.io/gorm"
)

type WorkflowDetailSignerItem struct {
	SignerID  string `json:"signerId"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	StepIndex int    `json:"stepIndex"`
	Status    string `json:"status"`
}

type WorkflowDetailResult struct {
	WorkflowID      uint                       `json:"workflowId"`
	DocumentID      uint                       `json:"documentId"`
	Title           string                     `json:"title"`
	CurrentStep     int                        `json:"currentStep"`
	WorkflowStatus  model.WorkflowStatus       `json:"workflowStatus"`
	DocumentStatus  model.DocumentStatus       `json:"documentStatus"`
	DocumentVersion int                        `json:"documentVersion"`
	CurrentSignerID string                     `json:"currentSignerId"`
	InitiatorID     string                     `json:"initiatorId"`
	Initiator       string                     `json:"initiator"`
	InitiatorAvatar string                     `json:"initiatorAvatar"`
	Description     string                     `json:"description"`
	FileName        string                     `json:"fileName"`
	CreatedAt       string                     `json:"createdAt"`
	Signers         []WorkflowDetailSignerItem `json:"signers"`
}

func (s *workflowQueryServiceImpl) GetDetail(workflowID uint) (*WorkflowDetailResult, error) {
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

	wfSigners, err := dao.WorkflowSignerDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, 1+len(wfSigners))
	if ic := strings.TrimSpace(workflow.InitiatorID); ic != "" {
		codes = append(codes, ic)
	}
	for i := range wfSigners {
		codes = append(codes, wfSigners[i].SignerID)
	}
	userMap, err := usersvc.UserService.BatchGetMapByUserCodes(codes)
	if err != nil {
		return nil, err
	}

	initiatorID := strings.TrimSpace(workflow.InitiatorID)
	initiatorName := ""
	initiatorAvatar := ""
	if initiatorID != "" {
		if u, ok := userMap[initiatorID]; ok {
			initiatorName = u.Name
			initiatorAvatar = u.Avatar
		}
	}

	signerItems := make([]WorkflowDetailSignerItem, 0, len(wfSigners))
	for i := range wfSigners {
		ws := wfSigners[i]
		name := ""
		avatar := ""
		if u, ok := userMap[ws.SignerID]; ok {
			name = u.Name
			avatar = u.Avatar
		}
		signerItems = append(signerItems, WorkflowDetailSignerItem{
			SignerID:  ws.SignerID,
			Name:      name,
			Avatar:    avatar,
			StepIndex: ws.StepIndex,
			Status:    util.BuildSignerStepStatus(workflow, ws.StepIndex),
		})
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
		InitiatorID:     initiatorID,
		Initiator:       initiatorName,
		InitiatorAvatar: initiatorAvatar,
		Description:     document.Description,
		FileName:        document.FileName,
		CreatedAt:       util.FormatWorkflowCreatedAt(workflow.CreatedAt),
		Signers:         signerItems,
	}, nil
}

// --- 签署执行页详情（子功能 A）---

type SigningDetailSignerItem struct {
	SignerID  string `json:"signerId"`
	Name      string `json:"name"`
	Avatar    string `json:"avatar"`
	StepIndex int    `json:"stepIndex"`
	Status    string `json:"status"`
}

type SigningProgress struct {
	CompletedCount int `json:"completedCount"`
	TotalCount     int `json:"totalCount"`
}

type SigningDetailResult struct {
	WorkflowID      uint                      `json:"workflowId"`
	DocumentID      uint                      `json:"documentId"`
	WorkflowTitle   string                    `json:"workflowTitle"`
	DocumentName    string                    `json:"documentName"`
	CurrentStep     int                       `json:"currentStep"`
	TotalSteps      int                       `json:"totalSteps"`
	CurrentSignerID string                    `json:"currentSignerId"`
	WorkflowStatus  model.WorkflowStatus      `json:"workflowStatus"`
	DocumentStatus  model.DocumentStatus      `json:"documentStatus"`
	CreatedAt       string                    `json:"createdAt"`
	DueDate         string                    `json:"dueDate"`
	Signers         []SigningDetailSignerItem `json:"signers"`
	Progress        SigningProgress           `json:"progress"`
}

func (s *workflowQueryServiceImpl) GetSigningDetail(workflowID uint) (*SigningDetailResult, error) {
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

	wfSigners, err := dao.WorkflowSignerDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(wfSigners))
	for i := range wfSigners {
		codes = append(codes, wfSigners[i].SignerID)
	}
	userMap, err := usersvc.UserService.BatchGetMapByUserCodes(codes)
	if err != nil {
		return nil, err
	}

	signerItems := make([]SigningDetailSignerItem, 0, len(wfSigners))
	for i := range wfSigners {
		ws := wfSigners[i]
		name := ""
		avatar := ""
		if u, ok := userMap[ws.SignerID]; ok {
			name = u.Name
			avatar = u.Avatar
		}
		signerItems = append(signerItems, SigningDetailSignerItem{
			SignerID:  ws.SignerID,
			Name:      name,
			Avatar:    avatar,
			StepIndex: ws.StepIndex,
			Status:    util.BuildSignerStepStatus(workflow, ws.StepIndex),
		})
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

	tasks, err := dao.TaskDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}
	completedCount := 0
	for _, t := range tasks {
		if t.Status == model.TaskStatusSigned {
			completedCount++
		}
	}
	totalCount := len(wfSigners)
	if totalCount == 0 && len(tasks) > 0 {
		totalCount = len(tasks)
	}

	return &SigningDetailResult{
		WorkflowID:      workflow.ID,
		DocumentID:      document.ID,
		WorkflowTitle:   document.Title,
		DocumentName:    document.FileName,
		CurrentStep:     workflow.CurrentStep,
		TotalSteps:      len(wfSigners),
		CurrentSignerID: currentSignerID,
		WorkflowStatus:  workflow.Status,
		DocumentStatus:  document.Status,
		CreatedAt:       util.FormatWorkflowCreatedAt(workflow.CreatedAt),
		DueDate:         "",
		Signers:         signerItems,
		Progress: SigningProgress{
			CompletedCount: completedCount,
			TotalCount:     totalCount,
		},
	}, nil
}

// --- 签署字段列表（子功能 A）---

type SignFieldItem struct {
	FieldID              uint    `json:"fieldId"`
	DocumentID           uint    `json:"documentId"`
	WorkflowID           uint    `json:"workflowId"`
	SignerID             string  `json:"signerId"`
	FieldType            string  `json:"fieldType"`
	PageNumber           int     `json:"pageNumber"`
	X                    float64 `json:"x"`
	Y                    float64 `json:"y"`
	Width                float64 `json:"width"`
	Height               float64 `json:"height"`
	Required             bool    `json:"required"`
	Status               string  `json:"status"`
	Value                string  `json:"value"`
	IsCurrentSignerField bool    `json:"isCurrentSignerField"`
	Clickable            bool    `json:"clickable"`
}

type SignFieldListResult struct {
	WorkflowID      uint            `json:"workflowId"`
	DocumentID      uint            `json:"documentId"`
	CurrentSignerID string          `json:"currentSignerId"`
	Items           []SignFieldItem `json:"items"`
}

func (s *workflowQueryServiceImpl) GetSignFields(workflowID uint) (*SignFieldListResult, error) {
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

	documentID := workflow.DocumentID

	currentSignerID := ""
	currentTask, err := dao.TaskDao.SelectCurrentPendingByWorkflowID(workflowID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		currentSignerID = strings.TrimSpace(currentTask.SignerID)
	}

	fields, err := dao.DocumentFieldDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}

	items := make([]SignFieldItem, 0, len(fields))
	for i := range fields {
		f := fields[i]
		sid := strings.TrimSpace(f.SignerID)
		isCurrent := currentSignerID != "" && sid == currentSignerID
		clickable := workflow.Status == model.WorkflowStatusPending &&
			strings.EqualFold(strings.TrimSpace(f.Status), string(model.DocumentFieldStatusPending)) &&
			strings.EqualFold(strings.TrimSpace(f.FieldType), "signature") &&
			isCurrent

		items = append(items, SignFieldItem{
			FieldID:              f.ID,
			DocumentID:           f.DocumentID,
			WorkflowID:           f.WorkflowID,
			SignerID:             f.SignerID,
			FieldType:            f.FieldType,
			PageNumber:           f.PageNumber,
			X:                    f.X,
			Y:                    f.Y,
			Width:                f.Width,
			Height:               f.Height,
			Required:             f.Required,
			Status:               f.Status,
			Value:                f.Value,
			IsCurrentSignerField: isCurrent,
			Clickable:            clickable,
		})
	}

	return &SignFieldListResult{
		WorkflowID:      workflow.ID,
		DocumentID:      documentID,
		CurrentSignerID: currentSignerID,
		Items:           items,
	}, nil
}
