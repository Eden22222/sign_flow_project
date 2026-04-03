package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/router"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

type createWorkflowResp struct {
	DocumentID uint `json:"documentId"`
	WorkflowID uint `json:"workflowId"`
}

type submitResp struct {
	WorkflowID      uint   `json:"workflowId"`
	DocumentID      uint   `json:"documentId"`
	SignedStep      int    `json:"signedStep"`
	NextStep        int    `json:"nextStep"`
	NextSignerID    string `json:"nextSignerId"`
	WorkflowStatus  string `json:"workflowStatus"`
	DocumentStatus  string `json:"documentStatus"`
	DocumentVersion int    `json:"documentVersion"`
}

func TestSubmitSigningThreeSignersFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	gdb, err := db.PostgresSetup()
	if err != nil {
		t.Fatalf("init postgres failed: %v", err)
	}

	if err := gdb.AutoMigrate(&model.WorkflowSignerModel{}); err != nil {
		t.Fatalf("migrate workflow signer failed: %v", err)
	}

	cleanupTables(t, gdb)

	engine := gin.New()
	router.RegisterRoutes(engine)
	seedWorkflowTestUsers(t, engine)

	// 1) 上传 PDF → 创建草稿 → 字段 → 激活，得到 pending + 首条 task
	created := createPendingWorkflowViaDraftAPI(t, engine, "thesis-flow-doc", "A", []string{"A", "B", "C"})

	// 2) A 提交，nextSigner 应为 B
	aSubmit := performJSON(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(created.WorkflowID)+"/submit", map[string]any{
		"signerId": "A",
	})
	assertSubmitOK(t, aSubmit, "A submit")
	aData := mustParseSubmitData(t, aSubmit)
	if aData.NextSignerID != "B" || aData.NextStep != 2 {
		t.Fatalf("A submit expect nextSigner=B nextStep=2, got nextSigner=%s nextStep=%d", aData.NextSignerID, aData.NextStep)
	}

	// 3) B 提交，nextSigner 应为 C
	bSubmit := performJSON(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(created.WorkflowID)+"/submit", map[string]any{
		"signerId": "B",
	})
	assertSubmitOK(t, bSubmit, "B submit")
	bData := mustParseSubmitData(t, bSubmit)
	if bData.NextSignerID != "C" || bData.NextStep != 3 {
		t.Fatalf("B submit expect nextSigner=C nextStep=3, got nextSigner=%s nextStep=%d", bData.NextSignerID, bData.NextStep)
	}

	// 4) C 提交，workflow/document 应 completed
	cSubmit := performJSON(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(created.WorkflowID)+"/submit", map[string]any{
		"signerId": "C",
	})
	assertSubmitOK(t, cSubmit, "C submit")
	cData := mustParseSubmitData(t, cSubmit)
	if cData.WorkflowStatus != string(model.WorkflowStatusCompleted) {
		t.Fatalf("C submit expect workflow completed, got %s", cData.WorkflowStatus)
	}
	if cData.DocumentStatus != string(model.DocumentStatusCompleted) {
		t.Fatalf("C submit expect document completed, got %s", cData.DocumentStatus)
	}
}

func cleanupTables(t *testing.T, gdb *gorm.DB) {
	t.Helper()
	if err := gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.TaskModel{}).Error; err != nil {
		t.Fatalf("cleanup tasks failed: %v", err)
	}
	if err := gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.WorkflowSignerModel{}).Error; err != nil {
		t.Fatalf("cleanup workflow signers failed: %v", err)
	}
	if err := gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.WorkflowModel{}).Error; err != nil {
		t.Fatalf("cleanup workflows failed: %v", err)
	}
	if err := gdb.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.DocumentModel{}).Error; err != nil {
		t.Fatalf("cleanup documents failed: %v", err)
	}
	// user_code 唯一索引：软删仍会占位，测试清理需物理删除
	if err := gdb.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.UserModel{}).Error; err != nil {
		t.Fatalf("cleanup users failed: %v", err)
	}
}

// seedWorkflowTestUsers 创建 A/B/C 三个测试用户（与历史签署人 string 占位一致）。
func seedWorkflowTestUsers(t *testing.T, engine *gin.Engine) {
	t.Helper()
	for _, code := range []string{"A", "B", "C"} {
		rec := performJSON(engine, http.MethodPost, "/api/v1/users", map[string]any{
			"userCode": code,
			"name":     "User " + code,
			"email":    strings.ToLower(code) + "@test.local",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("seed user %s status=%d body=%s", code, rec.Code, rec.Body.String())
		}
		var wrapper apiResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
			t.Fatalf("seed user %s unmarshal wrapper: %v", code, err)
		}
		if wrapper.Code != http.StatusOK {
			t.Fatalf("seed user %s code=%d msg=%s", code, wrapper.Code, wrapper.Msg)
		}
	}
}

func performJSON(r http.Handler, method, path string, body map[string]any) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func assertSubmitOK(t *testing.T, rec *httptest.ResponseRecorder, title string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", title, rec.Code, rec.Body.String())
	}
	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("%s unmarshal wrapper failed: %v", title, err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("%s code=%d msg=%s", title, wrapper.Code, wrapper.Msg)
	}
}

func mustParseSubmitData(t *testing.T, rec *httptest.ResponseRecorder) submitResp {
	t.Helper()
	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal submit wrapper failed: %v", err)
	}
	var data submitResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal submit data failed: %v", err)
	}
	return data
}

func uintToString(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}
