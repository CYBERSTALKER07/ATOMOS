package payment

import (
	"context"
	"fmt"
	"strings"
)

// ExecutionAction describes a provider-directed next step such as a redirect.
type ExecutionAction struct {
	Type        string         `json:"type"`
	RedirectURL string         `json:"redirect_url,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// StoredMethodChargeRequest contains the provider-agnostic inputs needed to
// charge a previously saved payment method.
type StoredMethodChargeRequest struct {
	CardToken                string
	Amount                   int64
	Currency                 string
	OrderID                  string
	SessionID                string
	AttemptID                string
	ExternalID               string
	ReturnURL                string
	ShopperReference         string
	RecurringProcessingModel string
	Recipients               []SplitRecipient
}

// StoredMethodChargeResult contains the normalized outcome of a saved-method charge.
type StoredMethodChargeResult struct {
	ProviderPaymentID string           `json:"provider_payment_id,omitempty"`
	ProviderReference string           `json:"provider_reference,omitempty"`
	Status            string           `json:"status,omitempty"`
	Action            *ExecutionAction `json:"action,omitempty"`
	Paid              bool             `json:"paid"`
	StoredMethodRef   string           `json:"stored_method_ref,omitempty"`
	RecurringCapable  bool             `json:"recurring_capable,omitempty"`
}

// StoredMethodAuthorizationRequest contains the normalized inputs required to
// place an authorization hold on a saved payment method.
type StoredMethodAuthorizationRequest struct {
	CardToken                string
	Amount                   int64
	Currency                 string
	OrderID                  string
	SessionID                string
	AttemptID                string
	ExternalID               string
	ReturnURL                string
	ShopperReference         string
	RecurringProcessingModel string
	Recipients               []SplitRecipient
}

// StoredMethodAuthorizationResult contains the normalized outcome of a saved-method authorization.
type StoredMethodAuthorizationResult struct {
	AuthorizationID   string           `json:"authorization_id,omitempty"`
	ProviderReference string           `json:"provider_reference,omitempty"`
	Status            string           `json:"status,omitempty"`
	Action            *ExecutionAction `json:"action,omitempty"`
	Authorized        bool             `json:"authorized"`
}

// CaptureAuthorizationRequest contains the provider-agnostic inputs needed to
// capture a previously authorized payment hold.
type CaptureAuthorizationRequest struct {
	AuthorizationID string
	Amount          int64
	Currency        string
}

// CaptureAuthorizationResult contains the normalized outcome of a capture request.
type CaptureAuthorizationResult struct {
	ProviderPaymentID string `json:"provider_payment_id,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
	Status            string `json:"status,omitempty"`
	Captured          bool   `json:"captured"`
}

// ProviderRefundRequest contains the provider-agnostic inputs needed to refund
// a previously executed payment.
type ProviderRefundRequest struct {
	OrderID   string
	PaymentID string
	Amount    int64
	Currency  string
}

// ProviderRefundResult contains the normalized outcome of a provider refund.
type ProviderRefundResult struct {
	ProviderRefundID  string `json:"provider_refund_id,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
	Status            string `json:"status,omitempty"`
}

// ProviderExecutionClient executes direct, saved-method, and refund operations
// behind a gateway-normalized interface.
type ProviderExecutionClient interface {
	GatewayName() string
	ChargeStoredMethod(ctx context.Context, req StoredMethodChargeRequest) (*StoredMethodChargeResult, error)
	AuthorizeStoredMethod(ctx context.Context, req StoredMethodAuthorizationRequest) (*StoredMethodAuthorizationResult, error)
	CaptureAuthorization(ctx context.Context, req CaptureAuthorizationRequest) (*CaptureAuthorizationResult, error)
	VoidAuthorization(ctx context.Context, authorizationID string) error
	RefundPayment(ctx context.Context, req ProviderRefundRequest) (*ProviderRefundResult, error)
}

// ProviderExecutionCredentials carries provider-specific credential payloads
// for execution resolvers without leaking concrete gateway types to callers.
type ProviderExecutionCredentials struct {
	globalPay *GlobalPayCredentials
	adyen     *AdyenCredentials
}

// NewGlobalPayExecutionCredentials wraps Global Pay credentials for the
// execution resolver.
func NewGlobalPayExecutionCredentials(creds GlobalPayCredentials) ProviderExecutionCredentials {
	return ProviderExecutionCredentials{globalPay: &creds}
}

// NewAdyenExecutionCredentials wraps Adyen credentials for the execution resolver.
func NewAdyenExecutionCredentials(creds AdyenCredentials) ProviderExecutionCredentials {
	return ProviderExecutionCredentials{adyen: &creds}
}

// ProviderExecutionRouter resolves a normalized execution client for a gateway.
type ProviderExecutionRouter struct {
	GlobalPayDirect *GlobalPayDirectClient
}

// NewProviderExecutionRouter creates a provider execution router using the
// currently configured direct-payment clients.
func NewProviderExecutionRouter(globalPayDirect *GlobalPayDirectClient) *ProviderExecutionRouter {
	return &ProviderExecutionRouter{GlobalPayDirect: globalPayDirect}
}

// Resolve returns a gateway-normalized execution client.
func (r *ProviderExecutionRouter) Resolve(gateway string, creds ProviderExecutionCredentials) (ProviderExecutionClient, error) {
	normalized := normalizeGateway(gateway)
	switch normalized {
	case "GLOBAL_PAY":
		globalPayCreds, err := creds.globalPayCredentials()
		if err != nil {
			return nil, err
		}
		if r == nil || r.GlobalPayDirect == nil {
			return nil, fmt.Errorf("provider execution unavailable for gateway %s", normalized)
		}
		return &globalPayExecutionClient{client: r.GlobalPayDirect, creds: globalPayCreds}, nil
	case "ADYEN":
		adyenCreds, err := creds.adyenCredentials()
		if err != nil {
			return nil, err
		}
		return &adyenExecutionClient{creds: adyenCreds}, nil
	case "CASH":
		return &unsupportedProviderExecutionClient{gateway: normalized}, nil
	default:
		return nil, fmt.Errorf("unsupported payment gateway: %s", strings.TrimSpace(gateway))
	}
}

func (c ProviderExecutionCredentials) globalPayCredentials() (GlobalPayCredentials, error) {
	if c.globalPay == nil {
		return GlobalPayCredentials{}, fmt.Errorf("global pay execution credentials missing")
	}
	return *c.globalPay, nil
}

func (c ProviderExecutionCredentials) adyenCredentials() (AdyenCredentials, error) {
	if c.adyen == nil {
		return AdyenCredentials{}, fmt.Errorf("adyen execution credentials missing")
	}
	return *c.adyen, nil
}

type unsupportedProviderExecutionClient struct {
	gateway string
}

func (c *unsupportedProviderExecutionClient) GatewayName() string {
	return c.gateway
}

func (c *unsupportedProviderExecutionClient) ChargeStoredMethod(ctx context.Context, req StoredMethodChargeRequest) (*StoredMethodChargeResult, error) {
	return nil, fmt.Errorf("stored-method charge unsupported for gateway %s", c.gateway)
}

func (c *unsupportedProviderExecutionClient) AuthorizeStoredMethod(ctx context.Context, req StoredMethodAuthorizationRequest) (*StoredMethodAuthorizationResult, error) {
	return nil, fmt.Errorf("stored-method authorization unsupported for gateway %s", c.gateway)
}

func (c *unsupportedProviderExecutionClient) CaptureAuthorization(ctx context.Context, req CaptureAuthorizationRequest) (*CaptureAuthorizationResult, error) {
	return nil, fmt.Errorf("capture unsupported for gateway %s", c.gateway)
}

func (c *unsupportedProviderExecutionClient) VoidAuthorization(ctx context.Context, authorizationID string) error {
	return fmt.Errorf("authorization void unsupported for gateway %s", c.gateway)
}

func (c *unsupportedProviderExecutionClient) RefundPayment(ctx context.Context, req ProviderRefundRequest) (*ProviderRefundResult, error) {
	return nil, fmt.Errorf("refund unsupported for gateway %s", c.gateway)
}

type globalPayExecutionClient struct {
	client *GlobalPayDirectClient
	creds  GlobalPayCredentials
}

func (c *globalPayExecutionClient) GatewayName() string {
	return "GLOBAL_PAY"
}

func (c *globalPayExecutionClient) ChargeStoredMethod(ctx context.Context, req StoredMethodChargeRequest) (*StoredMethodChargeResult, error) {
	if strings.TrimSpace(req.CardToken) == "" {
		return nil, fmt.Errorf("stored-method charge requires card token")
	}
	initResult, err := c.client.InitPayment(ctx, c.creds, DirectPaymentInitRequest{
		CardToken:  req.CardToken,
		Amount:     req.Amount,
		OrderID:    req.OrderID,
		SessionID:  req.SessionID,
		ExternalID: firstNonEmpty(strings.TrimSpace(req.ExternalID), strings.TrimSpace(req.AttemptID), strings.TrimSpace(req.SessionID), strings.TrimSpace(req.OrderID)),
		Recipients: req.Recipients,
	})
	if err != nil {
		return nil, err
	}

	result := &StoredMethodChargeResult{
		ProviderPaymentID: initResult.PaymentID,
		ProviderReference: initResult.PaymentID,
		Status:            strings.TrimSpace(initResult.Status),
		StoredMethodRef:   strings.TrimSpace(req.CardToken),
		RecurringCapable:  strings.TrimSpace(req.CardToken) != "",
	}
	if redirectURL := strings.TrimSpace(initResult.SecurityCheckURL); redirectURL != "" {
		result.Action = &ExecutionAction{Type: "REDIRECT", RedirectURL: redirectURL}
		return result, nil
	}

	performResult, err := c.client.PerformPayment(ctx, c.creds, initResult.PaymentID)
	if err != nil {
		return nil, err
	}
	result.Status = firstNonEmpty(strings.TrimSpace(performResult.Status), result.Status)
	result.Paid = performResult.Paid
	return result, nil
}

func (c *globalPayExecutionClient) AuthorizeStoredMethod(ctx context.Context, req StoredMethodAuthorizationRequest) (*StoredMethodAuthorizationResult, error) {
	if strings.TrimSpace(req.CardToken) == "" {
		return nil, fmt.Errorf("stored-method authorization requires card token")
	}
	authResult, err := c.client.AuthorizePayment(ctx, c.creds, DirectPaymentInitRequest{
		CardToken:  req.CardToken,
		Amount:     req.Amount,
		OrderID:    req.OrderID,
		SessionID:  req.SessionID,
		ExternalID: firstNonEmpty(strings.TrimSpace(req.ExternalID), strings.TrimSpace(req.AttemptID), strings.TrimSpace(req.SessionID), strings.TrimSpace(req.OrderID)),
		Recipients: req.Recipients,
	})
	if err != nil {
		return nil, err
	}

	result := &StoredMethodAuthorizationResult{
		AuthorizationID:   authResult.PaymentID,
		ProviderReference: authResult.PaymentID,
		Status:            strings.TrimSpace(authResult.Status),
		Authorized:        strings.TrimSpace(authResult.PaymentID) != "" && strings.TrimSpace(authResult.HoldURL) == "",
	}
	if redirectURL := strings.TrimSpace(authResult.HoldURL); redirectURL != "" {
		result.Action = &ExecutionAction{Type: "REDIRECT", RedirectURL: redirectURL}
	}
	return result, nil
}

func (c *globalPayExecutionClient) CaptureAuthorization(ctx context.Context, req CaptureAuthorizationRequest) (*CaptureAuthorizationResult, error) {
	if strings.TrimSpace(req.AuthorizationID) == "" {
		return nil, fmt.Errorf("capture requires authorization id")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("capture requires positive amount")
	}
	captureResult, err := c.client.CapturePayment(ctx, c.creds, req.AuthorizationID, req.Amount*100)
	if err != nil {
		return nil, err
	}
	return &CaptureAuthorizationResult{
		ProviderPaymentID: strings.TrimSpace(captureResult.PaymentID),
		ProviderReference: strings.TrimSpace(captureResult.PaymentID),
		Status:            strings.TrimSpace(captureResult.Status),
		Captured:          captureResult.Captured,
	}, nil
}

func (c *globalPayExecutionClient) VoidAuthorization(ctx context.Context, authorizationID string) error {
	if strings.TrimSpace(authorizationID) == "" {
		return fmt.Errorf("void authorization requires authorization id")
	}
	return c.client.VoidAuthorization(ctx, c.creds, authorizationID)
}

func (c *globalPayExecutionClient) RefundPayment(ctx context.Context, req ProviderRefundRequest) (*ProviderRefundResult, error) {
	if strings.TrimSpace(req.PaymentID) == "" {
		return nil, fmt.Errorf("refund requires provider payment id")
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("refund requires positive amount")
	}
	if err := c.client.RevertPayment(ctx, c.creds, req.PaymentID, req.Amount*100); err != nil {
		return nil, err
	}
	providerRef := strings.TrimSpace(req.PaymentID)
	return &ProviderRefundResult{
		ProviderRefundID:  providerRef,
		ProviderReference: providerRef,
		Status:            "REFUNDED",
	}, nil
}

type adyenExecutionClient struct {
	creds AdyenCredentials
}

func (c *adyenExecutionClient) GatewayName() string {
	return "ADYEN"
}

func (c *adyenExecutionClient) ChargeStoredMethod(ctx context.Context, req StoredMethodChargeRequest) (*StoredMethodChargeResult, error) {
	if !c.creds.DirectExecutionEnabled {
		return nil, fmt.Errorf("%w for order %s", ErrAdyenDirectOperationUnsupported, strings.TrimSpace(req.OrderID))
	}
	return nil, fmt.Errorf("adyen direct charge not yet implemented")
}

func (c *adyenExecutionClient) AuthorizeStoredMethod(ctx context.Context, req StoredMethodAuthorizationRequest) (*StoredMethodAuthorizationResult, error) {
	if !c.creds.DirectExecutionEnabled {
		return nil, fmt.Errorf("%w for order %s", ErrAdyenDirectOperationUnsupported, strings.TrimSpace(req.OrderID))
	}
	return nil, fmt.Errorf("adyen direct authorize not yet implemented")
}

func (c *adyenExecutionClient) CaptureAuthorization(ctx context.Context, req CaptureAuthorizationRequest) (*CaptureAuthorizationResult, error) {
	if !c.creds.DirectExecutionEnabled {
		return nil, fmt.Errorf("%w for authorization %s", ErrAdyenDirectOperationUnsupported, strings.TrimSpace(req.AuthorizationID))
	}
	return nil, fmt.Errorf("adyen direct capture not yet implemented")
}

func (c *adyenExecutionClient) VoidAuthorization(ctx context.Context, authorizationID string) error {
	if !c.creds.DirectExecutionEnabled {
		return fmt.Errorf("%w for authorization %s", ErrAdyenDirectOperationUnsupported, strings.TrimSpace(authorizationID))
	}
	return fmt.Errorf("adyen direct void not yet implemented")
}

func (c *adyenExecutionClient) RefundPayment(ctx context.Context, req ProviderRefundRequest) (*ProviderRefundResult, error) {
	if !c.creds.DirectExecutionEnabled {
		return nil, fmt.Errorf("%w for order %s", ErrAdyenDirectOperationUnsupported, strings.TrimSpace(req.OrderID))
	}
	client, err := CreateAdyenDirectClient(c.creds)
	if err != nil {
		return nil, err
	}
	res, err := client.Refund(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}
