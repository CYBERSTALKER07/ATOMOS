// Package order — Unified Checkout with Cart Fan-Out (Phase 1)
//
// POST /v1/checkout/unified
//
// The retailer sees ONE cart. The backend shatters it into isolated supplier
// fragments inside a single ACID Spanner transaction, writes durable outbox
// events in the same transaction, then the relay publishes to Kafka after commit.
package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/cart"
	"backend-go/dispatch"
	apierrors "backend-go/errors"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/telemetry"
	"backend-go/workers"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ─── Request / Response Types ─────────────────────────────────────────────────

// UnifiedCheckoutRequest is the single payload from the native app's checkout.
type UnifiedCheckoutRequest struct {
	RetailerID     string               `json:"retailer_id"`
	PaymentGateway string               `json:"payment_gateway"` // optional client hint; backend resolves the effective gateway
	Latitude       float64              `json:"latitude"`
	Longitude      float64              `json:"longitude"`
	Items          []cart.OrderLineItem `json:"items"`
}

// SupplierOrderResult represents one supplier's slice of the unified checkout.
type SupplierOrderResult struct {
	OrderID       string `json:"order_id"`
	SupplierID    string `json:"supplier_id"`
	SupplierName  string `json:"supplier_name"`
	WarehouseID   string `json:"warehouse_id,omitempty"`
	WarehouseName string `json:"warehouse_name,omitempty"`
	Total         int64  `json:"total"`
	Currency      string `json:"currency"`
	ItemCount     int    `json:"item_count"`
}

// UnifiedCheckoutResponse is returned on HTTP 201.
type UnifiedCheckoutResponse struct {
	Status               string                `json:"status"`
	InvoiceID            string                `json:"invoice_id"`
	Total                int64                 `json:"total"`
	Currency             string                `json:"currency"`
	SupplierOrders       []SupplierOrderResult `json:"supplier_orders"`
	BackorderedItemCount int                   `json:"backordered_item_count,omitempty"`
}

// ─── Internal: supplier metadata resolved from Spanner ────────────────────────

type supplierMeta struct {
	SupplierID   string
	SupplierName string
	BasePrice    int64
}

// ─── Handler ──────────────────────────────────────────────────────────────────

// HandleUnifiedCheckout handles POST /v1/checkout/unified.
//
// Flow:
//  1. Decode + validate the request.
//  2. Resolve SupplierId for every SKU via SupplierProducts table.
//  3. Group line items by SupplierId.
//  4. Price each supplier group via the B2B pricing engine.
//  5. Single Spanner ReadWriteTransaction:
//     a. INSERT MasterInvoice
//     b. For each supplier group: INSERT Order + INSERT N OrderLineItems
//  6. Inside transaction: write per-supplier outbox ORDER_CREATED events.
//  7. Outbox relay publishes committed events to Kafka.
func (s *OrderService) HandleUnifiedCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierrors.MethodNotAllowed(w, r)
		return
	}

	var req UnifiedCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, r, "Malformed checkout payload")
		return
	}
	if claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims); ok && claims != nil {
		if req.RetailerID != "" && req.RetailerID != claims.UserID {
			log.Printf("[UNIFIED_CHECKOUT] retailer_id mismatch: payload=%s, claims=%s", req.RetailerID, claims.UserID)
			http.Error(w, `{"error":"RetailerID mismatch. Do not trust request body."}`, http.StatusForbidden)
			return
		}
		req.RetailerID = claims.UserID
	}

	// ── Validation ────────────────────────────────────────────────────────────
	if req.RetailerID == "" {
		http.Error(w, `{"error":"retailer_id is required"}`, http.StatusUnprocessableEntity)
		return
	}
	if len(req.Items) == 0 {
		http.Error(w, `{"error":"items must not be empty"}`, http.StatusUnprocessableEntity)
		return
	}
	for _, item := range req.Items {
		if item.SkuId == "" || item.Quantity <= 0 {
			http.Error(w, `{"error":"each item must have sku_id and positive quantity"}`, http.StatusUnprocessableEntity)
			return
		}
	}

	ctx := r.Context()

	var retailerTIN string // fetched for fiscalization

	// ── Step 0: Resolve retailer coordinates & TIN ────────────────────────────
	// Client may send 0,0 (e.g. older app versions). Fall back to the stored
	// Retailers table coordinates so the proximity resolver always has input.
	retailerLat, retailerLng := req.Latitude, req.Longitude
	{
		row, errR := s.Client.Single().ReadRow(ctx, "Retailers",
			spanner.Key{req.RetailerID},
			[]string{"Latitude", "Longitude", "TaxIdentificationNumber"},
		)
		if errR == nil {
			var lat, lng spanner.NullFloat64
			var tin spanner.NullString
			if errC := row.Columns(&lat, &lng, &tin); errC == nil {
				if lat.Valid && lng.Valid {
					retailerLat, retailerLng = lat.Float64, lng.Float64
				}
				if tin.Valid {
					retailerTIN = tin.StringVal
				}
			}
		}
	}

	// ── Step 1: Resolve supplier ownership for every SKU ──────────────────────
	skuIDs := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		skuIDs = append(skuIDs, item.SkuId)
	}

	supplierBySku, err := s.resolveSuppliers(ctx, skuIDs)
	if err != nil {
		log.Printf("[UNIFIED_CHECKOUT] Supplier resolution failed: %v", err)
		apierrors.InternalError(w, r, "Failed to resolve product suppliers.")
		return
	}

	// Verify every SKU has a supplier
	for _, item := range req.Items {
		if _, ok := supplierBySku[item.SkuId]; !ok {
			http.Error(w, fmt.Sprintf(`{"error":"SKU %s has no supplier assignment"}`, item.SkuId), http.StatusUnprocessableEntity)
			return
		}
	}

	// ── Step 2: Group items by SupplierID ─────────────────────────────────────
	supplierGroups := make(map[string][]cart.OrderLineItem)
	for _, item := range req.Items {
		sid := supplierBySku[item.SkuId].SupplierID
		supplierGroups[sid] = append(supplierGroups[sid], item)
	}

	supplierIDs := make([]string, 0, len(supplierGroups))
	for supplierID := range supplierGroups {
		supplierIDs = append(supplierIDs, supplierID)
	}
	resolvedGateway, gatewayErr := s.resolveCheckoutGateway(ctx, supplierIDs, req.PaymentGateway)
	if gatewayErr != nil {
		var policyErr *ErrGatewayPolicy
		if errors.As(gatewayErr, &policyErr) {
			http.Error(w, fmt.Sprintf(`{"error":%s}`, mustJSON(policyErr.Error())), http.StatusUnprocessableEntity)
			return
		}
		log.Printf("[UNIFIED_CHECKOUT] Gateway resolution failed: %v", gatewayErr)
		apierrors.InternalError(w, r, "Failed to resolve payment gateway policy.")
		return
	}
	req.PaymentGateway = resolvedGateway

	// ── Step 3: Price each supplier group (with per-retailer overrides) ───
	supplierTotals := make(map[string]int64, len(supplierGroups))
	supplierCurrencyByID := make(map[string]string, len(supplierGroups))
	invoiceCurrency := ""

	for sid, items := range supplierGroups {
		var groupTotal int64
		for i, item := range items {
			basePrice := item.UnitPrice
			if basePrice <= 0 {
				basePrice = supplierBySku[item.SkuId].BasePrice
			}
			if basePrice <= 0 {
				http.Error(w, fmt.Sprintf(`{"error":"SKU %s has no base price"}`, item.SkuId), http.StatusUnprocessableEntity)
				return
			}
			effectivePrice, isOverride, priceErr := cart.ResolveRetailerPrice(
				ctx, s.Client, sid, req.RetailerID, item.SkuId, basePrice)
			if priceErr != nil {
				log.Printf("[UNIFIED_CHECKOUT] Price resolution failed for %s/%s: %v — using base", sid, item.SkuId, priceErr)
				effectivePrice = basePrice
			}
			items[i].UnitPrice = effectivePrice
			_ = isOverride // tracked for future analytics
			groupTotal += effectivePrice * item.Quantity
		}
		supplierTotals[sid] = groupTotal
		supplierGroups[sid] = items

		supplierCurrency := s.resolveSupplierCurrency(ctx, sid)
		supplierCurrencyByID[sid] = supplierCurrency
		if invoiceCurrency == "" {
			invoiceCurrency = supplierCurrency
		} else if invoiceCurrency != supplierCurrency {
			http.Error(w, fmt.Sprintf(`{"error":"MIXED_SUPPLIER_CURRENCIES_NOT_SUPPORTED","invoice_currency":"%s","supplier_currency":"%s","supplier_id":"%s"}`,
				invoiceCurrency, supplierCurrency, sid), http.StatusUnprocessableEntity)
			return
		}
	}
	if invoiceCurrency == "" {
		invoiceCurrency = "UZS"
	}

	// ── Step 3a: Stock pre-flight (read-only, non-blocking) ───────────────
	// Quick read-only check to give an early 409 for total OOS scenarios.
	// The authoritative lock still happens inside the ReadWriteTransaction.
	{
		oosItems := make([]string, 0)
		for _, item := range req.Items {
			row, readErr := s.Client.Single().ReadRow(ctx, "SupplierInventory",
				spanner.Key{item.SkuId}, []string{"QuantityAvailable"})
			if readErr != nil {
				continue // missing inventory record — will be caught in transaction
			}
			var avail int64
			if colErr := row.Columns(&avail); colErr == nil && avail <= 0 {
				oosItems = append(oosItems, item.SkuId)
			}
		}
		if len(oosItems) == len(req.Items) {
			// ALL items are OOS — fast-reject.
			// Publish best-effort notification via the canonical emitter path.
			workers.EventPool.Submit(func() {
				shortMap := make(map[string]int64, len(req.Items))
				for _, item := range req.Items {
					shortMap[item.SkuId] = item.Quantity
				}
				event := kafkaEvents.OutOfStockEvent{
					WarehouseId:  "",
					SupplierId:   supplierBySku[req.Items[0].SkuId].SupplierID,
					RetailerID:   req.RetailerID,
					ShortfallMap: shortMap,
					Timestamp:    time.Now(),
				}
				kafkaEvents.EmitNotification(kafkaEvents.EventOutOfStock, event)
			})
			http.Error(w, `{"error":"ALL_ITEMS_OUT_OF_STOCK","oos_skus":`+mustJSON(oosItems)+`}`, http.StatusConflict)
			return
		}
	}

	// ── Step 3b: Resolve nearest warehouse per supplier ───────────────────────
	// Uses the 3-tier proximity resolver (Grid cell → Redis GEOSEARCH → Spanner).
	// No-coverage returns nil — the order proceeds with empty WarehouseId.
	warehouseBySupplier := make(map[string]*proximity.WarehouseMatch, len(supplierGroups))
	for sid := range supplierGroups {
		match, whErr := proximity.ResolveWarehouseWithRouter(ctx, s.Client, s.ReadRouter, sid, retailerLat, retailerLng)
		if whErr != nil {
			log.Printf("[UNIFIED_CHECKOUT] Warehouse resolution failed for supplier %s: %v — proceeding without warehouse", sid, whErr)
		}
		if match != nil {
			warehouseBySupplier[sid] = match
		}
	}

	// ── Step 4: Single ACID Spanner transaction ───────────────────────────────
	invoiceID := hotspot.NewInvoiceID()
	wkt := fmt.Sprintf("POINT(%f %f)", retailerLng, retailerLat)

	// Pre-generate all order IDs and delivery tokens BEFORE the transaction
	// so we can reference them in the Kafka events after commit.
	type supplierOrderPlan struct {
		OrderID       string
		SupplierID    string
		SupplierName  string
		Currency      string
		WarehouseID   string
		WarehouseName string
		Total         int64
		Items         []cart.OrderLineItem
	}
	type supplierPayoutPolicySnapshot struct {
		PayoutMode       string
		FeePolicyVersion string
	}

	plans := make([]supplierOrderPlan, 0, len(supplierGroups))
	planSupplierIDs := make([]string, 0, len(supplierGroups))
	for sid, items := range supplierGroups {
		var whID, whName string
		if m := warehouseBySupplier[sid]; m != nil {
			whID = m.WarehouseId
			whName = m.Name
		}
		planSupplierIDs = append(planSupplierIDs, sid)
		plans = append(plans, supplierOrderPlan{
			OrderID:       hotspot.NewOrderID(),
			SupplierID:    sid,
			SupplierName:  supplierBySku[items[0].SkuId].SupplierName,
			Currency:      supplierCurrencyByID[sid],
			WarehouseID:   whID,
			WarehouseName: whName,
			Total:         supplierTotals[sid],
			Items:         items,
		})
	}

	warehouseIDs := make([]string, 0, len(plans))
	for _, plan := range plans {
		warehouseIDs = append(warehouseIDs, plan.WarehouseID)
	}
	regionalFeeDefaultsBySupplier, regionalFeeDefaultsErr := s.resolveRegionalDegressiveFeeDefaults(ctx, planSupplierIDs)
	if regionalFeeDefaultsErr != nil {
		log.Printf("[UNIFIED_CHECKOUT] regional fee defaults unavailable; falling back to legacy fee policy: %v", regionalFeeDefaultsErr)
		regionalFeeDefaultsBySupplier = nil
	}
	invoiceSettlementTarget := settlementTargetForWarehouseIDs(warehouseIDs)

	// Partial-fill state: populated inside the transaction, read after commit.
	effectiveQtyBySku := make(map[string]int64, len(req.Items))
	shortfallBySku := make(map[string]int64, len(req.Items))
	processedPlans := make([]supplierOrderPlan, 0, len(plans))
	processedTotals := make(map[string]int64, len(plans)) // OrderID → effective total
	var effectiveGrandTotal int64

	_, err = s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// ── INVENTORY LOCK: Read + validate stock inside the transaction ──────
		// Spanner serializes concurrent ReadWriteTransactions that touch the
		// same SupplierInventory row. If two retailers race for the last pallet,
		// the loser's transaction aborts instantly.
		allSkuIDs := make([]string, 0, len(req.Items))
		skuQtyRequested := make(map[string]int64, len(req.Items))
		for _, item := range req.Items {
			allSkuIDs = append(allSkuIDs, item.SkuId)
			skuQtyRequested[item.SkuId] += item.Quantity
		}

		// Read current inventory inside the transaction (acquires locks)
		inventoryKeys := make([]spanner.KeySet, 0, len(allSkuIDs))
		for _, sku := range allSkuIDs {
			inventoryKeys = append(inventoryKeys, spanner.Key{sku})
		}
		invIter := txn.Read(ctx, "SupplierInventory",
			spanner.KeySets(inventoryKeys...),
			[]string{"ProductId", "QuantityAvailable"},
		)
		defer invIter.Stop()

		currentStock := make(map[string]int64, len(allSkuIDs))
		for {
			row, iterErr := invIter.Next()
			if iterErr == iterator.Done {
				break
			}
			if iterErr != nil {
				return fmt.Errorf("inventory read failed: %w", iterErr)
			}
			var productID string
			var qty int64
			if colErr := row.Columns(&productID, &qty); colErr != nil {
				return fmt.Errorf("inventory row parse failed: %w", colErr)
			}
			currentStock[productID] = qty
		}

		type inventoryV2Key struct {
			supplierID  string
			warehouseID string
			productID   string
		}
		v2Keys := make(map[inventoryV2Key]struct{})
		v2Stock := make(map[inventoryV2Key]int64)
		for _, plan := range plans {
			if plan.WarehouseID == "" {
				continue
			}
			for _, item := range plan.Items {
				v2Keys[inventoryV2Key{supplierID: plan.SupplierID, warehouseID: plan.WarehouseID, productID: item.SkuId}] = struct{}{}
			}
		}
		if len(v2Keys) > 0 {
			v2KeySets := make([]spanner.KeySet, 0, len(v2Keys))
			for key := range v2Keys {
				v2KeySets = append(v2KeySets, spanner.Key{key.supplierID, key.warehouseID, key.productID})
			}
			invV2Iter := txn.Read(ctx, "SupplierInventoryV2",
				spanner.KeySets(v2KeySets...),
				[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityAvailable"},
			)
			defer invV2Iter.Stop()
			for {
				row, iterErr := invV2Iter.Next()
				if iterErr == iterator.Done {
					break
				}
				if iterErr != nil {
					return fmt.Errorf("inventory v2 read failed: %w", iterErr)
				}
				var supplierID, warehouseID, productID string
				var qty int64
				if colErr := row.Columns(&supplierID, &warehouseID, &productID, &qty); colErr != nil {
					return fmt.Errorf("inventory v2 row parse failed: %w", colErr)
				}
				v2Stock[inventoryV2Key{supplierID: supplierID, warehouseID: warehouseID, productID: productID}] = qty
			}
		}

		supplierPolicies := make(map[string]supplierPayoutPolicySnapshot, len(supplierGroups))
		supplierIDs := make([]string, 0, len(supplierGroups))
		for sid := range supplierGroups {
			supplierIDs = append(supplierIDs, sid)
			supplierPolicies[sid] = supplierPayoutPolicySnapshot{
				PayoutMode:       PayoutModeHQSupplier,
				FeePolicyVersion: FeePolicyVersionLegacyCheckout,
			}
		}
		if len(supplierIDs) > 0 {
			policyIter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT SupplierId, PayoutMode, FeePolicyVersion
				      FROM SupplierPayoutPolicies
				      WHERE SupplierId IN UNNEST(@supplierIds)
				        AND IsActive = TRUE`,
				Params: map[string]interface{}{"supplierIds": supplierIDs},
			})
			defer policyIter.Stop()

			for {
				row, iterErr := policyIter.Next()
				if iterErr == iterator.Done {
					break
				}
				if iterErr != nil {
					lowerErr := strings.ToLower(iterErr.Error())
					if strings.Contains(lowerErr, "supplierpayoutpolicies") && strings.Contains(lowerErr, "not found") {
						log.Printf("[UNIFIED_CHECKOUT] SupplierPayoutPolicies unavailable; defaulting to HQ payout mode: %v", iterErr)
						break
					}
					return fmt.Errorf("supplier payout policy lookup failed: %w", iterErr)
				}

				var supplierID string
				var payoutMode string
				var feePolicyVersion spanner.NullString
				if err := row.Columns(&supplierID, &payoutMode, &feePolicyVersion); err != nil {
					return fmt.Errorf("supplier payout policy row parse failed: %w", err)
				}

				supplierPolicies[supplierID] = supplierPayoutPolicySnapshot{
					PayoutMode:       normalizePayoutMode(payoutMode),
					FeePolicyVersion: normalizeFeePolicyVersion(feePolicyVersion.StringVal),
				}
			}
		}

		// ── Partial-fill: cap each SKU to available stock ───────────────────
		// Shortfall quantities become BACKORDERED orders in a second transaction.
		for sku, requested := range skuQtyRequested {
			available := currentStock[sku] // 0 if no inventory record
			if available >= requested {
				effectiveQtyBySku[sku] = requested
			} else {
				effectiveQtyBySku[sku] = available
				shortfallBySku[sku] = requested - available
			}
		}

		// ── Build mutations (inventory decrement + orders) ───────────────────
		mutations := make([]*spanner.Mutation, 0, len(allSkuIDs)+1+len(plans)*2)

		// Decrement inventory only for effective (filled) quantities
		for sku, effective := range effectiveQtyBySku {
			if effective == 0 {
				continue
			}
			mutations = append(mutations, spanner.Update("SupplierInventory",
				[]string{"ProductId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{sku, currentStock[sku] - effective, spanner.CommitTimestamp},
			))
		}

		for key := range v2Keys {
			effective := effectiveQtyBySku[key.productID]
			if effective == 0 {
				continue
			}
			baseline, ok := v2Stock[key]
			if !ok {
				baseline = currentStock[key.productID]
			}
			nextQty := baseline - effective
			if nextQty < 0 {
				nextQty = 0
			}
			mutations = append(mutations, spanner.InsertOrUpdate("SupplierInventoryV2",
				[]string{"SupplierId", "WarehouseId", "ProductId", "QuantityAvailable", "UpdatedAt"},
				[]interface{}{key.supplierID, key.warehouseID, key.productID, nextQty, spanner.CommitTimestamp},
			))
			v2Stock[key] = nextQty
		}

		// Compute effective grand total from capped quantities
		for _, plan := range plans {
			for _, item := range plan.Items {
				effectiveGrandTotal += effectiveQtyBySku[item.SkuId] * item.UnitPrice
			}
		}

		// 4a. INSERT MasterInvoice (uses effective total)
		mutations = append(mutations, spanner.Insert("MasterInvoices",
			[]string{"InvoiceId", "RetailerId", "Total", "Currency", "SettlementTarget", "State", "CreatedAt"},
			[]interface{}{invoiceID, req.RetailerID, effectiveGrandTotal, invoiceCurrency, invoiceSettlementTarget, "PENDING", spanner.CommitTimestamp},
		))

		feeBasisPoints := s.feeBasisPoints()
		invoiceFeePolicyVersion := FeePolicyVersionLegacyCheckout
		var invoiceFeeAmount int64
		var invoiceNetPayoutAmount int64
		var settlementSliceCount int64
		validatedWarehouseGateway := make(map[string]struct{})

		ensureWarehouseLocalGateway := func(supplierID, warehouseID string) error {
			trimmedWarehouseID := strings.TrimSpace(warehouseID)
			if trimmedWarehouseID == "" {
				return fmt.Errorf("warehouse-local payout requires warehouse assignment for supplier %s", supplierID)
			}

			cacheKey := supplierID + "|" + trimmedWarehouseID + "|" + req.PaymentGateway
			if _, ok := validatedWarehouseGateway[cacheKey]; ok {
				return nil
			}

			warehouseRow, warehouseErr := txn.ReadRow(ctx, "Warehouses", spanner.Key{trimmedWarehouseID}, []string{"SupplierId", "PaymentConfigId"})
			if warehouseErr != nil {
				return fmt.Errorf("warehouse-local payout config lookup failed for warehouse %s: %w", trimmedWarehouseID, warehouseErr)
			}

			var ownerSupplierID string
			var paymentConfigID spanner.NullString
			if err := warehouseRow.Columns(&ownerSupplierID, &paymentConfigID); err != nil {
				return fmt.Errorf("warehouse-local payout config row parse failed for warehouse %s: %w", trimmedWarehouseID, err)
			}
			if ownerSupplierID != supplierID {
				return fmt.Errorf("warehouse %s does not belong to supplier %s", trimmedWarehouseID, supplierID)
			}
			if !paymentConfigID.Valid || strings.TrimSpace(paymentConfigID.StringVal) == "" {
				return fmt.Errorf("warehouse-local payout requires active payment config for warehouse %s", trimmedWarehouseID)
			}

			cfgIter := txn.Query(ctx, spanner.Statement{
				SQL: `SELECT ConfigId, MerchantId, SecretKey
				      FROM SupplierPaymentConfigs
				      WHERE ConfigId = @configId
				        AND SupplierId = @supplierId
				        AND GatewayName = @gateway
				        AND IsActive = TRUE
				      LIMIT 1`,
				Params: map[string]interface{}{
					"configId":   paymentConfigID.StringVal,
					"supplierId": supplierID,
					"gateway":    req.PaymentGateway,
				},
			})
			defer cfgIter.Stop()

			cfgRow, cfgErr := cfgIter.Next()
			if cfgErr == iterator.Done {
				return fmt.Errorf("warehouse-local payout requires active %s credentials for warehouse %s", req.PaymentGateway, trimmedWarehouseID)
			}
			if cfgErr != nil {
				return fmt.Errorf("warehouse-local gateway credential lookup failed for warehouse %s: %w", trimmedWarehouseID, cfgErr)
			}

			var configID string
			var merchantID string
			var secretKey []byte
			if err := cfgRow.Columns(&configID, &merchantID, &secretKey); err != nil {
				return fmt.Errorf("warehouse-local gateway credential parse failed for warehouse %s: %w", trimmedWarehouseID, err)
			}
			if strings.TrimSpace(merchantID) == "" || len(secretKey) == 0 {
				return fmt.Errorf("warehouse-local gateway credentials incomplete for warehouse %s", trimmedWarehouseID)
			}

			validatedWarehouseGateway[cacheKey] = struct{}{}
			return nil
		}

		// 4b. For each supplier group: INSERT Order + INSERT N OrderLineItems (effective qty only)
		for _, plan := range plans {
			// Compute effective supplier total and check if any items are fulfillable
			var planEffectiveTotal int64
			var hasEffectiveItems bool
			for _, item := range plan.Items {
				eq := effectiveQtyBySku[item.SkuId]
				if eq > 0 {
					planEffectiveTotal += eq * item.UnitPrice
					hasEffectiveItems = true
				}
			}
			if !hasEffectiveItems {
				continue // All items for this supplier are backordered — skip primary order
			}

			policy := supplierPolicies[plan.SupplierID]
			payoutMode := normalizePayoutMode(policy.PayoutMode)
			feePolicyVersion := normalizeFeePolicyVersion(policy.FeePolicyVersion)
			var regionalFeeDefaults *regionalDegressiveFeeDefaults
			if defaults, ok := regionalFeeDefaultsBySupplier[plan.SupplierID]; ok {
				regionalFeeDefaults = &defaults
			}
			feeComputation := computeCheckoutFee(planEffectiveTotal, plan.Currency, feeBasisPoints, feePolicyVersion, regionalFeeDefaults)
			sliceSettlementTarget := settlementTargetForWarehouseID(plan.WarehouseID)
			payoutOwnerType := PayoutOwnerTypeSupplier
			payoutOwnerID := plan.SupplierID

			if payoutMode == PayoutModeWarehouseLocal {
				if err := ensureWarehouseLocalGateway(plan.SupplierID, plan.WarehouseID); err != nil {
					return err
				}
				payoutOwnerType = PayoutOwnerTypeWarehouse
				payoutOwnerID = plan.WarehouseID
				sliceSettlementTarget = SettlementTargetLocalWarehouse
			}

			if strings.TrimSpace(payoutOwnerID) == "" {
				return fmt.Errorf("missing payout owner id for supplier %s", plan.SupplierID)
			}

			invoiceFeeAmount += feeComputation.FeeAmount
			invoiceNetPayoutAmount += feeComputation.NetPayoutAmount
			settlementSliceCount++
			if settlementSliceCount == 1 {
				invoiceFeePolicyVersion = feeComputation.PolicyVersion
			} else if invoiceFeePolicyVersion != feeComputation.PolicyVersion {
				invoiceFeePolicyVersion = FeePolicyVersionMixed
			}

			sliceID := hotspot.NewOrderID()
			mutations = append(mutations, spanner.Insert("InvoiceSettlementSlices",
				[]string{
					"InvoiceId", "SliceId", "SupplierId", "WarehouseId",
					"SettlementTarget", "PayoutOwnerType", "PayoutOwnerId",
					"GrossAmount", "Currency", "FeePolicyVersion", "SelectedTierKey",
					"FeeBasisPoints", "FeeCapApplied", "FeeAmount", "NetPayoutAmount",
					"CreatedAt", "UpdatedAt",
				},
				[]interface{}{
					invoiceID, sliceID, plan.SupplierID, spanner.NullString{StringVal: plan.WarehouseID, Valid: strings.TrimSpace(plan.WarehouseID) != ""},
					sliceSettlementTarget, payoutOwnerType, payoutOwnerID,
					planEffectiveTotal, plan.Currency, feeComputation.PolicyVersion, feeComputation.SelectedTierKey,
					feeComputation.BasisPoints, feeComputation.CapApplied, feeComputation.FeeAmount, feeComputation.NetPayoutAmount,
					spanner.CommitTimestamp, spanner.CommitTimestamp,
				},
			))

			processedPlans = append(processedPlans, plan)
			processedTotals[plan.OrderID] = planEffectiveTotal

			// Phase 2 dispatch optimisation: pre-compute VolumeVU at INSERT so the
			// optimiser does not need a JOIN to OrderLineItems on the hot path.
			// SKU-level VolumePerUnit is not yet a column; default 1.0 VU/unit.
			planQuantities := make([]int, 0, len(plan.Items))
			planVolumes := make([]float64, 0, len(plan.Items))
			for _, item := range plan.Items {
				eq := effectiveQtyBySku[item.SkuId]
				if eq == 0 {
					continue
				}
				planQuantities = append(planQuantities, int(eq))
				planVolumes = append(planVolumes, dispatch.DefaultUnitVolumeVU)
			}
			planVolumeVU := dispatch.ComputeOrderVolume(planQuantities, planVolumes)

			mutations = append(mutations, spanner.Insert("Orders",
				[]string{
					"OrderId", "RetailerId", "SupplierId", "InvoiceId",
					"Amount", "Currency", "PaymentGateway", "State", "ScheduleShard",
					"ShopLocation", "OrderSource", "DeliveryToken", "WarehouseId",
					"VolumeVU", "CreatedAt",
				},
				[]interface{}{
					plan.OrderID, req.RetailerID, plan.SupplierID, invoiceID,
					planEffectiveTotal, plan.Currency, req.PaymentGateway, "PENDING", hotspot.ShardForKey(plan.OrderID),
					wkt, "UNIFIED_CHECKOUT",
					spanner.NullString{Valid: false}, // JIT: token generated at dispatch, not checkout
					plan.WarehouseID,
					planVolumeVU,
					spanner.CommitTimestamp,
				},
			))

			for _, item := range plan.Items {
				eq := effectiveQtyBySku[item.SkuId]
				if eq == 0 {
					continue // SKU fully backordered — not in primary order
				}
				lineItemID := fmt.Sprintf("LI-%s", GenerateSecureToken())
				mutations = append(mutations, spanner.Insert("OrderLineItems",
					[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Currency", "Status"},
					[]interface{}{lineItemID, plan.OrderID, item.SkuId, eq, item.UnitPrice, plan.Currency, "PENDING"},
				))
			}
		}

		if settlementSliceCount == 0 {
			invoiceFeePolicyVersion = FeePolicyVersionLegacyCheckout
		}
		mutations = append(mutations, spanner.Update("MasterInvoices",
			[]string{"InvoiceId", "FeePolicyVersion", "FeeAmount", "NetPayoutAmount", "SettlementSliceCount"},
			[]interface{}{invoiceID, invoiceFeePolicyVersion, invoiceFeeAmount, invoiceNetPayoutAmount, settlementSliceCount},
		))

		// ── Spanner mutation count guard ─────────────────────────────────────
		// Spanner enforces a 20,000 mutation limit per transaction. Each mutation
		// row * column counts, but we guard at the row level conservatively.
		const maxMutationRows = 5000
		if len(mutations) > maxMutationRows {
			return fmt.Errorf("checkout too large: %d mutations exceeds safety limit of %d — split into smaller orders", len(mutations), maxMutationRows)
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return err
		}

		now := time.Now().UTC()
		for _, plan := range processedPlans {
			// Temporary Resolver Logic for Unified Checkout
			terminalID := resolveTerminalForWarehouse(ctx, plan.WarehouseID, "globalpay")

			event := kafkaEvents.OrderCreatedEvent{
				InvoiceID:     invoiceID,
				OrderID:       plan.OrderID,
				SupplierID:    plan.SupplierID,
				RetailerID:    req.RetailerID,
				WarehouseID:   plan.WarehouseID,
				WarehouseName: plan.WarehouseName,
				TIN:           retailerTIN,
				TerminalID:    terminalID,
				Total:         processedTotals[plan.OrderID],
				Currency:      plan.Currency,
				Items:         plan.Items,
				Timestamp:     now,
			}
			if err := outbox.EmitJSON(txn, "Order", plan.OrderID, string(kafkaEvents.EventOrderCreated), kafkaEvents.TopicMain, event, telemetry.TraceIDFromContext(ctx)); err != nil {
				return err
			}
		}
		return outbox.EmitJSON(txn, "Invoice", invoiceID, string(kafkaEvents.EventUnifiedCheckoutCompleted), kafkaEvents.TopicMain, kafkaEvents.UnifiedCheckoutCompletedEvent{
			InvoiceID:  invoiceID,
			RetailerID: req.RetailerID,
			Total:      effectiveGrandTotal,
			Currency:   invoiceCurrency,
			OrderCount: len(processedPlans),
			Timestamp:  now,
		}, telemetry.TraceIDFromContext(ctx))
	})

	if err != nil {
		log.Printf("[UNIFIED_CHECKOUT] Spanner transaction failed: %v", err)

		// New Target: Inject OrderValidationFailed for UI feedback
		// Must be isolated since main checkout txn failed.
		_, _ = s.Client.ReadWriteTransaction(ctx, func(innerCtx context.Context, errTxn *spanner.ReadWriteTransaction) error {
			return outbox.EmitJSON(errTxn, "Order", invoiceID, kafkaEvents.EventOrderValidationFailed, kafkaEvents.TopicMain, kafkaEvents.OrderValidationFailedEvent{
				OrderID:    invoiceID,
				InvoiceID:  invoiceID,
				RetailerID: req.RetailerID,
				Reason:     "INVENTORY_LOCK_OR_CAPACITY_FAILED",
				Timestamp:  time.Now().UTC(),
			}, telemetry.TraceIDFromContext(ctx))
		})

		apierrors.WriteOperational(w, r, apierrors.ProblemDetail{
			Type:       "error/order/checkout-failed",
			Title:      "Checkout Failed",
			Status:     http.StatusInternalServerError,
			Detail:     "Checkout transaction failed. Please try again.",
			Code:       apierrors.CodeSpannerAborted,
			MessageKey: apierrors.MsgKeyInternalError,
			Retryable:  true,
			Action:     apierrors.ActionRetry,
		})
		return
	}

	// ── Step 5a: BACKORDERED second transaction (shortfall quantities) ─────────
	var totalBackorderedItems int
	for _, qty := range shortfallBySku {
		totalBackorderedItems += int(qty)
	}

	if totalBackorderedItems > 0 {
		// Build one BACKORDERED order per supplier that has shortfall
		type backorderPlan struct {
			OrderID       string
			SupplierID    string
			Currency      string
			WarehouseID   string
			WarehouseName string
			Total         int64
			Items         []cart.OrderLineItem
		}
		backordersBySup := make(map[string]*backorderPlan)
		for _, plan := range plans {
			for _, item := range plan.Items {
				shortfall := shortfallBySku[item.SkuId]
				if shortfall == 0 {
					continue
				}
				bp, ok := backordersBySup[plan.SupplierID]
				if !ok {
					bp = &backorderPlan{
						OrderID:       hotspot.NewOrderID(),
						SupplierID:    plan.SupplierID,
						Currency:      plan.Currency,
						WarehouseID:   plan.WarehouseID,
						WarehouseName: plan.WarehouseName,
					}
					backordersBySup[plan.SupplierID] = bp
				}
				bp.Items = append(bp.Items, cart.OrderLineItem{
					SkuId:     item.SkuId,
					Quantity:  shortfall,
					UnitPrice: item.UnitPrice,
				})
				bp.Total += shortfall * item.UnitPrice
			}
		}

		_, backErr := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			var bMutations []*spanner.Mutation
			for _, bp := range backordersBySup {
				bpQuantities := make([]int, 0, len(bp.Items))
				bpVolumes := make([]float64, 0, len(bp.Items))
				for _, item := range bp.Items {
					bpQuantities = append(bpQuantities, int(item.Quantity))
					bpVolumes = append(bpVolumes, dispatch.DefaultUnitVolumeVU)
				}
				bpVolumeVU := dispatch.ComputeOrderVolume(bpQuantities, bpVolumes)

				bMutations = append(bMutations, spanner.Insert("Orders",
					[]string{
						"OrderId", "RetailerId", "SupplierId", "InvoiceId",
						"Amount", "Currency", "PaymentGateway", "State", "ScheduleShard",
						"ShopLocation", "OrderSource", "DeliveryToken", "WarehouseId",
						"VolumeVU", "CreatedAt",
					},
					[]interface{}{
						bp.OrderID, req.RetailerID, bp.SupplierID, invoiceID,
						bp.Total, bp.Currency, req.PaymentGateway, "BACKORDERED", hotspot.ShardForKey(bp.OrderID),
						wkt, "UNIFIED_CHECKOUT",
						spanner.NullString{Valid: false},
						bp.WarehouseID,
						bpVolumeVU,
						spanner.CommitTimestamp,
					},
				))
				for _, item := range bp.Items {
					liID := fmt.Sprintf("LI-%s", GenerateSecureToken())
					bMutations = append(bMutations, spanner.Insert("OrderLineItems",
						[]string{"LineItemId", "OrderId", "SkuId", "Quantity", "UnitPrice", "Currency", "Status"},
						[]interface{}{liID, bp.OrderID, item.SkuId, item.Quantity, item.UnitPrice, bp.Currency, "PENDING"},
					))
				}
			}
			if err := txn.BufferWrite(bMutations); err != nil {
				return err
			}
			now := time.Now().UTC()
			for _, bp := range backordersBySup {
				event := kafkaEvents.StockBackorderedEvent{
					InvoiceID:         invoiceID,
					BackOrderID:       bp.OrderID,
					BackOrderLegacyID: bp.OrderID,
					SupplierID:        bp.SupplierID,
					RetailerID:        req.RetailerID,
					WarehouseID:       bp.WarehouseID,
					WarehouseName:     bp.WarehouseName,
					Items:             bp.Items,
					Total:             bp.Total,
					Currency:          bp.Currency,
					Timestamp:         now,
				}
				if err := outbox.EmitJSON(txn, "Order", bp.OrderID, string(kafkaEvents.EventStockBackordered), kafkaEvents.TopicMain, event, telemetry.TraceIDFromContext(ctx)); err != nil {
					return err
				}
			}
			return nil
		})
		if backErr != nil {
			// Primary order already committed — log and continue
			log.Printf("[UNIFIED_CHECKOUT] BACKORDERED transaction failed (primary ok): %v", backErr)
		}
	}

	// ── CACHE INVALIDATION After Commit ────────────────────────────────────────
	s.Cache.Invalidate(ctx, cache.PrefixActiveOrders+req.RetailerID)

	log.Printf("[UNIFIED_CHECKOUT] InvoiceID=%s RetailerId=%s Total=%d SupplierOrders=%d BackorderedItems=%d",
		invoiceID, req.RetailerID, effectiveGrandTotal, len(processedPlans), totalBackorderedItems)

	// ── Step 5c: Authorize GLOBAL_PAY orders (saved card → hold funds at checkout) ──
	// Best-effort: if authorization fails, orders remain PENDING and fall back to
	// delivery-time payment via TriggerSupplierFulfillmentPayment / CardCheckout.
	if req.PaymentGateway == "GLOBAL_PAY" && s.DirectClient != nil && s.CardTokenSvc != nil {
		savedCard, _ := s.CardTokenSvc.GetDefaultCard(ctx, req.RetailerID, "GLOBAL_PAY")
		if savedCard != nil {
			for _, plan := range processedPlans {
				orderTotal := processedTotals[plan.OrderID]
				if orderTotal <= 0 {
					continue
				}
				if authErr := s.authorizeAtCheckout(ctx, plan.OrderID, plan.SupplierID, req.RetailerID,
					invoiceID, orderTotal, savedCard.ProviderCardToken); authErr != nil {
					log.Printf("[UNIFIED_CHECKOUT] GP authorization failed for order %s (non-fatal, delivery fallback): %v",
						plan.OrderID, authErr)
				} else {
					log.Printf("[UNIFIED_CHECKOUT] GP authorization succeeded for order %s — amount %d held",
						plan.OrderID, orderTotal)
				}
			}
		}
	}

	// ── Step 6: HTTP 201 Response ─────────────────────────────────────────────
	supplierResults := make([]SupplierOrderResult, 0, len(processedPlans))
	for _, plan := range processedPlans {
		supplierResults = append(supplierResults, SupplierOrderResult{
			OrderID:       plan.OrderID,
			SupplierID:    plan.SupplierID,
			SupplierName:  plan.SupplierName,
			WarehouseID:   plan.WarehouseID,
			WarehouseName: plan.WarehouseName,
			Total:         processedTotals[plan.OrderID],
			Currency:      plan.Currency,
			ItemCount:     len(plan.Items),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(UnifiedCheckoutResponse{
		Status:               "CHECKOUT_LOCKED",
		InvoiceID:            invoiceID,
		Total:                effectiveGrandTotal,
		Currency:             invoiceCurrency,
		SupplierOrders:       supplierResults,
		BackorderedItemCount: totalBackorderedItems,
	}); err != nil {
		log.Printf("[UNIFIED_CHECKOUT] Response encode error: %v", err)
	}
}

// authorizeAtCheckout places a Global Pay authorization hold on a single order
// using the retailer's saved card token. On success it creates a PaymentSession
// with status AUTHORIZED and updates Orders.PaymentStatus = AUTHORIZED.
// Failure is non-fatal — callers log and continue (delivery-time fallback).
func (s *OrderService) authorizeAtCheckout(ctx context.Context, orderID, supplierID, retailerID, invoiceID string, amount int64, cardToken string) error {
	// Resolve per-supplier GP credentials.
	var merchantID, serviceID, secretKey, recipientID string
	if s.Vault != nil {
		cfg, vaultErr := s.Vault.GetDecryptedConfigByOrder(ctx, orderID, "GLOBAL_PAY")
		if vaultErr == nil {
			merchantID = cfg.MerchantId
			serviceID = cfg.ServiceId
			secretKey = cfg.SecretKey
			recipientID = cfg.RecipientId
		}
	}
	creds, credErr := payment.ResolveGlobalPayCredentials(merchantID, serviceID, secretKey)
	if credErr != nil {
		return fmt.Errorf("credential resolution: %w", credErr)
	}

	splitRecipients := payment.ComputeSplitRecipients(amount, recipientID, s.feeBasisPoints())
	if snapshot, ok, snapErr := payment.LoadSupplierSettlementSnapshot(ctx, s.Client, orderID, supplierID); snapErr != nil {
		log.Printf("[UNIFIED_CHECKOUT] snapshot lookup failed for order %s supplier %s: %v", orderID, supplierID, snapErr)
	} else if ok {
		effectiveAmount := snapshot.GrossAmount
		if effectiveAmount <= 0 {
			effectiveAmount = amount
		}
		splitRecipients = payment.ComputeSplitRecipientsWithFeeAmount(effectiveAmount, recipientID, snapshot.FeeAmount)
	}

	authResult, authErr := s.DirectClient.AuthorizePayment(ctx, creds, payment.DirectPaymentInitRequest{
		CardToken:  cardToken,
		Amount:     amount,
		OrderID:    orderID,
		SessionID:  "", // Session created below after auth succeeds.
		ExternalID: fmt.Sprintf("AUTH-%s", GenerateSecureToken()),
		Recipients: splitRecipients,
	})
	if authErr != nil {
		return fmt.Errorf("authorize call: %w", authErr)
	}

	// Create a PaymentSession in AUTHORIZED state.
	if s.SessionSvc != nil {
		currencyCode := s.resolveSupplierCurrency(ctx, supplierID)
		session, sessErr := s.SessionSvc.CreateSession(ctx, payment.CreateSessionRequest{
			OrderID:    orderID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			Gateway:    "GLOBAL_PAY",
			Amount:     amount,
			Currency:   currencyCode,
			InvoiceID:  invoiceID,
		})
		if sessErr != nil {
			log.Printf("[CHECKOUT-AUTH] Session creation failed for order %s after auth: %v", orderID, sessErr)
		} else {
			// Transition session to AUTHORIZED with auth metadata.
			_, updateErr := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite([]*spanner.Mutation{
					spanner.Update("PaymentSessions",
						[]string{"SessionId", "Status", "AuthorizationId", "AuthorizedAmount", "ProviderReference"},
						[]interface{}{session.SessionID, payment.SessionAuthorized, authResult.PaymentID, amount, authResult.PaymentID},
					),
				})
			})
			if updateErr != nil {
				log.Printf("[CHECKOUT-AUTH] Session AUTHORIZED update failed for %s: %v", session.SessionID, updateErr)
			}
		}
	}

	// Mark order PaymentStatus = AUTHORIZED.
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(ctx, spanner.Statement{
			SQL:    `UPDATE Orders SET PaymentStatus = 'AUTHORIZED' WHERE OrderId = @oid AND PaymentStatus IN ('PENDING', '')`,
			Params: map[string]interface{}{"oid": orderID},
		})
		if err != nil {
			return err
		}

		// Atomic Outbox Emission: Payment Processed
		if eErr := outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventPaymentCleared, kafkaEvents.TopicMain, kafkaEvents.PaymentClearedEvent{
			OrderID:    orderID,
			InvoiceID:  invoiceID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			Status:     "CLEARED",
			Timestamp:  time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx)); eErr != nil {
			return eErr
		}

		// Atomic Outbox Emission: Order Finalized (Post-Fiscalization)
		return outbox.EmitJSON(txn, "Order", orderID, kafkaEvents.EventOrderFinalized, kafkaEvents.TopicMain, kafkaEvents.OrderFinalizedEvent{
			OrderID:    orderID,
			InvoiceID:  invoiceID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			FiscalSign: "PENDING_SOLIQ_SIGNATURE",
			Timestamp:  time.Now().UTC(),
		}, telemetry.TraceIDFromContext(ctx))
	})
	if err != nil {
		log.Printf("[CHECKOUT-AUTH] Orders.PaymentStatus update failed for %s: %v", orderID, err)
	}

	return nil
}

// ─── Supplier Resolution ──────────────────────────────────────────────────────

// resolveSuppliers queries the SupplierProducts table to map each SKU to its
// owning supplier. Returns a map of SkuId → supplierMeta.
func (s *OrderService) resolveSuppliers(ctx context.Context, skuIDs []string) (map[string]supplierMeta, error) {
	if len(skuIDs) == 0 {
		return nil, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT sp.SkuId, sp.SupplierId, sp.Name, sp.BasePrice
		      FROM SupplierProducts sp
		      WHERE sp.SkuId IN UNNEST(@skuIds)
		        AND sp.IsActive = TRUE`,
		Params: map[string]interface{}{
			"skuIds": skuIDs,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	result := make(map[string]supplierMeta, len(skuIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("supplier resolution query failed: %w", err)
		}

		var skuID, supplierID, productName string
		var basePrice int64
		if err := row.Columns(&skuID, &supplierID, &productName, &basePrice); err != nil {
			return nil, fmt.Errorf("supplier resolution row parse failed: %w", err)
		}

		result[skuID] = supplierMeta{
			SupplierID:   supplierID,
			SupplierName: productName, // Product name used as display; real supplier name can be joined later
			BasePrice:    basePrice,
		}
	}

	return result, nil
}

// ── Checkout helpers ──────────────────────────────────────────────────────────

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// resolveTerminalForWarehouse wraps gateway terminal resolution logic.
func resolveTerminalForWarehouse(ctx context.Context, warehouseID, provider string) string {
	// Temporary Resolver Logic for Unified Checkout
	return fmt.Sprintf("term_%s_%s", warehouseID, provider)
}
