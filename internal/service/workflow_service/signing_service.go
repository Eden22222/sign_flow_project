package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"sign_flow_project/internal/dao"
	infradb "sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type signingServiceImpl struct{}

var SigningService = new(signingServiceImpl)

// SignerID 须由已挂 JWTAuth 的 handler 从 token 注入；body 中的 signerId 会被覆盖，勿信任前端。
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

// SignerID 须由已挂 JWTAuth 的 handler 从 token 注入；body 中的 signerId 会被覆盖，勿信任前端。
type FillSignFieldRequest struct {
	SignerID string `json:"signerId"`
	// Mode 仅用于与前端约定输入合法性（draw/type/upload），validateFillSignFieldMode 校验；不入库、不驱动分支逻辑。
	Mode  string `json:"mode"`
	Value string `json:"value"`
}

type FillSignFieldResult struct {
	WorkflowID      uint   `json:"workflowId"`
	FieldID         uint   `json:"fieldId"`
	SignerID        string `json:"signerId"`
	FieldType       string `json:"fieldType"`
	Status          string `json:"status"`
	Value           string `json:"value"`
	AutoFilledDates int    `json:"autoFilledDates"`
}

func validateFillSignFieldMode(mode string) error {
	m := strings.TrimSpace(strings.ToLower(mode))
	if m == "" {
		return nil
	}
	if m != "draw" && m != "type" && m != "upload" {
		return fmt.Errorf("mode must be draw, type or upload")
	}
	return nil
}

// FillSignField 保存当前签署人签名域值，并自动填充其待填 date 域；不推进流程。
func (s *signingServiceImpl) FillSignField(workflowID, fieldID uint, req FillSignFieldRequest) (*FillSignFieldResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	if fieldID == 0 {
		return nil, fmt.Errorf("fieldId is required")
	}
	signerID := strings.TrimSpace(req.SignerID)
	if signerID == "" {
		return nil, fmt.Errorf("signerId is required")
	}
	value := strings.TrimSpace(req.Value)
	if value == "" {
		return nil, fmt.Errorf("value is required")
	}
	if err := validateFillSignFieldMode(req.Mode); err != nil {
		return nil, err
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	today := time.Now().Format("2006-01-02")
	var result *FillSignFieldResult

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

		currentTask, err := dao.TaskDao.SelectCurrentPendingByWorkflowIDTx(tx, workflowID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("current pending task not found")
			}
			return err
		}
		if strings.TrimSpace(currentTask.SignerID) != signerID {
			return fmt.Errorf("current pending task does not belong to signer")
		}
		if currentTask.StepIndex != workflow.CurrentStep {
			return fmt.Errorf("workflow current step does not match pending task")
		}

		field, err := dao.DocumentFieldDao.SelectByIDTx(tx, fieldID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("field not found")
			}
			return err
		}
		if field.WorkflowID != workflowID {
			return fmt.Errorf("field does not belong to this workflow")
		}
		if strings.TrimSpace(field.SignerID) != signerID {
			return fmt.Errorf("field signer mismatch")
		}
		if !strings.EqualFold(strings.TrimSpace(field.FieldType), "signature") {
			return fmt.Errorf("field must be signature type")
		}
		if !strings.EqualFold(strings.TrimSpace(field.Status), string(model.DocumentFieldStatusPending)) {
			return fmt.Errorf("field is not pending")
		}

		field.Value = value
		field.Status = string(model.DocumentFieldStatusFilled)
		if err := dao.DocumentFieldDao.UpdateTx(tx, field); err != nil {
			return err
		}

		autoFilled := 0
		signerFields, err := dao.DocumentFieldDao.SelectByWorkflowIDAndSignerIDTx(tx, workflowID, signerID)
		if err != nil {
			return err
		}
		for i := range signerFields {
			f := &signerFields[i]
			if !strings.EqualFold(strings.TrimSpace(f.FieldType), "date") {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(f.Status), string(model.DocumentFieldStatusPending)) {
				continue
			}
			f.Value = today
			f.Status = string(model.DocumentFieldStatusFilled)
			if err := dao.DocumentFieldDao.UpdateTx(tx, f); err != nil {
				return err
			}
			autoFilled++
		}

		ft := strings.TrimSpace(strings.ToLower(field.FieldType))
		result = &FillSignFieldResult{
			WorkflowID:      workflowID,
			FieldID:         field.ID,
			SignerID:        signerID,
			FieldType:       ft,
			Status:          string(model.DocumentFieldStatusFilled),
			Value:           value,
			AutoFilledDates: autoFilled,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *signingServiceImpl) Submit(workflowID uint, req SubmitSigningRequest) (*SubmitSigningResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	signerID := strings.TrimSpace(req.SignerID)
	if signerID == "" {
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

		if strings.TrimSpace(currentTask.SignerID) != signerID {
			return fmt.Errorf("current pending task does not belong to signer")
		}

		signerFields, err := dao.DocumentFieldDao.SelectByWorkflowIDAndSignerIDTx(tx, workflowID, signerID)
		if err != nil {
			return err
		}
		for _, f := range signerFields {
			if !f.Required {
				continue
			}
			st := strings.TrimSpace(f.Status)
			if !strings.EqualFold(st, string(model.DocumentFieldStatusFilled)) {
				return fmt.Errorf("required sign fields not completed")
			}
			if strings.TrimSpace(f.Value) == "" {
				return fmt.Errorf("required sign fields not completed")
			}
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
