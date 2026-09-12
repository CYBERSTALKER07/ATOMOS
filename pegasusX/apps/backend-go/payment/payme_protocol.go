package payment

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Payme Merchant API (Payme → merchant) and Subscribe API (merchant → Payme).
// Amounts are tiyin int64 — same as SessionRecord.AmountMinor for UZS.
// Official: https://developer.help.paycom.uz/metody-merchant-api/
//
// These types and helpers are the full adapter. Live checkout still uses
// catalogHonestyExecutorFor("PAYME") until an explicit wire decision.

const (
	paymeMethodCheckPerform = "CheckPerformTransaction"
	paymeMethodCreate       = "CreateTransaction"
	paymeMethodPerform      = "PerformTransaction"
	paymeMethodCancel       = "CancelTransaction"
	paymeMethodCheck        = "CheckTransaction"
	paymeMethodStatement    = "GetStatement"
	paymeMethodSetFiscal    = "SetFiscalData"

	paymeStateCreated   = 1
	paymeStatePerformed = 2
	paymeStateCancelled = -1
	paymeStateReverted  = -2

	paymeErrParse          = -32700
	paymeErrInvalidRequest = -32600
	paymeErrMethod         = -32601
	paymeErrInvalidParams  = -32602
	paymeErrInvalidAmount  = -31001
	paymeErrNotFound       = -31003
	paymeErrCannotCancel   = -31007
	paymeErrCannotPerform  = -31008
	paymeErrAccount        = -31050
	paymeErrOrderState     = -31099
	paymeErrFiscalNotFound = -32001
)

type paymeRPCError struct {
	Code    int             `json:"code"`
	Message paymeRPCMessage `json:"message"`
	Data    string          `json:"data,omitempty"`
}

type paymeRPCMessage struct {
	Ru string `json:"ru"`
	Uz string `json:"uz"`
	En string `json:"en"`
}

func paymeError(code int, en, data string) *paymeRPCError {
	return &paymeRPCError{
		Code: code,
		Message: paymeRPCMessage{
			En: en,
			Ru: en,
			Uz: en,
		},
		Data: data,
	}
}

type paymeRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      any             `json:"id"`
	Params  json.RawMessage `json:"params"`
}

type paymeMerchantParams struct {
	PaymeID    string
	Time       int64
	Amount     int64
	OrderID    string
	Account    map[string]string
	Reason     int
	HasReason  bool
	From       int64
	To         int64
	Type       string
	FiscalData json.RawMessage
}

func parsePaymeMerchantParams(raw json.RawMessage) (paymeMerchantParams, *paymeRPCError) {
	var out paymeMerchantParams
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return out, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return out, paymeError(paymeErrParse, "parse error", "params")
	}
	out.PaymeID = jsonValueString(m["id"])
	if out.PaymeID == "" {
		out.PaymeID = jsonValueString(m["transaction"])
	}
	out.Time = jsonValueInt64(m["time"])
	out.Amount = jsonValueInt64(m["amount"])
	out.From = jsonValueInt64(m["from"])
	out.To = jsonValueInt64(m["to"])
	if _, ok := m["reason"]; ok {
		out.HasReason = true
		out.Reason = int(jsonValueInt64(m["reason"]))
	}
	out.Account = map[string]string{}
	if acc, ok := m["account"].(map[string]any); ok {
		for k, v := range acc {
			out.Account[k] = jsonValueString(v)
		}
	}
	out.OrderID = strings.TrimSpace(out.Account["order_id"])
	if out.OrderID == "" {
		out.OrderID = jsonValueString(m["order_id"])
	}
	out.Type = jsonValueString(m["type"])
	if fd, ok := m["fiscal_data"]; ok && fd != nil {
		raw, err := json.Marshal(fd)
		if err == nil {
			out.FiscalData = raw
		}
	}
	return out, nil
}

func jsonValueString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return strings.TrimSpace(t.String())
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func jsonValueInt64(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0
		}
		return n
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

func paymeCheckoutBaseURL(env string) string {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod", "live":
		return "https://checkout.paycom.uz"
	default:
		return "https://test.paycom.uz"
	}
}

func paymeSubscribeBaseURL(env string) string {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod", "live":
		return "https://checkout.paycom.uz/api"
	default:
		return "https://checkout.test.paycom.uz/api"
	}
}

// paymeHostedCheckoutURL builds the Merchant API checkout link
// (m=merchant;ac.order_id=…;a=tiyin). Payme then calls our Merchant methods.
func paymeHostedCheckoutURL(checkoutBase, merchantID, orderID string, amountMinor int64, returnURL string) string {
	var b strings.Builder
	b.WriteString("m=")
	b.WriteString(strings.TrimSpace(merchantID))
	b.WriteString(";ac.order_id=")
	b.WriteString(strings.TrimSpace(orderID))
	b.WriteString(";a=")
	b.WriteString(strconv.FormatInt(amountMinor, 10))
	if strings.TrimSpace(returnURL) != "" {
		b.WriteString(";c=")
		b.WriteString(strings.TrimSpace(returnURL))
	}
	enc := base64.StdEncoding.EncodeToString([]byte(b.String()))
	return strings.TrimRight(checkoutBase, "/") + "/" + enc
}

func paymeStatusFromState(state int) string {
	switch state {
	case paymeStatePerformed:
		return "PAID"
	case paymeStateCancelled, paymeStateReverted:
		return "FAILED"
	default:
		return "PENDING"
	}
}
