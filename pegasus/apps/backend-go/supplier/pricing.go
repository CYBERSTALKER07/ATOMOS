// Package supplier implements the Supplier Control API for Pegasus.
// Each supplier manages their own discount rules — isolated by SupplierId from
// their JWT. Nestle cannot read or overwrite Coca-Cola's pricing matrix.
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

	"cloud.google.com/go/spanner"
)

// ── Domain Types ──────────────────────────────────────────────────────────────

// AdvancedPricingRule is the payload from the Supplier Next.js Dashboard.
// SupplierId is NOT accepted from the body — it is extracted from the verified JWT.
type AdvancedPricingRule struct {
	TierId             string    `json:"tier_id"`              // UUID, client-supplied idempotency key
	SkuId              string    `json:"sku_id"`               // e.g. "SKU-COKE-001"
	MinPallets         int64     `json:"min_pallets"`          // Minimum quantity to activate this tier
	DiscountPercent    int64     `json:"discount_percent"`     // Integer 1–40; enforced server-side
	TargetRetailerTier string    `json:"target_retailer_tier"` // "ALL" | "BRONZE" | "SILVER" | "GOLD"
	ValidUntil         time.Time `json:"valid_until"`          // Hard expiry; zero-value = no expiry
}

// ── Service ───────────────────────────────────────────────────────────────────

// PricingService wraps the Spanner client for supplier-scoped mutations.
type PricingService struct {
	Client *spanner.Client
}

// NewPricingService constructs the service.
func NewPricingService(client *spanner.Client) *PricingService {
	return &PricingService{Client: client}
}

// UpsertPricingRule writes (or overwrites) a single pricing tier into Spanner.
// Uses InsertOrUpdate so the supplier can iterate on the same TierId without
// creating duplicate rows.
func (s *PricingService) UpsertPricingRule(ctx context.Context, supplierId string, rule AdvancedPricingRule) error {
	// Determine ValidUntil — use spanner.NullTime so zero-value encodes as NULL.
	validUntil := spanner.NullTime{}
	if !rule.ValidUntil.IsZero() {
		validUntil = spanner.NullTime{Time: rule.ValidUntil.UTC(), Valid: true}
	}

	// Normalise retailer tier: default to "ALL".
	target := rule.TargetRetailerTier
	if target == "" {
		target = "ALL"
	}

	m := spanner.InsertOrUpdate(
		"PricingTiers",
		[]string{
			"TierId", "SupplierId", "SkuId",
			"MinPallets", "DiscountPct",
			"TargetRetailerTier", "ValidUntil",
			"IsActive",
		},
		[]interface{}{
			rule.TierId, supplierId, rule.SkuId,
			rule.MinPallets, rule.DiscountPercent,
			target, validUntil,
			true,
		},
	)

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("supplier: spanner upsert failed: %w", err)
	}

	log.Printf("[SUPPLIER_MATRIX] SupplierId=%s locked %d%% discount on SkuId=%s for RetailerTier=%s MinPallets=%d",
		supplierId, rule.DiscountPercent, rule.SkuId, target, rule.MinPallets)

	return nil
}

// ── HTTP Handler ──────────────────────────────────────────────────────────────

// HandleUpsertPricingRule is the HTTP entrypoint for POST /v1/supplier/pricing/rules.
// GET lists all rules for the authenticated supplier.
// The SUPPLIER JWT role is enforced by the auth middleware upstream — this
// handler trusts that user_id in context is an authenticated supplier.
func (s *PricingService) HandleUpsertPricingRule(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.UserID == "" {
		http.Error(w, "Unauthorized: missing supplier identity", http.StatusUnauthorized)
		return
	}
	supplierId := claims.ResolveSupplierID()

	switch r.Method {
	case http.MethodGet:
		s.listPricingRules(w, r, supplierId)
	case http.MethodPost:
		s.upsertPricingRule(w, r, supplierId)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *PricingService) listPricingRules(w http.ResponseWriter, r *http.Request, supplierId string) {
	ctx := r.Context()
	stmt := spanner.Statement{
		SQL: `SELECT TierId, SkuId, MinPallets, DiscountPct, TargetRetailerTier, ValidUntil, IsActive
              FROM PricingTiers WHERE SupplierId = @sid ORDER BY SkuId`,
		Params: map[string]interface{}{"sid": supplierId},
	}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type RuleRow struct {
		TierId             string  `json:"tier_id"`
		SkuId              string  `json:"sku_id"`
		MinPallets         int64   `json:"min_pallets"`
		DiscountPercent    int64   `json:"discount_percent"`
		TargetRetailerTier string  `json:"target_retailer_tier"`
		ValidUntil         *string `json:"valid_until,omitempty"`
		IsActive           bool    `json:"is_active"`
	}

	var rules []RuleRow
	for {
		row, err := iter.Next()
		if err != nil {
			break
		}
		var tierId, skuId, target string
		var minPallets, discountPct int64
		var validUntil spanner.NullTime
		var isActive bool
		if err := row.Columns(&tierId, &skuId, &minPallets, &discountPct, &target, &validUntil, &isActive); err != nil {
			continue
		}
		r := RuleRow{
			TierId: tierId, SkuId: skuId, MinPallets: minPallets,
			DiscountPercent: discountPct, TargetRetailerTier: target, IsActive: isActive,
		}
		if validUntil.Valid {
			s := validUntil.Time.Format(time.RFC3339)
			r.ValidUntil = &s
		}
		rules = append(rules, r)
	}
	if rules == nil {
		rules = []RuleRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (s *PricingService) upsertPricingRule(w http.ResponseWriter, r *http.Request, supplierId string) {
	var rule AdvancedPricingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "Malformed rule payload", http.StatusBadRequest)
		return
	}

	// ── Validation ────────────────────────────────────────────────────────────
	if rule.TierId == "" {
		http.Error(w, "tier_id is required (use a UUID for idempotent upserts)", http.StatusUnprocessableEntity)
		return
	}
	if rule.SkuId == "" {
		http.Error(w, "sku_id is required", http.StatusUnprocessableEntity)
		return
	}
	if rule.MinPallets <= 0 {
		http.Error(w, "min_pallets must be >= 1", http.StatusUnprocessableEntity)
		return
	}
	// Hard cap: suppliers cannot accidentally give product away.
	if rule.DiscountPercent <= 0 || rule.DiscountPercent > 40 {
		http.Error(w, "discount_percent must be between 1 and 40", http.StatusUnprocessableEntity)
		return
	}
	validTiers := map[string]bool{"ALL": true, "BRONZE": true, "SILVER": true, "GOLD": true}
	if rule.TargetRetailerTier != "" && !validTiers[rule.TargetRetailerTier] {
		http.Error(w, "target_retailer_tier must be ALL, BRONZE, SILVER, or GOLD", http.StatusUnprocessableEntity)
		return
	}

	// ── Persist ───────────────────────────────────────────────────────────────

	// Verify the SKU belongs to this supplier before persisting
	skuCheck := spanner.Statement{
		SQL:    `SELECT 1 FROM SupplierProducts WHERE SkuId = @skuId AND SupplierId = @sid`,
		Params: map[string]interface{}{"skuId": rule.SkuId, "sid": supplierId},
	}
	skuIter := s.Client.Single().Query(r.Context(), skuCheck)
	_, skuErr := skuIter.Next()
	skuIter.Stop()
	if skuErr != nil {
		http.Error(w, `{"error":"sku_id does not belong to this supplier or does not exist"}`, http.StatusBadRequest)
		return
	}

	if err := s.UpsertPricingRule(r.Context(), supplierId, rule); err != nil {
		log.Printf("[ERROR] supplier.UpsertPricingRule: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "RULE_ACTIVE",
		"tier_id":  rule.TierId,
		"sku_id":   rule.SkuId,
		"discount": fmt.Sprintf("%d%%", rule.DiscountPercent),
		"target":   rule.TargetRetailerTier,
	})
}

// HandlePricingRuleAction handles DELETE /v1/supplier/pricing/rules/{tier_id}
func (s *PricingService) HandlePricingRuleAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
	if !ok || claims == nil || claims.UserID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	supplierId := claims.ResolveSupplierID()

	switch r.Method {
	case http.MethodDelete:
		s.deactivatePricingRule(w, r, supplierId)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *PricingService) deactivatePricingRule(w http.ResponseWriter, r *http.Request, supplierId string) {
	tierId := strings.TrimPrefix(r.URL.Path, "/v1/supplier/pricing/rules/")
	if tierId == "" {
		http.Error(w, `{"error":"tier_id required in path"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := s.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "PricingTiers", spanner.Key{tierId}, []string{"SupplierId"})
		if err != nil {
			return fmt.Errorf("pricing rule not found")
		}
		var ownerSid string
		if err := row.Columns(&ownerSid); err != nil {
			return err
		}
		if ownerSid != supplierId {
			return fmt.Errorf("access denied")
		}

		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("PricingTiers",
				[]string{"TierId", "SupplierId", "IsActive"},
				[]interface{}{tierId, supplierId, false},
			),
		})
		return nil
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			http.Error(w, `{"error":"pricing rule not found"}`, http.StatusNotFound)
		} else if strings.Contains(errMsg, "access denied") {
			http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		} else {
			log.Printf("[SUPPLIER PRICING] deactivate error for %s/%s: %v", supplierId, tierId, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "RULE_DEACTIVATED",
		"tier_id": tierId,
	})
}
