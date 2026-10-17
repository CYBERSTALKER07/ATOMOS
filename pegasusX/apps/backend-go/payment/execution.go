package payment

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

// ExecutionAction is the payment execution intent routed to a provider adapter.
type ExecutionAction string

const (
	ExecutionActionCheckoutInit       ExecutionAction = "CHECKOUT_INIT"
	ExecutionActionCheckoutCapture    ExecutionAction = "CHECKOUT_CAPTURE"
	ExecutionActionRefund             ExecutionAction = "REFUND"
	ExecutionActionChargebackRecord   ExecutionAction = "CHARGEBACK_RECORD"
	ExecutionActionChargebackReversal ExecutionAction = "CHARGEBACK_REVERSAL"
	ExecutionActionStatusCheck        ExecutionAction = "STATUS_CHECK"
)

// ExecutionMode describes the execution branch selected by a provider adapter.
type ExecutionMode string

const (
	ExecutionModeHostedRedirect ExecutionMode = "HOSTED_REDIRECT"
	ExecutionModeDirect         ExecutionMode = "DIRECT"
	ExecutionModeManual         ExecutionMode = "MANUAL"
)

// ExecutionRequest carries provider execution inputs.
type ExecutionRequest struct {
	Gateway         string
	FallbackGateway string
	Action          ExecutionAction
	OrderID         string
	SessionID       string
	AmountMinor     int64
	Currency        string
	AttemptNo       int
	CardToken       string
	PayerPhone      string
}

// ExecutionResult carries normalized provider execution outputs.
type ExecutionResult struct {
	ResolvedGateway string
	Mode            ExecutionMode
	PolicySource    string
	RedirectURL     string
	ProviderRef     string
}

// ProviderExecutor is the provider adapter execution seam.
type ProviderExecutor interface {
	Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error)
}

// ProviderExecutionRouterConfig configures retry and provider capabilities.
type ProviderExecutionRouterConfig struct {
	MaxAttempts                     int
	BaseBackoff                     time.Duration
	MaxBackoff                      time.Duration
	AirwallexDirectExecutionEnabled bool

	GlobalPayEnv       string
	GlobalPayServiceID string
	GlobalPayUsername  string
	GlobalPayPassword  string
	PaymentBreaker     *circuit.Breaker
}

// ProviderExecutionRouter routes payment actions to provider adapters with
// bounded retry and jitter for retryable failures.
type ProviderExecutionRouter struct {
	executors   map[string]ProviderExecutor
	maxAttempts int
	baseBackoff time.Duration
	maxBackoff  time.Duration
	randInt63n  func(int64) int64
	sleepFn     func(context.Context, time.Duration) error
}

// GatewayPolicyError represents policy or capability violations (422 class).
type GatewayPolicyError struct {
	Code             string
	Message          string
	RequestedGateway string
	ResolvedGateway  string
	PolicySource     string
}

func (e *GatewayPolicyError) Error() string {
	if e == nil {
		return "gateway policy violation"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.Code) != "" {
		return e.Code
	}
	return "gateway policy violation"
}

// RetryableExecutionError marks provider failures as retryable.
type RetryableExecutionError struct {
	err error
}

func (e *RetryableExecutionError) Error() string {
	if e == nil || e.err == nil {
		return "retryable execution error"
	}
	return e.err.Error()
}

func (e *RetryableExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// MarkRetryable wraps an error as retryable.
func MarkRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableExecutionError{err: err}
}

// NewProviderExecutionRouter constructs provider execution routing with defaults.
func NewProviderExecutionRouter(cfg ProviderExecutionRouterConfig) *ProviderExecutionRouter {
	cfg = normalizeExecutionRouterConfig(cfg)
	router := &ProviderExecutionRouter{
		executors: map[string]ProviderExecutor{
			"GLOBAL_PAY": newGlobalPayProviderExecutorWithOptions(
				cfg.GlobalPayEnv,
				cfg.GlobalPayServiceID,
				cfg.GlobalPayUsername,
				cfg.GlobalPayPassword,
				"",
				cfg.PaymentBreaker,
			),
			"ADYEN":  catalogHonestyExecutorFor("ADYEN"),
			"STRIPE": catalogHonestyExecutorFor("STRIPE"),
			// PAYME / CLICK adapters exist (payme_executor.go, click_executor.go,
			// Merchant/SHOP inbound handlers) but are NOT wired. UZ launch is
			// Cash + Global Pay + MySoliq + bank-file. Checkout stays honesty
			// 501 no_live_keys. Uncomment only after an explicit wire decision:
			// "PAYME": newPaymeProviderExecutor(merchantID, key, env),
			// "CLICK": newClickProviderExecutor(merchantID, serviceID, merchantUserID, secret),
			"PAYME": catalogHonestyExecutorFor("PAYME"),
			"CLICK": catalogHonestyExecutorFor("CLICK"),
			"CASH": &staticProviderExecutor{
				gateway:      "CASH",
				checkoutMode: ExecutionModeManual,
			},
			// Cash-on-delivery / in-platform sessions with no card PSP.
			"INTERNAL": &staticProviderExecutor{
				gateway:      "INTERNAL",
				checkoutMode: ExecutionModeManual,
			},
			"CREDIT": &staticProviderExecutor{
				gateway:      "CREDIT",
				checkoutMode: ExecutionModeManual,
			},
			"AIRWALLEX": &airwallexProviderExecutor{
				enabled: cfg.AirwallexDirectExecutionEnabled,
			},
		},
		maxAttempts: cfg.MaxAttempts,
		baseBackoff: cfg.BaseBackoff,
		maxBackoff:  cfg.MaxBackoff,
		randInt63n:  rand.Int63n,
		sleepFn:     sleepWithContext,
	}
	return router
}

func normalizeExecutionRouterConfig(cfg ProviderExecutionRouterConfig) ProviderExecutionRouterConfig {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = 25 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 250 * time.Millisecond
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		cfg.MaxBackoff = cfg.BaseBackoff
	}
	return cfg
}

// SetExecutor overrides or adds a gateway executor.
func (r *ProviderExecutionRouter) SetExecutor(gateway string, executor ProviderExecutor) {
	if r == nil {
		return
	}
	key := strings.ToUpper(strings.TrimSpace(gateway))
	if key == "" || executor == nil {
		return
	}
	r.executors[key] = executor
}

// Execute routes the request to the gateway adapter with bounded retries.
// If the primary gateway fails and FallbackGateway is provided, it attempts failover.
func (r *ProviderExecutionRouter) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if r == nil {
		return ExecutionResult{}, fmt.Errorf("execution router is nil")
	}

	if req.Action == "" {
		req.Action = ExecutionActionCheckoutInit
	}

	executeWithRetries := func(targetGateway string, execReq ExecutionRequest) (ExecutionResult, error) {
		targetGateway = strings.ToUpper(strings.TrimSpace(targetGateway))
		if targetGateway == "GLOBALPAY" || targetGateway == "GP" {
			targetGateway = "GLOBAL_PAY"
		}

		if targetGateway == "" {
			return ExecutionResult{}, &GatewayPolicyError{
				Code:         "gateway_required",
				Message:      "gateway_required",
				PolicySource: "MARKET_PACK",
			}
		}
		executor, ok := r.executors[targetGateway]
		if !ok {
			return ExecutionResult{}, &GatewayPolicyError{
				Code:             "payment_gateway_policy_violation",
				Message:          fmt.Sprintf("unsupported payment gateway: %s", targetGateway),
				RequestedGateway: targetGateway,
				ResolvedGateway:  targetGateway,
				PolicySource:     "ROUTER_CAPABILITY",
			}
		}

		execReq.Gateway = targetGateway

		var lastErr error
		for attempt := 1; attempt <= r.maxAttempts; attempt++ {
			execReq.AttemptNo = attempt
			result, err := executor.Execute(ctx, execReq)
			if err == nil {
				result.ResolvedGateway = strings.ToUpper(strings.TrimSpace(result.ResolvedGateway))
				if result.ResolvedGateway == "" {
					result.ResolvedGateway = targetGateway
				}
				if result.PolicySource == "" {
					result.PolicySource = "SUPPLIER_DEFAULT"
				}
				if result.Mode == "" {
					result.Mode = ExecutionModeHostedRedirect
				}
				return result, nil
			}
			lastErr = err
			if !isRetryableExecutionError(err) || attempt == r.maxAttempts {
				return ExecutionResult{}, err
			}
			if err := r.sleepFn(ctx, r.backoffForAttempt(attempt)); err != nil {
				return ExecutionResult{}, err
			}
		}
		return ExecutionResult{}, lastErr
	}

	primaryGateway := strings.ToUpper(strings.TrimSpace(req.Gateway))
	if primaryGateway == "" {
		resolved, resolveErr := resolveDefaultCardGateway(ctx)
		if resolveErr != nil {
			return ExecutionResult{}, resolveErr
		}
		primaryGateway = resolved
	}

	result, err := executeWithRetries(primaryGateway, req)
	if err != nil {
		fallback := strings.ToUpper(strings.TrimSpace(req.FallbackGateway))
		if fallback != "" && fallback != strings.ToUpper(strings.TrimSpace(primaryGateway)) {
			fallbackResult, fallbackErr := executeWithRetries(fallback, req)
			if fallbackErr == nil {
				fallbackResult.PolicySource = "ROUTER_FAILOVER"
				return fallbackResult, nil
			}
		}
	}

	return result, err
}

func (r *ProviderExecutionRouter) backoffForAttempt(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	delay := r.baseBackoff << (attempt - 1)
	if delay > r.maxBackoff {
		delay = r.maxBackoff
	}
	if delay <= 0 {
		return 0
	}
	jitterMax := int64(delay / 2)
	if jitterMax > 0 {
		delay += time.Duration(r.randInt63n(jitterMax))
	}
	return delay
}

func isRetryableExecutionError(err error) bool {
	var retryable *RetryableExecutionError
	return errors.As(err, &retryable)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func resolveDefaultCardGateway(ctx context.Context) (string, error) {
	pack, err := auth.CheckoutPackFromContext(ctx)
	if err != nil {
		return "", &GatewayPolicyError{
			Code:         "gateway_required",
			Message:      err.Error(),
			PolicySource: "MARKET_PACK",
		}
	}
	for _, gw := range LivePackGateways(pack) {
		if gw != "CASH" {
			return gw, nil
		}
	}
	return "", &GatewayPolicyError{
		Code:         "gateway_required",
		Message:      "no live card gateway in pack",
		PolicySource: "MARKET_PACK",
	}
}

func catalogHonestyExecutorFor(gateway string) ProviderExecutor {
	status := PSPStatusUnkeyed
	if adapter, ok := LookupPSP(gateway); ok {
		status = adapter.Status
	}
	return &catalogHonestyExecutor{gateway: gateway, status: status}
}

type catalogHonestyExecutor struct {
	gateway string
	status  string
}

func (e *catalogHonestyExecutor) Execute(_ context.Context, req ExecutionRequest) (ExecutionResult, error) {
	gateway := ""
	if e != nil {
		gateway = e.gateway
	}
	switch req.Action {
	case ExecutionActionChargebackRecord, ExecutionActionChargebackReversal:
		// Local ledger only — not a PSP charge and not a redirect.
		return ExecutionResult{
			ResolvedGateway: gateway,
			Mode:            ExecutionModeDirect,
			PolicySource:    "PSP_CATALOG",
		}, nil
	}
	code := "no_live_keys"
	if e != nil && e.status == PSPStatusPlanned {
		code = "adapter_planned"
	}
	return ExecutionResult{}, &GatewayPolicyError{
		Code:             code,
		Message:          code,
		RequestedGateway: strings.ToUpper(strings.TrimSpace(req.Gateway)),
		ResolvedGateway:  gateway,
		PolicySource:     "PSP_CATALOG",
	}
}

type staticProviderExecutor struct {
	gateway      string
	checkoutMode ExecutionMode
	urlPrefix    string
}

func (e *staticProviderExecutor) Execute(_ context.Context, req ExecutionRequest) (ExecutionResult, error) {
	result := ExecutionResult{
		ResolvedGateway: e.gateway,
		Mode:            e.checkoutMode,
		PolicySource:    "SUPPLIER_DEFAULT",
	}
	switch req.Action {
	case ExecutionActionCheckoutInit:
		if e.urlPrefix != "" {
			sessionID := strings.TrimSpace(req.SessionID)
			if sessionID == "" {
				sessionID = strings.TrimSpace(req.OrderID)
			}
			result.RedirectURL = e.urlPrefix + sessionID
		}
		return result, nil
	case ExecutionActionChargebackRecord, ExecutionActionChargebackReversal:
		result.Mode = ExecutionModeDirect
		return result, nil
	default:
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          fmt.Sprintf("unsupported execution action %s for gateway %s", req.Action, e.gateway),
			RequestedGateway: req.Gateway,
			ResolvedGateway:  e.gateway,
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}
}

type airwallexProviderExecutor struct {
	enabled bool
}

func (e *airwallexProviderExecutor) Execute(_ context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if !e.enabled {
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "card_tokenization_gateway_unsupported",
			Message:          "AIRWALLEX direct execution is disabled",
			RequestedGateway: strings.ToUpper(strings.TrimSpace(req.Gateway)),
			ResolvedGateway:  strings.ToUpper(strings.TrimSpace(req.Gateway)),
			PolicySource:     "FEATURE_FLAG",
		}
	}
	switch req.Action {
	case ExecutionActionCheckoutInit:
		return ExecutionResult{
			ResolvedGateway: "AIRWALLEX",
			Mode:            ExecutionModeHostedRedirect,
			PolicySource:    "SUPPLIER_DEFAULT",
			RedirectURL:     "/v1/payment/redirect/airwallex/" + strings.TrimSpace(req.OrderID),
		}, nil
	case ExecutionActionChargebackRecord, ExecutionActionChargebackReversal:
		return ExecutionResult{
			ResolvedGateway: "AIRWALLEX",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
		}, nil
	default:
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          fmt.Sprintf("unsupported execution action %s for gateway AIRWALLEX", req.Action),
			RequestedGateway: strings.ToUpper(strings.TrimSpace(req.Gateway)),
			ResolvedGateway:  "AIRWALLEX",
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}
}
