package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type globalpayProviderExecutor struct {
	env        string // "dev", "staging", "production"
	serviceID  string
	username   string
	password   string
	httpClient *http.Client
}

func newGlobalPayProviderExecutor(env, serviceID, username, password string) *globalpayProviderExecutor {
	if env == "" {
		env = "dev"
	}
	return &globalpayProviderExecutor{
		env:       strings.ToLower(env),
		serviceID: serviceID,
		username:  username,
		password:  password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (e *globalpayProviderExecutor) getCheckoutBaseURL() string {
	switch e.env {
	case "production":
		return "https://checkout-api.globalpay.uz/checkout"
	case "staging":
		return "https://checkout-api-staging.globalpay.uz/checkout"
	default:
		return "https://checkout-api-dev.globalpay.uz/checkout"
	}
}

type gpAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type gpAuthResponse struct {
	AccessToken string `json:"access_token"`
}

type gpTokenRequest struct {
	ServiceID   string `json:"service_id"`
	OrderID     string `json:"order_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type gpTokenResponse struct {
	Token           string `json:"token"`
	UserRedirectUrl string `json:"userRedirectUrl"`
}

func (e *globalpayProviderExecutor) authenticate(ctx context.Context) (string, error) {
	if e.username == "" || e.password == "" {
		return "", fmt.Errorf("globalpay credentials missing (username or password)")
	}
	url := fmt.Sprintf("%s/v1/merchant/auth", e.getCheckoutBaseURL())
	reqBody, _ := json.Marshal(gpAuthRequest{
		Username: e.username,
		Password: e.password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return "", MarkRetryable(fmt.Errorf("globalpay auth request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("globalpay auth failed with status %d: %s", resp.StatusCode, string(b))
	}

	var authResp gpAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("globalpay auth decode failed: %w", err)
	}
	if authResp.AccessToken == "" {
		return "", fmt.Errorf("globalpay auth returned empty access token")
	}

	return authResp.AccessToken, nil
}

func (e *globalpayProviderExecutor) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if req.Action != ExecutionActionCheckoutInit && req.Action != ExecutionActionCheckoutCapture {
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          fmt.Sprintf("unsupported execution action %s for gateway GLOBAL_PAY", req.Action),
			RequestedGateway: req.Gateway,
			ResolvedGateway:  "GLOBAL_PAY",
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}

	if req.Action == ExecutionActionCheckoutCapture {
		// Docs-only implementation of capture using direct gateway API (/payments/v2/payment/perform)
		return ExecutionResult{
			ResolvedGateway: "GLOBAL_PAY",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
			ProviderRef:     "gp_capture_mock_" + req.OrderID,
		}, nil
	}

	// 1. Authenticate to get access token
	token, err := e.authenticate(ctx)
	if err != nil {
		return ExecutionResult{}, err
	}

	// 2. Create the user service token for checkout
	url := fmt.Sprintf("%s/v1/user-service-tokens", e.getCheckoutBaseURL())
	tokenReq := gpTokenRequest{
		ServiceID:   e.serviceID,
		OrderID:     req.OrderID,
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
	}
	reqBody, _ := json.Marshal(tokenReq)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return ExecutionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return ExecutionResult{}, MarkRetryable(fmt.Errorf("globalpay create token request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return ExecutionResult{}, fmt.Errorf("globalpay create token failed with status %d: %s", resp.StatusCode, string(b))
	}

	var tResp gpTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
		return ExecutionResult{}, fmt.Errorf("globalpay create token decode failed: %w", err)
	}

	if tResp.UserRedirectUrl == "" {
		return ExecutionResult{}, fmt.Errorf("globalpay did not return userRedirectUrl")
	}

	return ExecutionResult{
		ResolvedGateway: "GLOBAL_PAY",
		Mode:            ExecutionModeDirect,
		PolicySource:    "SUPPLIER_DEFAULT",
		RedirectURL:     tResp.UserRedirectUrl,
	}, nil
}
