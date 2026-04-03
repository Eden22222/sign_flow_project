package model

type WorkflowSignerModel struct {
	Model
	WorkflowID uint   `json:"workflowId" gorm:"column:workflow_id;type:integer;not null;index"`
	SignerID   uint   `json:"signerId" gorm:"column:signer_id;type:integer;not null"`
	StepIndex  int    `json:"stepIndex" gorm:"column:step_index;type:integer;not null"`
}