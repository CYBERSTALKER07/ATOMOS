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

	"github.com/pegasusx/pegasusx/apps/backend-go/pkg/circuit"
)

// paymeProviderExecutor is the outbound Subscribe API + hosted Merchant checkout
// adapter. It is compiled and tested but MUST NOT be registered on the live
// router (see NewProviderExecutionRouter). Launch path is Cash + Global Pay.
type paymeProviderExecutor struct {
	env           string
	merchantID    string
	key           string
	subscribeBase string
	httpClient    *http.Client
	breaker       *circuit.Breaker
}

func newPaymeProviderExecutor(merchantID, key, env string) *paymeProviderExecutor {
	return newPaymeProviderExecutorWithOptions(merchantID, key, env, "", nil)
}

func newPaymeProviderExecutorWithOptions(merchantID, key, env, subscribeBase string, breaker *circuit.Breaker) *paymeProviderExecutor {
	if env == "" {
		env = "dev"
	}
	return &paymeProviderExecutor{
		env:           strings.ToLower(env),
		merchantID:    strings.TrimSpace(merchantID),
		key:           strings.TrimSpace(key),
		subscribeBase: strings.TrimSpace(subscribeBase),
		breaker:       breaker,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

var errPaymeCredentialsMissing = fmt.Errorf("payme credentials missing (PAYME_MERCHANT_ID) — adapter is implemented but not wired")

func errPaymeUnkeyed() error {
	return fmt.Errorf("%w: %w", errPaymeCredentialsMissing, &GatewayPolicyError{
		Code:             "no_live_keys",
		Message:          "no_live_keys",
		RequestedGateway: "PAYME",
		ResolvedGateway:  "PAYME",
		PolicySource:     "PSP_CATALOG",
	})
}

func (e *paymeProviderExecutor) keyed() bool {
	return e != nil && strings.TrimSpace(e.merchantID) != ""
}

func (e *paymeProviderExecutor) subscribeKeyed() bool {
	return e.keyed() && strings.TrimSpace(e.key) != ""
}

func (e *paymeProviderExecutor) doHTTP(ctx context.Context, call func(context.Context) error) error {
	if e.breaker != nil {
		return e.breaker.Do(ctx, call)
	}
	return call(ctx)
}

func (e *paymeProviderExecutor) subscribeEndpoint() string {
	if e.subscribeBase != "" {
		return strings.TrimRight(e.subscribeBase, "/")
	}
	return paymeSubscribeBaseURL(e.env)
}

func (e *paymeProviderExecutor) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	switch req.Action {
	case ExecutionActionChargebackRecord, ExecutionActionChargebackReversal:
		return ExecutionResult{
			ResolvedGateway: "PAYME",
			Mode:            ExecutionModeDirect,
			PolicySource:    "SUPPLIER_DEFAULT",
		}, nil
	case ExecutionActionCheckoutInit, ExecutionActionCheckoutCapture, ExecutionActionStatusCheck, ExecutionActionRefund:
	default:
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          fmt.Sprintf("unsupported execution action %s for gateway PAYME", req.Action),
			RequestedGateway: req.Gateway,
			ResolvedGateway:  "PAYME",
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}
	if !e.keyed() {
		return ExecutionResult{}, errPaymeUnkeyed()
	}
	switch req.Action {
	case ExecutionActionCheckoutInit:
		return e.executeCheckoutInit(ctx, req)
	case ExecutionActionStatusCheck:
		return e.executeReceiptsCheck(ctx, req)
	case ExecutionActionRefund:
		return e.executeReceiptsCancel(ctx, req)
	case ExecutionActionCheckoutCapture:
		return e.executeReceiptsPay(ctx, req)
	default:
		return ExecutionResult{}, errPaymeUnkeyed()
	}
}

func (e *paymeProviderExecutor) executeCheckoutInit(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	_ = ctx
	url := paymeHostedCheckoutURL(
		paymeCheckoutBaseURL(e.env),
		e.merchantID,
		req.OrderID,
		req.AmountMinor,
		"",
	)
	ref := strings.TrimSpace(req.OrderID)
	if e.subscribeKeyed() {
		receiptID, err := e.receiptsCreate(ctx, req)
		if err != nil {
			return ExecutionResult{}, err
		}
		if receiptID != "" {
			ref = receiptID
		}
	}
	return ExecutionResult{
		ResolvedGateway: "PAYME",
		Mode:            ExecutionModeHostedRedirect,
		PolicySource:    "PAYME_MERCHANT",
		RedirectURL:     url,
		ProviderRef:     ref,
	}, nil
}

func (e *paymeProviderExecutor) receiptsCreate(ctx context.Context, req ExecutionRequest) (string, error) {
	params := map[string]any{
		"amount": req.AmountMinor,
		"account": map[string]string{
			"order_id": req.OrderID,
		},
	}
	raw, err := e.callSubscribe(ctx, "receipts.create", params)
	if err != nil {
		return "", err
	}
	var wrap struct {
		Result struct {
			Receipt struct {
				ID string `json:"_id"`
			} `json:"receipt"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &wrap)
	return strings.TrimSpace(wrap.Result.Receipt.ID), nil
}

func (e *paymeProviderExecutor) executeReceiptsCheck(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if !e.subscribeKeyed() {
		return ExecutionResult{}, errPaymeUnkeyed()
	}
	id := strings.TrimSpace(req.SessionID)
	if id == "" {
		id = strings.TrimSpace(req.OrderID)
	}
	raw, err := e.callSubscribe(ctx, "receipts.check", map[string]any{"id": id})
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		Result struct {
			State int `json:"state"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &wrap)
	return ExecutionResult{
		ResolvedGateway: "PAYME",
		Mode:            ExecutionModeDirect,
		PolicySource:    "PAYME_SUBSCRIBE",
		ProviderRef:     fmt.Sprintf("state:%d", wrap.Result.State),
	}, nil
}

func (e *paymeProviderExecutor) executeReceiptsCancel(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if !e.subscribeKeyed() {
		return ExecutionResult{}, errPaymeUnkeyed()
	}
	id := strings.TrimSpace(req.SessionID)
	if id == "" {
		id = strings.TrimSpace(req.OrderID)
	}
	raw, err := e.callSubscribe(ctx, "receipts.cancel", map[string]any{"id": id})
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		Result struct {
			Receipt struct {
				ID string `json:"_id"`
			} `json:"receipt"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &wrap)
	ref := strings.TrimSpace(wrap.Result.Receipt.ID)
	if ref == "" {
		ref = id
	}
	return ExecutionResult{
		ResolvedGateway: "PAYME",
		Mode:            ExecutionModeDirect,
		PolicySource:    "PAYME_SUBSCRIBE",
		ProviderRef:     ref,
	}, nil
}

func (e *paymeProviderExecutor) callSubscribe(ctx context.Context, method string, params any) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"id":     time.Now().UnixNano(),
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.subscribeEndpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Auth", e.merchantID+":"+e.key)

	var respBody []byte
	err = e.doHTTP(ctx, func(callCtx context.Context) error {
		resp, err := e.httpClient.Do(httpReq.WithContext(callCtx))
		if err != nil {
			return MarkRetryable(fmt.Errorf("payme subscribe request failed: %w", err))
		}
		defer resp.Body.Close()
		respBody, _ = io.ReadAll(io.LimitReader(resp.Body, 256*1024))
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("payme subscribe %s status %d: %s", method, resp.StatusCode, string(respBody))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var rpc struct {
		Error *paymeRPCError `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpc); err != nil {
		return nil, fmt.Errorf("payme subscribe decode: %w", err)
	}
	if rpc.Error != nil && rpc.Error.Code != 0 {
		return nil, fmt.Errorf("payme subscribe %s: %d %s", method, rpc.Error.Code, rpc.Error.Message.En)
	}
	return json.RawMessage(respBody), nil
}

func (e *paymeProviderExecutor) receiptsSend(ctx context.Context, receiptID, phone string) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "receipts.send", map[string]any{"id": receiptID, "phone": phone})
}

func (e *paymeProviderExecutor) receiptsGet(ctx context.Context, receiptID string) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "receipts.get", map[string]any{"id": receiptID})
}

func (e *paymeProviderExecutor) receiptsGetAll(ctx context.Context, from, to int64) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "receipts.get_all", map[string]any{"from": from, "to": to})
}

func (e *paymeProviderExecutor) executeReceiptsPay(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	if !e.subscribeKeyed() {
		return ExecutionResult{}, errPaymeUnkeyed()
	}
	token := strings.TrimSpace(req.CardToken)
	if token == "" {
		return ExecutionResult{}, &GatewayPolicyError{
			Code:             "payment_gateway_policy_violation",
			Message:          "payme receipts.pay needs card token (Subscribe cards.verify)",
			RequestedGateway: req.Gateway,
			ResolvedGateway:  "PAYME",
			PolicySource:     "ROUTER_CAPABILITY",
		}
	}
	id := strings.TrimSpace(req.SessionID)
	if id == "" {
		id = strings.TrimSpace(req.OrderID)
	}
	raw, err := e.callSubscribe(ctx, "receipts.pay", map[string]any{
		"id":    id,
		"token": token,
	})
	if err != nil {
		return ExecutionResult{}, err
	}
	var wrap struct {
		Result struct {
			Receipt struct {
				ID    string `json:"_id"`
				State int    `json:"state"`
			} `json:"receipt"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &wrap)
	ref := strings.TrimSpace(wrap.Result.Receipt.ID)
	if ref == "" {
		ref = id
	}
	return ExecutionResult{
		ResolvedGateway: "PAYME",
		Mode:            ExecutionModeDirect,
		PolicySource:    "PAYME_SUBSCRIBE",
		ProviderRef:     ref,
	}, nil
}

func (e *paymeProviderExecutor) cardsCreate(ctx context.Context, number, expire string, save bool) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	params := map[string]any{
		"card": map[string]any{
			"number": strings.TrimSpace(number),
			"expire": strings.TrimSpace(expire),
		},
		"save": save,
	}
	return e.callSubscribe(ctx, "cards.create", params)
}

func (e *paymeProviderExecutor) cardsGetVerifyCode(ctx context.Context, token string) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "cards.get_verify_code", map[string]any{"token": token})
}

func (e *paymeProviderExecutor) cardsVerify(ctx context.Context, token, code string) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "cards.verify", map[string]any{"token": token, "code": code})
}

func (e *paymeProviderExecutor) cardsCheck(ctx context.Context, token string) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "cards.check", map[string]any{"token": token})
}

func (e *paymeProviderExecutor) cardsRemove(ctx context.Context, token string) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	return e.callSubscribe(ctx, "cards.remove", map[string]any{"token": token})
}

func (e *paymeProviderExecutor) receiptsSetFiscalData(ctx context.Context, receiptID string, fiscalData map[string]any) (json.RawMessage, error) {
	if !e.subscribeKeyed() {
		return nil, errPaymeUnkeyed()
	}
	if strings.TrimSpace(receiptID) == "" || len(fiscalData) == 0 {
		return nil, fmt.Errorf("payme receipts.set_fiscal_data needs receipt id and fiscal_data from MY_SOLIQ — will not invent")
	}
	return e.callSubscribe(ctx, "receipts.set_fiscal_data", map[string]any{
		"id":          receiptID,
		"fiscal_data": fiscalData,
	})
}
