package model

type WorkflowModel struct {
	Model
	DocumentId  uint   `json:"documentId" gorm:"column:document_id;type:integer;not null"`
	CurrentStep int    `json:"currentStep" gorm:"column:current_step;type:integer;not null;default:0"`
	Status      string `json:"status" gorm:"column:status;type:varchar(255);not null;default:'pending'"`
}
