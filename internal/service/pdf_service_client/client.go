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

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("pdf service request marshal failed: %w", err)
	}

	url := c.baseURL + ApplyFieldsPath
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pdf service request build failed: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pdf service request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pdf service read body failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = fmt.Sprintf("http status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("pdf service returned status %d: %s", resp.StatusCode, msg)
	}

	var env envelopeResponse
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("pdf service response is not valid json: %w; body=%s", err, truncateForErr(respBody))
	}

	if env.Code != http.StatusOK {
		msg := strings.TrimSpace(env.Msg)
		if msg == "" {
			msg = "pdf service business error"
		}
		return nil, fmt.Errorf("pdf service error (code=%d, msg=%s)", env.Code, msg)
	}

	return &ApplyFieldsResponse{RawData: env.Data}, nil
}

func truncateForErr(b []byte) string {
	const max = 512
	s := string(b)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
