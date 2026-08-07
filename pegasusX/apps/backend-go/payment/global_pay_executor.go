package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

type globalpayProviderExecutor struct {
	env           string // "local", "dev", "staging", "production"
	serviceID     string
	username      string
	password      string
	simulatorBase string // overrides base URL for local/dev simulation
	allowStub     bool   // test-only escape hatch; production can never stub
	httpClient    *http.Client
	breaker       *circuit.Breaker
}

func newGlobalPayProviderExecutor(env, serviceID, username, password string) *globalpayProviderExecutor {
	return newGlobalPayProviderExecutorWithOptions(env, serviceID, username, password, "", nil)
}

func newGlobalPayProviderExecutorWithSimulator(env, serviceID, username, password, simulatorBase string) *globalpayProviderExecutor {
	return newGlobalPayProviderExecutorWithOptions(env, serviceID, username, password, simulatorBase, nil)
}

func newGlobalPayProviderExecutorWithOptions(env, serviceID, username, password, simulatorBase string, breaker *circuit.Breaker) *globalpayProviderExecutor {
	if env == "" {
		env = "dev"
	}
	return &globalpayProviderExecutor{
		env:           strings.ToLower(env),
		serviceID:     serviceID,
		username:      username,
		password:      password,
		simulatorBase: simulatorBase,
		breaker:       breaker,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (e *globalpayProviderExecutor) doHTTP(ctx context.Context, call func(context.Context) error) error {
	if e.breaker != nil {
		return e.breaker.Do(ctx, call)
	}
	return call(ctx)
}

// errGlobalPayCredentialsMissing is returned when a gateway call is attempted
// without merchant credentials and stub mode is not explicitly enabled.
var errGlobalPayCredentialsMissing = fmt.Errorf("globalpay credentials missing (GLOBAL_PAY_SERVICE_ID/GLOBAL_PAY_USERNAME/GLOBAL_PAY_PASSWORD) and GLOBAL_PAY_STUB_MODE is not enabled")

// stubMode reports whether fabricated-success stub responses are permitted.
// Stubs exist only for non-production load testing and must be opted into
// explicitly; production never stubs — missing credentials are hard errors.
func (e *globalpayProviderExecutor) stubMode() bool {
	if e.env == "production" {
		return false
	}
	if e.allowStub {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GLOBAL_PAY_STUB_MODE"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (e *globalpayProviderExecutor) getCheckoutBaseURL() string {
	if e.simulatorBase != "" {
		return strings.TrimRight(e.simulatorBase, "/") + "/sim/globalpay"
	}
	switch e.env {
	case "production":
		return "https://checkout-api.globalpay.uz/checkout"
	case "staging":
		return "https://checkout-api-staging.globalpay.uz/checkout"
	default:
		// "dev" or "local": use the in-process simulator
		return "http://localhost:8080/sim/globalpay"
	}
}

func (e *globalpayProviderExecutor) getBackofficeBaseURL() string {
	if e.simulatorBase != "" {
		return strings.TrimRight(e.simulatorBase, "/") + "/sim/globalpay"
	}
	switch e.env {
	case "production":
		return "https://backoffice-api.globalpay.uz"
	case "staging":
		return "https://backoffice-api-staging.globalpay.uz"
	default:
		return "http://localhost:8080/sim/globalpay"
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

// executeRefund issues a full or partial refund via backoffice perform.
// Global Pay marketing confirms partial refunds; perform action code is
// env-overridable (GLOBAL_PAY_REFUND_ACTION, default "RF") until merchant docs confirm.
func (e *globalpayProviderExecutor) executeRefund(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if e.username == "" || e.password == "" {
		if !e.stubMode() {
			return ExecutionResult{}, errGlobalPayCredentialsMissing
		}
		return ExecutionResult{
			ResolvedGateway: "GLOBAL_PAY",
			Mode:            ExecutionModeDirect,
			PolicySource:    "GLOBAL_PAY_STUB",
			ProviderRef:     "gp_refund_stub_" + req.OrderID,
		}, nil
	}
	token, err := e.authenticate(ctx)
	if err != nil {
		return ExecutionResult{}, err
	}
	paymentID := strings.TrimSpace(req.SessionID)
	if paymentID == "" {
		paymentID = strings.TrimSpace(req.OrderID)
	}
	action := strings.TrimSpace(os.Getenv("GLOBAL_PAY_REFUND_ACTION"))
	if action == "" {
		action = "RF" // reverse funds — confirm with Global Pay merchant support
	}
	url := fmt.Sprintf("%s/payments/v2/payment/%s/perform", e.getBackofficeBaseURL(), paymentID)
	body, _ := json.Marshal(map[string]any{
		"action":       action,
		"amount_minor": req.AmountMinor,
		"currency":     req.Currency,
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ExecutionResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	var performResp struct {
		PaymentID    string `json:"paymentId"`
		Status       string `json:"status"`
		IsSuccessful bool   `json:"isSuccessful"`
	}
	err = e.doHTTP(ctx, func(callCtx context.Context) error {
		resp, err := e.httpClient.Do(httpReq.WithContext(callCtx))
		if err != nil {
			return MarkRetryable(fmt.Errorf("globalpay refund request failed: %w", err))
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("globalpay refund failed with status %d: %s", resp.StatusCode, string(respBody))
		}
		_ = json.Unmarshal(respBody, &performResp)
		return nil
	})
	if err != nil {
		return ExecutionResult{}, err
	}
	ref := performResp.PaymentID
	if ref == "" {
		ref = paymentID
	}
	return ExecutionResult{
		ResolvedGateway: "GLOBAL_PAY",
		Mode:            ExecutionModeDirect,
		PolicySource:    "SUPPLIER_DEFAULT",
		ProviderRef:     ref,
	}, nil
}

func (e *globalpayProviderExecutor) authenticate(ctx context.Context) (string, error) {
	username := e.username
	password := e.password

	if username == "" || password == "" {
		return "", fmt.Errorf("globalpay credentials missing (username or password)")
	}
	url := fmt.Sprintf("%s/v1/merchant/auth", e.getCheckoutBaseURL())
	reqBody, _ := json.Marshal(gpAuthRequest{
		Username: username,
		Password: password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	var authResp gpAuthResponse
	err = e.doHTTP(ctx, func(callCtx context.Context) error {
		resp, err := e.httpClient.Do(req.WithContext(callCtx))
		if err != nil {
			return MarkRetryable(fmt.Errorf("globalpay auth request failed: %w", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("globalpay auth failed with status %d: %s", resp.StatusCode, string(b))
		}

		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return fmt.Errorf("globalpay auth decode failed: %w", err)
		}
		if authResp.AccessToken == "" {
			return fmt.Errorf("globalpay auth returned empty access token")
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return authResp.AccessToken, nil
}

func (e *globalpayProviderExecutor) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	switch req.Action {
	case ExecutionActionChargebackRecord, ExecutionActionChargebackReversal:
		// Internal marketplace debit/credit ledger only (supplier settlement).
		return ExecutionResult{
			ResolvedGateway: "GLOBAL_PAY",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
		}, nil
	case ExecutionActionCheckoutInit, ExecutionActionCheckoutCapture, ExecutionActionStatusCheck, ExecutionActionRefund:
		// handled below
	default:
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          fmt.Sprintf("unsupported execution action %s for gateway GLOBAL_PAY", req.Action),
			RequestedGateway: req.Gateway,
			ResolvedGateway:  "GLOBAL_PAY",
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}

	if req.Action == ExecutionActionRefund {
		return e.executeRefund(ctx, req)
	}

	if req.Action == ExecutionActionCheckoutCapture {
		if e.username == "" || e.password == "" {
			if !e.stubMode() {
				return ExecutionResult{}, errGlobalPayCredentialsMissing
			}
			return ExecutionResult{
				ResolvedGateway: "GLOBAL_PAY",
				Mode:            ExecutionModeDirect,
				PolicySource:    "GLOBAL_PAY_STUB",
				ProviderRef:     "gp_capture_stub_" + req.OrderID,
			}, nil
		}
		token, err := e.authenticate(ctx)
		if err != nil {
			return ExecutionResult{}, err
		}
		paymentID := strings.TrimSpace(req.SessionID)
		if paymentID == "" {
			paymentID = strings.TrimSpace(req.OrderID)
		}
		url := fmt.Sprintf("%s/payments/v2/payment/%s/perform", e.getBackofficeBaseURL(), paymentID)
		body, _ := json.Marshal(map[string]any{
			"action":       "CP",
			"amount_minor": req.AmountMinor,
			"currency":     req.Currency,
		})
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return ExecutionResult{}, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)
		var performResp struct {
			PaymentID    string `json:"paymentId"`
			Status       string `json:"status"`
			IsSuccessful bool   `json:"isSuccessful"`
		}
		err = e.doHTTP(ctx, func(callCtx context.Context) error {
			resp, err := e.httpClient.Do(httpReq.WithContext(callCtx))
			if err != nil {
				return MarkRetryable(fmt.Errorf("globalpay capture request failed: %w", err))
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("globalpay capture failed with status %d: %s", resp.StatusCode, string(respBody))
			}
			_ = json.Unmarshal(respBody, &performResp)
			return nil
		})
		if err != nil {
			return ExecutionResult{}, err
		}
		ref := performResp.PaymentID
		if ref == "" {
			ref = paymentID
		}
		return ExecutionResult{
			ResolvedGateway: "GLOBAL_PAY",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
			ProviderRef:     ref,
		}, nil
	}

	if req.Action == ExecutionActionStatusCheck {
		if e.username == "" || e.password == "" {
			if !e.stubMode() {
				return ExecutionResult{}, errGlobalPayCredentialsMissing
			}
			return ExecutionResult{
				ResolvedGateway: "GLOBAL_PAY",
				Mode:            ExecutionModeDirect,
				PolicySource:    "GLOBAL_PAY_STUB",
				ProviderRef:     "gp_status_stub_paid", // Stub paid status (load-test only)
			}, nil
		}
		token, err := e.authenticate(ctx)
		if err != nil {
			return ExecutionResult{}, err
		}
		paymentID := strings.TrimSpace(req.SessionID)
		if paymentID == "" {
			paymentID = strings.TrimSpace(req.OrderID)
		}
		url := fmt.Sprintf("%s/payments/v2/payment/%s", e.getBackofficeBaseURL(), paymentID)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return ExecutionResult{}, err
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)
		var statusResp struct {
			Status string `json:"status"`
		}
		err = e.doHTTP(ctx, func(callCtx context.Context) error {
			resp, err := e.httpClient.Do(httpReq.WithContext(callCtx))
			if err != nil {
				return MarkRetryable(fmt.Errorf("globalpay status request failed: %w", err))
			}
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("globalpay status failed with status %d: %s", resp.StatusCode, string(respBody))
			}
			_ = json.Unmarshal(respBody, &statusResp)
			return nil
		})
		if err != nil {
			return ExecutionResult{}, err
		}
		return ExecutionResult{
			ResolvedGateway: "GLOBAL_PAY",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
			ProviderRef:     statusResp.Status, // Using ProviderRef to carry status
		}, nil
	}

	// Stub mode: with no merchant credentials, return a mock redirect URL to
	// allow non-production load testing without a real gateway contract. Must be
	// explicitly enabled via GLOBAL_PAY_STUB_MODE; never available in production.
	if e.username == "" || e.password == "" {
		if !e.stubMode() {
			return ExecutionResult{}, errGlobalPayCredentialsMissing
		}
		return ExecutionResult{
			ResolvedGateway: "GLOBAL_PAY",
			Mode:            ExecutionModeHostedRedirect,
			PolicySource:    "GLOBAL_PAY_STUB",
			RedirectURL:     fmt.Sprintf("https://test.globalpay.uz/checkout-stub/%s", req.OrderID),
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

	var tResp gpTokenResponse
	err = e.doHTTP(ctx, func(callCtx context.Context) error {
		resp, err := e.httpClient.Do(httpReq.WithContext(callCtx))
		if err != nil {
			return MarkRetryable(fmt.Errorf("globalpay create token request failed: %w", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("globalpay create token failed with status %d: %s", resp.StatusCode, string(b))
		}

		if err := json.NewDecoder(resp.Body).Decode(&tResp); err != nil {
			return fmt.Errorf("globalpay create token decode failed: %w", err)
		}
		if tResp.UserRedirectUrl == "" {
			return fmt.Errorf("globalpay did not return userRedirectUrl")
		}
		return nil
	})
	if err != nil {
		return ExecutionResult{}, err
	}

	return ExecutionResult{
		ResolvedGateway: "GLOBAL_PAY",
		Mode:            ExecutionModeHostedRedirect,
		PolicySource:    "SUPPLIER_DEFAULT",
		RedirectURL:     tResp.UserRedirectUrl,
	}, nil
}
