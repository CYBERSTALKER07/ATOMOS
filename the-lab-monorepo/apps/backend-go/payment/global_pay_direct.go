// Package payment — Global Pay Payments Service Public (direct gateway) wrapper.
// Enables charging a saved card token directly without hosting a checkout redirect.
// Supports split payments via recipients[] for supplier + platform distribution.
//
// DOCS-ONLY: All field names and endpoint paths are based on provider documentation
// and must be validated during sandbox testing.
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
)

// SplitRecipient represents a single recipient in a split payment.
// Global Pay distributes the total charge across all recipients automatically.
type SplitRecipient struct {
	MerchantID string `json:"merchant_id"` // DOCS: validate field name with sandbox
	Amount     int64  `json:"amount"`      // Amount in tiyins (UZS * 100)
}

// GlobalPayDirectClient wraps the Global Pay Payments Service Public API.
// Requires GLOBAL_PAY_GATEWAY_BASE_URL to be set.
type GlobalPayDirectClient struct {
	gatewayBaseURL string
	httpClient     *http.Client
}

// NewGlobalPayDirectClient creates a Payments Service Public client.
// Returns nil if the gateway base URL is not configured (feature disabled).
func NewGlobalPayDirectClient() *GlobalPayDirectClient {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("GLOBAL_PAY_GATEWAY_BASE_URL")), "/")
	if baseURL == "" {
		return nil
	}
	return &GlobalPayDirectClient{
		gatewayBaseURL: baseURL,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// DirectPaymentInitRequest contains the data required to initiate a direct charge.
type DirectPaymentInitRequest struct {
	CardToken  string           // Saved card token from Cards Service
	Amount     int64            // Order amount in UZS (converted to tiyins internally)
	OrderID    string           // For reference / idempotency
	SessionID  string           // Durable payment session
	ExternalID string           // Attempt ID for idempotency
	Recipients []SplitRecipient // Optional split recipients (nil = no split)
}

// DirectPaymentInitResult is returned by InitPayment.
type DirectPaymentInitResult struct {
	PaymentID        string `json:"payment_id"`
	Status           string `json:"status"`
	SecurityCheckURL string `json:"security_check_url,omitempty"` // Non-empty if 3DS verification required
}

// DirectPaymentPerformResult is returned by PerformPayment.
type DirectPaymentPerformResult struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Paid      bool   `json:"paid"`
}

// AuthorizeResult is returned by AuthorizePayment (hold without capture).
type AuthorizeResult struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"` // "AUTHORIZED" or 3DS-pending
	HoldURL   string `json:"hold_url,omitempty"`
}

// CaptureResult is returned by CapturePayment (partial/full capture of a held auth).
type CaptureResult struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Captured  bool   `json:"captured"`
}

// RecipientRegistration holds the supplier legal and bank details required by
// Global Pay to register a subordinate recipient for split-payment distribution.
type RecipientRegistration struct {
	Name         string // Legal entity name
	TIN          string // Tax Identification Number
	BankAccount  string // Bank settlement account
	BankMFO      string // Bank MFO code (Uzbekistan)
	ContactPhone string // Primary contact number
	ContactEmail string // Contact email (optional)
	OKED         string // Economic activity classifier (optional)
	LegalAddress string // Registered legal address (optional)
}

// RecipientResult is returned by RegisterRecipient.
type RecipientResult struct {
	RecipientID string `json:"recipient_id"`
	Status      string `json:"status"`
}

// InitPayment initiates a charge against a saved card token.
// If 3DS is required, SecurityCheckURL will be non-empty and the retailer must
// complete verification before calling PerformPayment.
func (c *GlobalPayDirectClient) InitPayment(ctx context.Context, creds GlobalPayCredentials, req DirectPaymentInitRequest) (*DirectPaymentInitResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay direct payments not configured (GLOBAL_PAY_GATEWAY_BASE_URL is empty)")
	}
	if strings.TrimSpace(req.CardToken) == "" {
		return nil, fmt.Errorf("card_token is required for direct payment")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	amountTiyin := req.Amount * 100

	// DOCS: validate all field names with sandbox
	payload := map[string]interface{}{
		"cardToken":  req.CardToken,  // DOCS: validate with sandbox
		"card_token": req.CardToken,  // DOCS: validate with sandbox
		"amount":     amountTiyin,    // DOCS: validate with sandbox (expected in tiyins)
		"externalId": req.ExternalID, // DOCS: validate with sandbox
		"orderId":    req.OrderID,    // DOCS: validate with sandbox
		"order_id":   req.OrderID,    // DOCS: validate with sandbox
	}

	// Add split recipients if configured
	if len(req.Recipients) > 0 {
		recipients := make([]map[string]interface{}, len(req.Recipients))
		for i, r := range req.Recipients {
			recipients[i] = map[string]interface{}{
				"merchantId":  r.MerchantID, // DOCS: validate with sandbox
				"merchant_id": r.MerchantID, // DOCS: validate with sandbox
				"amount":      r.Amount,     // DOCS: validate with sandbox (tiyins)
			}
		}
		payload["recipients"] = recipients // DOCS: validate with sandbox
	}

	endpoint := c.gatewayBaseURL + "/payments/v2/payment/init" // DOCS: validate with sandbox
	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("direct payment init failed: %w", err)
	}

	paymentID := globalPayLookupString(body,
		"paymentId", "payment_id", "id",
	)
	status := globalPayLookupString(body,
		"status", "state", "paymentStatus", "payment_status",
	)
	securityURL := globalPayLookupString(body,
		"securityUrl", "security_url",
		"securityCheckUrl", "security_check_url",
		"redirectUrl", "redirect_url",
		"threeDsUrl", "three_ds_url",
	)

	if paymentID == "" {
		return nil, fmt.Errorf("direct payment init response missing payment_id")
	}

	return &DirectPaymentInitResult{
		PaymentID:        paymentID,
		Status:           status,
		SecurityCheckURL: securityURL,
	}, nil
}

// PerformPayment finalizes a charge after 3DS verification (or immediately if no 3DS).
func (c *GlobalPayDirectClient) PerformPayment(ctx context.Context, creds GlobalPayCredentials, paymentID string) (*DirectPaymentPerformResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay direct payments not configured")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	// DOCS: validate endpoint and field names with sandbox
	endpoint := c.gatewayBaseURL + "/payments/v2/payment/perform"
	payload := map[string]interface{}{
		"paymentId":  paymentID, // DOCS: validate with sandbox
		"payment_id": paymentID, // DOCS: validate with sandbox
	}

	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("direct payment perform failed: %w", err)
	}

	status := globalPayLookupString(body,
		"status", "state", "paymentStatus", "payment_status",
	)
	paid := globalPayLookupBool(body, "paid", "isPaid", "is_paid", "success")
	if !paid {
		paid = isGlobalPayPaidStatus(status)
	}

	return &DirectPaymentPerformResult{
		PaymentID: paymentID,
		Status:    status,
		Paid:      paid,
	}, nil
}

// RevertPayment initiates a refund/reversal for a direct payment.
func (c *GlobalPayDirectClient) RevertPayment(ctx context.Context, creds GlobalPayCredentials, paymentID string, amountTiyin int64) error {
	if c == nil {
		return fmt.Errorf("global pay direct payments not configured")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return err
	}

	// DOCS: validate endpoint and field names with sandbox
	endpoint := c.gatewayBaseURL + "/payments/v2/payment/revert"
	payload := map[string]interface{}{
		"paymentId":  paymentID,   // DOCS: validate with sandbox
		"payment_id": paymentID,   // DOCS: validate with sandbox
		"amount":     amountTiyin, // DOCS: validate with sandbox
	}

	_, err = c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return fmt.Errorf("direct payment revert failed: %w", err)
	}
	return nil
}

// VerifyPaymentDirect checks the status of a direct payment.
func (c *GlobalPayDirectClient) VerifyPaymentDirect(ctx context.Context, creds GlobalPayCredentials, paymentID string) (*GlobalPayPaymentStatus, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay direct payments not configured")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	// DOCS: validate endpoint with sandbox
	endpoint := fmt.Sprintf("%s/payments/v1/payment/%s", c.gatewayBaseURL, paymentID)
	body, err := c.doJSON(ctx, http.MethodGet, endpoint, accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("direct payment status lookup failed: %w", err)
	}

	rawStatus := globalPayLookupString(body,
		"paymentStatus", "payment_status", "status", "state",
	)
	failureCode := globalPayLookupString(body, "errorCode", "error_code", "code")
	failureMessage := globalPayLookupString(body, "errorMessage", "error_message", "message")
	paid := globalPayLookupBool(body, "paid", "isPaid", "is_paid", "success")
	if !paid {
		paid = isGlobalPayPaidStatus(rawStatus)
	}

	return &GlobalPayPaymentStatus{
		ProviderPaymentID: paymentID,
		RawStatus:         rawStatus,
		Paid:              paid,
		FailureCode:       failureCode,
		FailureMessage:    failureMessage,
	}, nil
}

// AuthorizePayment places a hold on the retailer's card without capturing funds.
// The authorization reserves the full amount; actual capture happens later via
// CapturePayment (possibly for a lesser amount after driver edits).
func (c *GlobalPayDirectClient) AuthorizePayment(ctx context.Context, creds GlobalPayCredentials, req DirectPaymentInitRequest) (*AuthorizeResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay direct payments not configured (GLOBAL_PAY_GATEWAY_BASE_URL is empty)")
	}
	if strings.TrimSpace(req.CardToken) == "" {
		return nil, fmt.Errorf("card_token is required for authorization")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("authorize auth: %w", err)
	}

	amountTiyin := req.Amount * 100

	// DOCS: validate field names + "hold" or "authorize" action discriminator with sandbox
	payload := map[string]interface{}{
		"cardToken":  req.CardToken,
		"card_token": req.CardToken,
		"amount":     amountTiyin,
		"externalId": req.ExternalID,
		"orderId":    req.OrderID,
		"order_id":   req.OrderID,
		"hold":       true,
		"authorize":  true,
	}

	if len(req.Recipients) > 0 {
		recipients := make([]map[string]interface{}, len(req.Recipients))
		for i, r := range req.Recipients {
			recipients[i] = map[string]interface{}{
				"merchantId":  r.MerchantID,
				"merchant_id": r.MerchantID,
				"amount":      r.Amount,
			}
		}
		payload["recipients"] = recipients
	}

	endpoint := c.gatewayBaseURL + "/payments/v2/payment/init" // DOCS: validate auth-hold variant
	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("authorize payment failed: %w", err)
	}

	paymentID := globalPayLookupString(body, "paymentId", "payment_id", "id")
	status := globalPayLookupString(body, "status", "state", "paymentStatus", "payment_status")
	holdURL := globalPayLookupString(body,
		"securityUrl", "security_url",
		"securityCheckUrl", "security_check_url",
		"redirectUrl", "redirect_url",
		"threeDsUrl", "three_ds_url",
	)

	if paymentID == "" {
		return nil, fmt.Errorf("authorize payment response missing payment_id")
	}

	return &AuthorizeResult{
		PaymentID: paymentID,
		Status:    status,
		HoldURL:   holdURL,
	}, nil
}

// CapturePayment captures a previously authorized payment hold.
// captureAmountTiyin may be ≤ the authorized amount — Global Pay auto-scales
// split recipients proportionally when capturing a partial amount.
func (c *GlobalPayDirectClient) CapturePayment(ctx context.Context, creds GlobalPayCredentials, paymentID string, captureAmountTiyin int64) (*CaptureResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay direct payments not configured")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("capture auth: %w", err)
	}

	// DOCS: validate endpoint and field names with sandbox
	endpoint := c.gatewayBaseURL + "/payments/v2/payment/capture"
	payload := map[string]interface{}{
		"paymentId":  paymentID,
		"payment_id": paymentID,
		"amount":     captureAmountTiyin,
	}

	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("capture payment failed: %w", err)
	}

	status := globalPayLookupString(body, "status", "state", "paymentStatus", "payment_status")
	captured := globalPayLookupBool(body, "captured", "isCaptured", "is_captured", "paid", "success")
	if !captured {
		captured = isGlobalPayPaidStatus(status)
	}

	return &CaptureResult{
		PaymentID: paymentID,
		Status:    status,
		Captured:  captured,
	}, nil
}

// VoidAuthorization releases a held authorization without capturing any funds.
// Used when an order is cancelled after authorization but before delivery.
func (c *GlobalPayDirectClient) VoidAuthorization(ctx context.Context, creds GlobalPayCredentials, paymentID string) error {
	if c == nil {
		return fmt.Errorf("global pay direct payments not configured")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return fmt.Errorf("void auth: %w", err)
	}

	// DOCS: validate endpoint — may be /revert with amount=0 or a dedicated /void endpoint
	endpoint := c.gatewayBaseURL + "/payments/v2/payment/void"
	payload := map[string]interface{}{
		"paymentId":  paymentID,
		"payment_id": paymentID,
	}

	_, err = c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return fmt.Errorf("void authorization failed: %w", err)
	}
	return nil
}

// ComputeSplitRecipients calculates the supplier + platform split from an order total.
// feePercent is the platform commission in basis points (e.g., 500 = 5%).
// Returns nil if split is not possible (missing supplier recipient ID or platform merchant ID).
func ComputeSplitRecipients(amount int64, supplierRecipientID string, feePercent int64) []SplitRecipient {
	platformMerchantID := strings.TrimSpace(os.Getenv("GLOBAL_PAY_PLATFORM_MERCHANT_ID"))
	supplierRecipientID = strings.TrimSpace(supplierRecipientID)
	if supplierRecipientID == "" || platformMerchantID == "" {
		return nil // Split not configured — charge goes entirely to service account
	}
	if feePercent < 0 || feePercent > 10000 {
		return nil // Impossible commission rate — would produce negative amounts
	}

	totalTiyin := amount * 100
	platformTiyin := totalTiyin * feePercent / 10000
	supplierTiyin := totalTiyin - platformTiyin // Remainder to supplier (avoids rounding loss)

	return []SplitRecipient{
		{MerchantID: supplierRecipientID, Amount: supplierTiyin},
		{MerchantID: platformMerchantID, Amount: platformTiyin},
	}
}

// authenticate obtains an OAuth access token from the gateway.
func (c *GlobalPayDirectClient) authenticate(ctx context.Context, creds GlobalPayCredentials) (string, error) {
	authURL := c.gatewayBaseURL + "/payments/v1/merchant/auth" // DOCS: validate with sandbox
	payload := map[string]interface{}{
		"username":    creds.Username,
		"merchant_id": creds.Username,
		"login":       creds.Username,
		"password":    creds.Password,
		"secret_key":  creds.Password,
	}

	body, err := c.doJSON(ctx, http.MethodPost, authURL, "", payload)
	if err != nil {
		return "", fmt.Errorf("global pay direct auth failed: %w", err)
	}

	accessToken := globalPayLookupString(body,
		"accessToken", "access_token",
		"token", "jwt",
	)
	if accessToken == "" {
		return "", fmt.Errorf("global pay direct auth response missing access token")
	}
	return accessToken, nil
}

// doJSON performs a JSON HTTP request and returns the parsed response body.
func (c *GlobalPayDirectClient) doJSON(ctx context.Context, method, target, accessToken string, payload interface{}) (map[string]interface{}, error) {
	var rawBody []byte
	if payload != nil {
		var err error
		rawBody, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	buildReq := func() (*http.Request, error) {
		var bodyReader io.Reader
		if len(rawBody) > 0 {
			bodyReader = bytes.NewReader(rawBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if accessToken != "" {
			req.Header.Set("Authorization", globalPayAuthScheme()+" "+accessToken)
		}
		return req, nil
	}

	resp, err := retryDo(c.httpClient, buildReq, 2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBytes)))
	}

	var body map[string]interface{}
	if err := json.Unmarshal(responseBytes, &body); err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}
	return body, nil
}

// RegisterRecipient onboards a supplier as a split-payment recipient with Global Pay.
// Returns the provider-assigned RecipientID that must be stored on SupplierPaymentConfigs.
func (c *GlobalPayDirectClient) RegisterRecipient(ctx context.Context, creds GlobalPayCredentials, reg RecipientRegistration) (*RecipientResult, error) {
	if c == nil {
		return nil, fmt.Errorf("global pay direct payments not configured (GLOBAL_PAY_GATEWAY_BASE_URL is empty)")
	}
	if strings.TrimSpace(reg.Name) == "" || strings.TrimSpace(reg.TIN) == "" {
		return nil, fmt.Errorf("recipient name and TIN are required")
	}
	if strings.TrimSpace(reg.BankAccount) == "" || strings.TrimSpace(reg.BankMFO) == "" {
		return nil, fmt.Errorf("bank account and MFO are required")
	}

	accessToken, err := c.authenticate(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("register recipient auth: %w", err)
	}

	// DOCS: validate all field names with sandbox — GP recipient registration endpoint
	payload := map[string]interface{}{
		"name":          reg.Name,
		"tin":           reg.TIN,
		"bank_account":  reg.BankAccount,
		"bank_mfo":      reg.BankMFO,
		"contact_phone": reg.ContactPhone,
	}
	if reg.ContactEmail != "" {
		payload["contact_email"] = reg.ContactEmail
	}
	if reg.OKED != "" {
		payload["oked"] = reg.OKED
	}
	if reg.LegalAddress != "" {
		payload["legal_address"] = reg.LegalAddress
	}

	// DOCS: validate endpoint path with sandbox
	endpoint := c.gatewayBaseURL + "/payments/v1/recipient/register"
	body, err := c.doJSON(ctx, http.MethodPost, endpoint, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("register recipient failed: %w", err)
	}

	recipientID := globalPayLookupString(body,
		"recipientId", "recipient_id", "id", "merchantId", "merchant_id",
	)
	status := globalPayLookupString(body, "status", "state")

	if recipientID == "" {
		return nil, fmt.Errorf("register recipient response missing recipient_id")
	}

	return &RecipientResult{
		RecipientID: recipientID,
		Status:      status,
	}, nil
}
