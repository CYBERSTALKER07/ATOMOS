package payout

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/grpc/codes"
)

const (
	PayoutModeHQSupplier     = "HQ_SUPPLIER"
	PayoutModeWarehouseLocal = "WAREHOUSE_LOCAL"

	FeePolicyVersionLegacyCheckout = "LEGACY_PLATFORM_FEE_BPS_V1"

	payoutPolicyActorSupplier = "SUPPLIER"
	payoutPolicySourceDefault = "DEFAULT"
	payoutPolicySourceCustom  = "SUPPLIER_POLICY"
)

type Policy struct {
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

type policyPatch struct {
	PayoutMode       string `json:"payout_mode"`
	FeePolicyVersion string `json:"fee_policy_version"`
	Reason           string `json:"reason"`
}

func defaultPolicy(supplierID string) Policy {
	return Policy{
		SupplierID:       supplierID,
		PayoutMode:       PayoutModeHQSupplier,
		FeePolicyVersion: FeePolicyVersionLegacyCheckout,
		IsActive:         true,
		Source:           payoutPolicySourceDefault,
	}
}

func normalizePayoutMode(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case PayoutModeHQSupplier:
		return PayoutModeHQSupplier, true
	case PayoutModeWarehouseLocal:
		return PayoutModeWarehouseLocal, true
	default:
		return "", false
	}
}

func normalizeFeePolicyVersion(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return FeePolicyVersionLegacyCheckout
	}
	return v
}

func (h *Handlers) HandlePayoutPolicy(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Svc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payout_unavailable"})
		return
	}
	supplierID := payoutSupplierID(r)
	if supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		policy, err := h.Svc.GetPolicy(r.Context(), supplierID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_failed"})
			return
		}
		writeJSON(w, http.StatusOK, policy)
	case http.MethodPatch:
		claims, ok := auth.FromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		var req policyPatch
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		policy, err := h.Svc.PatchPolicy(r.Context(), supplierID, claims.Subject, string(claims.Role), req)
		if err != nil {
			switch {
			case errors.Is(err, errPolicyReasonRequired):
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reason_required"})
			case errors.Is(err, errPolicyInvalidMode):
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_payout_mode"})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
			}
			return
		}
		writeJSON(w, http.StatusOK, policy)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

var (
	errPolicyReasonRequired = errors.New("reason_required")
	errPolicyInvalidMode    = errors.New("invalid_payout_mode")
)

func (s *Service) GetPolicy(ctx context.Context, supplierID string) (Policy, error) {
	if s == nil || s.repo == nil || s.repo.client == nil {
		return Policy{}, errors.New("payout_unavailable")
	}
	row, err := s.repo.client.Single().ReadRow(ctx, "SupplierPayoutPolicies", spanner.Key{supplierID}, []string{
		"PayoutMode", "FeePolicyVersion", "EffectiveAt", "UpdatedBy", "UpdatedByType", "Reason", "IsActive",
	})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return defaultPolicy(supplierID), nil
		}
		return Policy{}, err
	}
	return scanPolicy(supplierID, row)
}

func (s *Service) PatchPolicy(ctx context.Context, supplierID, actorID, actorRole string, req policyPatch) (Policy, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return Policy{}, errPolicyReasonRequired
	}
	mode, ok := normalizePayoutMode(req.PayoutMode)
	if !ok {
		return Policy{}, errPolicyInvalidMode
	}
	if s == nil || s.repo == nil || s.repo.client == nil {
		return Policy{}, errors.New("payout_unavailable")
	}
	fee := normalizeFeePolicyVersion(req.FeePolicyVersion)
	now := time.Now().UTC()
	if s.now != nil {
		now = s.now().UTC()
	}
	resp := Policy{
		SupplierID:       supplierID,
		PayoutMode:       mode,
		FeePolicyVersion: fee,
		EffectiveAt:      &now,
		UpdatedBy:        actorID,
		UpdatedByType:    payoutPolicyActorSupplier,
		Reason:           reason,
		IsActive:         true,
		Source:           payoutPolicySourceCustom,
	}
	err := spannerutils.RunReadWriteTransaction(ctx, s.repo.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		createdAt := interface{}(spanner.CommitTimestamp)
		before := defaultPolicy(supplierID)
		row, readErr := txn.ReadRow(ctx, "SupplierPayoutPolicies", spanner.Key{supplierID}, []string{
			"PayoutMode", "FeePolicyVersion", "EffectiveAt", "UpdatedBy", "UpdatedByType", "Reason", "IsActive", "CreatedAt",
		})
		if readErr == nil {
			if scanned, scanErr := scanPolicy(supplierID, row); scanErr == nil {
				before = scanned
			}
			var existingCreated spanner.NullTime
			_ = row.ColumnByName("CreatedAt", &existingCreated)
			if existingCreated.Valid {
				createdAt = existingCreated.Time
			}
		} else if spanner.ErrCode(readErr) != codes.NotFound {
			return readErr
		}
		meta, _ := json.Marshal(map[string]any{"before": before, "after": resp})
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("SupplierPayoutPolicies", map[string]any{
				"SupplierId":       supplierID,
				"PayoutMode":       mode,
				"FeePolicyVersion": fee,
				"EffectiveAt":      now,
				"UpdatedBy":        actorID,
				"UpdatedByType":    payoutPolicyActorSupplier,
				"Reason":           reason,
				"IsActive":         true,
				"CreatedAt":        createdAt,
				"UpdatedAt":        spanner.CommitTimestamp,
			}),
			spanner.InsertMap("AuditLog", map[string]any{
				"AuditId":       uuid.NewString(),
				"SupplierId":    supplierID,
				"ActorId":       actorID,
				"ActorRole":     actorRole,
				"Action":        "SUPPLIER_PAYOUT_POLICY_UPDATED",
				"AggregateType": events.AggregatePayoutPolicy,
				"AggregateId":   supplierID,
				"DetailsJson":   meta,
				"TraceId":       outbox.TraceIDFromContext(ctx),
				"CreatedAt":     spanner.CommitTimestamp,
			}),
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(ctx, buf, events.AggregatePayoutPolicy, supplierID, events.TopicMain, map[string]any{
			"type":               events.EventPayoutPolicyUpdated,
			"supplier_id":        supplierID,
			"payout_mode":        mode,
			"fee_policy_version": fee,
			"reason":             reason,
			"timestamp":          now.Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		if err := buf.Flush(ctx); err != nil {
			return err
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return Policy{}, err
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "supplier:"+supplierID)
	}
	return resp, nil
}

func scanPolicy(supplierID string, row *spanner.Row) (Policy, error) {
	var mode, fee string
	var effective time.Time
	var active bool
	var ub, ubt, rs spanner.NullString
	if err := row.ColumnByName("PayoutMode", &mode); err != nil {
		return Policy{}, err
	}
	if err := row.ColumnByName("FeePolicyVersion", &fee); err != nil {
		return Policy{}, err
	}
	if err := row.ColumnByName("EffectiveAt", &effective); err != nil {
		return Policy{}, err
	}
	_ = row.ColumnByName("UpdatedBy", &ub)
	_ = row.ColumnByName("UpdatedByType", &ubt)
	_ = row.ColumnByName("Reason", &rs)
	_ = row.ColumnByName("IsActive", &active)
	src := payoutPolicySourceCustom
	if mode == "" {
		mode = PayoutModeHQSupplier
		src = payoutPolicySourceDefault
	}
	eff := effective.UTC()
	return Policy{
		SupplierID:       supplierID,
		PayoutMode:       mode,
		FeePolicyVersion: fee,
		EffectiveAt:      &eff,
		UpdatedBy:        ub.StringVal,
		UpdatedByType:    ubt.StringVal,
		Reason:           rs.StringVal,
		IsActive:         active,
		Source:           src,
	}, nil
}
