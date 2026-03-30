package test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sign_flow_project/internal/router"

	"github.com/gin-gonic/gin"
)

type uploadFileResp struct {
	FileKey  string `json:"fileKey"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileType string `json:"fileType"`
}

func TestUploadFilePDF_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router.RegisterRoutes(engine)

	pdfPath := filepath.Join("testdata", "sample.pdf")
	rec := performMultipartFile(t, engine, http.MethodPost, "/api/v1/files/upload", "file", pdfPath, "application/pdf")

	if rec.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", rec.Code, rec.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("upload code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data uploadFileResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal data failed: %v", err)
	}

	if data.FileKey == "" {
		t.Fatalf("fileKey is empty")
	}
	if !strings.HasSuffix(strings.ToLower(data.FileKey), ".pdf") {
		t.Fatalf("expect fileKey endswith .pdf, got %s", data.FileKey)
	}
	if data.FileName != "sample.pdf" {
		t.Fatalf("expect fileName=sample.pdf, got %s", data.FileName)
	}
	if data.FileSize <= 0 {
		t.Fatalf("expect fileSize>0, got %d", data.FileSize)
	}
	if data.FileType != "application/pdf" {
		t.Fatalf("expect fileType=application/pdf, got %s", data.FileType)
	}

	absStored := filepath.Join("storage", filepath.FromSlash(data.FileKey))
	if _, err := os.Stat(absStored); err != nil {
		t.Fatalf("stored file not found at %s, err=%v", absStored, err)
	}
}

func TestUploadFilePDF_InternshipTemplate_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router.RegisterRoutes(engine)

	pdfPath := filepath.Join("testdata", "internship_contract_template.pdf")
	rec := performMultipartFile(t, engine, http.MethodPost, "/api/v1/files/upload", "file", pdfPath, "application/pdf")

	if rec.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", rec.Code, rec.Body.String())
	}

	var wrapper apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("unmarshal wrapper failed: %v", err)
	}
	if wrapper.Code != http.StatusOK {
		t.Fatalf("upload code=%d msg=%s", wrapper.Code, wrapper.Msg)
	}

	var data uploadFileResp
	if err := json.Unmarshal(wrapper.Data, &data); err != nil {
		t.Fatalf("unmarshal data failed: %v", err)
	}

	if data.FileName != "internship_contract_template.pdf" {
		t.Fatalf("expect fileName=internship_contract_template.pdf, got %s", data.FileName)
	}
	if data.FileType != "application/pdf" {
		t.Fatalf("expect fileType=application/pdf, got %s", data.FileType)
	}
	if data.FileSize <= 0 {
		t.Fatalf("expect fileSize>0, got %d", data.FileSize)
	}
	if data.FileKey == "" {
		t.Fatalf("fileKey is empty")
	}

	absStored := filepath.Join("storage", filepath.FromSlash(data.FileKey))
	if _, err := os.Stat(absStored); err != nil {
		t.Fatalf("stored file not found at %s, err=%v", absStored, err)
	}
}

func TestUploadFile_MissingFile_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router.RegisterRoutes(engine)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expect status=400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUploadFile_NotPDF_ByExt_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router.RegisterRoutes(engine)

	txtPath := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	rec := performMultipartFile(t, engine, http.MethodPost, "/api/v1/files/upload", "file", txtPath, "application/pdf")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expect status=400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUploadFile_NotPDF_ByContentType_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	router.RegisterRoutes(engine)

	pdfPath := filepath.Join("testdata", "sample.pdf")
	rec := performMultipartFile(t, engine, http.MethodPost, "/api/v1/files/upload", "file", pdfPath, "text/plain")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expect status=400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func performMultipartFile(t *testing.T, r http.Handler, method, urlPath, fieldName, filePath, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+filepath.Base(filePath)+`"`)
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		t.Fatalf("create multipart part failed: %v", err)
	}
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open file failed: %v", err)
	}
	defer f.Close()
	if _, err := io.Copy(part, f); err != nil {
		t.Fatalf("copy file to multipart failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req := httptest.NewRequest(method, urlPath, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func removeEmptyParents(dir string, stopAt string) error {
	stopAt = filepath.Clean(stopAt)
	for {
		d := filepath.Clean(dir)
		if d == "." || d == string(filepath.Separator) {
			return nil
		}
		if d == stopAt {
			return nil
		}
		err := os.Remove(d)
		if err != nil {
			return nil
		}
		dir = filepath.Dir(d)
	}
}

