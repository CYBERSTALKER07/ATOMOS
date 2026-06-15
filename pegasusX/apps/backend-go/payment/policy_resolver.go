package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// PolicyResolver resolves allowed gateways for checkout.
type PolicyResolver interface {
	Resolve(ctx context.Context, supplierID, warehouseID string) (GatewayPolicy, error)
}

// SpannerPolicyResolver reads supplier and warehouse payment configuration from Spanner.
type SpannerPolicyResolver struct {
	client *spanner.Client
}

// NewSpannerPolicyResolver constructs a policy resolver backed by Spanner.
func NewSpannerPolicyResolver(client *spanner.Client) *SpannerPolicyResolver {
	return &SpannerPolicyResolver{client: client}
}

// Resolve implements PolicyResolver.
func (r *SpannerPolicyResolver) Resolve(ctx context.Context, supplierID, warehouseID string) (GatewayPolicy, error) {
	if r == nil || r.client == nil {
		return NormalizeGatewayPolicy(PaymentAcceptorSupplier, nil, "SUPPLIER_DEFAULT"), nil
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return GatewayPolicy{}, fmt.Errorf("supplier_id required")
	}

	acceptor, gateways, err := r.loadSupplierPolicy(ctx, supplierID)
	if err != nil {
		return GatewayPolicy{}, err
	}
	policySource := "SUPPLIER_DEFAULT"
	if normalizePaymentAcceptor(acceptor) == PaymentAcceptorWarehouse {
		warehouseID = strings.TrimSpace(warehouseID)
		if warehouseID != "" {
			warehouseGateways, found, err := r.loadWarehouseGateways(ctx, supplierID, warehouseID)
			if err != nil {
				return GatewayPolicy{}, err
			}
			if found && len(warehouseGateways) > 0 {
				gateways = warehouseGateways
				policySource = "WAREHOUSE_CONFIG"
			}
		}
	}
	return NormalizeGatewayPolicy(acceptor, gateways, policySource), nil
}

// AllowedGateways exposes a narrow read surface for order event enrichment.
func (r *SpannerPolicyResolver) AllowedGateways(ctx context.Context, supplierID, warehouseID string) ([]string, string, error) {
	policy, err := r.Resolve(ctx, supplierID, warehouseID)
	if err != nil {
		return nil, "", err
	}
	return policy.AllowedGateways, policy.Acceptor, nil
}

func (r *SpannerPolicyResolver) loadSupplierPolicy(ctx context.Context, supplierID string) (string, []string, error) {
	row, err := r.client.Single().ReadRow(ctx, "SupplierProfiles", spanner.Key{supplierID}, []string{
		"PaymentAcceptor",
		"SelectedGatewaysJson",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return PaymentAcceptorSupplier, nil, nil
		}
		return "", nil, fmt.Errorf("read supplier payment policy: %w", err)
	}
	var acceptor spanner.NullString
	var gatewaysJSON []byte
	if err := row.Columns(&acceptor, &gatewaysJSON); err != nil {
		return "", nil, fmt.Errorf("decode supplier payment policy: %w", err)
	}
	gateways, err := decodeGatewaySlice(gatewaysJSON)
	if err != nil {
		return "", nil, err
	}
	return acceptor.StringVal, gateways, nil
}

func (r *SpannerPolicyResolver) loadWarehouseGateways(ctx context.Context, supplierID, warehouseID string) ([]string, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SelectedGatewaysJson
			FROM PaymentConfigs@{FORCE_INDEX=UQ_PaymentConfigs_ByWarehouse}
			WHERE WarehouseId = @warehouseId AND SupplierId = @supplierId
			LIMIT 1`,
		Params: map[string]any{
			"warehouseId": warehouseID,
			"supplierId":  supplierID,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read warehouse payment config: %w", err)
	}
	var gatewaysJSON []byte
	if err := row.Columns(&gatewaysJSON); err != nil {
		return nil, false, fmt.Errorf("decode warehouse payment config: %w", err)
	}
	gateways, err := decodeGatewaySlice(gatewaysJSON)
	if err != nil {
		return nil, false, err
	}
	return gateways, true, nil
}

func decodeGatewaySlice(payload []byte) ([]string, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var gateways []string
	if err := json.Unmarshal(payload, &gateways); err != nil {
		return nil, fmt.Errorf("decode gateway list: %w", err)
	}
	return gateways, nil
}

func encodeGatewaySlice(gateways []string) ([]byte, error) {
	return json.Marshal(normalizeAllowedGateways(gateways))
}
