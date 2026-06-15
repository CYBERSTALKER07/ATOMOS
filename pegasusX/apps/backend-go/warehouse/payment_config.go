package warehouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

type paymentGatewayWire struct {
	GatewayName string `json:"gateway_name"`
	Provider    string `json:"provider"`
	IsActive    bool   `json:"is_active"`
	Mode        string `json:"mode"`
}

type warehousePaymentConfigResponse struct {
	Gateways          []paymentGatewayWire `json:"gateways"`
	SelectedGateways  []string             `json:"selected_gateways"`
	PaymentAcceptor   string               `json:"payment_acceptor,omitempty"`
	PaymentConfigID   string               `json:"payment_config_id,omitempty"`
}

type warehousePaymentConfigRequest struct {
	SelectedGateways []string `json:"selected_gateways"`
}

var allowedWarehouseGateways = map[string]struct{}{
	"GLOBAL_PAY": {},
	"ADYEN":      {},
	"AIRWALLEX":  {},
	"CASH":       {},
}

// HandleOpsPaymentConfig serves GET/POST /v1/warehouse/ops/payment-config.
func (s *Service) HandleOpsPaymentConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetOpsPaymentConfig(w, r)
	case http.MethodPost:
		s.handlePostOpsPaymentConfig(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleGetOpsPaymentConfig(w http.ResponseWriter, r *http.Request) {
	warehouseID := warehouseIDFromRequest(r)
	resp, err := s.loadPaymentConfigResponse(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_payment_config_failed"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) handlePostOpsPaymentConfig(w http.ResponseWriter, r *http.Request) {
	warehouseID := warehouseIDFromRequest(r)
	body, ok := readMutationBody(w, r, 16*1024)
	if !ok {
		return
	}
	if _, handled := s.guardMutationReplay(w, r, body); handled {
		return
	}

	var req warehousePaymentConfigRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if len(req.SelectedGateways) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "selected_gateways required"})
		return
	}
	for _, gateway := range req.SelectedGateways {
		if _, ok := allowedWarehouseGateways[strings.ToUpper(strings.TrimSpace(gateway))]; !ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": fmt.Sprintf("unknown gateway %q", gateway)})
			return
		}
	}

	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment_config_unavailable"})
		return
	}

	normalized := payment.NormalizeGatewayPolicy(payment.PaymentAcceptorWarehouse, req.SelectedGateways, "WAREHOUSE_CONFIG").AllowedGateways
	configID, err := s.upsertWarehousePaymentConfig(r.Context(), warehouseID, normalized)
	if err != nil {
		s.log.ErrorContext(r.Context(), "persist warehouse payment config failed", "warehouse_id", warehouseID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_payment_config_failed"})
		return
	}
	if key := idempotencyKeyFromRequest(r); key != "" {
		respBytes, _ := json.Marshal(map[string]any{
			"payment_config_id": configID,
			"selected_gateways": normalized,
		})
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	}

	resp, err := s.loadPaymentConfigResponse(r.Context(), warehouseID)
	if err != nil {
		writeJSON(w, http.StatusOK, warehousePaymentConfigResponse{
			PaymentConfigID:  configID,
			SelectedGateways: normalized,
			Gateways:         gatewaysToWire(normalized),
		})
		return
	}
	resp.PaymentConfigID = configID
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) loadPaymentConfigResponse(ctx context.Context, warehouseID string) (warehousePaymentConfigResponse, error) {
	warehouseID = strings.TrimSpace(warehouseID)
	if s.spannerClient == nil || warehouseID == "" {
		defaultGateways := payment.NormalizeGatewayPolicy(payment.PaymentAcceptorSupplier, nil, "SUPPLIER_DEFAULT").AllowedGateways
		return warehousePaymentConfigResponse{
			Gateways:         gatewaysToWire(defaultGateways),
			SelectedGateways: defaultGateways,
			PaymentAcceptor:  payment.PaymentAcceptorSupplier,
		}, nil
	}

	acceptor, supplierGateways, err := s.loadSupplierPaymentAcceptor(ctx)
	if err != nil {
		return warehousePaymentConfigResponse{}, err
	}

	gateways := supplierGateways
	configID := ""
	if normalizePaymentAcceptor(acceptor) == payment.PaymentAcceptorWarehouse {
		warehouseGateways, id, found, err := s.loadWarehousePaymentConfig(ctx, warehouseID)
		if err != nil {
			return warehousePaymentConfigResponse{}, err
		}
		if found {
			gateways = warehouseGateways
			configID = id
		}
	}
	if len(gateways) == 0 {
		gateways = payment.NormalizeGatewayPolicy(acceptor, nil, "SUPPLIER_DEFAULT").AllowedGateways
	}

	return warehousePaymentConfigResponse{
		Gateways:         gatewaysToWire(gateways),
		SelectedGateways: gateways,
		PaymentAcceptor:  normalizePaymentAcceptor(acceptor),
		PaymentConfigID:  configID,
	}, nil
}

func (s *Service) loadSupplierPaymentAcceptor(ctx context.Context) (string, []string, error) {
	row, err := s.spannerClient.Single().ReadRow(ctx, "SupplierProfiles", spanner.Key{s.supplierID}, []string{
		"PaymentAcceptor",
		"SelectedGatewaysJson",
	})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return payment.PaymentAcceptorSupplier, nil, nil
		}
		return "", nil, err
	}
	var acceptor spanner.NullString
	var gatewaysJSON []byte
	if err := row.Columns(&acceptor, &gatewaysJSON); err != nil {
		return "", nil, err
	}
	var gateways []string
	if len(gatewaysJSON) > 0 {
		_ = json.Unmarshal(gatewaysJSON, &gateways)
	}
	return acceptor.StringVal, gateways, nil
}

func (s *Service) loadWarehousePaymentConfig(ctx context.Context, warehouseID string) ([]string, string, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT PaymentConfigId, SelectedGatewaysJson
			FROM PaymentConfigs@{FORCE_INDEX=UQ_PaymentConfigs_ByWarehouse}
			WHERE WarehouseId = @warehouseId AND SupplierId = @supplierId
			LIMIT 1`,
		Params: map[string]any{
			"warehouseId": warehouseID,
			"supplierId":  s.supplierID,
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil, "", false, nil
		}
		return nil, "", false, err
	}
	var configID string
	var gatewaysJSON []byte
	if err := row.Columns(&configID, &gatewaysJSON); err != nil {
		return nil, "", false, err
	}
	var gateways []string
	if len(gatewaysJSON) > 0 {
		if err := json.Unmarshal(gatewaysJSON, &gateways); err != nil {
			return nil, "", false, err
		}
	}
	return gateways, configID, true, nil
}

func (s *Service) upsertWarehousePaymentConfig(ctx context.Context, warehouseID string, gateways []string) (string, error) {
	_, existingID, found, err := s.loadWarehousePaymentConfig(ctx, warehouseID)
	if err != nil {
		return "", err
	}
	configID := existingID
	if !found || configID == "" {
		configID = uuid.NewString()
	}
	gatewaysJSON, err := json.Marshal(gateways)
	if err != nil {
		return "", err
	}
	now := s.now()
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("PaymentConfigs", map[string]any{
				"PaymentConfigId":      configID,
				"SupplierId":           s.supplierID,
				"WarehouseId":          warehouseID,
				"SelectedGatewaysJson": gatewaysJSON,
				"CreatedAt":            now,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
			spanner.UpdateMap("Warehouses", map[string]any{
				"WarehouseId":     warehouseID,
				"SupplierId":      s.supplierID,
				"PaymentConfigId": configID,
				"UpdatedAt":       spanner.CommitTimestamp,
			}),
		}
		return txn.BufferWrite(mutations)
	})
	return configID, err
}

func gatewaysToWire(gateways []string) []paymentGatewayWire {
	out := make([]paymentGatewayWire, 0, len(gateways))
	for _, gateway := range gateways {
		gateway = strings.ToUpper(strings.TrimSpace(gateway))
		if gateway == "" {
			continue
		}
		mode := "LIVE"
		if gateway == "GLOBAL_PAY" {
			mode = "PRODUCTION"
		}
		out = append(out, paymentGatewayWire{
			GatewayName: gateway,
			Provider:    gateway,
			IsActive:    true,
			Mode:        mode,
		})
	}
	return out
}

func normalizePaymentAcceptor(acceptor string) string {
	switch strings.ToUpper(strings.TrimSpace(acceptor)) {
	case payment.PaymentAcceptorWarehouse:
		return payment.PaymentAcceptorWarehouse
	default:
		return payment.PaymentAcceptorSupplier
	}
}
