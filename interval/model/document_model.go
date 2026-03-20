package model

type DocumentModel struct {
	Model
	Title          string `json:"title" gorm:"column:title;type:varchar(255);not null"`
	CurrentVersion int    `json:"currentVersion" gorm:"column:current_version;type:integer;not null"`
	Status         string `json:"status" gorm:"column:status;type:varchar(255);not null"`
}
