package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/infra/db"
	"sign_flow_project/internal/model"
	"sign_flow_project/internal/router"

	"github.com/gin-gonic/gin"
)

func TestDocumentDownload_InvalidDocumentID_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router.RegisterRoutes(engine)

	for _, path := range []string{
		"/api/v1/documents/0/download",
		"/api/v1/documents/abc/download",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s expect 400, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestDocumentVersionDownload_InvalidVersionID_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	router.RegisterRoutes(engine)

	for _, path := range []string{
		"/api/v1/document-versions/0/download",
		"/api/v1/document-versions/xyz/download",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s expect 400, got %d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestDocumentDownload_NotFound(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/999999999/download", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expect 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDocumentDownload_Latest_OK(t *testing.T) {
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

	title := "doc-download-latest-" + uniqueTestSuffix()
	created := createWorkflowDraftViaAPI(
		t,
		engine,
		title,
		users["A"].Token,
		users["A"].ID,
		[]uint{users["A"].ID, users["B"].ID},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+uintToString(created.DocumentID)+"/download", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assertDownloadPDFResponse(t, rec, "latest document download", "in.pdf")
}

func TestDocumentVersionDownload_OK(t *testing.T) {
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

	title := "doc-download-version-" + uniqueTestSuffix()
	created := createWorkflowDraftViaAPI(
		t,
		engine,
		title,
		users["A"].Token,
		users["A"].ID,
		[]uint{users["A"].ID, users["B"].ID},
	)

	rows, err := dao.DocumentVersionDao.SelectByDocumentID(created.DocumentID)
	if err != nil {
		t.Fatalf("select versions: %v", err)
	}
	var versionID uint
	for i := range rows {
		if rows[i].VersionNo == 1 {
			versionID = rows[i].ID
			break
		}
	}
	if versionID == 0 {
		t.Fatalf("no initial version row for documentId=%d", created.DocumentID)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/document-versions/"+uintToString(versionID)+"/download", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assertDownloadPDFResponse(t, rec, "history version download", "in.pdf")
}

func assertDownloadPDFResponse(t *testing.T, rec *httptest.ResponseRecorder, label string, wantFileNameSubstr string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status=%d body=%s", label, rec.Code, rec.Body.String())
	}
	ct := strings.ToLower(rec.Header().Get("Content-Type"))
	if !strings.Contains(ct, "application/pdf") {
		t.Fatalf("%s: Content-Type want pdf, got %q", label, rec.Header().Get("Content-Type"))
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(strings.ToLower(cd), "attachment") {
		t.Fatalf("%s: Content-Disposition want attachment, got %q", label, cd)
	}
	if wantFileNameSubstr != "" && !strings.Contains(cd, wantFileNameSubstr) {
		t.Fatalf("%s: Content-Disposition should mention %q, got %q", label, wantFileNameSubstr, cd)
	}
	body := rec.Body.Bytes()
	if len(body) < 4 || string(body[:4]) != "%PDF" {
		t.Fatalf("%s: body should start with %%PDF, len=%d", label, len(body))
	}
}
