package model

type WorkflowSignerModel struct {
	Model
	WorkflowID uint   `json:"workflowId" gorm:"column:workflow_id;type:integer;not null;index"`
	SignerID   string `json:"signerId" gorm:"column:signer_id;type:varchar(50);not null"`
	StepIndex  int    `json:"stepIndex" gorm:"column:step_index;type:integer;not null"`
}