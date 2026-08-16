package payment

import (
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	PaymentAcceptorSupplier  = "SUPPLIER"
	PaymentAcceptorWarehouse = "WAREHOUSE"

	DefaultPaymentMethod   = "CASH"
	DefaultCardGatewayName = "GLOBAL_PAY"
)

// GatewayPolicy is the resolved checkout policy for a supplier/warehouse pair.
type GatewayPolicy struct {
	Acceptor             string
	AllowedGateways      []string
	DefaultPaymentMethod string
	DefaultCardGateway   string
	PolicySource         string
}

// NormalizeGatewayPolicy ensures CASH is always allowed and picks a card default.
func NormalizeGatewayPolicy(acceptor string, gateways []string, policySource string) GatewayPolicy {
	acceptor = normalizePaymentAcceptor(acceptor)
	allowed := normalizeAllowedGateways(gateways)
	cardDefault := DefaultCardGatewayName
	for _, gateway := range allowed {
		if gateway != DefaultPaymentMethod {
			cardDefault = gateway
			break
		}
	}
	if policySource == "" {
		policySource = "SUPPLIER_DEFAULT"
	}
	return GatewayPolicy{
		Acceptor:             acceptor,
		AllowedGateways:      allowed,
		DefaultPaymentMethod: DefaultPaymentMethod,
		DefaultCardGateway:   cardDefault,
		PolicySource:         policySource,
	}
}

func normalizePaymentAcceptor(acceptor string) string {
	switch strings.ToUpper(strings.TrimSpace(acceptor)) {
	case PaymentAcceptorWarehouse:
		return PaymentAcceptorWarehouse
	default:
		return PaymentAcceptorSupplier
	}
}

func normalizeAllowedGateways(gateways []string) []string {
	seen := make(map[string]struct{}, len(gateways)+2)
	out := make([]string, 0, len(gateways)+2)
	add := func(gateway string) {
		gateway = strings.ToUpper(strings.TrimSpace(gateway))
		if gateway == "" {
			return
		}
		if _, ok := seen[gateway]; ok {
			return
		}
		seen[gateway] = struct{}{}
		out = append(out, gateway)
	}
	for _, gateway := range gateways {
		add(gateway)
	}
	if len(out) == 0 {
		add(DefaultPaymentMethod)
		add(DefaultCardGatewayName)
		return out
	}
	add(DefaultPaymentMethod)
	return out
}

// CardGateways returns non-cash gateways exposed to retailer card checkout.
func (p GatewayPolicy) CardGateways() []string {
	out := make([]string, 0, len(p.AllowedGateways))
	for _, gateway := range p.AllowedGateways {
		if gateway == DefaultPaymentMethod {
			continue
		}
		out = append(out, gateway)
	}
	if len(out) == 0 {
		return []string{DefaultCardGatewayName}
	}
	return out
}

// ResolveCardGateway picks the gateway for card checkout.
func (p GatewayPolicy) ResolveCardGateway(requested string) string {
	requested = strings.ToUpper(strings.TrimSpace(requested))
	if requested == "" {
		return p.DefaultCardGateway
	}
	return requested
}

// ValidateCardGateway rejects gateways outside the configured policy.
func (p GatewayPolicy) ValidateCardGateway(requested string) error {
	requested = strings.ToUpper(strings.TrimSpace(requested))
	if requested == "" {
		requested = p.DefaultCardGateway
	}
	if requested == DefaultPaymentMethod {
		return nil
	}
	for _, allowed := range p.AllowedGateways {
		if allowed == requested {
			return nil
		}
	}
	return &GatewayPolicyError{
		Code:             "payment_gateway_policy_violation",
		Message:          fmt.Sprintf("gateway %s is not enabled for this supplier", requested),
		RequestedGateway: requested,
		ResolvedGateway:  requested,
		PolicySource:     p.PolicySource,
	}
}

// applyPackToGatewayPolicy intersects supplier policy with the shipped pack PSP list (GS-M1).
func applyPackToGatewayPolicy(policy GatewayPolicy, pack auth.MarketPack) GatewayPolicy {
	filtered := make([]string, 0, len(policy.AllowedGateways))
	seen := map[string]struct{}{}
	for _, g := range policy.AllowedGateways {
		canon := auth.CanonicalPSP(g)
		if !auth.PackAllowsPSP(pack, canon) {
			continue
		}
		if _, ok := seen[canon]; ok {
			continue
		}
		seen[canon] = struct{}{}
		filtered = append(filtered, canon)
	}
	if len(filtered) == 0 {
		return GatewayPolicy{
			Acceptor:             policy.Acceptor,
			DefaultPaymentMethod: DefaultPaymentMethod,
			PolicySource:         "MARKET_PACK",
		}
	}
	out := NormalizeGatewayPolicy(policy.Acceptor, filtered, "MARKET_PACK")
	if !auth.PackAllowsPSP(pack, DefaultPaymentMethod) {
		kept := make([]string, 0, len(out.AllowedGateways))
		for _, g := range out.AllowedGateways {
			if g == DefaultPaymentMethod {
				continue
			}
			kept = append(kept, g)
		}
		out.AllowedGateways = kept
	}
	return out
}
