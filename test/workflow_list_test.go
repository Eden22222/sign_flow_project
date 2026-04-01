package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/router"

	"github.com/gin-gonic/gin"
)

type workflowListItemResp struct {
	WorkflowID     uint   `json:"workflowId"`
	Title          string `json:"title"`
	FileName       string `json:"fileName"`
	DocumentStatus string `json:"documentStatus"`
	Initiator      string `json:"initiator"`
	SignerCount    int    `json:"signerCount"`
	CurrentStep    int    `json:"currentStep"`
	TotalSteps     int    `json:"totalSteps"`
	WorkflowStatus string `json:"workflowStatus"`
	CreatedAt      string `json:"createdAt"`
}

type workflowListResp struct {
	List     []workflowListItemResp `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

func TestListWorkflows_OK(t *testing.T) {
	engine := setupListTestEngine(t)

	createWorkflowForListTest(t, engine, "list-doc-1", []string{"A", "B"})
	createWorkflowForListTest(t, engine, "list-doc-2", []string{"A", "B", "C"})

	getRes := performJSON(engine, http.MethodGet, "/api/v1/workflows?page=1&pageSize=10", nil)
	if getRes.Code != http.StatusOK {
		t.Fatalf("list workflows status=%d body=%s", getRes.Code, getRes.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal list wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("list workflows code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data workflowListResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal list data failed: %v", err)
	}

	if data.Total != 2 {
		t.Fatalf("expect total=2, got %d", data.Total)
	}
	if data.Page != 1 {
		t.Fatalf("expect page=1, got %d", data.Page)
	}
	if data.PageSize != 10 {
		t.Fatalf("expect pageSize=10, got %d", data.PageSize)
	}
	if len(data.List) != 2 {
		t.Fatalf("expect list length=2, got %d", len(data.List))
	}

	itemsByTitle := make(map[string]workflowListItemResp, len(data.List))
	for _, item := range data.List {
		itemsByTitle[item.Title] = item
	}

	item1, ok := itemsByTitle["list-doc-1"]
	if !ok {
		t.Fatalf("expect list contains title=list-doc-1")
	}
	if item1.SignerCount != 2 || item1.TotalSteps != 2 {
		t.Fatalf("expect list-doc-1 signerCount/totalSteps=2, got signerCount=%d totalSteps=%d", item1.SignerCount, item1.TotalSteps)
	}
	if item1.WorkflowStatus != string(model.WorkflowStatusDraft) {
		t.Fatalf("expect list-doc-1 workflowStatus=%s, got %s", model.WorkflowStatusDraft, item1.WorkflowStatus)
	}
	if item1.DocumentStatus != string(model.DocumentStatusDraft) {
		t.Fatalf("expect list-doc-1 documentStatus=%s, got %s", model.DocumentStatusDraft, item1.DocumentStatus)
	}
	if item1.Initiator != "User A" {
		t.Fatalf("expect list-doc-1 initiator=User A, got %q", item1.Initiator)
	}

	item2, ok := itemsByTitle["list-doc-2"]
	if !ok {
		t.Fatalf("expect list contains title=list-doc-2")
	}
	if item2.SignerCount != 3 || item2.TotalSteps != 3 {
		t.Fatalf("expect list-doc-2 signerCount/totalSteps=3, got signerCount=%d totalSteps=%d", item2.SignerCount, item2.TotalSteps)
	}
	if item2.Initiator != "User A" {
		t.Fatalf("expect list-doc-2 initiator=User A, got %q", item2.Initiator)
	}
	if item2.WorkflowStatus != string(model.WorkflowStatusDraft) {
		t.Fatalf("expect list-doc-2 workflowStatus=%s, got %s", model.WorkflowStatusDraft, item2.WorkflowStatus)
	}
	if item2.DocumentStatus != string(model.DocumentStatusDraft) {
		t.Fatalf("expect list-doc-2 documentStatus=%s, got %s", model.DocumentStatusDraft, item2.DocumentStatus)
	}
}

func TestListWorkflows_PageNormalize_OK(t *testing.T) {
	engine := setupListTestEngine(t)
	createWorkflowForListTest(t, engine, "normalize-doc", []string{"A"})

	getRes := performJSON(engine, http.MethodGet, "/api/v1/workflows?page=0&pageSize=0", nil)
	if getRes.Code != http.StatusOK {
		t.Fatalf("list workflows normalize status=%d body=%s", getRes.Code, getRes.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal normalize wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("list workflows normalize code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data workflowListResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal normalize data failed: %v", err)
	}
	if data.Page != 1 {
		t.Fatalf("expect normalized page=1, got %d", data.Page)
	}
	if data.PageSize != 10 {
		t.Fatalf("expect normalized pageSize=10, got %d", data.PageSize)
	}
	if data.Total != 1 {
		t.Fatalf("expect total=1, got %d", data.Total)
	}
	if len(data.List) != 1 {
		t.Fatalf("expect list length=1, got %d", len(data.List))
	}
}

func setupListTestEngine(t *testing.T) *gin.Engine {
	t.Helper()

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
	return engine
}

func createWorkflowForListTest(t *testing.T, engine *gin.Engine, title string, signers []string) {
	t.Helper()
	createWorkflowDraftViaAPI(t, engine, title, signers[0], signers)
}

