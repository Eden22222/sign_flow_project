package model

import (
	"gorm.io/gorm"
	"time"
)

type Model struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt *time.Time     `json:"createdAt" time_format:"sql_datetime" time_utc:"false"`
	UpdatedAt *time.Time     `json:"updatedAt" time_format:"sql_datetime" time_utc:"false"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index"`
}
