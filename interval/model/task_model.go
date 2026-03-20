package model

type TaskModel struct {
	Model
	WorkflowId uint   `json:"workflowId" gorm:"column:workflow_id;type:integer;not null"`
	SignerId   uint   `json:"signerId" gorm:"column:signer_id;type:integer;not null"`
	StepIndex  int    `json:"stepIndex" gorm:"column:step_index;type:integer;not null"`
	Status     string `json:"status" gorm:"column:status;type:varchar(255);not null"`
}
