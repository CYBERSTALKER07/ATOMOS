package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/kafka"
	"backend-go/proximity"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

// ── Per-Retailer Pricing Override DTOs ────────────────────────────────────────

type CreateRetailerPriceOverrideRequest struct {
	RetailerId string `json:"retailer_id"`
	SkuId      string `json:"sku_id"`
	Price      int64  `json:"price"`      // Absolute price in smallest unit (e.g. UZS)
	Notes      string `json:"notes"`      // Optional label/reason
	ExpiresAt  string `json:"expires_at"` // RFC3339 or empty
}

type RetailerPriceOverrideResponse struct {
	OverrideId string  `json:"override_id"`
	SupplierId string  `json:"supplier_id"`
	RetailerId string  `json:"retailer_id"`
	SkuId      string  `json:"sku_id"`
	Price      int64   `json:"price"`
	SetBy      string  `json:"set_by"`
	SetByRole  string  `json:"set_by_role"`
	IsActive   bool    `json:"is_active"`
	Notes      string  `json:"notes,omitempty"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// ── Service ───────────────────────────────────────────────────────────────────

type RetailerPricingService struct {
	Client     *spanner.Client
	ReadRouter proximity.ReadRouter
	Producer   *kafkago.Writer
}

func NewRetailerPricingService(client *spanner.Client, readRouter proximity.ReadRouter, producer *kafkago.Writer) *RetailerPricingService {
	return &RetailerPricingService{Client: client, ReadRouter: readRouter, Producer: producer}
}

// ── HTTP Handlers ─────────────────────────────────────────────────────────────

// HandleRetailerPricingOverrides handles:
//
//	GET  /v1/supplier/pricing/retailer-overrides           — list overrides
//	GET  /v1/supplier/pricing/retailer-overrides?retailer_id=X — list for specific retailer
//	POST /v1/supplier/pricing/retailer-overrides           — create/update override
func (s *RetailerPricingService) HandleRetailerPricingOverrides(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.listOverrides(w, r, claims)
	case http.MethodPost:
		s.createOverride(w, r, claims)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// HandleRetailerPricingOverrideAction handles DELETE /v1/supplier/pricing/retailer-overrides/{id}
func (s *RetailerPricingService) HandleRetailerPricingOverrideAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	overrideID := strings.TrimPrefix(r.URL.Path, "/v1/supplier/pricing/retailer-overrides/")
	if overrideID == "" {
		http.Error(w, `{"error":"override_id required in path"}`, http.StatusBadRequest)
		return
	}
	s.deactivateOverride(w, r, claims, overrideID)
}

// ── List Overrides ────────────────────────────────────────────────────────────

func (s *RetailerPricingService) listOverrides(w http.ResponseWriter, r *http.Request, claims *auth.LabClaims) {
	supplierID := claims.ResolveSupplierID()
	retailerID := r.URL.Query().Get("retailer_id")
	skuID := r.URL.Query().Get("sku_id")

	sql := `SELECT OverrideId, SupplierId, RetailerId, SkuId, OverridePrice,
	               SetBy, SetByRole, IsActive, Notes, ExpiresAt, CreatedAt
	        FROM RetailerPricingOverrides
	        WHERE SupplierId = @supplierId AND IsActive = true`
	params := map[string]interface{}{"supplierId": supplierID}

	if retailerID != "" {
		sql += ` AND RetailerId = @retailerId`
		params["retailerId"] = retailerID
	}
	if skuID != "" {
		sql += ` AND SkuId = @skuId`
		params["skuId"] = skuID
	}

	// NODE_ADMIN: only see overrides for retailers in their warehouse
	scope := auth.GetWarehouseScope(r.Context())
	if scope != nil && scope.IsNodeAdmin {
		sql += ` AND WarehouseId = @warehouseId`
		params["warehouseId"] = scope.WarehouseID
	}

	sql += ` ORDER BY CreatedAt DESC LIMIT 500`

	readClient := s.readClientForWarehouseScope(r.Context(), supplierID, auth.EffectiveWarehouseID(r.Context()))
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := readClient.Single().Query(r.Context(), stmt)
	defer iter.Stop()

	var overrides []RetailerPriceOverrideResponse
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("[RETAILER_PRICING] List error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		var o RetailerPriceOverrideResponse
		var expiresAt spanner.NullTime
		var createdAt spanner.NullTime
		var notes spanner.NullString
		if err := row.Columns(&o.OverrideId, &o.SupplierId, &o.RetailerId, &o.SkuId,
			&o.Price, &o.SetBy, &o.SetByRole, &o.IsActive, &notes, &expiresAt, &createdAt); err != nil {
			log.Printf("[RETAILER_PRICING] Row parse: %v", err)
			continue
		}
		if notes.Valid {
			o.Notes = notes.StringVal
		}
		if expiresAt.Valid {
			s := expiresAt.Time.Format(time.RFC3339)
			o.ExpiresAt = &s
		}
		if createdAt.Valid {
			o.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}
		overrides = append(overrides, o)
	}

	if overrides == nil {
		overrides = []RetailerPriceOverrideResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"overrides": overrides,
		"total":     len(overrides),
	})
}

// ── Create Override ───────────────────────────────────────────────────────────

func (s *RetailerPricingService) createOverride(w http.ResponseWriter, r *http.Request, claims *auth.LabClaims) {
	supplierID := claims.ResolveSupplierID()

	var req CreateRetailerPriceOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// ── Validation ────────────────────────────────────────────────────────────
	if req.RetailerId == "" {
		http.Error(w, `{"error":"retailer_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.SkuId == "" {
		http.Error(w, `{"error":"sku_id is required"}`, http.StatusBadRequest)
		return
	}
	if req.Price <= 0 {
		http.Error(w, `{"error":"price must be positive"}`, http.StatusBadRequest)
		return
	}

	// Resolve caller's role
	scope := auth.GetWarehouseScope(r.Context())
	callerRole := "GLOBAL_ADMIN"
	var warehouseID *string
	if scope != nil && scope.IsNodeAdmin {
		callerRole = "NODE_ADMIN"
		warehouseID = &scope.WarehouseID
	}

	ctx := r.Context()

	// Verify SKU belongs to this supplier
	skuCheck := spanner.Statement{
		SQL:    `SELECT 1 FROM SupplierProducts WHERE SkuId = @skuId AND SupplierId = @sid`,
		Params: map[string]interface{}{"skuId": req.SkuId, "sid": supplierID},
	}
	skuIter := s.Client.Single().Query(ctx, skuCheck)
	_, skuErr := skuIter.Next()
	skuIter.Stop()
	if skuErr != nil {
		http.Error(w, `{"error":"sku_id does not belong to this supplier"}`, http.StatusBadRequest)
		return
	}

	// NODE_ADMIN: verify retailer is within their warehouse scope
	if scope != nil && scope.IsNodeAdmin {
		if !s.retailerInWarehouseScope(ctx, supplierID, req.RetailerId, scope.WarehouseID) {
			http.Error(w, `{"error":"retailer is not within your warehouse scope"}`, http.StatusForbidden)
			return
		}
	}

	// Parse optional expiry
	var expiresAt spanner.NullTime
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			http.Error(w, `{"error":"expires_at must be RFC3339 format"}`, http.StatusBadRequest)
			return
		}
		expiresAt = spanner.NullTime{Time: t.UTC(), Valid: true}
	}

	overrideID := uuid.New().String()

	// Upsert: deactivate any existing active override for this supplier+retailer+sku,
	// then insert new one. Uses ReadWriteTransaction for atomicity.
	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Deactivate existing active override (if any)
		deactivateStmt := spanner.Statement{
			SQL: `SELECT OverrideId FROM RetailerPricingOverrides
			      WHERE SupplierId = @sid AND RetailerId = @rid AND SkuId = @skuId AND IsActive = true`,
			Params: map[string]interface{}{
				"sid":   supplierID,
				"rid":   req.RetailerId,
				"skuId": req.SkuId,
			},
		}
		deactIter := txn.Query(ctx, deactivateStmt)
		defer deactIter.Stop()

		var mutations []*spanner.Mutation
		for {
			row, err := deactIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var oldID string
			if err := row.Columns(&oldID); err != nil {
				continue
			}
			mutations = append(mutations, spanner.Update("RetailerPricingOverrides",
				[]string{"OverrideId", "IsActive", "UpdatedAt"},
				[]interface{}{oldID, false, spanner.CommitTimestamp},
			))
		}

		// Insert new override
		cols := []string{"OverrideId", "SupplierId", "RetailerId", "SkuId",
			"OverridePrice", "SetBy", "SetByRole", "IsActive", "CreatedAt", "UpdatedAt"}
		vals := []interface{}{overrideID, supplierID, req.RetailerId, req.SkuId,
			req.Price, claims.UserID, callerRole, true, spanner.CommitTimestamp, spanner.CommitTimestamp}

		if warehouseID != nil {
			cols = append(cols, "WarehouseId")
			vals = append(vals, *warehouseID)
		}
		if req.Notes != "" {
			cols = append(cols, "Notes")
			vals = append(vals, req.Notes)
		}
		if expiresAt.Valid {
			cols = append(cols, "ExpiresAt")
			vals = append(vals, expiresAt.Time)
		}

		mutations = append(mutations, spanner.Insert("RetailerPricingOverrides", cols, vals))
		return txn.BufferWrite(mutations)
	})

	if err != nil {
		log.Printf("[RETAILER_PRICING] Create failed: %v", err)
		http.Error(w, "Failed to create price override", http.StatusInternalServerError)
		return
	}

	// Emit Kafka event (non-blocking)
	go s.emitPriceOverrideEvent(overrideID, supplierID, req.RetailerId, req.SkuId, req.Price, "CREATED", claims.UserID, callerRole)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "OVERRIDE_ACTIVE",
		"override_id": overrideID,
		"retailer_id": req.RetailerId,
		"sku_id":      req.SkuId,
		"price":       req.Price,
	})
}

// ── Deactivate Override ───────────────────────────────────────────────────────

func (s *RetailerPricingService) deactivateOverride(w http.ResponseWriter, r *http.Request, claims *auth.LabClaims, overrideID string) {
	supplierID := claims.ResolveSupplierID()
	ctx := r.Context()

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "RetailerPricingOverrides", spanner.Key{overrideID},
			[]string{"SupplierId", "WarehouseId", "RetailerId", "SkuId", "OverridePrice"})
		if err != nil {
			return fmt.Errorf("override not found")
		}

		var ownerSid string
		var warehouseID spanner.NullString
		var retailerID, skuID string
		var price int64
		if err := row.Columns(&ownerSid, &warehouseID, &retailerID, &skuID, &price); err != nil {
			return err
		}
		if ownerSid != supplierID {
			return fmt.Errorf("access denied")
		}

		// NODE_ADMIN: can only deactivate overrides for their warehouse
		scope := auth.GetWarehouseScope(ctx)
		if scope != nil && scope.IsNodeAdmin {
			if !warehouseID.Valid || warehouseID.StringVal != scope.WarehouseID {
				return fmt.Errorf("access denied: warehouse scope violation")
			}
		}

		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("RetailerPricingOverrides",
				[]string{"OverrideId", "IsActive", "UpdatedAt"},
				[]interface{}{overrideID, false, spanner.CommitTimestamp},
			),
		})

		// Emit event after commit
		go s.emitPriceOverrideEvent(overrideID, supplierID, retailerID, skuID, price, "DEACTIVATED", claims.UserID, "GLOBAL_ADMIN")

		return nil
	})

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "access denied") {
			http.Error(w, `{"error":"Access denied"}`, http.StatusForbidden)
			return
		}
		if strings.Contains(errStr, "not found") {
			http.Error(w, `{"error":"Override not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("[RETAILER_PRICING] Deactivate error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deactivated", "override_id": overrideID})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// retailerInWarehouseScope checks if a retailer's location falls within the
// warehouse's H3 coverage, or if the retailer has PrimaryWarehouseId matching.
func (s *RetailerPricingService) retailerInWarehouseScope(ctx context.Context, supplierID, retailerID, warehouseID string) bool {
	readClient := s.readClientForWarehouseScope(ctx, supplierID, warehouseID)

	// Check SupplierRetailerClients materialised view first
	stmt := spanner.Statement{
		SQL: `SELECT 1 FROM SupplierRetailerClients
		      WHERE SupplierId = @sid AND RetailerId = @rid AND PrimaryWarehouseId = @whId`,
		Params: map[string]interface{}{
			"sid":  supplierID,
			"rid":  retailerID,
			"whId": warehouseID,
		},
	}
	iter := readClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == nil {
		return true // Found in materialised assignment
	}

	// Fallback: H3 check
	retailerRow, err := readClient.Single().ReadRow(ctx, "Retailers",
		spanner.Key{retailerID}, []string{"H3Index"})
	if err != nil {
		return false
	}
	var h3 spanner.NullString
	if err := retailerRow.Columns(&h3); err != nil || !h3.Valid {
		return false
	}

	whStmt := spanner.Statement{
		SQL: `SELECT 1 FROM Warehouses
		      WHERE WarehouseId = @whId AND SupplierId = @sid
		        AND @h3cell IN UNNEST(H3Indexes)`,
		Params: map[string]interface{}{
			"whId":   warehouseID,
			"sid":    supplierID,
			"h3cell": h3.StringVal,
		},
	}
	whIter := readClient.Single().Query(ctx, whStmt)
	defer whIter.Stop()
	_, err = whIter.Next()
	return err == nil
}

func (s *RetailerPricingService) readClientForWarehouseScope(ctx context.Context, supplierID, warehouseID string) *spanner.Client {
	if s == nil || s.Client == nil {
		return nil
	}
	if s.ReadRouter == nil || warehouseID == "" {
		return s.Client
	}

	stmt := spanner.Statement{
		SQL: `SELECT Lat, Lng
		      FROM Warehouses
		      WHERE WarehouseId = @warehouseId AND SupplierId = @supplierId
		      LIMIT 1`,
		Params: map[string]interface{}{
			"warehouseId": warehouseID,
			"supplierId":  supplierID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return s.Client
	}

	var lat, lng spanner.NullFloat64
	if row.Columns(&lat, &lng) != nil || !lat.Valid || !lng.Valid {
		return s.Client
	}

	return proximity.ReadClientForRetailer(s.Client, s.ReadRouter, lat.Float64, lng.Float64)
}

// emitPriceOverrideEvent fires a Kafka event for audit and analytics.
func (s *RetailerPricingService) emitPriceOverrideEvent(overrideID, supplierID, retailerID, skuID string, price int64, action, setBy, setByRole string) {
	if s.Producer == nil {
		return
	}
	event := kafka.RetailerPriceOverrideEvent{
		OverrideId: overrideID,
		SupplierId: supplierID,
		RetailerId: retailerID,
		SkuId:      skuID,
		Price:      price,
		Action:     action,
		SetBy:      setBy,
		SetByRole:  setByRole,
		Timestamp:  time.Now(),
	}
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("[RETAILER_PRICING] marshal event: %v", err)
		return
	}
	_ = s.Producer.WriteMessages(context.Background(), kafkago.Message{
		Key:   []byte(kafka.EventRetailerPriceOverride),
		Value: payload,
	})
}
