package supplier

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

const (
	payoutPolicyActorSupplier = "SUPPLIER"
	payoutPolicySourceDefault = "DEFAULT"
	payoutPolicySourceCustom  = "SUPPLIER_POLICY"
)

type supplierPayoutPolicyPatchRequest struct {
	PayoutMode       string `json:"payout_mode"`
	FeePolicyVersion string `json:"fee_policy_version"`
	Reason           string `json:"reason"`
}

type supplierPayoutPolicyResponse struct {
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

// HandleSupplierPayoutPolicy serves supplier self-service payout-policy reads
// and updates.
//
//	GET   /v1/supplier/payout-policy
//	PATCH /v1/supplier/payout-policy
func HandleSupplierPayoutPolicy(spannerClient *spanner.Client, rc *cache.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		supplierID := strings.TrimSpace(claims.ResolveSupplierID())
		if supplierID == "" {
			http.Error(w, `{"error":"supplier_id_unavailable"}`, http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleGetSupplierPayoutPolicy(w, r, spannerClient, supplierID)
		case http.MethodPatch:
			handlePatchSupplierPayoutPolicy(w, r, spannerClient, rc, claims, supplierID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetSupplierPayoutPolicy(w http.ResponseWriter, r *http.Request, spannerClient *spanner.Client, supplierID string) {
	policy, err := readSupplierPayoutPolicy(r.Context(), spannerClient, supplierID)
	if err != nil {
		slog.Error("supplier.payout_policy_read_failed", "supplier_id", supplierID, "err", err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(policy)
}

func handlePatchSupplierPayoutPolicy(
	w http.ResponseWriter,
	r *http.Request,
	spannerClient *spanner.Client,
	rc *cache.Cache,
	claims *auth.PegasusClaims,
	supplierID string,
) {
	var req supplierPayoutPolicyPatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
		return
	}

	payoutMode, ok := parsePayoutMode(req.PayoutMode)
	if !ok {
		http.Error(w, `{"error":"payout_mode must be HQ_SUPPLIER or WAREHOUSE_LOCAL"}`, http.StatusBadRequest)
		return
	}

	feePolicyVersion := normalizePolicyVersion(req.FeePolicyVersion)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		http.Error(w, `{"error":"reason required"}`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	effectiveAt := now
	resp := supplierPayoutPolicyResponse{
		SupplierID:       supplierID,
		PayoutMode:       payoutMode,
		FeePolicyVersion: feePolicyVersion,
		EffectiveAt:      &effectiveAt,
		UpdatedBy:        claims.UserID,
		UpdatedByType:    payoutPolicyActorSupplier,
		Reason:           reason,
		IsActive:         true,
		Source:           payoutPolicySourceCustom,
	}

	_, err := spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		before, createdAt, err := loadPolicyBefore(txn, ctx, supplierID)
		if err != nil {
			return err
		}

		metadata, err := json.Marshal(map[string]interface{}{
			"source": "SUPPLIER_SELF_SERVICE",
			"before": before,
			"after":  resp,
		})
		if err != nil {
			return fmt.Errorf("encode payout policy audit metadata: %w", err)
		}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdate("SupplierPayoutPolicies",
				[]string{"SupplierId", "PayoutMode", "FeePolicyVersion", "EffectiveAt", "UpdatedBy", "UpdatedByType", "Reason", "IsActive", "CreatedAt", "UpdatedAt"},
				[]interface{}{supplierID, payoutMode, feePolicyVersion, effectiveAt, claims.UserID, payoutPolicyActorSupplier, reason, true, createdAt, spanner.CommitTimestamp},
			),
			spanner.Insert("AuditLog",
				[]string{"LogId", "ActorId", "ActorRole", "Action", "ResourceType", "ResourceId", "Metadata", "CreatedAt"},
				[]interface{}{
					fmt.Sprintf("AUDIT-SPP-%d-%s", now.UnixNano(), shortID(supplierID)),
					claims.UserID,
					claims.Role,
					"SUPPLIER_PAYOUT_POLICY_UPDATED",
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
		slog.Error("supplier.payout_policy_update_failed", "supplier_id", supplierID, "err", err)
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	if rc != nil {
		rc.Invalidate(r.Context(), cache.SupplierProfile(supplierID))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func readSupplierPayoutPolicy(ctx context.Context, client *spanner.Client, supplierID string) (supplierPayoutPolicyResponse, error) {
	row, err := client.Single().ReadRow(
		ctx,
		"SupplierPayoutPolicies",
		spanner.Key{supplierID},
		[]string{"PayoutMode", "FeePolicyVersion", "EffectiveAt", "UpdatedBy", "UpdatedByType", "Reason", "IsActive"},
	)
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return defaultPayoutPolicy(supplierID), nil
		}
		return supplierPayoutPolicyResponse{}, err
	}

	var payoutMode string
	var feePolicyVersion string
	var effectiveAt time.Time
	var updatedBy spanner.NullString
	var updatedByType spanner.NullString
	var reason spanner.NullString
	var isActive bool
	if err := row.Columns(&payoutMode, &feePolicyVersion, &effectiveAt, &updatedBy, &updatedByType, &reason, &isActive); err != nil {
		return supplierPayoutPolicyResponse{}, err
	}

	resp := supplierPayoutPolicyResponse{
		SupplierID:       supplierID,
		PayoutMode:       payoutMode,
		FeePolicyVersion: feePolicyVersion,
		EffectiveAt:      &effectiveAt,
		UpdatedBy:        strings.TrimSpace(updatedBy.StringVal),
		UpdatedByType:    strings.TrimSpace(updatedByType.StringVal),
		Reason:           strings.TrimSpace(reason.StringVal),
		IsActive:         isActive,
		Source:           payoutPolicySourceCustom,
	}

	return resp, nil
}

func defaultPayoutPolicy(supplierID string) supplierPayoutPolicyResponse {
	return supplierPayoutPolicyResponse{
		SupplierID:       supplierID,
		PayoutMode:       order.PayoutModeHQSupplier,
		FeePolicyVersion: order.FeePolicyVersionLegacyCheckout,
		IsActive:         true,
		Source:           payoutPolicySourceDefault,
	}
}

func loadPolicyBefore(txn *spanner.ReadWriteTransaction, ctx context.Context, supplierID string) (supplierPayoutPolicyResponse, interface{}, error) {
	before := defaultPayoutPolicy(supplierID)
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
		return supplierPayoutPolicyResponse{}, nil, fmt.Errorf("read SupplierPayoutPolicies row: %w", err)
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
		return supplierPayoutPolicyResponse{}, nil, fmt.Errorf("decode SupplierPayoutPolicies row: %w", err)
	}

	before = supplierPayoutPolicyResponse{
		SupplierID:       supplierID,
		PayoutMode:       payoutMode,
		FeePolicyVersion: feePolicyVersion,
		EffectiveAt:      &effectiveAt,
		UpdatedBy:        strings.TrimSpace(updatedBy.StringVal),
		UpdatedByType:    strings.TrimSpace(updatedByType.StringVal),
		Reason:           strings.TrimSpace(reason.StringVal),
		IsActive:         isActive,
		Source:           payoutPolicySourceCustom,
	}
	if existingCreatedAt.Valid {
		createdAt = existingCreatedAt.Time
	}

	return before, createdAt, nil
}

func parsePayoutMode(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case order.PayoutModeHQSupplier:
		return order.PayoutModeHQSupplier, true
	case order.PayoutModeWarehouseLocal:
		return order.PayoutModeWarehouseLocal, true
	default:
		return "", false
	}
}

func normalizePolicyVersion(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return order.FeePolicyVersionLegacyCheckout
	}
	return trimmed
}

func shortID(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 8 {
		return trimmed
	}
	return trimmed[:8]
}
