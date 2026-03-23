package model

type DocumentModel struct {
	Model
	Title          string `json:"title" gorm:"column:title;type:varchar(255);not null"`
	CurrentVersion int    `json:"currentVersion" gorm:"column:current_version;type:integer;not null;default:1"`
	Status         DocumentStatus `json:"status" gorm:"column:status;type:varchar(50);not null;default:'draft'"`
}

type DocumentStatus string

const (
	DocumentStatusDraft     DocumentStatus = "draft"
	DocumentStatusSigning   DocumentStatus = "signing"
	DocumentStatusCompleted DocumentStatus = "completed"
)