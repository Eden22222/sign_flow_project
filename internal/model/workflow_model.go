package model

type WorkflowModel struct {
	Model
	DocumentID  uint           `json:"documentId" gorm:"column:document_id;type:integer;not null"`
	InitiatorID string         `json:"initiatorId" gorm:"column:initiator_id;type:varchar(50);not null;default:''"`
	CurrentStep int            `json:"currentStep" gorm:"column:current_step;type:integer;not null;default:1"`
	Status      WorkflowStatus `json:"status" gorm:"column:status;type:varchar(50);not null;default:'pending'"`
}

type WorkflowStatus string

const (
	WorkflowStatusDraft     WorkflowStatus = "draft"
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)