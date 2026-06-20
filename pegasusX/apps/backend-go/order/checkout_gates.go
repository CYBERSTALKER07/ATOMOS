package order

import (
	"strings"
	"time"
)

func (s *Service) resolveCheckoutPolicyOverride(live WarehouseOpsPolicy, token string) (string, *CheckoutPolicySnapshot) {
	token = strings.TrimSpace(token)
	if token == "" || strings.TrimSpace(s.jwtSecret) == "" {
		return "", nil
	}
	snap, err := ResolveCheckoutPolicyToken(s.jwtSecret, token, s.now())
	if err != nil {
		return "", nil
	}
	if snap.WarehouseID != "" && snap.WarehouseID != live.WarehouseID {
		return "", nil
	}
	effective := EffectiveWarehouseStockPolicy(live.DefaultOutOfStockPolicy, &snap)
	if effective == live.DefaultOutOfStockPolicy {
		return "", &snap
	}
	return effective, &snap
}

func checkOrderAcceptanceGate(policy WarehouseOpsPolicy, now time.Time) (bool, string, *time.Time, string) {
	open, label, nextOpen := EvaluateOrderAcceptance(policy.OperatingSchedule, now)
	if open {
		return true, label, nextOpen, ""
	}
	return false, label, nextOpen, orderAcceptanceClosedMessage(label)
}

func applyAcceptancePreviewFields(resp *CheckoutPreviewResponse, open bool, label string, nextOpen *time.Time) {
	resp.OrderAcceptanceOpen = open
	if label != "" {
		resp.OrderAcceptanceWindowLabel = label
	}
	if nextOpen != nil {
		ts := nextOpen.Format(time.RFC3339)
		resp.NextOrderAcceptanceAt = &ts
	}
}
