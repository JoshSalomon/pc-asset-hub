package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
)

const (
	protocolVersion = "v1"
	maxResponseSize = 10 * 1024 * 1024 // 10 MB
	minTimeout      = 3
)

type WebhookExporter struct {
	name            string
	description     string
	baseURL         string
	paramSchema     []ParameterDef
	validateTimeout time.Duration
	exportTimeout   time.Duration
	healthMu        sync.RWMutex
	healthStatus    string
	httpClient      *http.Client
	tokenGetter     func() string
}

func NewWebhookExporter(name, description, baseURL string, paramSchema []ParameterDef, timeoutSeconds int, tokenGetter func() string) (*WebhookExporter, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if u.Path != "" && u.Path != "/" {
		return nil, fmt.Errorf("base URL must not contain a path (got %q)", u.Path)
	}

	if timeoutSeconds < minTimeout {
		timeoutSeconds = minTimeout
	}
	validateTimeout := timeoutSeconds / 2
	if validateTimeout < minTimeout {
		validateTimeout = minTimeout
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("unexpected redirect to %s", req.URL)
		},
	}

	return &WebhookExporter{
		name:            name,
		description:     description,
		baseURL:         strings.TrimSuffix(baseURL, "/"),
		paramSchema:     paramSchema,
		validateTimeout: time.Duration(validateTimeout) * time.Second,
		exportTimeout:   time.Duration(timeoutSeconds) * time.Second,
		healthStatus:    "Unknown",
		httpClient:      client,
		tokenGetter:     tokenGetter,
	}, nil
}

func (w *WebhookExporter) Name() string               { return w.name }
func (w *WebhookExporter) Description() string         { return w.description }
func (w *WebhookExporter) ParameterSchema() []ParameterDef { return w.paramSchema }

func (w *WebhookExporter) HealthStatus() string {
	w.healthMu.RLock()
	defer w.healthMu.RUnlock()
	return w.healthStatus
}

func (w *WebhookExporter) SetHealthStatus(s string) {
	w.healthMu.Lock()
	defer w.healthMu.Unlock()
	w.healthStatus = s
}

func (w *WebhookExporter) Timeouts() (validate, export time.Duration) {
	return w.validateTimeout, w.exportTimeout
}

func (w *WebhookExporter) ValidateSchema(ctx context.Context, params map[string]string, schema SchemaInfo) error {
	reqBody := WebhookValidateRequest{
		Parameters: params,
		Schema:     SchemaInfoToWebhook(schema),
	}

	ctx, cancel := context.WithTimeout(ctx, w.validateTimeout)
	defer cancel()

	resp, err := w.doPost(ctx, "/validate", reqBody, w.validateTimeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := w.readLimitedBody(resp)
	if err != nil {
		return err
	}

	if err := w.handleErrorResponse(body, resp.StatusCode); err != nil {
		return err
	}

	var valResp WebhookValidateResponse
	if err := json.Unmarshal(body, &valResp); err != nil {
		return domainerrors.NewValidation("plugin returned 200 but response is not valid JSON")
	}
	if !valResp.Valid {
		return domainerrors.NewValidation(valResp.Error)
	}
	return nil
}

func (w *WebhookExporter) Export(ctx context.Context, input ExportInput) (*ExportOutput, error) {
	reqBody := ExportInputToWebhookRequest(input)

	exportCtx, cancel := context.WithTimeout(ctx, w.exportTimeout)
	defer cancel()

	resp, err := w.doPost(exportCtx, "/export", reqBody, w.exportTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := w.readLimitedBody(resp)
	if err != nil {
		return nil, err
	}

	if err := w.handleErrorResponse(body, resp.StatusCode); err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("plugin returned 200 but empty response body")
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.Contains(ct, "application/json") {
		return nil, fmt.Errorf("plugin returned 200 but response is not valid JSON (Content-Type: %s)", ct)
	}

	var exportResp WebhookExportResponse
	if err := json.Unmarshal(body, &exportResp); err != nil {
		return nil, fmt.Errorf("plugin returned 200 but response is not valid JSON: %s", err)
	}

	return WebhookExportResponseToOutput(exportResp), nil
}

func (w *WebhookExporter) handleErrorResponse(body []byte, statusCode int) error {
	if statusCode < 400 {
		return nil
	}

	var errResp WebhookErrorResponse
	hasMessage := json.Unmarshal(body, &errResp) == nil && errResp.Error != ""

	if statusCode < 500 {
		if hasMessage {
			return domainerrors.NewValidation(errResp.Error)
		}
		return domainerrors.NewValidation(fmt.Sprintf("plugin returned HTTP %d", statusCode))
	}

	if hasMessage {
		return fmt.Errorf("webhook plugin error: %s", errResp.Error)
	}
	return fmt.Errorf("webhook plugin error: HTTP %d", statusCode)
}

func (w *WebhookExporter) doPost(ctx context.Context, path string, body any, timeout time.Duration) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AssetHub-Protocol-Version", protocolVersion)
	if token := w.tokenGetter(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("webhook plugin timed out after %s", timeout)
		}
		if strings.Contains(err.Error(), "redirect") {
			return nil, fmt.Errorf("webhook plugin returned unexpected redirect: %s", err)
		}
		return nil, fmt.Errorf("webhook plugin unreachable: %s", err)
	}

	return resp, nil
}

func (w *WebhookExporter) readLimitedBody(resp *http.Response) ([]byte, error) {
	limited := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if len(body) > maxResponseSize {
		return nil, domainerrors.NewValidation("response exceeds 10 MB limit")
	}
	return body, nil
}
