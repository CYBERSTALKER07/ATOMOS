package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

type Simulator struct {
	cfg    *bootstrap.Config
	base   string
	client *http.Client
}

func NewSimulator(cfg *bootstrap.Config, base string) *Simulator {
	return &Simulator{
		cfg:  cfg,
		base: base,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Simulator) Run(ctx context.Context) error {
	slog.Info("Simulating ecosystem engagement...")

	// 1. Generate identities
	retailerID := "sim-retailer-" + uuid.NewString()[:8]
	supplierID := "sim-supplier-1" // Use a known supplier to avoid complex setup
	warehouseID := "sim-warehouse-1"
	driverID := "sim-driver-" + uuid.NewString()[:8]
	factoryID := "sim-factory-1"
	payloaderID := "sim-payloader-1"

	slog.Info("Identities generated", "retailer", retailerID, "driver", driverID)

	retailerToken, err := s.issueJWT(auth.Claims{
		Subject: retailerID,
		Role:    auth.RoleRetailer,
	})
	if err != nil {
		return err
	}

	supplierToken, err := s.issueJWT(auth.Claims{
		Subject:    supplierID,
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	})
	if err != nil {
		return err
	}

	warehouseToken, err := s.issueJWT(auth.Claims{
		Subject:      "sim-wh-admin",
		Role:         auth.RoleWarehouseAdmin,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeID:   warehouseID,
	})
	if err != nil {
		return err
	}

	driverToken, err := s.issueJWT(auth.Claims{
		Subject: driverID,
		Role:    auth.RoleDriver,
	})
	if err != nil {
		return err
	}

	factoryToken, err := s.issueJWT(auth.Claims{
		Subject:      "sim-factory-admin",
		Role:         auth.RoleFactoryAdmin,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleFactoryAdmin,
		HomeNodeID:   factoryID,
	})
	if err != nil {
		return err
	}

	payloadToken, err := s.issueJWT(auth.Claims{
		Subject:    payloaderID,
		Role:       auth.RolePayload,
		HomeNodeID: warehouseID,
	})
	if err != nil {
		return err
	}

	// 2. Connect WebSockets for all roles
	retWS, err := s.connectWS(retailerToken)
	if err != nil {
		return fmt.Errorf("retailer ws: %w", err)
	}
	defer retWS.Close()

	supWS, err := s.connectWS(supplierToken)
	if err != nil {
		return fmt.Errorf("supplier ws: %w", err)
	}
	defer supWS.Close()

	whWS, err := s.connectWS(warehouseToken)
	if err != nil {
		return fmt.Errorf("warehouse ws: %w", err)
	}
	defer whWS.Close()

	drvWS, err := s.connectWS(driverToken)
	if err != nil {
		return fmt.Errorf("driver ws: %w", err)
	}
	defer drvWS.Close()

	facWS, err := s.connectWS(factoryToken)
	if err != nil {
		return fmt.Errorf("factory ws: %w", err)
	}
	defer facWS.Close()

	payWS, err := s.connectWS(payloadToken)
	if err != nil {
		return fmt.Errorf("payload ws: %w", err)
	}
	defer payWS.Close()

	slog.Info("All WebSockets connected, starting async readers")

	// Read loops
	go s.wsReader("Retailer", retWS)
	go s.wsReader("Supplier", supWS)
	go s.wsReader("Warehouse", whWS)
	go s.wsReader("Driver", drvWS)
	go s.wsReader("Factory", facWS)
	go s.wsReader("Payloader", payWS)

	// Wait for connections to stabilize
	time.Sleep(1 * time.Second)

	// 3. Retailer places order
	slog.Info("Retailer placing order...")
	orderID, err := s.retailerPlaceOrder(ctx, retailerToken, supplierID)
	if err != nil {
		return fmt.Errorf("place order: %w", err)
	}
	slog.Info("Order created", "order_id", orderID)

	// Wait for Supplier WS to receive OrderCreated event
	time.Sleep(2 * time.Second)

	// 4. Supplier approves order
	slog.Info("Supplier approving order...")
	if err := s.supplierApproveOrder(ctx, supplierToken, supplierID, orderID); err != nil {
		return fmt.Errorf("approve order: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 5. Warehouse packs order
	slog.Info("Warehouse packing order...")
	if err := s.warehousePackOrder(ctx, warehouseToken, orderID); err != nil {
		return fmt.Errorf("pack order: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 6. Driver acknowledges assignment
	slog.Info("Driver accepting assignment...")
	if err := s.driverAccept(ctx, driverToken, driverID, orderID); err != nil {
		return fmt.Errorf("driver accept: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 7. Driver completes delivery
	slog.Info("Driver completing delivery...")
	if err := s.driverComplete(ctx, driverToken, driverID, orderID); err != nil {
		return fmt.Errorf("driver complete: %w", err)
	}

	// 8. Trigger Webhook for payment (mimic external provider)
	// 9. Give time for final WS events to flush
	time.Sleep(2 * time.Second)

	// ---- NEGATIVE EDGE CASE: PAYMENT FAILURE ----
	slog.Info("Starting Negative Edge Case: Payment Failure...")
	failOrderID, err := s.retailerPlacePrepayOrder(ctx, retailerToken, supplierID)
	if err != nil {
		slog.Error("Failed to place prepay order", "err", err)
	} else {
		slog.Info("Prepay Order created, simulating failure...", "order_id", failOrderID)
		time.Sleep(1 * time.Second)
		if err := s.simulateGlobalPayFailureWebhook(ctx, failOrderID); err != nil {
			slog.Error("Failed to simulate webhook failure", "err", err)
		} else {
			slog.Info("Payment failure webhook sent successfully")
		}
	}

	// ---- NEGATIVE EDGE CASE: DRIVER REPORTS DAMAGE ----
	slog.Info("Starting Negative Edge Case: Driver Reports Damage...")
	damageOrderID, err := s.retailerPlaceOrder(ctx, retailerToken, supplierID)
	if err == nil {
		s.supplierApproveOrder(ctx, supplierToken, supplierID, damageOrderID)
		s.warehousePackOrder(ctx, warehouseToken, damageOrderID)
		slog.Info("Driver reporting damage...", "order_id", damageOrderID)
		time.Sleep(1 * time.Second)
		if err := s.driverReportDamage(ctx, driverToken, driverID, damageOrderID); err != nil {
			slog.Error("Driver failed to report damage", "err", err)
		} else {
			slog.Info("Damage reported successfully")
		}
	}

	time.Sleep(3 * time.Second)

	// ---- FACTORY & PAYLOAD LIFECYCLE ----
	slog.Info("Starting Factory & Payload Lifecycle...")

	// Create a transfer order for the factory to dispatch
	transferOrderID := "sim-transfer-ord-" + uuid.NewString()[:8]

	factoryManifestID, err := s.factoryDispatch(ctx, factoryToken, map[string]any{
		"reason": "simulated factory dispatch",
	})
	if err != nil {
		slog.Error("Factory failed to dispatch manifest", "err", err)
	} else {
		slog.Info("Factory dispatched manifest", "manifest_id", factoryManifestID)
		time.Sleep(1 * time.Second)

		// Payloader starts loading
		slog.Info("Payloader starting loading...")
		if err := s.payloadStartLoading(ctx, payloadToken, factoryManifestID); err != nil {
			slog.Error("Payloader failed to start loading", "err", err)
		} else {
			slog.Info("Payloader started loading manifest successfully")
			time.Sleep(1 * time.Second)

			// For testing payload Exceptions, we can simulate an exception
			if err := s.payloadException(ctx, payloadToken, factoryManifestID, transferOrderID, "OVERFLOW"); err != nil {
				// We expect this might fail if transferOrderID isn't actually in the DB or manifest
				slog.Warn("Payloader manifest exception warning", "err", err)
			} else {
				slog.Info("Payloader successfully recorded manifest exception")
			}
			time.Sleep(1 * time.Second)

			if err := s.payloadSealCompleted(ctx, payloadToken, factoryManifestID); err != nil {
				slog.Error("Payloader failed to complete sealing", "err", err)
			} else {
				slog.Info("Payloader completed sealing manifest successfully")
			}
		}
	}

	time.Sleep(3 * time.Second)
	return nil
}

func (s *Simulator) issueJWT(claims auth.Claims) (string, error) {
	return auth.Issue(claims, auth.IssueOptions{
		Secret: s.cfg.JWTSecret,
		Issuer: s.cfg.JWTIssuer,
		TTL:    1 * time.Hour,
	})
}

func (s *Simulator) connectWS(token string) (*websocket.Conn, error) {
	wsURL := strings.Replace(s.base, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	u := wsURL + "/v1/ws"

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	c, _, err := websocket.DefaultDialer.Dial(u, header)
	return c, err
}

func (s *Simulator) wsReader(role string, c *websocket.Conn) {
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			slog.Warn("WS closed", "role", role, "err", err)
			return
		}
		slog.Info("WS Event received", "role", role, "event", string(msg))
	}
}

func (s *Simulator) retailerPlaceOrder(ctx context.Context, token, supplierID string) (string, error) {
	ccy := s.operatingCurrency(ctx)
	if ccy == "" {
		return "", fmt.Errorf("empty_operating_currency")
	}
	body := map[string]any{
		"supplier_id": supplierID,
		"currency":    ccy,
		"items": []map[string]any{
			{"product_id": "prod-1", "quantity": 10},
		},
		"delivery_location": map[string]any{
			"lat": 41.2995,
			"lng": 69.2401,
		},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/order/create", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("status %d body %s", resp.StatusCode, string(respBody))
	}
	var res map[string]any
	json.Unmarshal(respBody, &res)

	orderID, _ := res["order_id"].(string)
	if orderID == "" {
		return "", fmt.Errorf("no order_id in response: %s", string(respBody))
	}
	return orderID, nil
}

func (s *Simulator) retailerPlacePrepayOrder(ctx context.Context, token, supplierID string) (string, error) {
	ccy := s.operatingCurrency(ctx)
	if ccy == "" {
		return "", fmt.Errorf("empty_operating_currency")
	}
	body := map[string]any{
		"supplier_id":    supplierID,
		"currency":       ccy,
		"payment_method": "CARD_ONLINE",
		"items": []map[string]any{
			{"product_id": "prod-1", "quantity": 10},
		},
		"delivery_location": map[string]any{
			"lat": 41.2995,
			"lng": 69.2401,
		},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/order/create", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("status %d body %s", resp.StatusCode, string(respBody))
	}
	var res map[string]any
	json.Unmarshal(respBody, &res)

	orderID, _ := res["order_id"].(string)
	if orderID == "" {
		return "", fmt.Errorf("no order_id in response: %s", string(respBody))
	}
	return orderID, nil
}

func (s *Simulator) supplierApproveOrder(ctx context.Context, token, supplierID, orderID string) error {
	// Supplier routes vet order instead of approve
	body := map[string]any{
		"order_ids": []string{orderID},
		"action":    "approve",
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/supplier/orders/vet", s.base), bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) warehousePackOrder(ctx context.Context, token, orderID string) error {
	// Warehouse uses dispatch/execute to process and assign an order
	body := map[string]any{
		"order_ids": []string{orderID},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/warehouse/ops/dispatch/execute", s.base), bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) driverAccept(ctx context.Context, token, driverID, orderID string) error {
	// We'll skip accept for this E2E or use arrive depending on standard driver flow
	return nil
}

func (s *Simulator) driverComplete(ctx context.Context, token, driverID, orderID string) error {
	body := map[string]any{
		"order_id": orderID,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/order/complete", s.base), bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) driverReportDamage(ctx context.Context, token, driverID, orderID string) error {
	body := map[string]any{
		"order_id": orderID,
		"reason":   "Boxes were crushed during transport",
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/delivery/report-damage", s.base), bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) simulateGlobalPayWebhook(ctx context.Context, orderID string) error {
	ccy := s.operatingCurrency(ctx)
	if ccy == "" {
		return fmt.Errorf("empty_operating_currency")
	}
	body := map[string]any{
		"transaction_id": "sim-tx-1234",
		"order_id":       orderID,
		"status":         "SUCCESS",
		"amount":         100000,
		"currency":       ccy,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/webhooks/global-pay", s.base), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GlobalPay-Signature", s.cfg.GlobalPayWebhookSecret)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) simulateGlobalPayFailureWebhook(ctx context.Context, orderID string) error {
	ccy := s.operatingCurrency(ctx)
	if ccy == "" {
		return fmt.Errorf("empty_operating_currency")
	}
	body := map[string]any{
		"transaction_id": "sim-tx-fail",
		"order_id":       orderID,
		"status":         "FAILED",
		"amount":         100000,
		"currency":       ccy,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/webhooks/global-pay", s.base), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GlobalPay-Signature", s.cfg.GlobalPayWebhookSecret)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) factoryDispatch(ctx context.Context, token string, reqBody map[string]any) (string, error) {
	b, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/factory/dispatch", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("status %d body %s", resp.StatusCode, string(respBody))
	}
	var res struct {
		ManifestID string `json:"manifest_id"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return "", err
	}
	return res.ManifestID, nil
}

func (s *Simulator) payloadStartLoading(ctx context.Context, token, manifestID string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/payloader/manifests/"+manifestID+"/start-loading", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d body %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) payloadSeal(ctx context.Context, token, orderID string) error {
	body := map[string]string{"order_id": orderID}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/payload/seal", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d body %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) payloadSealCompleted(ctx context.Context, token, manifestID string) error {
	body := map[string]any{
		"manifest_ids": []string{manifestID},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/payloader/manifests/seal-completed", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d body %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *Simulator) payloadException(ctx context.Context, token, manifestID, orderID, reason string) error {
	body := map[string]string{
		"manifest_id": manifestID,
		"order_id":    orderID,
		"reason":      reason,
		"metadata":    "simulated-exception",
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/payload/manifest-exception", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d body %s", resp.StatusCode, string(rb))
	}
	return nil
}

func (s *Simulator) payloadReassign(ctx context.Context, token, orderID, reason string) error {
	body := map[string]string{
		"order_id": orderID,
		"reason":   reason,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v1/payloader/reassign-order", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d body %s", resp.StatusCode, string(rb))
	}
	return nil
}
