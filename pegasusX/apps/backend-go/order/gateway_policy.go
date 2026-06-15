package order

import "context"

// GatewayPolicyReader resolves allowed payment gateways for order events.
type GatewayPolicyReader interface {
	AllowedGateways(ctx context.Context, supplierID, warehouseID string) (gateways []string, acceptor string, err error)
}
