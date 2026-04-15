package pdf_service_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"sign_flow_project/internal/config"

	log "github.com/sirupsen/logrus"
)

type envelopeResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

// Client 调用 sign_flow_pdf_service 的轻量 HTTP 客户端。
type Client struct {
	baseURL    string
	httpClient *http.Client
}

var (
	defaultClient     *Client
	defaultClientOnce sync.Once
)

// DefaultClient 懒加载单例；base 取自环境变量 PDF_SERVICE_BASE_URL（去掉末尾 /）。
func DefaultClient() *Client {
	defaultClientOnce.Do(func() {
		base := strings.TrimSpace(config.GetEnv("PDF_SERVICE_BASE_URL", ""))
		base = strings.TrimRight(base, "/")
		defaultClient = &Client{
			baseURL: base,
			httpClient: &http.Client{
				Timeout: 5 * time.Minute,
			},
		}
	})
	return defaultClient
}

// ResetDefaultClientForTest 仅测试使用：重置懒加载单例。
func ResetDefaultClientForTest() {
	defaultClient = nil
	defaultClientOnce = sync.Once{}
}

func (c *Client) ApplyFields(req ApplyFieldsRequest) (*ApplyFieldsResponse, error) {
	if c == nil || c.baseURL == "" {
		return nil, fmt.Errorf("pdf service is not configured (PDF_SERVICE_BASE_URL)")
	}
	start := time.Now()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("pdf service request marshal failed: %w", err)
	}

	url := c.baseURL + ApplyFieldsPath
	log.WithFields(log.Fields{
		"requestId":   req.RequestID,
		"workflowId":  req.WorkflowID,
		"documentId":  req.DocumentID,
		"signerId":    req.SignerID,
		"sourceFile":  req.SourceFile,
		"targetFile":  req.TargetFile,
		"fieldCount":  len(req.Fields),
		"writeMode":   req.WriteMode,
		"url":         url,
		"fieldBriefs": buildFieldBriefs(req.Fields),
	}).Info("pdf service apply-fields request")

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pdf service request build failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"requestId":  req.RequestID,
			"workflowId": req.WorkflowID,
			"documentId": req.DocumentID,
			"signerId":   req.SignerID,
			"sourceFile": req.SourceFile,
			"targetFile": req.TargetFile,
			"url":        url,
			"elapsedMs":  time.Since(start).Milliseconds(),
		}).Error("pdf service request failed")
		return nil, fmt.Errorf("pdf service request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"requestId":  req.RequestID,
			"workflowId": req.WorkflowID,
			"documentId": req.DocumentID,
			"signerId":   req.SignerID,
			"targetFile": req.TargetFile,
			"url":        url,
			"httpStatus": resp.StatusCode,
			"elapsedMs":  time.Since(start).Milliseconds(),
		}).Error("pdf service read body failed")
		return nil, fmt.Errorf("pdf service read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(truncateForErr(respBody))
		if msg == "" {
			msg = fmt.Sprintf("http status %d", resp.StatusCode)
		}
		err := fmt.Errorf("pdf service returned status %d: %s", resp.StatusCode, msg)
		log.WithError(err).WithFields(log.Fields{
			"requestId":   req.RequestID,
			"workflowId":  req.WorkflowID,
			"documentId":  req.DocumentID,
			"signerId":    req.SignerID,
			"sourceFile":  req.SourceFile,
			"targetFile":  req.TargetFile,
			"url":         url,
			"httpStatus":  resp.StatusCode,
			"respBodyCut": truncateForErr(respBody),
			"elapsedMs":   time.Since(start).Milliseconds(),
		}).Error("pdf service returned non-200 status")
		return nil, err
	}

	var env envelopeResponse
	if err := json.Unmarshal(respBody, &env); err != nil {
		wrapErr := fmt.Errorf("pdf service response is not valid json: %w; body=%s", err, truncateForErr(respBody))
		log.WithError(wrapErr).WithFields(log.Fields{
			"requestId":   req.RequestID,
			"workflowId":  req.WorkflowID,
			"documentId":  req.DocumentID,
			"signerId":    req.SignerID,
			"targetFile":  req.TargetFile,
			"url":         url,
			"httpStatus":  resp.StatusCode,
			"respBodyCut": truncateForErr(respBody),
			"elapsedMs":   time.Since(start).Milliseconds(),
		}).Error("pdf service response parse failed")
		return nil, wrapErr
	}

	if env.Code != http.StatusOK {
		msg := strings.TrimSpace(env.Msg)
		if msg == "" {
			msg = "pdf service business error"
		}
		err := fmt.Errorf("pdf service error (code=%d, msg=%s)", env.Code, msg)
		log.WithError(err).WithFields(log.Fields{
			"requestId":    req.RequestID,
			"workflowId":   req.WorkflowID,
			"documentId":   req.DocumentID,
			"signerId":     req.SignerID,
			"sourceFile":   req.SourceFile,
			"targetFile":   req.TargetFile,
			"url":          url,
			"httpStatus":   resp.StatusCode,
			"envelopeCode": env.Code,
			"envelopeMsg":  env.Msg,
			"respBodyCut":  truncateForErr(respBody),
			"elapsedMs":    time.Since(start).Milliseconds(),
		}).Error("pdf service business error")
		return nil, err
	}

	log.WithFields(log.Fields{
		"requestId":    req.RequestID,
		"workflowId":   req.WorkflowID,
		"documentId":   req.DocumentID,
		"signerId":     req.SignerID,
		"targetFile":   req.TargetFile,
		"url":          url,
		"httpStatus":   resp.StatusCode,
		"envelopeCode": env.Code,
		"envelopeMsg":  env.Msg,
		"elapsedMs":    time.Since(start).Milliseconds(),
	}).Info("pdf service apply-fields success")

	return &ApplyFieldsResponse{RawData: env.Data}, nil
}

func buildFieldBriefs(fields []ApplyFieldPayload) []string {
	if len(fields) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(fields))
	for i := range fields {
		out = append(out, fmt.Sprintf("%d:%s", fields[i].FieldID, strings.TrimSpace(fields[i].FieldType)))
	}
	return out
}

func truncateForErr(b []byte) string {
	const max = 512
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
