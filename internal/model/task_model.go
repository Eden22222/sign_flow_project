package model

type TaskModel struct {
	Model
	WorkflowID uint   `json:"workflowId" gorm:"column:workflow_id;type:integer;not null"`
	SignerID   string `json:"signerId" gorm:"column:signer_id;type:varchar(100);not null"`
	StepIndex  int    `json:"stepIndex" gorm:"column:step_index;type:integer;not null"`
	Status     TaskStatus `json:"status" gorm:"column:status;type:varchar(50);not null;default:'pending'"`
}

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusSigned  TaskStatus = "signed"
)
