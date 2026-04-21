package model

import "time"

type TaskModel struct {
	Model
	WorkflowID uint       `json:"workflowId" gorm:"column:workflow_id;type:integer;not null"`
	SignerID   uint       `json:"signerId" gorm:"column:signer_id;type:integer;not null"`
	StepIndex  int        `json:"stepIndex" gorm:"column:step_index;type:integer;not null"`
	Status     TaskStatus `json:"status" gorm:"column:status;type:varchar(50);not null;default:'pending'"`
	SignedAt   *time.Time `json:"signedAt" gorm:"column:signed_at"`
}

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusSigned  TaskStatus = "signed"
)
