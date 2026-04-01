package model

type DocumentFieldStatus string

const (
	DocumentFieldStatusPending DocumentFieldStatus = "pending"
	DocumentFieldStatusFilled  DocumentFieldStatus = "filled"
)

type DocumentFieldModel struct {
	Model

	DocumentID uint   `json:"documentId" gorm:"column:document_id;type:integer;not null;index"`
	WorkflowID uint   `json:"workflowId" gorm:"column:workflow_id;type:integer;not null;index"`
	SignerID   string `json:"signerId" gorm:"column:signer_id;type:varchar(100);not null;index"`

	FieldType  string  `json:"fieldType" gorm:"column:field_type;type:varchar(50);not null"`
	PageNumber int     `json:"pageNumber" gorm:"column:page_number;type:integer;not null"`
	X          float64 `json:"x" gorm:"column:x;type:double precision;not null"`
	Y          float64 `json:"y" gorm:"column:y;type:double precision;not null"`
	Width      float64 `json:"width" gorm:"column:width;type:double precision;not null"`
	Height     float64 `json:"height" gorm:"column:height;type:double precision;not null"`

	Required bool   `json:"required" gorm:"column:required;type:boolean;not null;default:false"`
	Status   string `json:"status" gorm:"column:status;type:varchar(50);not null;default:'pending'"`
	// Value 存签署文本或 draw/upload 的 base64、data URL 等；不做独立资源表。
	Value string `json:"value" gorm:"column:value;type:text"`
}
