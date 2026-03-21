package model

type WorkflowModel struct {
	Model
	DocumentID  uint   `json:"documentId" gorm:"column:document_id;type:integer;not null"`
	CurrentStep int    `json:"currentStep" gorm:"column:current_step;type:integer;not null;default:1"`
	Status      string `json:"status" gorm:"column:status;type:varchar(50);not null;default:'pending'"`
}
