package model

const (
	DocumentVersionActionInitial     = "initial"
	DocumentVersionActionSignApplied = "sign_applied"
)

// DocumentVersionModel 文档文件版本元数据（fileKey 指向 storage 下实际 PDF）。
type DocumentVersionModel struct {
	Model

	DocumentID uint   `json:"documentId" gorm:"column:document_id;type:integer;not null;index;uniqueIndex:ux_document_versions_doc_ver"`
	WorkflowID uint   `json:"workflowId" gorm:"column:workflow_id;type:integer;not null;index"`
	VersionNo  int    `json:"versionNo" gorm:"column:version_no;type:integer;not null;uniqueIndex:ux_document_versions_doc_ver"`
	FilePath   string `json:"filePath" gorm:"column:file_path;type:text;not null"`
	FileName   string `json:"fileName" gorm:"column:file_name;type:varchar(255);not null"`
	FileSize   int64  `json:"fileSize" gorm:"column:file_size;type:bigint;not null;default:0"`
	SignerID   uint   `json:"signerId" gorm:"column:signer_id;type:integer;not null;default:0;index"`
	ActionType string `json:"actionType" gorm:"column:action_type;type:varchar(50);not null"`
	Remark     string `json:"remark" gorm:"column:remark;type:text"`
}

func (DocumentVersionModel) TableName() string {
	return "document_versions"
}
