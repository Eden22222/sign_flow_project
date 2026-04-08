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

type workflowDetailResp struct {
	WorkflowID      uint   `json:"workflowId"`
	DocumentID      uint   `json:"documentId"`
	Title           string `json:"title"`
	CurrentStep     int    `json:"currentStep"`
	WorkflowStatus  string `json:"workflowStatus"`
	DocumentStatus  string `json:"documentStatus"`
	DocumentVersion int    `json:"documentVersion"`
	CurrentSignerID uint   `json:"currentSignerId"`
}

type workflowTaskItemResp struct {
	TaskID     uint   `json:"taskId"`
	WorkflowID uint   `json:"workflowId"`
	SignerID   uint   `json:"signerId"`
	StepIndex  int    `json:"stepIndex"`
	Status     string `json:"status"`
}

type workflowTaskListResp struct {
	WorkflowID uint                   `json:"workflowId"`
	Tasks      []workflowTaskItemResp `json:"tasks"`
}

type workflowSignerItemResp struct {
	SignerID  uint   `json:"signerId"`
	StepIndex int    `json:"stepIndex"`
}

type workflowSignerListResp struct {
	WorkflowID uint                     `json:"workflowId"`
	Signers    []workflowSignerItemResp `json:"signers"`
}

func TestGetWorkflowDetail(t *testing.T) {
	title := "query-test-doc-" + uniqueTestSuffix()
	engine, workflowID, users := setupQueryTestEngineAndWorkflow(t, title)

	// 推进一步后，当前签署人应变为 B，版本应 +1，状态应为 signing/pending
	submitRes := performJSONWithAuth(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(workflowID)+"/submit", map[string]any{}, users["A"].Token)
	assertSubmitOK(t, submitRes, "A submit for detail test")

	getRes := performJSON(engine, http.MethodGet, "/api/v1/workflows/"+uintToString(workflowID), nil)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get workflow detail status=%d body=%s", getRes.Code, getRes.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal detail wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("get workflow detail code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data workflowDetailResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal detail data failed: %v", err)
	}
	t.Logf("GET /api/v1/workflows/%d data: %+v", workflowID, data)

	if data.WorkflowID != workflowID {
		t.Fatalf("expect workflowId=%d, got %d", workflowID, data.WorkflowID)
	}
	if data.Title != title {
		t.Fatalf("expect title=%s, got %s", title, data.Title)
	}
	if data.CurrentStep != 2 {
		t.Fatalf("expect currentStep=2, got %d", data.CurrentStep)
	}
	if data.CurrentSignerID != users["B"].ID {
		t.Fatalf("expect currentSignerId=%d, got %d", users["B"].ID, data.CurrentSignerID)
	}
	if data.WorkflowStatus != string(model.WorkflowStatusPending) {
		t.Fatalf("expect workflowStatus=%s, got %s", model.WorkflowStatusPending, data.WorkflowStatus)
	}
	if data.DocumentStatus != string(model.DocumentStatusSigning) {
		t.Fatalf("expect documentStatus=%s, got %s", model.DocumentStatusSigning, data.DocumentStatus)
	}
	if data.DocumentVersion != 2 {
		t.Fatalf("expect documentVersion=2, got %d", data.DocumentVersion)
	}
}

func TestGetWorkflowTasks(t *testing.T) {
	engine, workflowID, users := setupQueryTestEngineAndWorkflow(t, "query-test-doc-"+uniqueTestSuffix())

	// 先签一步，这样应该有两条 task：A signed, B pending
	submitRes := performJSONWithAuth(engine, http.MethodPost, "/api/v1/workflows/"+uintToString(workflowID)+"/submit", map[string]any{}, users["A"].Token)
	assertSubmitOK(t, submitRes, "A submit for tasks test")

	getRes := performJSON(engine, http.MethodGet, "/api/v1/workflows/"+uintToString(workflowID)+"/tasks", nil)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get workflow tasks status=%d body=%s", getRes.Code, getRes.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal tasks wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("get workflow tasks code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data workflowTaskListResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal tasks data failed: %v", err)
	}
	t.Logf("GET /api/v1/workflows/%d/tasks data: %+v", workflowID, data)

	if data.WorkflowID != workflowID {
		t.Fatalf("expect workflowId=%d, got %d", workflowID, data.WorkflowID)
	}
	if len(data.Tasks) != 2 {
		t.Fatalf("expect tasks length=2, got %d", len(data.Tasks))
	}

	first := data.Tasks[0]
	second := data.Tasks[1]
	if first.StepIndex != 1 || first.SignerID != users["A"].ID || first.Status != string(model.TaskStatusSigned) {
		t.Fatalf("expect first task step=1 signer=%d status=signed, got step=%d signer=%d status=%s", users["A"].ID, first.StepIndex, first.SignerID, first.Status)
	}
	if second.StepIndex != 2 || second.SignerID != users["B"].ID || second.Status != string(model.TaskStatusPending) {
		t.Fatalf("expect second task step=2 signer=%d status=pending, got step=%d signer=%d status=%s", users["B"].ID, second.StepIndex, second.SignerID, second.Status)
	}
}

func TestGetWorkflowSigners(t *testing.T) {
	engine, workflowID, users := setupQueryTestEngineAndWorkflow(t, "query-test-doc-"+uniqueTestSuffix())

	getRes := performJSON(engine, http.MethodGet, "/api/v1/workflows/"+uintToString(workflowID)+"/signers", nil)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get workflow signers status=%d body=%s", getRes.Code, getRes.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(getRes.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal signers wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("get workflow signers code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data workflowSignerListResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal signers data failed: %v", err)
	}
	t.Logf("GET /api/v1/workflows/%d/signers data: %+v", workflowID, data)

	if data.WorkflowID != workflowID {
		t.Fatalf("expect workflowId=%d, got %d", workflowID, data.WorkflowID)
	}
	if len(data.Signers) != 3 {
		t.Fatalf("expect signers length=3, got %d", len(data.Signers))
	}
	if data.Signers[0].StepIndex != 1 || data.Signers[0].SignerID != users["A"].ID {
		t.Fatalf("expect signer1 step=1 signer=%d, got step=%d signer=%d", users["A"].ID, data.Signers[0].StepIndex, data.Signers[0].SignerID)
	}
	if data.Signers[1].StepIndex != 2 || data.Signers[1].SignerID != users["B"].ID {
		t.Fatalf("expect signer2 step=2 signer=%d, got step=%d signer=%d", users["B"].ID, data.Signers[1].StepIndex, data.Signers[1].SignerID)
	}
	if data.Signers[2].StepIndex != 3 || data.Signers[2].SignerID != users["C"].ID {
		t.Fatalf("expect signer3 step=3 signer=%d, got step=%d signer=%d", users["C"].ID, data.Signers[2].StepIndex, data.Signers[2].SignerID)
	}
}

func setupQueryTestEngineAndWorkflow(t *testing.T, workflowTitle string) (*gin.Engine, uint, map[string]testUserSeed) {
	t.Helper()

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

	created := createPendingWorkflowViaDraftAPI(
		t,
		engine,
		workflowTitle,
		users["A"].Token,
		users["A"].ID,
		[]uint{users["A"].ID, users["B"].ID, users["C"].ID},
	)

	return engine, created.WorkflowID, users
}
