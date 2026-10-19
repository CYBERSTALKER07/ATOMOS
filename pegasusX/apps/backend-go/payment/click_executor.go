package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

// clickProviderExecutor is the outbound Merchant API + hosted SHOP pay URL.
// Compiled and tested; MUST NOT be registered on the live router.
type clickProviderExecutor struct {
	merchantID     string
	serviceID      string
	merchantUserID string
	secret         string
	apiBase        string
	httpClient     *http.Client
	breaker        *circuit.Breaker
	nowUnix        func() int64
}

func newClickProviderExecutor(merchantID, serviceID, merchantUserID, secret string) *clickProviderExecutor {
	return newClickProviderExecutorWithOptions(merchantID, serviceID, merchantUserID, secret, "", nil)
}

func newClickProviderExecutorWithOptions(merchantID, serviceID, merchantUserID, secret, apiBase string, breaker *circuit.Breaker) *clickProviderExecutor {
	return &clickProviderExecutor{
		merchantID:     strings.TrimSpace(merchantID),
		serviceID:      strings.TrimSpace(serviceID),
		merchantUserID: strings.TrimSpace(merchantUserID),
		secret:         strings.TrimSpace(secret),
		apiBase:        strings.TrimSpace(apiBase),
		breaker:        breaker,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
		nowUnix:        func() int64 { return time.Now().Unix() },
	}
}

var errClickCredentialsMissing = fmt.Errorf("click credentials missing (CLICK_MERCHANT_ID/CLICK_SERVICE_ID) — adapter is implemented but not wired")

func errClickUnkeyed() error {
	return fmt.Errorf("%w: %w", errClickCredentialsMissing, &GatewayPolicyError{
		Code:             "no_live_keys",
		Message:          "no_live_keys",
		RequestedGateway: "CLICK",
		ResolvedGateway:  "CLICK",
		PolicySource:     "PSP_CATALOG",
	})
}

func (e *clickProviderExecutor) keyed() bool {
	return e != nil && e.merchantID != "" && e.serviceID != ""
}

func (e *clickProviderExecutor) merchantAPIKeyed() bool {
	return e.keyed() && e.merchantUserID != "" && e.secret != ""
}

func (e *clickProviderExecutor) doHTTP(ctx context.Context, call func(context.Context) error) error {
	if e.breaker != nil {
		return e.breaker.Do(ctx, call)
	}
	return call(ctx)
}

func (e *clickProviderExecutor) baseURL() string {
	if e.apiBase != "" {
		return strings.TrimRight(e.apiBase, "/")
	}
	return clickMerchantBaseURL()
}

func (e *clickProviderExecutor) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	switch req.Action {
	case ExecutionActionChargebackRecord, ExecutionActionChargebackReversal:
		return ExecutionResult{
			ResolvedGateway: "CLICK",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
		}, nil
	case ExecutionActionCheckoutInit, ExecutionActionCheckoutCapture, ExecutionActionStatusCheck, ExecutionActionRefund:
	default:
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          fmt.Sprintf("unsupported execution action %s for gateway CLICK", req.Action),
			RequestedGateway: req.Gateway,
			ResolvedGateway:  "CLICK",
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}
	if !e.keyed() {
		return ExecutionResult{}, errClickUnkeyed()
	}
	switch req.Action {
	case ExecutionActionCheckoutInit:
		return e.executeCheckoutInit(req)
	case ExecutionActionStatusCheck:
		return e.executePaymentStatus(ctx, req)
	case ExecutionActionRefund:
		return e.executeReversal(ctx, req)
	case ExecutionActionCheckoutCapture:
		return e.executeCheckoutCapture(ctx, req)
	default:
		return ExecutionResult{}, errClickUnkeyed()
	}
}

func (e *clickProviderExecutor) executeCheckoutInit(req ExecutionRequest) (ExecutionResult, error) {
	url := clickHostedPayURL(
		e.serviceID,
		e.merchantID,
		req.OrderID,
		clickMinorToSom(req.AmountMinor),
		"",
	)
	return ExecutionResult{
		ResolvedGateway: "CLICK",
		Mode:            ExecutionModeHostedRedirect,
		PolicySource:    "CLICK_SHOP",
		RedirectURL:     url,
		ProviderRef:     strings.TrimSpace(req.OrderID),
	}, nil
}

func (e *clickProviderExecutor) executeCreateInvoice(ctx context.Context, req ExecutionRequest, phone string) (ExecutionResult, error) {
	if !e.merchantAPIKeyed() {
		return ExecutionResult{}, errClickUnkeyed()
	}
	body := clickInvoiceJSON(e.serviceID, req.AmountMinor, phone, req.OrderID)
	raw, err := e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/invoice/create", body)
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		ErrorCode int    `json:"error_code"`
		ErrorNote string `json:"error_note"`
		InvoiceID int64  `json:"invoice_id"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return ExecutionResult{}, fmt.Errorf("click invoice decode: %w", err)
	}
	if wrap.ErrorCode != 0 {
		return ExecutionResult{}, fmt.Errorf("click invoice: %d %s", wrap.ErrorCode, wrap.ErrorNote)
	}
	return ExecutionResult{
		ResolvedGateway: "CLICK",
		Mode:            ExecutionModeDirect,
		PolicySource:    "CLICK_MERCHANT",
		ProviderRef:     strconvInt64(wrap.InvoiceID),
	}, nil
}

func clickInvoiceJSON(serviceID string, amountMinor int64, phone, orderID string) []byte {
	var b bytes.Buffer
	b.WriteString(`{"service_id":`)
	b.WriteString(jsonNumberOrQuoted(serviceID))
	b.WriteString(`,"amount":`)
	b.WriteString(clickMinorToSom(amountMinor))
	b.WriteString(`,"phone_number":`)
	enc, _ := json.Marshal(phone)
	b.Write(enc)
	b.WriteString(`,"merchant_trans_id":`)
	enc, _ = json.Marshal(orderID)
	b.Write(enc)
	b.WriteByte('}')
	return b.Bytes()
}

func jsonNumberOrQuoted(s string) string {
	s = strings.TrimSpace(s)
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return s
	}
	enc, _ := json.Marshal(s)
	return string(enc)
}

func strconvInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

func (e *clickProviderExecutor) executeInvoiceStatus(ctx context.Context, invoiceID string) (ExecutionResult, error) {
	if !e.merchantAPIKeyed() {
		return ExecutionResult{}, errClickUnkeyed()
	}
	path := fmt.Sprintf("%s/invoice/status/%s/%s", e.baseURL(), e.serviceID, strings.TrimSpace(invoiceID))
	raw, err := e.callMerchant(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		ErrorCode int    `json:"error_code"`
		ErrorNote string `json:"error_note"`
		Status    int    `json:"status"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return ExecutionResult{}, err
	}
	if wrap.ErrorCode != 0 {
		return ExecutionResult{}, fmt.Errorf("click invoice status: %d %s", wrap.ErrorCode, wrap.ErrorNote)
	}
	return ExecutionResult{
		ResolvedGateway: "CLICK",
		Mode:            ExecutionModeDirect,
		PolicySource:    "CLICK_MERCHANT",
		ProviderRef:     fmt.Sprintf("invoice_status:%d", wrap.Status),
	}, nil
}

func (e *clickProviderExecutor) executePaymentStatusByMTI(ctx context.Context, merchantTransID, date string) (ExecutionResult, error) {
	if !e.merchantAPIKeyed() {
		return ExecutionResult{}, errClickUnkeyed()
	}
	path := fmt.Sprintf("%s/payment/status_by_mti/%s/%s/%s", e.baseURL(), e.serviceID, merchantTransID, date)
	raw, err := e.callMerchant(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		ErrorCode     int `json:"error_code"`
		PaymentStatus int `json:"payment_status"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return ExecutionResult{}, err
	}
	if wrap.ErrorCode != 0 {
		return ExecutionResult{}, fmt.Errorf("click status_by_mti: %d", wrap.ErrorCode)
	}
	return ExecutionResult{
		ResolvedGateway: "CLICK",
		Mode:            ExecutionModeDirect,
		PolicySource:    "CLICK_MERCHANT",
		ProviderRef:     fmt.Sprintf("status:%d", wrap.PaymentStatus),
	}, nil
}

func (e *clickProviderExecutor) executeCardTokenRequest(ctx context.Context, cardNumber, expireDate string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	body := []byte(fmt.Sprintf(
		`{"service_id":%s,"card_number":%s,"expire_date":%s,"temporary":1}`,
		jsonNumberOrQuoted(e.serviceID),
		mustJSONString(cardNumber),
		mustJSONString(expireDate),
	))
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/card_token/request", body)
}

func (e *clickProviderExecutor) executeCardTokenVerify(ctx context.Context, cardToken, smsCode string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	body := []byte(fmt.Sprintf(
		`{"service_id":%s,"card_token":%s,"sms_code":%s}`,
		jsonNumberOrQuoted(e.serviceID),
		mustJSONString(cardToken),
		mustJSONString(smsCode),
	))
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/card_token/verify", body)
}

func (e *clickProviderExecutor) executeCardTokenPayment(ctx context.Context, cardToken string, amountMinor int64, merchantTransID string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	body := []byte(fmt.Sprintf(
		`{"service_id":%s,"card_token":%s,"amount":%s,"transaction_parameter":%s}`,
		jsonNumberOrQuoted(e.serviceID),
		mustJSONString(cardToken),
		clickMinorToSom(amountMinor),
		mustJSONString(merchantTransID),
	))
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/card_token/payment", body)
}

func (e *clickProviderExecutor) executeCardTokenDelete(ctx context.Context, cardToken string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	path := fmt.Sprintf("%s/card_token/%s/%s", e.baseURL(), e.serviceID, cardToken)
	return e.callMerchant(ctx, http.MethodDelete, path, nil)
}

func mustJSONString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (e *clickProviderExecutor) executePaymentStatus(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if !e.merchantAPIKeyed() {
		return ExecutionResult{}, errClickUnkeyed()
	}
	paymentID := strings.TrimSpace(req.SessionID)
	if paymentID == "" {
		paymentID = strings.TrimSpace(req.OrderID)
	}
	path := fmt.Sprintf("%s/payment/status/%s/%s", e.baseURL(), e.serviceID, paymentID)
	raw, err := e.callMerchant(ctx, http.MethodGet, path, nil)
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		ErrorCode     int    `json:"error_code"`
		ErrorNote     string `json:"error_note"`
		PaymentStatus int    `json:"payment_status"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return ExecutionResult{}, fmt.Errorf("click status decode: %w", err)
	}
	if wrap.ErrorCode != 0 {
		return ExecutionResult{}, fmt.Errorf("click status: %d %s", wrap.ErrorCode, wrap.ErrorNote)
	}
	return ExecutionResult{
		ResolvedGateway: "CLICK",
		Mode:            ExecutionModeDirect,
		PolicySource:    "CLICK_MERCHANT",
		ProviderRef:     fmt.Sprintf("status:%d", wrap.PaymentStatus),
	}, nil
}

func (e *clickProviderExecutor) executeReversal(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if !e.merchantAPIKeyed() {
		return ExecutionResult{}, errClickUnkeyed()
	}
	paymentID := strings.TrimSpace(req.SessionID)
	if paymentID == "" {
		paymentID = strings.TrimSpace(req.OrderID)
	}
	path := fmt.Sprintf("%s/payment/reversal/%s/%s", e.baseURL(), e.serviceID, paymentID)
	raw, err := e.callMerchant(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		ErrorCode int    `json:"error_code"`
		ErrorNote string `json:"error_note"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return ExecutionResult{}, fmt.Errorf("click reversal decode: %w", err)
	}
	if wrap.ErrorCode != 0 {
		return ExecutionResult{}, fmt.Errorf("click reversal: %d %s", wrap.ErrorCode, wrap.ErrorNote)
	}
	return ExecutionResult{
		ResolvedGateway: "CLICK",
		Mode:            ExecutionModeDirect,
		PolicySource:    "CLICK_MERCHANT",
		ProviderRef:     paymentID,
	}, nil
}

func (e *clickProviderExecutor) callMerchant(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	unix := time.Now().Unix()
	if e.nowUnix != nil {
		unix = e.nowUnix()
	}
	httpReq.Header.Set("Auth", clickMerchantAuthHeader(e.merchantUserID, e.secret, unix))

	var respBody []byte
	err = e.doHTTP(ctx, func(callCtx context.Context) error {
		resp, err := e.httpClient.Do(httpReq.WithContext(callCtx))
		if err != nil {
			return MarkRetryable(fmt.Errorf("click merchant request failed: %w", err))
		}
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(io.LimitReader(resp.Body, 256*1024))
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("click merchant %s %s status %d: %s", method, url, resp.StatusCode, string(respBody))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (e *clickProviderExecutor) executeCheckoutCapture(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if token := strings.TrimSpace(req.CardToken); token != "" {
		raw, err := e.executeCardTokenPayment(ctx, token, req.AmountMinor, req.OrderID)
		if err != nil {
			return ExecutionResult{}, err
		}
		return ExecutionResult{
			ResolvedGateway: "CLICK",
			Mode:            ExecutionModeDirect,
			PolicySource:    "CLICK_MERCHANT",
			ProviderRef:     string(raw),
		}, nil
	}
	if phone := strings.TrimSpace(req.PayerPhone); phone != "" {
		return e.executeCreateInvoice(ctx, req, phone)
	}
	return ExecutionResult{}, &GatewayPolicyError{
		Code:             "payment_gateway_policy_violation",
		Message:          "click capture needs card token, payer phone (invoice), or SHOP Complete action=1",
		RequestedGateway: req.Gateway,
		ResolvedGateway:  "CLICK",
		PolicySource:     "ROUTER_CAPABILITY",
	}
}

func (e *clickProviderExecutor) executePartialReversal(ctx context.Context, paymentID string, amountMinor int64) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	if amountMinor <= 0 {
		return nil, fmt.Errorf("click partial reversal amount_minor must be > 0")
	}
	path := fmt.Sprintf(
		"%s/payment/partial_reversal/%s/%s/%s",
		e.baseURL(),
		e.serviceID,
		strings.TrimSpace(paymentID),
		clickMinorToSom(amountMinor),
	)
	return e.callMerchant(ctx, http.MethodDelete, path, nil)
}

func (e *clickProviderExecutor) executeOFDGet(ctx context.Context, paymentID string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	path := fmt.Sprintf("%s/payment/ofd_data/%s/%s", e.baseURL(), e.serviceID, strings.TrimSpace(paymentID))
	return e.callMerchant(ctx, http.MethodGet, path, nil)
}

func (e *clickProviderExecutor) executeOFDSubmitItems(ctx context.Context, paymentID string, items []clickOFDItem) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	if strings.TrimSpace(paymentID) == "" || len(items) == 0 {
		return nil, fmt.Errorf("click ofd submit_items needs payment_id and items — will not invent fiscal lines")
	}
	body, err := json.Marshal(map[string]any{
		"service_id": json.RawMessage(jsonNumberOrQuoted(e.serviceID)),
		"payment_id": json.RawMessage(jsonNumberOrQuoted(paymentID)),
		"items":      items,
	})
	if err != nil {
		return nil, err
	}
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/payment/ofd_data/submit_items", body)
}

func (e *clickProviderExecutor) executeOFDSubmitQR(ctx context.Context, paymentID, qrCodeURL string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	if strings.TrimSpace(paymentID) == "" || strings.TrimSpace(qrCodeURL) == "" {
		return nil, fmt.Errorf("click ofd submit_qrcode needs payment_id and qr_code_url from MY_SOLIQ — will not invent")
	}
	body, err := json.Marshal(map[string]any{
		"service_id":  json.RawMessage(jsonNumberOrQuoted(e.serviceID)),
		"payment_id":  json.RawMessage(jsonNumberOrQuoted(paymentID)),
		"qr_code_url": qrCodeURL,
	})
	if err != nil {
		return nil, err
	}
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/payment/ofd_data/submit_qrcode", body)
}

func (e *clickProviderExecutor) executeClickPassPayment(ctx context.Context, req ExecutionRequest, phone string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	if strings.TrimSpace(phone) == "" {
		return nil, fmt.Errorf("click_pass payment needs phone")
	}
	body := clickInvoiceJSON(e.serviceID, req.AmountMinor, phone, req.OrderID)
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/click_pass/payment", body)
}

func (e *clickProviderExecutor) executeClickPassConfirm(ctx context.Context, paymentID string) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	body := []byte(fmt.Sprintf(
		`{"service_id":%s,"payment_id":%s}`,
		jsonNumberOrQuoted(e.serviceID),
		jsonNumberOrQuoted(paymentID),
	))
	return e.callMerchant(ctx, http.MethodPost, e.baseURL()+"/click_pass/confirm", body)
}

func (e *clickProviderExecutor) executeClickPassEnableConfirmation(ctx context.Context) (json.RawMessage, error) {
	if !e.merchantAPIKeyed() {
		return nil, errClickUnkeyed()
	}
	path := fmt.Sprintf("%s/click_pass/confirmation/%s", e.baseURL(), e.serviceID)
	return e.callMerchant(ctx, http.MethodPut, path, nil)
}

// clickOFDItem is one fiscal line for Click OFD submit_items. PriceMinor is
// int64 (tiyin / UZS minor). Do not use float64.
type clickOFDItem struct {
	Name       string `json:"name"`
	PriceMinor int64  `json:"price"`
	Quantity   int64  `json:"quantity"`
	VATPercent int64  `json:"vat_percent,omitempty"`
	Package    string `json:"package_code,omitempty"`
	SPIC       string `json:"spic,omitempty"`
}
