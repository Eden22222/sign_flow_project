package pdf_service_client

// ApplyFieldsPath 为 pdf-service 上 apply-fields 的路径（拼在 PDF_SERVICE_BASE_URL 之后）。
const ApplyFieldsPath = "/api/v1/pdf/apply-fields"

// ApplyFieldsRequest 与 sign_flow_pdf_service 约定一致；sourceFile/targetFile 使用主服务 fileKey。
type ApplyFieldsRequest struct {
	RequestID  string              `json:"requestId"`
	WorkflowID uint                `json:"workflowId"`
	DocumentID uint                `json:"documentId"`
	SignerID   uint                `json:"signerId"`
	SourceFile string              `json:"sourceFile"`
	TargetFile string              `json:"targetFile"`
	WriteMode  string              `json:"writeMode"`
	Fields     []ApplyFieldPayload `json:"fields"`
}

// ApplyFieldPayload 单个写入 PDF 的字段。
type ApplyFieldPayload struct {
	FieldID    uint    `json:"fieldId"`
	FieldType  string  `json:"fieldType"`
	PageNumber int     `json:"pageNumber"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Required   bool    `json:"required"`
	Value      string  `json:"value"`
}

// ApplyFieldsResponse pdf-service 成功时 data 可为空对象。
type ApplyFieldsResponse struct {
	RawData []byte
}
