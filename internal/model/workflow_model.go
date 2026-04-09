package model

import "time"

type WorkflowModel struct {
	Model
	DocumentID  uint             `json:"documentId" gorm:"column:document_id;type:integer;not null"`
	InitiatorID uint             `json:"initiatorId" gorm:"column:initiator_id;type:integer;not null"`
	DueAt       *time.Time       `json:"dueAt" gorm:"column:due_at;type:timestamp"`
	Priority    WorkflowPriority `json:"priority" gorm:"column:priority;type:varchar(20);not null;default:'normal'"`
	CurrentStep int              `json:"currentStep" gorm:"column:current_step;type:integer;not null;default:1"`
	Status      WorkflowStatus   `json:"status" gorm:"column:status;type:varchar(50);not null;default:'pending'"`
}

type WorkflowStatus string
type WorkflowPriority string

const (
	WorkflowStatusDraft     WorkflowStatus = "draft"
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

const (
	WorkflowPriorityNormal WorkflowPriority = "normal"
	WorkflowPriorityHigh   WorkflowPriority = "high"
	WorkflowPriorityUrgent WorkflowPriority = "urgent"
)
