package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"sign_flow_project/internal/dao"
	infradb "sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/service/file_service"
	pdfclient "sign_flow_project/internal/service/pdf_service_client"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type fillSignFieldContext struct {
	sigField              *model.DocumentFieldModel
	document              *model.DocumentModel
	pendingDateFields     []model.DocumentFieldModel
	today                 string
	sigValue              string
	expectedDocVersion    int
	expectedSourceFileKey string
}

// FillSignField 写入签名字段与日期到 PDF（经 pdf-service），成功后落库并记录 document_versions；不推进流程。
func (s *signingServiceImpl) FillSignField(workflowID, fieldID uint, req FillSignFieldRequest) (*FillSignFieldResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}
	if fieldID == 0 {
		return nil, fmt.Errorf("fieldId is required")
	}
	signerID := req.SignerID
	if signerID == 0 {
		return nil, fmt.Errorf("signerId is required")
	}
	sigValue := strings.TrimSpace(req.Value)
	if sigValue == "" {
		return nil, fmt.Errorf("value is required")
	}
	if err := validateFillSignFieldMode(req.Mode); err != nil {
		return nil, err
	}

	db := infradb.GetPostgres()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	ctx, err := loadFillSignFieldContext(workflowID, fieldID, signerID, sigValue)
	if err != nil {
		return nil, err
	}

	nextVersion := ctx.document.CurrentVersion + 1
	targetKey := file_service.FileService.BuildVersionFileKey(workflowID, ctx.document.ID, nextVersion)
	log.WithFields(log.Fields{
		"workflowId":            workflowID,
		"fieldId":               fieldID,
		"signerId":              signerID,
		"documentId":            ctx.document.ID,
		"currentVersion":        ctx.document.CurrentVersion,
		"sourceFile":            ctx.document.FilePath,
		"nextVersion":           nextVersion,
		"targetFile":            targetKey,
		"pendingDateFieldCount": len(ctx.pendingDateFields),
	}).Info("fill sign field precheck success")

	if err := file_service.FileService.EnsureParentDirsForFileKey(targetKey); err != nil {
		return nil, fmt.Errorf("prepare version directory failed: %w", err)
	}

	pdfReq := buildApplyFieldsPDFRequest(ctx, workflowID, targetKey)
	if _, err := pdfclient.DefaultClient().ApplyFields(pdfReq); err != nil {
		return nil, fmt.Errorf("pdf service: %w", err)
	}

	absTarget := file_service.FileService.AbsPathFromFileKey(targetKey)
	st, err := os.Stat(absTarget)
	if err != nil || st.IsDir() {
		removeFileWithLog(absTarget, "cleanup target file when signed pdf missing")
		if err != nil {
			return nil, fmt.Errorf("signed pdf not found after pdf service: %w", err)
		}
		return nil, fmt.Errorf("signed pdf not found after pdf service")
	}
	newFileSize := st.Size()
	log.WithFields(log.Fields{
		"workflowId":     workflowID,
		"fieldId":        fieldID,
		"signerId":       signerID,
		"targetFile":     targetKey,
		"outputExists":   true,
		"outputFileSize": newFileSize,
	}).Info("fill sign field pdf output ready")

	var result *FillSignFieldResult
	err = db.Transaction(func(tx *gorm.DB) error {
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
		if currentTask.SignerID != signerID {
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
		if field.SignerID != signerID {
			return fmt.Errorf("field signer mismatch")
		}
		if !strings.EqualFold(strings.TrimSpace(field.FieldType), "signature") {
			return fmt.Errorf("field must be signature type")
		}
		if !strings.EqualFold(strings.TrimSpace(field.Status), string(model.DocumentFieldStatusPending)) {
			return fmt.Errorf("field is not pending")
		}

		document, err := dao.DocumentDao.SelectByIDTx(tx, workflow.DocumentID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("document not found")
			}
			return err
		}
		if document.CurrentVersion != ctx.expectedDocVersion {
			return fmt.Errorf("document version changed during signing, please retry")
		}
		if strings.TrimSpace(document.FilePath) != ctx.expectedSourceFileKey {
			return fmt.Errorf("document file path changed during signing, please retry")
		}

		for i := range ctx.pendingDateFields {
			df, err := dao.DocumentFieldDao.SelectByIDTx(tx, ctx.pendingDateFields[i].ID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("date field not found")
				}
				return err
			}
			if !strings.EqualFold(strings.TrimSpace(df.FieldType), "date") {
				return fmt.Errorf("field state changed, please retry")
			}
			if !strings.EqualFold(strings.TrimSpace(df.Status), string(model.DocumentFieldStatusPending)) {
				return fmt.Errorf("field state changed, please retry")
			}
			if df.SignerID != signerID || df.WorkflowID != workflowID {
				return fmt.Errorf("field state changed, please retry")
			}
		}

		field.Value = sigValue
		field.Status = string(model.DocumentFieldStatusFilled)
		if err := dao.DocumentFieldDao.UpdateTx(tx, field); err != nil {
			return err
		}

		autoFilled := 0
		for i := range ctx.pendingDateFields {
			df := &ctx.pendingDateFields[i]
			row, err := dao.DocumentFieldDao.SelectByIDTx(tx, df.ID)
			if err != nil {
				return err
			}
			row.Value = ctx.today
			row.Status = string(model.DocumentFieldStatusFilled)
			if err := dao.DocumentFieldDao.UpdateTx(tx, row); err != nil {
				return err
			}
			autoFilled++
		}

		document.FilePath = filepathToSlash(targetKey)
		document.CurrentVersion = nextVersion
		document.FileSize = newFileSize
		if document.Status == model.DocumentStatusReady {
			document.Status = model.DocumentStatusSigning
		}

		if err := dao.DocumentDao.UpdateTx(tx, document); err != nil {
			return err
		}

		ver := &model.DocumentVersionModel{
			DocumentID: document.ID,
			WorkflowID: workflowID,
			VersionNo:  nextVersion,
			FilePath:   filepathToSlash(targetKey),
			FileName:   versionFileNameFromKey(targetKey),
			FileSize:   newFileSize,
			SignerID:   signerID,
			ActionType: model.DocumentVersionActionSignApplied,
			Remark:     "",
		}
		if err := dao.DocumentVersionDao.CreateTx(tx, ver); err != nil {
			return err
		}

		ft := strings.TrimSpace(strings.ToLower(field.FieldType))
		result = &FillSignFieldResult{
			WorkflowID:      workflowID,
			FieldID:         field.ID,
			SignerID:        signerID,
			FieldType:       ft,
			Status:          string(model.DocumentFieldStatusFilled),
			Value:           sigValue,
			AutoFilledDates: autoFilled,
			DocumentID:      document.ID,
			DocumentVersion: document.CurrentVersion,
			FilePath:        document.FilePath,
		}
		return nil
	})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"workflowId": workflowID,
			"fieldId":    fieldID,
			"signerId":   signerID,
			"documentId": ctx.document.ID,
			"targetFile": targetKey,
		}).Error("fill sign field transaction failed, start cleanup")
		removeFileWithLog(absTarget, "rollback generated target file after db failure")
		return nil, err
	}
	log.WithFields(log.Fields{
		"workflowId":        workflowID,
		"fieldId":           fieldID,
		"signerId":          signerID,
		"documentId":        result.DocumentID,
		"newFilePath":       result.FilePath,
		"documentVersion":   result.DocumentVersion,
		"insertedVersionNo": result.DocumentVersion,
	}).Info("fill sign field committed successfully")
	return result, nil
}

func removeFileWithLog(absPath string, reason string) {
	if strings.TrimSpace(absPath) == "" {
		return
	}
	if err := os.Remove(absPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.WithError(err).WithFields(log.Fields{
			"path":   absPath,
			"reason": reason,
		}).Warn("remove temporary signed pdf failed")
	}
}

func filepathToSlash(key string) string {
	key = strings.TrimSpace(key)
	return strings.ReplaceAll(key, "\\", "/")
}

func versionFileNameFromKey(fileKey string) string {
	fileKey = strings.TrimSpace(fileKey)
	if fileKey == "" {
		return ""
	}
	idx := strings.LastIndex(strings.ReplaceAll(fileKey, "\\", "/"), "/")
	if idx < 0 || idx >= len(fileKey)-1 {
		return fileKey
	}
	return fileKey[idx+1:]
}

func loadFillSignFieldContext(workflowID, fieldID, signerID uint, sigValue string) (*fillSignFieldContext, error) {
	today := time.Now().Format("2006-01-02")

	workflow, err := dao.WorkflowDao.SelectByID(workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, err
	}
	if workflow.Status != model.WorkflowStatusPending {
		return nil, fmt.Errorf("workflow is not pending")
	}

	currentTask, err := dao.TaskDao.SelectCurrentPendingByWorkflowID(workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("current pending task not found")
		}
		return nil, err
	}
	if currentTask.SignerID != signerID {
		return nil, fmt.Errorf("current pending task does not belong to signer")
	}
	if currentTask.StepIndex != workflow.CurrentStep {
		return nil, fmt.Errorf("workflow current step does not match pending task")
	}

	field, err := dao.DocumentFieldDao.SelectByID(fieldID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("field not found")
		}
		return nil, err
	}
	if field.WorkflowID != workflowID {
		return nil, fmt.Errorf("field does not belong to this workflow")
	}
	if field.SignerID != signerID {
		return nil, fmt.Errorf("field signer mismatch")
	}
	if !strings.EqualFold(strings.TrimSpace(field.FieldType), "signature") {
		return nil, fmt.Errorf("field must be signature type")
	}
	if !strings.EqualFold(strings.TrimSpace(field.Status), string(model.DocumentFieldStatusPending)) {
		return nil, fmt.Errorf("field is not pending")
	}

	document, err := dao.DocumentDao.SelectByID(workflow.DocumentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("document not found")
		}
		return nil, err
	}
	srcKey := strings.TrimSpace(document.FilePath)
	if srcKey == "" {
		return nil, fmt.Errorf("document file path is empty")
	}
	if _, err := file_service.FileService.OpenDocumentByFileKey(srcKey); err != nil {
		return nil, fmt.Errorf("source document file not accessible: %w", err)
	}

	signerFields, err := dao.DocumentFieldDao.SelectByWorkflowIDAndSignerID(workflowID, signerID)
	if err != nil {
		return nil, err
	}
	var pendingDates []model.DocumentFieldModel
	for i := range signerFields {
		f := signerFields[i]
		if !strings.EqualFold(strings.TrimSpace(f.FieldType), "date") {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(f.Status), string(model.DocumentFieldStatusPending)) {
			continue
		}
		pendingDates = append(pendingDates, f)
	}

	return &fillSignFieldContext{
		sigField:              field,
		document:              document,
		pendingDateFields:     pendingDates,
		today:                 today,
		sigValue:              sigValue,
		expectedDocVersion:    document.CurrentVersion,
		expectedSourceFileKey: filepathToSlash(srcKey),
	}, nil
}

func buildApplyFieldsPDFRequest(ctx *fillSignFieldContext, workflowID uint, targetKey string) pdfclient.ApplyFieldsRequest {
	fields := make([]pdfclient.ApplyFieldPayload, 0, 1+len(ctx.pendingDateFields))

	sf := ctx.sigField
	fields = append(fields, pdfclient.ApplyFieldPayload{
		FieldID:    sf.ID,
		FieldType:  strings.TrimSpace(strings.ToLower(sf.FieldType)),
		PageNumber: sf.PageNumber,
		X:          sf.X,
		Y:          sf.Y,
		Width:      sf.Width,
		Height:     sf.Height,
		Required:   sf.Required,
		Value:      ctx.sigValue,
	})
	for i := range ctx.pendingDateFields {
		df := &ctx.pendingDateFields[i]
		fields = append(fields, pdfclient.ApplyFieldPayload{
			FieldID:    df.ID,
			FieldType:  strings.TrimSpace(strings.ToLower(df.FieldType)),
			PageNumber: df.PageNumber,
			X:          df.X,
			Y:          df.Y,
			Width:      df.Width,
			Height:     df.Height,
			Required:   df.Required,
			Value:      ctx.today,
		})
	}

	return pdfclient.ApplyFieldsRequest{
		RequestID:  newPDFRequestID(),
		WorkflowID: workflowID,
		DocumentID: ctx.document.ID,
		SignerID:   ctx.sigField.SignerID,
		SourceFile: filepathToSlash(strings.TrimSpace(ctx.document.FilePath)),
		TargetFile: filepathToSlash(targetKey),
		WriteMode:  "replace",
		Fields:     fields,
	}
}

func newPDFRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
