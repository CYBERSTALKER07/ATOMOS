package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/cashrecon"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
)

// FiscalOrderResolver triggers fiscal receipt retry for failed orders.
type FiscalOrderResolver interface {
	RetryFiscal(ctx context.Context, claims auth.Claims, orderID string) (order.CollectCashResponse, error)
}

// ExceptionResolveDeps wires finance, 1C posting, fiscal, and retailer dedup services for inline exception resolution.
type ExceptionResolveDeps struct {
	CashRecon      *cashrecon.Service
	CreditNote     *creditnote.Service
	Credit         *credit.Service
	Spanner        *spanner.Client
	OnecPostings   partner.OnecPostingRepository
	RetailerLinks  entityresolution.RetailerLinkRepository
	FiscalOrders   FiscalOrderResolver
	Service        *Service
	ExtraResolveFn func(ctx context.Context, claims auth.Claims, kind, id string, body ResolveExceptionRequest) (int, error)
}

// ResolveExceptionRequest captures payload parameters for single or batch exception resolution.
type ResolveExceptionRequest struct {
	Note             string `json:"note"`
	CreditNoteID     string `json:"credit_note_id"`
	Action           string `json:"action"`
	OnecRef          string `json:"onec_ref"`
	MergedIntoID     string `json:"merged_into_id"`
	TargetRetailerID string `json:"target_retailer_id"`
}

// BatchResolveItem represents one item in POST /v1/supplier/exceptions/batch-resolve.
type BatchResolveItem struct {
	Kind             string `json:"kind"`
	ID               string `json:"id"`
	Action           string `json:"action,omitempty"`
	Note             string `json:"note,omitempty"`
	CreditNoteID     string `json:"credit_note_id,omitempty"`
	OnecRef          string `json:"onec_ref,omitempty"`
	MergedIntoID     string `json:"merged_into_id,omitempty"`
	TargetRetailerID string `json:"target_retailer_id,omitempty"`
}

// BatchResolveFailure describes one failed item during batch resolution.
type BatchResolveFailure struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Error string `json:"error"`
}

// BatchResolveResult describes the outcome for each item in batch resolution.
type BatchResolveResult struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// HandleResolveException serves POST /v1/supplier/exceptions/{kind}/{id}/resolve.
func HandleResolveException(deps ExceptionResolveDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		claims, ok := auth.FromContext(r.Context())
		if !ok || claims.Role != auth.RoleAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		kind := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "kind")))
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if kind == "" || id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind_and_id_required"})
			return
		}
		var body ResolveExceptionRequest
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json_body"})
				return
			}
		}

		status, err := resolveSingleException(r.Context(), deps, claims, kind, id, body)
		if err != nil {
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	}
}

// HandleBatchResolveExceptions serves POST /v1/supplier/exceptions/batch-resolve.
func HandleBatchResolveExceptions(deps ExceptionResolveDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
			return
		}
		claims, ok := auth.FromContext(r.Context())
		if !ok || claims.Role != auth.RoleAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		if r.Body == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json_body"})
			return
		}
		var req struct {
			Items []BatchResolveItem `json:"items"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 256*1024)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json_body"})
			return
		}
		if len(req.Items) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items_required"})
			return
		}
		if len(req.Items) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "batch_limit_exceeded"})
			return
		}

		resolved := 0
		failed := make([]BatchResolveFailure, 0)
		results := make([]BatchResolveResult, 0, len(req.Items))
		for _, it := range req.Items {
			kind := strings.ToUpper(strings.TrimSpace(it.Kind))
			id := strings.TrimSpace(it.ID)
			if kind == "" || id == "" {
				errMsg := "kind_and_id_required"
				failed = append(failed, BatchResolveFailure{Kind: kind, ID: id, Error: errMsg})
				results = append(results, BatchResolveResult{Kind: kind, ID: id, Status: "failed", Error: errMsg})
				continue
			}
			_, err := resolveSingleException(r.Context(), deps, claims, kind, id, ResolveExceptionRequest{
				Note:             it.Note,
				CreditNoteID:     it.CreditNoteID,
				Action:           it.Action,
				OnecRef:          it.OnecRef,
				MergedIntoID:     it.MergedIntoID,
				TargetRetailerID: it.TargetRetailerID,
			})
			if err != nil {
				failed = append(failed, BatchResolveFailure{Kind: kind, ID: id, Error: err.Error()})
				results = append(results, BatchResolveResult{Kind: kind, ID: id, Status: "failed", Error: err.Error()})
				continue
			}
			resolved++
			results = append(results, BatchResolveResult{Kind: kind, ID: id, Status: "resolved"})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"resolved": resolved,
			"failed":   failed,
			"results":  results,
		})
	}
}

func resolveSingleException(ctx context.Context, deps ExceptionResolveDeps, claims auth.Claims, kind, id string, body ResolveExceptionRequest) (int, error) {
	if deps.ExtraResolveFn != nil {
		if code, err := deps.ExtraResolveFn(ctx, claims, kind, id, body); err != nil {
			if code == 0 {
				code = http.StatusBadRequest
			}
			return code, err
		} else if code == http.StatusOK {
			return http.StatusOK, nil
		}
	}

	switch kind {
	case "CASH_DISCREPANCY":
		if deps.CashRecon == nil {
			return http.StatusServiceUnavailable, errors.New("cash_recon_unavailable")
		}
		if err := deps.CashRecon.Accept(ctx, id, claims.Subject, body.Note); err != nil {
			return http.StatusBadRequest, err
		}
	case "CREDIT_NOTE_DRAFT":
		if deps.CreditNote == nil {
			return http.StatusServiceUnavailable, errors.New("credit_note_unavailable")
		}
		cnID := strings.TrimSpace(body.CreditNoteID)
		if cnID == "" {
			cnID = id
		}
		if err := deps.CreditNote.Issue(ctx, cnID, claims.Subject); err != nil {
			return http.StatusBadRequest, err
		}
	case "CREDIT_FREEZE":
		if deps.Credit == nil {
			return http.StatusServiceUnavailable, errors.New("credit_unavailable")
		}
		existing, found, err := deps.Credit.GetProfile(ctx, id, claims.SupplierID)
		if err != nil {
			return http.StatusInternalServerError, errors.New("load_profile_failed")
		}
		if !found {
			return http.StatusNotFound, errors.New("credit_profile_not_found")
		}
		existing.Status = credit.StatusActive
		if err := deps.Credit.UpsertProfile(ctx, existing, claims.Subject, body.Note); err != nil {
			return http.StatusBadRequest, err
		}
	case "EXPORT_REJECTED":
		postings := deps.OnecPostings
		if postings == nil && deps.Spanner != nil {
			postings = partner.NewSpannerOnecPostingRepository(deps.Spanner)
		}
		if postings == nil {
			return http.StatusServiceUnavailable, errors.New("onec_postings_unavailable")
		}
		now := time.Now().UTC()
		act := strings.ToLower(strings.TrimSpace(body.Action))
		var resolveErr error
		switch act {
		case "requeue", "retry":
			_, resolveErr = postings.Requeue(ctx, claims.SupplierID, id, now)
		case "", "ack", "acknowledge", "close", "resolve":
			ref := strings.TrimSpace(body.OnecRef)
			if ref == "" {
				if note := strings.TrimSpace(body.Note); note != "" {
					ref = "desk:" + note
				} else {
					ref = "desk:resolved"
				}
			}
			_, resolveErr = postings.Ack(ctx, claims.SupplierID, id, ref, now)
		default:
			return http.StatusBadRequest, errors.New("unsupported_export_resolve_action")
		}
		if resolveErr != nil {
			if errors.Is(resolveErr, partner.ErrOnecPostingNotFound) {
				return http.StatusNotFound, resolveErr
			}
			return http.StatusBadRequest, resolveErr
		}
	case "TIN_DEDUP_CANDIDATE":
		links := deps.RetailerLinks
		if links == nil && deps.Spanner != nil {
			links = entityresolution.NewSpannerRetailerLinkRepository(deps.Spanner)
		}
		if links == nil {
			return http.StatusServiceUnavailable, errors.New("retailer_links_unavailable")
		}
		mergedInto := strings.TrimSpace(body.MergedIntoID)
		if mergedInto == "" {
			mergedInto = strings.TrimSpace(body.TargetRetailerID)
		}
		act := strings.TrimSpace(body.Action)
		if act == "" {
			if mergedInto != "" {
				act = "merge_into"
			} else {
				act = "mark_clean"
			}
		}
		if _, _, err := links.Resolve(ctx, claims.SupplierID, id, act, mergedInto, time.Now().UTC()); err != nil {
			return mapRetailerLinkErrorStatus(err), err
		}
	case "FISCAL_FAILED", "FISCAL_DEAD_LETTER", "FISCAL_TIMEOUT", "FISCAL_FAILURE":
		act := strings.ToLower(strings.TrimSpace(body.Action))
		switch act {
		case "commercial_override", "override", "resolve", "close":
			if deps.Spanner == nil {
				if deps.Service == nil {
					return http.StatusServiceUnavailable, errors.New("fiscal_override_unavailable")
				}
				deps.Service.markInMemoryOrderExceptionResolved(kind, id)
				return http.StatusOK, nil
			}
			_, err := deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				if _, uErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE Orders
						SET FiscalStatus = 'COMMERCIAL_OVERRIDE',
						    Status = CASE WHEN Status = 'FISCAL_FAILED' THEN 'COMPLETED' ELSE Status END,
						    UpdatedAt = CURRENT_TIMESTAMP()
						WHERE OrderId = @oid AND SupplierId = @sid`,
					Params: map[string]any{"oid": id, "sid": claims.SupplierID},
				}); uErr != nil {
					return uErr
				}
				_, tErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE ExceptionTickets
						SET Status = 'RESOLVED'
						WHERE OrderId = @oid AND UPPER(Status) IN ('OPEN', 'PENDING')`,
					Params: map[string]any{"oid": id},
				})
				return tErr
			})
			if err != nil {
				return http.StatusInternalServerError, err
			}
		case "", "retry", "requeue":
			if deps.FiscalOrders == nil {
				return http.StatusServiceUnavailable, errors.New("fiscal_service_unavailable")
			}
			if _, err := deps.FiscalOrders.RetryFiscal(ctx, claims, id); err != nil {
				return http.StatusBadRequest, err
			}
			if deps.Spanner != nil {
				_, _ = deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
					_, uErr := txn.Update(txCtx, spanner.Statement{
						SQL: `UPDATE ExceptionTickets SET Status = 'RESOLVED'
							WHERE OrderId = @oid AND UPPER(Status) IN ('OPEN', 'PENDING')`,
						Params: map[string]any{"oid": id},
					})
					return uErr
				})
			}
		default:
			return http.StatusBadRequest, errors.New("unsupported_fiscal_resolve_action")
		}
	case "CLAIM_REVIEW", "BUYER_REJECTED":
		newStatus := "RESOLVED"
		switch strings.ToLower(strings.TrimSpace(body.Action)) {
		case "approve", "accept":
			newStatus = "APPROVED"
		case "reject", "deny":
			newStatus = "REJECTED"
		}
		if deps.Spanner != nil {
			_, err := deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, uErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE Claims
						SET Status = @st, UpdatedAt = CURRENT_TIMESTAMP()
						WHERE SupplierId = @sid AND (ClaimId = @id OR OrderId = @id)
						  AND UPPER(Status) IN ('OPEN', 'PENDING', 'UNDER_REVIEW', 'ESCALATED')`,
					Params: map[string]any{"sid": claims.SupplierID, "id": id, "st": newStatus},
				})
				return uErr
			})
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else if deps.Service != nil {
			deps.Service.markInMemoryOrderExceptionResolved(kind, id)
		} else {
			return http.StatusServiceUnavailable, errors.New("claims_service_unavailable")
		}
	case "PROMISE_BREACH":
		newStatus := "RESOLVED"
		switch strings.ToLower(strings.TrimSpace(body.Action)) {
		case "waive":
			newStatus = "WAIVED"
		case "extend", "reopen":
			newStatus = "ACTIVE"
		}
		if deps.Spanner != nil {
			_, err := deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, uErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE ArPaymentPromises
						SET Status = @st, UpdatedAt = CURRENT_TIMESTAMP()
						WHERE SupplierId = @sid AND PromiseId = @id`,
					Params: map[string]any{"sid": claims.SupplierID, "id": id, "st": newStatus},
				})
				return uErr
			})
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else if deps.Service != nil {
			deps.Service.markInMemoryOrderExceptionResolved(kind, id)
		} else {
			return http.StatusServiceUnavailable, errors.New("promises_service_unavailable")
		}
	case "SHOP_CLOSED":
		resolution := strings.ToUpper(strings.TrimSpace(body.Action))
		if resolution == "" {
			resolution = "RESOLVED"
		}
		if deps.Spanner != nil {
			_, err := deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, uErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE ShopClosedAttempts
						SET Resolution = @res, ResolvedAt = CURRENT_TIMESTAMP(), ResolvedBy = @actor
						WHERE (AttemptId = @id OR OrderId = @id) AND (Resolution IS NULL OR Resolution = '')`,
					Params: map[string]any{"res": resolution, "actor": claims.Subject, "id": id},
				})
				return uErr
			})
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else if deps.Service != nil {
			deps.Service.markInMemoryOrderExceptionResolved(kind, id)
		} else {
			return http.StatusServiceUnavailable, errors.New("shop_closed_service_unavailable")
		}
	case "MANIFEST_EXCEPTION":
		if deps.Spanner != nil {
			_, err := deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, uErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE ManifestExceptions
						SET ResolvedAt = CURRENT_TIMESTAMP()
						WHERE SupplierId = @sid AND (ExceptionId = @id OR OrderId = @id) AND ResolvedAt IS NULL`,
					Params: map[string]any{"sid": claims.SupplierID, "id": id},
				})
				return uErr
			})
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else if deps.Service != nil {
			deps.Service.markInMemoryOrderExceptionResolved(kind, id)
		} else {
			return http.StatusServiceUnavailable, errors.New("manifest_service_unavailable")
		}
	case "ORDER_CANCELLED", "VET_REJECTED", "AWAITING_REVIEW", "OPERATOR_NOTE":
		if deps.Spanner != nil {
			_, err := deps.Spanner.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				if kind == "OPERATOR_NOTE" {
					if _, uErr := txn.Update(txCtx, spanner.Statement{
						SQL: `UPDATE Orders SET WarehouseNotes = NULL, UpdatedAt = CURRENT_TIMESTAMP()
							WHERE SupplierId = @sid AND OrderId = @id`,
						Params: map[string]any{"sid": claims.SupplierID, "id": id},
					}); uErr != nil {
						return uErr
					}
				}
				_, tErr := txn.Update(txCtx, spanner.Statement{
					SQL: `UPDATE ExceptionTickets SET Status = 'RESOLVED'
						WHERE OrderId = @id AND UPPER(Status) IN ('OPEN', 'PENDING')`,
					Params: map[string]any{"id": id},
				})
				return tErr
			})
			if err != nil {
				return http.StatusInternalServerError, err
			}
		} else if deps.Service != nil {
			deps.Service.markInMemoryOrderExceptionResolved(kind, id)
		} else {
			return http.StatusServiceUnavailable, errors.New("order_exception_resolver_unavailable")
		}
	default:
		return http.StatusBadRequest, errors.New("unsupported_exception_kind")
	}
	return http.StatusOK, nil
}

func mapRetailerLinkErrorStatus(err error) int {
	switch {
	case errors.Is(err, entityresolution.ErrRetailerLinkNotFound):
		return http.StatusNotFound
	case errors.Is(err, entityresolution.ErrInvalidRetailerLinkInput),
		errors.Is(err, entityresolution.ErrRetailerMergeTargetNotFound):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
