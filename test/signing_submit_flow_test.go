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
	NextSignerID    uint   `json:"nextSignerId"`
	WorkflowStatus  string `json:"workflowStatus"`
	DocumentStatus  string `json:"documentStatus"`
	DocumentVersion int    `json:"documentVersion"`
}

type authResp struct {
	User struct {
		ID uint `json:"id"`
	} `json:"user"`
	AccessToken string `json:"accessToken"`
}

type testUserSeed struct {
	ID    uint
	Token string
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

	engine := gin.New()
	router.RegisterRoutes(engine)
	users := seedWorkflowTestUsers(t, engine)

	// 1) 上传 PDF → 创建草稿 → 字段 → 激活，得到 pending + 首条 task
	created := createPendingWorkflowViaDraftAPI(
		t,
		engine,
		"thesis-flow-doc",
		users["A"].Token,
		users["A"].ID,
		[]uint{users["A"].ID, users["B"].ID, users["C"].ID},
	)

	// 2) A 提交，nextSigner 应为 B
	aSubmit := performJSONWithAuth(
		engine,
		http.MethodPost,
		"/api/v1/workflows/"+uintToString(created.WorkflowID)+"/submit",
		map[string]any{},
		users["A"].Token,
	)
	assertSubmitOK(t, aSubmit, "A submit")
	aData := mustParseSubmitData(t, aSubmit)
	if aData.NextSignerID != users["B"].ID || aData.NextStep != 2 {
		t.Fatalf("A submit expect nextSigner=%d nextStep=2, got nextSigner=%d nextStep=%d", users["B"].ID, aData.NextSignerID, aData.NextStep)
	}

	// 3) B 提交，nextSigner 应为 C
	bSubmit := performJSONWithAuth(
		engine,
		http.MethodPost,
		"/api/v1/workflows/"+uintToString(created.WorkflowID)+"/submit",
		map[string]any{},
		users["B"].Token,
	)
	assertSubmitOK(t, bSubmit, "B submit")
	bData := mustParseSubmitData(t, bSubmit)
	if bData.NextSignerID != users["C"].ID || bData.NextStep != 3 {
		t.Fatalf("B submit expect nextSigner=%d nextStep=3, got nextSigner=%d nextStep=%d", users["C"].ID, bData.NextSignerID, bData.NextStep)
	}

	// 4) C 提交，workflow/document 应 completed
	cSubmit := performJSONWithAuth(
		engine,
		http.MethodPost,
		"/api/v1/workflows/"+uintToString(created.WorkflowID)+"/submit",
		map[string]any{},
		users["C"].Token,
	)
	assertSubmitOK(t, cSubmit, "C submit")
	cData := mustParseSubmitData(t, cSubmit)
	if cData.WorkflowStatus != string(model.WorkflowStatusCompleted) {
		t.Fatalf("C submit expect workflow completed, got %s", cData.WorkflowStatus)
	}
	if cData.DocumentStatus != string(model.DocumentStatusCompleted) {
		t.Fatalf("C submit expect document completed, got %s", cData.DocumentStatus)
	}
}

// seedWorkflowTestUsers 创建 A/B/C 三个测试用户，并返回 ID 与 token。
func seedWorkflowTestUsers(t *testing.T, engine *gin.Engine) map[string]testUserSeed {
	t.Helper()
	out := make(map[string]testUserSeed, 3)
	emailSuffix := uniqueTestSuffix()
	for _, code := range []string{"A", "B", "C"} {
		rec := performJSON(engine, http.MethodPost, "/api/v1/auth/register", map[string]any{
			"name":     "User " + code,
			"email":    strings.ToLower(code) + "-" + emailSuffix + "@test.local",
			"password": "password123",
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
		var authData authResp
		if err := json.Unmarshal(wrapper.Data, &authData); err != nil {
			t.Fatalf("register user %s unmarshal data: %v", code, err)
		}
		if authData.User.ID == 0 || authData.AccessToken == "" {
			t.Fatalf("login user %s get invalid auth data", code)
		}
		out[code] = testUserSeed{ID: authData.User.ID, Token: authData.AccessToken}
	}
	return out
}

func performJSON(r http.Handler, method, path string, body map[string]any) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func performJSONWithAuth(r http.Handler, method, path string, body map[string]any, token string) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
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
