package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// GlobalPayCredentials maps our existing vault fields onto Global Pay's
// username/password/service_id semantics.
type GlobalPayCredentials struct {
	Username  string
	Password  string
	ServiceID string
}

// GlobalPayCheckoutRequest contains the data required to initialize a hosted
// Global Pay checkout.
type GlobalPayCheckoutRequest struct {
	OrderID         string
	InvoiceID       string
	SessionID       string
	AttemptID       string
	Amount          int64
	Account         string
	CallbackBaseURL string
	Recipients      []SplitRecipient // Optional split recipients (nil = no split)
}

// GlobalPayCheckoutResult contains the provider redirect URL plus the durable
// provider reference required for callback verification.
type GlobalPayCheckoutResult struct {
	RedirectURL       string
	ProviderReference string
	ExpiresAt         *time.Time
}

// GlobalPayPaymentStatus is the normalized verification result returned from
// the provider status endpoint.
type GlobalPayPaymentStatus struct {
	ProviderPaymentID string
	ProviderReference string
	RawStatus         string
	Paid              bool
	FailureCode       string
	FailureMessage    string
}

type globalPayClient struct {
	authURL           string
	checkoutCreateURL string
	statusURL         string
	statusMethod      string
	httpClient        *http.Client
}

// ResolveGlobalPayCredentials merges vault-sourced values with environment
// fallbacks. MerchantId maps to username/login and SecretKey maps to password.
func ResolveGlobalPayCredentials(merchantID, serviceID, secretKey string) (GlobalPayCredentials, error) {
	creds := GlobalPayCredentials{
		Username:  firstNonEmpty(merchantID, os.Getenv("GLOBAL_PAY_USERNAME"), os.Getenv("GLOBAL_PAY_MERCHANT_ID")),
		Password:  firstNonEmpty(secretKey, os.Getenv("GLOBAL_PAY_PASSWORD"), os.Getenv("GLOBAL_PAY_SECRET_KEY")),
		ServiceID: firstNonEmpty(serviceID, os.Getenv("GLOBAL_PAY_SERVICE_ID")),
	}
	if creds.Username == "" || creds.Password == "" || creds.ServiceID == "" {
		return GlobalPayCredentials{}, fmt.Errorf("global pay credentials incomplete: username/login, password/secret_key, and service_id are required")
	}
	return creds, nil
}

func CreateGlobalPayHostedCheckout(ctx context.Context, creds GlobalPayCredentials, req GlobalPayCheckoutRequest) (*GlobalPayCheckoutResult, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return nil, fmt.Errorf("global pay checkout requires a durable session_id")
	}
	if strings.TrimSpace(req.InvoiceID) == "" {
		return nil, fmt.Errorf("global pay checkout requires an invoice_id")
	}
	if strings.TrimSpace(req.Account) == "" {
		return nil, fmt.Errorf("global pay checkout requires an account identifier")
	}

	client, err := newGlobalPayClient()
	if err != nil {
		return nil, err
	}

	accessToken, err := client.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	callbackURL, err := resolveGlobalPayCallbackURL(req)
	if err != nil {
		return nil, err
	}
	successURL := resolveOptionalGlobalPayURL(os.Getenv("GLOBAL_PAY_SUCCESS_REDIRECT_URL"), req)
	failURL := resolveOptionalGlobalPayURL(os.Getenv("GLOBAL_PAY_FAIL_REDIRECT_URL"), req)

	payload := map[string]interface{}{
		"serviceId":           creds.ServiceID,
		"service_id":          creds.ServiceID,
		"account":             req.Account,
		"phone":               req.Account,
		"amountTiyin":         req.Amount * 100,
		"amount_tiyin":        req.Amount * 100,
		"orderId":             req.OrderID,
		"order_id":            req.OrderID,
		"invoiceId":           req.InvoiceID,
		"invoice_id":          req.InvoiceID,
		"sessionId":           req.SessionID,
		"session_id":          req.SessionID,
		"clientTransactionId": req.AttemptID,
		"callbackUrl":         callbackURL,
		"callback_url":        callbackURL,
		"description":         fmt.Sprintf("Pegasus order %s", req.OrderID),
	}
	if successURL != "" {
		payload["successRedirectUrl"] = successURL
		payload["success_redirect_url"] = successURL
	}
	if failURL != "" {
		payload["failRedirectUrl"] = failURL
		payload["fail_redirect_url"] = failURL
	}

	// DOCS: validate split recipients field names with sandbox
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

	body, err := client.doJSON(ctx, http.MethodPost, client.checkoutCreateURL, accessToken, payload)
	if err != nil {
		return nil, fmt.Errorf("global pay checkout init failed: %w", err)
	}

	providerReference := globalPayLookupString(body,
		"userServiceToken", "user_service_token",
		"serviceToken", "service_token",
		"token", "id",
	)
	redirectURL := globalPayLookupString(body,
		"userRedirectUrl", "user_redirect_url",
		"redirectUrl", "redirect_url",
		"paymentUrl", "payment_url",
		"url",
	)
	if providerReference == "" || redirectURL == "" {
		return nil, fmt.Errorf("global pay checkout init returned incomplete payload")
	}

	return &GlobalPayCheckoutResult{
		RedirectURL:       redirectURL,
		ProviderReference: providerReference,
		ExpiresAt:         globalPayLookupTime(body, "expiresAt", "expires_at"),
	}, nil
}

func VerifyGlobalPayPayment(ctx context.Context, creds GlobalPayCredentials, providerReference, providerPaymentID string) (*GlobalPayPaymentStatus, error) {
	if strings.TrimSpace(providerReference) == "" {
		return nil, fmt.Errorf("global pay status verification requires provider reference")
	}

	client, err := newGlobalPayClient()
	if err != nil {
		return nil, err
	}
	accessToken, err := client.authenticate(ctx, creds)
	if err != nil {
		return nil, err
	}

	body, err := client.lookupPaymentStatus(ctx, accessToken, providerReference, providerPaymentID)
	if err != nil {
		return nil, fmt.Errorf("global pay payment status lookup failed: %w", err)
	}

	rawStatus := globalPayLookupString(body,
		"paymentStatus", "payment_status",
		"status", "state",
	)
	failureCode := globalPayLookupString(body, "errorCode", "error_code", "code")
	failureMessage := globalPayLookupString(body, "errorMessage", "error_message", "message")
	verifiedPaymentID := firstNonEmpty(providerPaymentID, globalPayLookupString(body, "paymentId", "payment_id", "id"))
	paid := globalPayLookupBool(body, "paid", "isPaid", "is_paid", "success", "completed")
	if !paid {
		paid = isGlobalPayPaidStatus(rawStatus)
	}

	return &GlobalPayPaymentStatus{
		ProviderPaymentID: verifiedPaymentID,
		ProviderReference: providerReference,
		RawStatus:         rawStatus,
		Paid:              paid,
		FailureCode:       failureCode,
		FailureMessage:    failureMessage,
	}, nil
}

func (s GlobalPayPaymentStatus) Failed() bool {
	if s.Paid {
		return false
	}
	if s.FailureCode != "" || s.FailureMessage != "" {
		return true
	}
	return isGlobalPayFailedStatus(s.RawStatus)
}

func newGlobalPayClient() (*globalPayClient, error) {
	authURL := strings.TrimSpace(os.Getenv("GLOBAL_PAY_AUTH_URL"))
	checkoutCreateURL := strings.TrimSpace(os.Getenv("GLOBAL_PAY_CHECKOUT_CREATE_URL"))
	statusURL := strings.TrimSpace(os.Getenv("GLOBAL_PAY_STATUS_URL"))
	if authURL == "" || checkoutCreateURL == "" || statusURL == "" {
		return nil, fmt.Errorf("global pay endpoints not configured: GLOBAL_PAY_AUTH_URL, GLOBAL_PAY_CHECKOUT_CREATE_URL, and GLOBAL_PAY_STATUS_URL are required")
	}
	statusMethod := strings.ToUpper(strings.TrimSpace(os.Getenv("GLOBAL_PAY_STATUS_METHOD")))
	if statusMethod == "" {
		statusMethod = http.MethodPost
	}
	return &globalPayClient{
		authURL:           authURL,
		checkoutCreateURL: checkoutCreateURL,
		statusURL:         statusURL,
		statusMethod:      statusMethod,
		httpClient:        &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (c *globalPayClient) authenticate(ctx context.Context, creds GlobalPayCredentials) (string, error) {
	body, err := c.doJSON(ctx, http.MethodPost, c.authURL, "", map[string]interface{}{
		"username":    creds.Username,
		"merchant_id": creds.Username,
		"login":       creds.Username,
		"password":    creds.Password,
		"secret_key":  creds.Password,
	})
	if err != nil {
		return "", fmt.Errorf("global pay auth failed: %w", err)
	}
	accessToken := globalPayLookupString(body,
		"accessToken", "access_token",
		"token", "jwt",
	)
	if accessToken == "" {
		return "", fmt.Errorf("global pay auth response missing access token")
	}
	return accessToken, nil
}

func (c *globalPayClient) lookupPaymentStatus(ctx context.Context, accessToken, providerReference, providerPaymentID string) (map[string]interface{}, error) {
	if c.statusMethod == http.MethodGet {
		parsedURL, err := url.Parse(c.statusURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GLOBAL_PAY_STATUS_URL: %w", err)
		}
		q := parsedURL.Query()
		q.Set("service_token", providerReference)
		q.Set("serviceToken", providerReference)
		if providerPaymentID != "" {
			q.Set("payment_id", providerPaymentID)
			q.Set("paymentId", providerPaymentID)
		}
		parsedURL.RawQuery = q.Encode()
		return c.doJSON(ctx, http.MethodGet, parsedURL.String(), accessToken, nil)
	}

	payload := map[string]interface{}{
		"serviceToken":  providerReference,
		"service_token": providerReference,
	}
	if providerPaymentID != "" {
		payload["paymentId"] = providerPaymentID
		payload["payment_id"] = providerPaymentID
	}
	return c.doJSON(ctx, c.statusMethod, c.statusURL, accessToken, payload)
}

func (c *globalPayClient) doJSON(ctx context.Context, method, target, accessToken string, payload interface{}) (map[string]interface{}, error) {
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

func resolveGlobalPayCallbackURL(req GlobalPayCheckoutRequest) (string, error) {
	template := strings.TrimSpace(os.Getenv("GLOBAL_PAY_CALLBACK_URL"))
	if template == "" {
		base := strings.TrimRight(strings.TrimSpace(req.CallbackBaseURL), "/")
		if base == "" {
			return "", fmt.Errorf("global pay callback URL unavailable: set GLOBAL_PAY_CALLBACK_URL or supply a request base URL")
		}
		template = base + "/v1/webhooks/global-pay?session_id={session_id}&invoice_id={invoice_id}&order_id={order_id}"
	}
	return renderGlobalPayURLTemplate(template, req), nil
}

func resolveOptionalGlobalPayURL(template string, req GlobalPayCheckoutRequest) string {
	template = strings.TrimSpace(template)
	if template == "" {
		return ""
	}
	return renderGlobalPayURLTemplate(template, req)
}

func renderGlobalPayURLTemplate(template string, req GlobalPayCheckoutRequest) string {
	replacer := strings.NewReplacer(
		"{order_id}", url.QueryEscape(req.OrderID),
		"{invoice_id}", url.QueryEscape(req.InvoiceID),
		"{session_id}", url.QueryEscape(req.SessionID),
		"{attempt_id}", url.QueryEscape(req.AttemptID),
		"{account}", url.QueryEscape(req.Account),
	)
	return replacer.Replace(template)
}

func globalPayAuthScheme() string {
	scheme := strings.TrimSpace(os.Getenv("GLOBAL_PAY_ACCESS_TOKEN_SCHEME"))
	if scheme == "" {
		return "Bearer"
	}
	return scheme
}

func globalPayLookupString(body map[string]interface{}, keys ...string) string {
	for _, container := range globalPayCandidateMaps(body) {
		for _, key := range keys {
			if value, ok := container[key]; ok {
				if stringValue := globalPayAnyToString(value); stringValue != "" {
					return stringValue
				}
			}
		}
	}
	return ""
}

func globalPayLookupBool(body map[string]interface{}, keys ...string) bool {
	for _, container := range globalPayCandidateMaps(body) {
		for _, key := range keys {
			if value, ok := container[key]; ok {
				switch typed := value.(type) {
				case bool:
					return typed
				case string:
					normalized := strings.ToLower(strings.TrimSpace(typed))
					return normalized == "true" || normalized == "1" || normalized == "yes"
				case float64:
					return typed != 0
				}
			}
		}
	}
	return false
}

func globalPayLookupTime(body map[string]interface{}, keys ...string) *time.Time {
	value := globalPayLookupString(body, keys...)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z07:00", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	if unixSeconds, err := strconv.ParseInt(value, 10, 64); err == nil && unixSeconds > 0 {
		parsed := time.Unix(unixSeconds, 0).UTC()
		return &parsed
	}
	return nil
}

func globalPayCandidateMaps(body map[string]interface{}) []map[string]interface{} {
	containers := []map[string]interface{}{body}
	for _, key := range []string{"data", "result", "payload", "response"} {
		if nested, ok := body[key].(map[string]interface{}); ok {
			containers = append(containers, nested)
		}
	}
	return containers
}

func globalPayAnyToString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func isGlobalPayPaidStatus(status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "SUCCESS", "SUCCEEDED", "PAID", "COMPLETED", "COMPLETE", "CONFIRMED", "APPROVED":
		return true
	default:
		return false
	}
}

func isGlobalPayFailedStatus(status string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "FAILED", "FAIL", "CANCELLED", "CANCELED", "DECLINED", "REJECTED", "EXPIRED", "ERROR":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
