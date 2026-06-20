package order

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const checkoutPolicyTokenTTL = 15 * time.Minute

// CheckoutPolicySnapshot is embedded in a signed checkout_policy_token.
type CheckoutPolicySnapshot struct {
	WarehouseID          string `json:"warehouse_id"`
	StockPolicy          string `json:"stock_policy"`
	OrderLineMin         *int64 `json:"order_line_min,omitempty"`
	OrderLineMax         *int64 `json:"order_line_max,omitempty"`
	AcceptanceWindowHash string `json:"acceptance_window_hash,omitempty"`
	ExpiresAt            int64  `json:"exp"`
}

// IssueCheckoutPolicyToken signs a snapshot for preview→submit grace.
func IssueCheckoutPolicyToken(secret string, snap CheckoutPolicySnapshot, now time.Time) (token string, expiresAt time.Time, err error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", time.Time{}, errors.New("checkout policy token: missing secret")
	}
	if strings.TrimSpace(snap.WarehouseID) == "" {
		return "", time.Time{}, errors.New("checkout policy token: warehouse_id required")
	}
	expiresAt = now.Add(checkoutPolicyTokenTTL)
	snap.ExpiresAt = expiresAt.Unix()
	payload, err := json.Marshal(snap)
	if err != nil {
		return "", time.Time{}, err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token = base64.RawURLEncoding.EncodeToString(payload) + "." + sig
	return token, expiresAt, nil
}

// ResolveCheckoutPolicyToken verifies and decodes a checkout policy token.
func ResolveCheckoutPolicyToken(secret, token string, now time.Time) (CheckoutPolicySnapshot, error) {
	secret = strings.TrimSpace(secret)
	token = strings.TrimSpace(token)
	if secret == "" || token == "" {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_invalid")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_invalid")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_invalid")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_invalid")
	}
	var snap CheckoutPolicySnapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_invalid")
	}
	if snap.ExpiresAt <= 0 || now.Unix() > snap.ExpiresAt {
		return CheckoutPolicySnapshot{}, errors.New("checkout_policy_token_expired")
	}
	return snap, nil
}

// EffectiveWarehouseStockPolicy picks live vs snapshotted stock policy (honor more permissive snapshot).
func EffectiveWarehouseStockPolicy(live string, snap *CheckoutPolicySnapshot) string {
	liveNorm := normalizeStockPolicy(live)
	if snap == nil {
		return liveNorm
	}
	snapNorm := normalizeStockPolicy(snap.StockPolicy)
	if snapNorm == outOfStockPolicyAcceptBackorder && liveNorm == outOfStockPolicyReject {
		return outOfStockPolicyAcceptBackorder
	}
	if liveNorm == outOfStockPolicyAcceptBackorder {
		return outOfStockPolicyAcceptBackorder
	}
	return liveNorm
}

func normalizeStockPolicy(p string) string {
	p = strings.ToUpper(strings.TrimSpace(p))
	if p == outOfStockPolicyAcceptBackorder {
		return outOfStockPolicyAcceptBackorder
	}
	return outOfStockPolicyReject
}

func snapshotFromWarehousePolicy(wh WarehouseOpsPolicy, sched OperatingSchedule) CheckoutPolicySnapshot {
	return CheckoutPolicySnapshot{
		WarehouseID:          wh.WarehouseID,
		StockPolicy:          wh.DefaultOutOfStockPolicy,
		OrderLineMin:         wh.OrderLineMinQuantity,
		OrderLineMax:         wh.OrderLineMaxQuantity,
		AcceptanceWindowHash: AcceptanceWindowHash(sched),
	}
}

func orderAcceptanceClosedMessage(label string) string {
	if label == "" {
		return "This supplier is not accepting orders at this time."
	}
	return fmt.Sprintf("This supplier accepts orders only from %s.", label)
}
