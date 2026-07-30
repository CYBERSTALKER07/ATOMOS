package soliq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SoliqClient interface {
	// Submit sends the signed EHF.
	// idempotencyKey = AttemptId (must be unique per attempt, stable across retries of the same attempt).
	Submit(ctx context.Context, signedPayload []byte, idempotencyKey string) (*SoliqResponse, error)
	// CheckStatus retrieves the status of an EHF document by its Soliq-assigned ID.
	CheckStatus(ctx context.Context, ehfId string) (*DocumentStatus, error)
}

type DocumentStatus struct {
	Status string
	Raw    []byte
}

type SoliqResponse struct {
	Success      bool
	EhfID        string // Soliq-assigned identifier on success
	StatusCode   int
	RawBody      []byte // always stored for audit
	ErrorCode    string // machine-readable if provided
	ErrorMessage string
	Permanent    bool // true = do not retry (validation / business reject)
}

type SoliqConfig struct {
	BaseURL    string // https://api.soliq.uz or operator endpoint
	Operator   string // "direct" | "didox" | "faktura" | ...
	APIKey     string
	Timeout    time.Duration
	TIN        string // taxpayer identification number
}

type client struct {
	cfg        SoliqConfig
	httpClient *http.Client
}

func NewClient(cfg SoliqConfig) SoliqClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

type soliqAPIResponse struct {
	Success bool   `json:"success"`
	Data    struct {
		EhfID string `json:"ehf_id"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type soliqStatusResponse struct {
	Success bool   `json:"success"`
	Data    struct {
		Status string `json:"status"`
	} `json:"data"`
}

func (c *client) Submit(ctx context.Context, signedPayload []byte, idempotencyKey string) (*SoliqResponse, error) {
	url := fmt.Sprintf("%s/v1/ehf/submit", c.cfg.BaseURL)
	if c.cfg.Operator == "didox" {
		url = fmt.Sprintf("%s/api/v1/documents", c.cfg.BaseURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(signedPayload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return &SoliqResponse{
			Success:    false,
			StatusCode: res.StatusCode,
			RawBody:    nil,
			Permanent:  false,
		}, fmt.Errorf("read body: %w", err)
	}

	out := &SoliqResponse{
		StatusCode: res.StatusCode,
		RawBody:    body,
	}

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		out.Success = true
		
		var parsed soliqAPIResponse
		if err := json.Unmarshal(body, &parsed); err == nil {
			out.EhfID = parsed.Data.EhfID
		}
		
		return out, nil
	}

	out.Success = false
	
	// Determine if permanent based on status code
	if res.StatusCode == http.StatusBadRequest || res.StatusCode == http.StatusUnprocessableEntity {
		out.Permanent = true
	} else if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		out.Permanent = true
	} else {
		out.Permanent = false
	}

	var parsed soliqAPIResponse
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Code != "" {
		out.ErrorCode = parsed.Error.Code
		out.ErrorMessage = parsed.Error.Message
	} else {
		out.ErrorMessage = fmt.Sprintf("HTTP %d", res.StatusCode)
	}

	return out, nil
}

func (c *client) CheckStatus(ctx context.Context, ehfId string) (*DocumentStatus, error) {
	url := fmt.Sprintf("%s/v1/ehf/%s/status", c.cfg.BaseURL, ehfId)
	if c.cfg.Operator == "didox" {
		url = fmt.Sprintf("%s/api/v1/documents/%s/status", c.cfg.BaseURL, ehfId)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("http error %d: %s", res.StatusCode, string(body))
	}

	var parsed soliqStatusResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	return &DocumentStatus{
		Status: parsed.Data.Status,
		Raw:    body,
	}, nil
}
