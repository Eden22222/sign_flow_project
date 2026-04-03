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

func normalizeSignerIDToUserCode(rawSignerID string, emailToUserCode map[string]string) string {
	sid := strings.TrimSpace(rawSignerID)
	if sid == "" {
		return sid
	}
	if !strings.Contains(sid, "@") {
		return sid
	}
	if code, ok := emailToUserCode[sid]; ok {
		return code
	}
	return sid
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

func (s *draftWorkflowServiceImpl) SaveWorkflowFields(workflowID uint, currentUserCode string, req SaveWorkflowFieldsRequest) (*SaveWorkflowFieldsResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	currentUserCode = strings.TrimSpace(currentUserCode)
	if currentUserCode == "" {
		return nil, fmt.Errorf("current user is required")
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
		if strings.TrimSpace(workflow.InitiatorID) != currentUserCode {
			return fmt.Errorf("only initiator can edit workflow fields")
		}

		signers, err := dao.WorkflowSignerDao.SelectByWorkflowIDTx(tx, workflowID)
		if err != nil {
			return err
		}
		signerSet := make(map[string]struct{}, len(signers))
		for _, s := range signers {
			signerSet[s.SignerID] = struct{}{}
		}

		fieldEmails := make([]string, 0, len(req.Fields))
		seenEmail := make(map[string]struct{}, len(req.Fields))
		for _, f := range req.Fields {
			sid := strings.TrimSpace(f.SignerID)
			if sid == "" || !strings.Contains(sid, "@") {
				continue
			}
			if _, exists := seenEmail[sid]; exists {
				continue
			}
			seenEmail[sid] = struct{}{}
			fieldEmails = append(fieldEmails, sid)
		}
		emailToUserCode := make(map[string]string, len(fieldEmails))
		if len(fieldEmails) > 0 {
			users, err := dao.UserDao.SelectByEmails(fieldEmails)
			if err != nil {
				return err
			}
			for _, u := range users {
				emailToUserCode[u.Email] = u.UserCode
			}
		}

		fields := make([]*model.DocumentFieldModel, 0, len(req.Fields))
		for i, f := range req.Fields {
			sid := normalizeSignerIDToUserCode(f.SignerID, emailToUserCode)
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
