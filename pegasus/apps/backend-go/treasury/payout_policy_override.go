package treasury

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/order"

	"cloud.google.com/go/spanner"
)

const internalPayoutPolicyActorType = "INTERNAL"

type internalSupplierPayoutPolicyOverrideRequest struct {
	SupplierID       string     `json:"supplier_id"`
	PayoutMode       string     `json:"payout_mode"`
	FeePolicyVersion string     `json:"fee_policy_version"`
	Reason           string     `json:"reason"`
	EffectiveAt      *time.Time `json:"effective_at,omitempty"`
}

type internalSupplierPayoutPolicyResponse struct {
	SupplierID       string     `json:"supplier_id"`
	PayoutMode       string     `json:"payout_mode"`
	FeePolicyVersion string     `json:"fee_policy_version"`
	EffectiveAt      *time.Time `json:"effective_at,omitempty"`
	UpdatedBy        string     `json:"updated_by,omitempty"`
	UpdatedByType    string     `json:"updated_by_type,omitempty"`
	Reason           string     `json:"reason,omitempty"`
	IsActive         bool       `json:"is_active"`
	Source           string     `json:"source"`
}

// HandleInternalSupplierPayoutPolicyOverride applies audited payout-policy
// overrides through the internal-service auth lane.
//
//	PATCH /v1/internal/treasury/supplier-payout-policy
func HandleInternalSupplierPayoutPolicyOverride(client *spanner.Client, rc *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if claims.Role != "INTERNAL" {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}

		var req internalSupplierPayoutPolicyOverrideRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}

		supplierID := strings.TrimSpace(req.SupplierID)
		if supplierID == "" {
			http.Error(w, `{"error":"supplier_id required"}`, http.StatusBadRequest)
			return
		}

		payoutMode, ok := parseInternalPayoutMode(req.PayoutMode)
		if !ok {
			http.Error(w, `{"error":"payout_mode must be HQ_SUPPLIER or WAREHOUSE_LOCAL"}`, http.StatusBadRequest)
			return
		}

		reason := strings.TrimSpace(req.Reason)
		if reason == "" {
			http.Error(w, `{"error":"reason required"}`, http.StatusBadRequest)
			return
		}

		feePolicyVersion := normalizeInternalPolicyVersion(req.FeePolicyVersion)
		now := time.Now().UTC()
		effectiveAt := now
		if req.EffectiveAt != nil {
			effectiveAt = req.EffectiveAt.UTC()
		}

		resp := internalSupplierPayoutPolicyResponse{
			SupplierID:       supplierID,
			PayoutMode:       payoutMode,
			FeePolicyVersion: feePolicyVersion,
			EffectiveAt:      &effectiveAt,
			UpdatedBy:        claims.UserID,
			UpdatedByType:    internalPayoutPolicyActorType,
			Reason:           reason,
			IsActive:         true,
			Source:           "INTERNAL_OVERRIDE",
		}

		_, err := client.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			before, createdAt, err := loadInternalPolicyBefore(txn, ctx, supplierID)
			if err != nil {
				return err
			}

			metadata, err := json.Marshal(map[string]interface{}{
				"source": "INTERNAL_OVERRIDE",
				"before": before,
				"after":  resp,
			})
			if err != nil {
				return fmt.Errorf("encode payout policy override audit metadata: %w", err)
			}

			mutations := []*spanner.Mutation{
				spanner.InsertOrUpdate("SupplierPayoutPolicies",
					[]string{"SupplierId", "PayoutMode", "FeePolicyVersion", "EffectiveAt", "UpdatedBy", "UpdatedByType", "Reason", "IsActive", "CreatedAt", "UpdatedAt"},
					[]interface{}{supplierID, payoutMode, feePolicyVersion, effectiveAt, claims.UserID, internalPayoutPolicyActorType, reason, true, createdAt, spanner.CommitTimestamp},
				),
				spanner.Insert("AuditLog",
					[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
					[]interface{}{
						fmt.Sprintf("AUDIT-SPP-OVR-%d-%s", now.UnixNano(), shortInternalID(supplierID)),
						claims.UserID,
						claims.Role,
						"SUPPLIER_PAYOUT_POLICY_OVERRIDE",
						"SUPPLIER_PAYOUT_POLICY",
						supplierID,
						string(metadata),
						spanner.CommitTimestamp,
					},
				),
			}

			return txn.BufferWrite(mutations)
		})
		if err != nil {
			slog.Error("treasury.internal_payout_policy_override_failed", "supplier_id", supplierID, "err", err)
			http.Error(w, `{"error":"override_failed"}`, http.StatusInternalServerError)
			return
		}

		if rc != nil {
			rc.Invalidate(r.Context(), cache.SupplierProfile(supplierID))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func loadInternalPolicyBefore(txn *spanner.ReadWriteTransaction, ctx context.Context, supplierID string) (internalSupplierPayoutPolicyResponse, interface{}, error) {
	before := internalSupplierPayoutPolicyResponse{
		SupplierID:       supplierID,
		PayoutMode:       order.PayoutModeHQSupplier,
		FeePolicyVersion: order.FeePolicyVersionLegacyCheckout,
		IsActive:         true,
		Source:           "DEFAULT",
	}
	createdAt := interface{}(spanner.CommitTimestamp)

	row, err := txn.ReadRow(
		ctx,
		"SupplierPayoutPolicies",
		spanner.Key{supplierID},
		[]string{"PayoutMode", "FeePolicyVersion", "EffectiveAt", "UpdatedBy", "UpdatedByType", "Reason", "IsActive", "CreatedAt"},
	)
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return before, createdAt, nil
		}
		return internalSupplierPayoutPolicyResponse{}, nil, fmt.Errorf("read SupplierPayoutPolicies row: %w", err)
	}

	var payoutMode string
	var feePolicyVersion string
	var effectiveAt time.Time
	var updatedBy spanner.NullString
	var updatedByType spanner.NullString
	var reason spanner.NullString
	var isActive bool
	var existingCreatedAt spanner.NullTime
	if err := row.Columns(&payoutMode, &feePolicyVersion, &effectiveAt, &updatedBy, &updatedByType, &reason, &isActive, &existingCreatedAt); err != nil {
		return internalSupplierPayoutPolicyResponse{}, nil, fmt.Errorf("decode SupplierPayoutPolicies row: %w", err)
	}

	before = internalSupplierPayoutPolicyResponse{
		SupplierID:       supplierID,
		PayoutMode:       payoutMode,
		FeePolicyVersion: feePolicyVersion,
		EffectiveAt:      &effectiveAt,
		UpdatedBy:        strings.TrimSpace(updatedBy.StringVal),
		UpdatedByType:    strings.TrimSpace(updatedByType.StringVal),
		Reason:           strings.TrimSpace(reason.StringVal),
		IsActive:         isActive,
		Source:           "SUPPLIER_POLICY",
	}
	if existingCreatedAt.Valid {
		createdAt = existingCreatedAt.Time
	}

	return before, createdAt, nil
}

func parseInternalPayoutMode(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case order.PayoutModeHQSupplier:
		return order.PayoutModeHQSupplier, true
	case order.PayoutModeWarehouseLocal:
		return order.PayoutModeWarehouseLocal, true
	default:
		return "", false
	}
}

func normalizeInternalPolicyVersion(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return order.FeePolicyVersionLegacyCheckout
	}
	return trimmed
}

func shortInternalID(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 8 {
		return trimmed
	}
	return trimmed[:8]
}
