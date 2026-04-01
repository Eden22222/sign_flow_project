package test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

// uploadSamplePDFFileKey 上传 testdata/sample.pdf，返回 fileKey（供 POST /workflows 使用）。
func uploadSamplePDFFileKey(t *testing.T, engine http.Handler) string {
	t.Helper()
	pdfPath := filepath.Join("testdata", "sample.pdf")
	rec := performMultipartFile(t, engine, http.MethodPost, "/api/v1/files/upload", "file", pdfPath, "application/pdf")
	if rec.Code != http.StatusOK {
		t.Fatalf("upload pdf status=%d body=%s", rec.Code, rec.Body.String())
	}
	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("upload unmarshal wrapper: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("upload code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}
	var data uploadFileResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("upload unmarshal data: %v", err)
	}
	if data.FileKey == "" {
		t.Fatalf("upload fileKey empty")
	}
	return data.FileKey
}

// createWorkflowDraftViaAPI 调用 POST /api/v1/workflows，返回草稿（workflow 为 draft，无 task）。
func createWorkflowDraftViaAPI(t *testing.T, engine http.Handler, title, initiator string, signers []string) createWorkflowResp {
	t.Helper()
	fileKey := uploadSamplePDFFileKey(t, engine)
	body := map[string]any{
		"title":       title,
		"description": "",
		"dueDate":     "",
		"priority":    "",
		"fileKey":     fileKey,
		"fileName":    "sample.pdf",
		"fileType":    "application/pdf",
		"fileSize":    0,
		"initiatorId": initiator,
		"signers":     signers,
	}
	createRes := performJSON(engine, http.MethodPost, "/api/v1/workflows", body)
	if createRes.Code != http.StatusOK {
		t.Fatalf("create workflow status=%d body=%s", createRes.Code, createRes.Body.String())
	}
	var wrapper apiResponse
	if err := json.Unmarshal(createRes.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal create wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("create workflow code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}
	var created createWorkflowResp
	if err := json.Unmarshal(wrapper.Data, &created); err != nil {
		t.Fatalf("unmarshal create data failed: %v", err)
	}
	return created
}

// putMinimalFieldAndActivate 保存一条签名字段并激活流程，使 workflow 进入 pending 且产生首条 task（与旧 admin 行为对齐，供签署/查询测试使用）。
func putMinimalFieldAndActivate(t *testing.T, engine http.Handler, workflowID uint, fieldSignerID string) {
	t.Helper()
	fieldsBody := map[string]any{
		"fields": []map[string]any{
			{
				"signerId":   fieldSignerID,
				"fieldType":  "signature",
				"pageNumber": 1,
				"x":          10.0,
				"y":          10.0,
				"width":      80.0,
				"height":     30.0,
				"required":   true,
			},
		},
	}
	putRes := performJSON(engine, http.MethodPut, "/api/v1/workflows/"+uintToString(workflowID)+"/fields", fieldsBody)
	if putRes.Code != http.StatusOK {
		t.Fatalf("save fields status=%d body=%s", putRes.Code, putRes.Body.String())
	}
	var wrap1 apiResponse
	if err := json.Unmarshal(putRes.Body.Bytes(), &wrap1); err != nil {
		t.Fatalf("save fields unmarshal: %v", err)
	}
	if wrap1.Code != http.StatusOK {
		t.Fatalf("save fields code=%d msg=%s", wrap1.Code, wrap1.Msg)
	}

	actRes := performJSON(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(workflowID)+"/activate", map[string]any{})
	if actRes.Code != http.StatusOK {
		t.Fatalf("activate status=%d body=%s", actRes.Code, actRes.Body.String())
	}
	var wrap2 apiResponse
	if err := json.Unmarshal(actRes.Body.Bytes(), &wrap2); err != nil {
		t.Fatalf("activate unmarshal: %v", err)
	}
	if wrap2.Code != http.StatusOK {
		t.Fatalf("activate code=%d msg=%s", wrap2.Code, wrap2.Msg)
	}
}

// createPendingWorkflowViaDraftAPI 通过「上传 → 创建草稿 → 字段 → 激活」得到可 Submit 的 pending 流程。
func createPendingWorkflowViaDraftAPI(t *testing.T, engine http.Handler, title, initiator string, signers []string) createWorkflowResp {
	t.Helper()
	created := createWorkflowDraftViaAPI(t, engine, title, initiator, signers)
	putMinimalFieldAndActivate(t, engine, created.WorkflowID, signers[0])
	return created
}
