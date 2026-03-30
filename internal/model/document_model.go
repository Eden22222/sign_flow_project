package model

type DocumentModel struct {
	Model
	Title          string `json:"title" gorm:"column:title;type:varchar(255);not null"`
	Description    string `json:"description" gorm:"column:description;type:text"`
	FileName       string `json:"fileName" gorm:"column:file_name;type:varchar(255)"`
	FilePath       string `json:"filePath" gorm:"column:file_path;type:text"`
	FileSize       int64  `json:"fileSize" gorm:"column:file_size;type:bigint;not null;default:0"`
	FileType       string `json:"fileType" gorm:"column:file_type;type:varchar(100)"`
	CurrentVersion int    `json:"currentVersion" gorm:"column:current_version;type:integer;not null;default:1"`
	Status         string `json:"status" gorm:"column:status;type:varchar(50);not null;default:'draft'"`
}

// 为了兼容现有 service / 响应结构体引用，保留 DocumentStatus 作为 string 的别名。
type DocumentStatus = string

const (
	DocumentStatusDraft     DocumentStatus = "draft"
	DocumentStatusReady     DocumentStatus = "ready"
	DocumentStatusSigning   DocumentStatus = "signing"
	DocumentStatusCompleted DocumentStatus = "completed"
)
