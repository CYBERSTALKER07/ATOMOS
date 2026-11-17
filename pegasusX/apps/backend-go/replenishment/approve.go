package replenishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/grpc/codes"
)

const SuggestionStatusApproved = "APPROVED"

// MemorySuggestionStore supports unit testing HandleApprove without live Spanner.
type MemorySuggestionStore struct {
	mu          sync.Mutex
	suggestions map[string]ReorderSuggestion // retailerID|sku -> row
	linked      map[string]bool              // supplierID|retailerID -> true
}

// NewMemorySuggestionStore constructs an in-memory suggestion store for tests.
func NewMemorySuggestionStore() *MemorySuggestionStore {
	return &MemorySuggestionStore{
		suggestions: make(map[string]ReorderSuggestion),
		linked:      make(map[string]bool),
	}
}

// Seed adds a suggestion row and optional supplier link to the memory store.
func (m *MemorySuggestionStore) Seed(supplierID string, s ReorderSuggestion) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suggestions[s.RetailerId+"|"+s.Sku] = s
	if strings.TrimSpace(supplierID) != "" {
		m.linked[supplierID+"|"+s.RetailerId] = true
	}
}

// SetMemoryStore attaches a MemorySuggestionStore for unit tests without Spanner.
func (a *SuggestionsAPI) SetMemoryStore(m *MemorySuggestionStore) {
	if a != nil {
		a.memStore = m
	}
}

// ApproveSuggestionRequest is the optional JSON body for POST /v1/replenishment/suggestions/{id}/approve.
type ApproveSuggestionRequest struct {
	RetailerID     string `json:"retailer_id,omitempty"`
	Sku            string `json:"sku,omitempty"`
	Qty            *int64 `json:"qty,omitempty"`
	Mode           string `json:"mode,omitempty"` // "order" (default) | "approve_only"
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// ApproveSuggestionResponse is returned by POST /v1/replenishment/suggestions/{id}/approve.
type ApproveSuggestionResponse struct {
	SuggestionID    string `json:"suggestion_id"`
	RetailerID      string `json:"retailer_id"`
	Sku             string `json:"sku"`
	Status          string `json:"status"`
	ApprovedQty     int64  `json:"approved_qty"`
	OrderID         string `json:"order_id,omitempty"`
	ApprovedBy      string `json:"approved_by"`
	AlreadyApproved bool   `json:"already_approved,omitempty"`
}

// HandleApprove serves POST /v1/replenishment/suggestions/{id}/approve (one-tap approve).
// Accepts {id} as "{retailerId}:{sku}" (or "{sku}" for authenticated retailer callers).
// Rejects AGENT and AUDITOR actors cleanly.
func (a *SuggestionsAPI) HandleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if auth.IsAgentActor(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "agent_approval_forbidden"})
		return
	}
	if auth.IsReadOnlyAuditor(claims) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "auditor_read_only"})
		return
	}

	var isRetailer bool
	switch claims.Role {
	case auth.RoleRetailer:
		if !auth.HasRetailerPerm(claims, auth.PermOrderPlace) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		isRetailer = true
	case auth.RoleAdmin, auth.RolePlatformAdmin, auth.RoleWarehouseAdmin:
		if (claims.SupplierRole != "" || len(claims.SupplierPerms) > 0) &&
			!auth.HasSupplierPerm(claims, auth.PermAgentProposalApprove) &&
			!auth.HasSupplierPerm(claims, auth.PermDraftApprove) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "missing_permission"})
			return
		}
	default:
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	if a == nil || (a.Spanner == nil && a.memStore == nil) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "suggestions_unavailable"})
		return
	}

	var req ApproveSuggestionRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, context.Canceled) && err.Error() != "EOF" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	}

	rawID := strings.TrimSpace(chi.URLParam(r, "id"))
	if unescaped, uErr := url.PathUnescape(rawID); uErr == nil && strings.TrimSpace(unescaped) != "" {
		rawID = strings.TrimSpace(unescaped)
	}

	callerRetailerID := ""
	if isRetailer {
		callerRetailerID = auth.ResolveRetailerOrgID(claims)
	}

	retailerID, sku := resolveSuggestionIDParts(rawID, strings.TrimSpace(req.RetailerID), strings.TrimSpace(req.Sku), callerRetailerID)
	if retailerID == "" || sku == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_and_sku_required"})
		return
	}
	if isRetailer && callerRetailerID != "" && retailerID != callerRetailerID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden_retailer_scope"})
		return
	}
	if req.Qty != nil && *req.Qty <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "qty_must_be_positive"})
		return
	}

	supplierID := ""
	if !isRetailer && a.SupplierID != nil {
		supplierID = strings.TrimSpace(a.SupplierID(r))
	}
	if supplierID == "" {
		supplierID = strings.TrimSpace(auth.PreferTenantSupplierID(r.Context(), claims.SupplierID))
	}
	if !isRetailer && supplierID == "" && claims.Role != auth.RolePlatformAdmin {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	actorID := auth.ActorLabel(claims)
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "order"
	}

	if a.memStore != nil {
		resp, statusCode, errCode := a.approveInMemory(supplierID, retailerID, sku, actorID, isRetailer, mode, req.Qty)
		if errCode != "" {
			writeJSON(w, statusCode, map[string]string{"error": errCode})
			return
		}
		audit.SetMutationContext(r.Context(), audit.MutationContext{
			TenantID:   firstNonEmptyRepl(supplierID, retailerID),
			Action:     "replenishment.suggestion.approved",
			ObjectType: "ReorderSuggestion",
			ObjectID:   resp.SuggestionID,
		})
		writeJSON(w, statusCode, resp)
		return
	}

	if !isRetailer && supplierID != "" && !a.retailerLinkedToSupplierStrong(r.Context(), supplierID, retailerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "retailer_not_linked"})
		return
	}

	sug, err := a.loadSuggestionStrong(r.Context(), retailerID, sku)
	if err != nil {
		if errors.Is(err, errSuggestionNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "suggestion_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_suggestion_failed"})
		return
	}
	if sug.Status == SuggestionStatusDismissed {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "suggestion_dismissed"})
		return
	}
	if sug.Status == SuggestionStatusConverted || sug.Status == SuggestionStatusApproved {
		writeJSON(w, http.StatusOK, ApproveSuggestionResponse{
			SuggestionID:    retailerID + ":" + sku,
			RetailerID:      retailerID,
			Sku:             sku,
			Status:          sug.Status,
			ApprovedQty:     sug.SuggestedQty,
			ApprovedBy:      actorID,
			AlreadyApproved: true,
		})
		return
	}

	approvedQty := sug.SuggestedQty
	if req.Qty != nil && *req.Qty > 0 {
		approvedQty = *req.Qty
	}
	if approvedQty <= 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "suggested_qty_zero"})
		return
	}

	var orderID string
	nextStatus := SuggestionStatusApproved
	httpStatus := http.StatusOK

	if mode != "approve_only" && a.Orders != nil {
		if lat, lng, h3, locErr := a.loadRetailerLocationStrong(r.Context(), retailerID); locErr == nil {
			deliveryDate := sug.SuggestedByDate.In(time.UTC)
			createReq := order.CreateRequest{
				LineItems: []order.LineItem{{
					SKU:      sku,
					Quantity: approvedQty,
				}},
				H3Cell:                h3,
				Lat:                   lat,
				Lng:                   lng,
				DeliveryMode:          order.DeliveryModeScheduled,
				RequestedDeliveryDate: deliveryDate.Format(time.RFC3339),
			}
			ordResp, oErr := a.Orders.Create(r.Context(), retailerID, createReq)
			if oErr != nil {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": oErr.Error()})
				return
			}
			orderID = ordResp.OrderID
			nextStatus = SuggestionStatusConverted
			httpStatus = http.StatusCreated
		}
	}

	_, updErr := a.Spanner.ReadWriteTransaction(r.Context(), func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ReorderSuggestions", map[string]any{
				"RetailerId":   retailerID,
				"Sku":          sku,
				"SuggestedQty": approvedQty,
				"Status":       nextStatus,
			}),
		}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(txCtx, buf, "ReorderSuggestion", retailerID+":"+sku, events.TopicMain, map[string]any{
			"type":         "replenishment.suggestion.approved",
			"supplier_id":  supplierID,
			"retailer_id":  retailerID,
			"sku":          sku,
			"approved_qty": approvedQty,
			"order_id":     orderID,
			"status":       nextStatus,
			"approved_by":  actorID,
		}); err != nil {
			return err
		}
		return buf.Flush(txCtx)
	})
	if updErr != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "approve_failed"})
		return
	}

	audit.SetMutationContext(r.Context(), audit.MutationContext{
		TenantID:   firstNonEmptyRepl(supplierID, retailerID),
		Action:     "replenishment.suggestion.approved",
		ObjectType: "ReorderSuggestion",
		ObjectID:   retailerID + ":" + sku,
	})

	writeJSON(w, httpStatus, ApproveSuggestionResponse{
		SuggestionID: retailerID + ":" + sku,
		RetailerID:   retailerID,
		Sku:          sku,
		Status:       nextStatus,
		ApprovedQty:  approvedQty,
		OrderID:      orderID,
		ApprovedBy:   actorID,
	})
}

func (a *SuggestionsAPI) approveInMemory(supplierID, retailerID, sku, actorID string, isRetailer bool, mode string, qtyOverride *int64) (ApproveSuggestionResponse, int, string) {
	a.memStore.mu.Lock()
	defer a.memStore.mu.Unlock()
	if !isRetailer && supplierID != "" && !a.memStore.linked[supplierID+"|"+retailerID] {
		return ApproveSuggestionResponse{}, http.StatusForbidden, "retailer_not_linked"
	}
	key := retailerID + "|" + sku
	sug, ok := a.memStore.suggestions[key]
	if !ok {
		return ApproveSuggestionResponse{}, http.StatusNotFound, "suggestion_not_found"
	}
	if sug.Status == SuggestionStatusDismissed {
		return ApproveSuggestionResponse{}, http.StatusConflict, "suggestion_dismissed"
	}
	if sug.Status == SuggestionStatusConverted || sug.Status == SuggestionStatusApproved {
		return ApproveSuggestionResponse{
			SuggestionID:    retailerID + ":" + sku,
			RetailerID:      retailerID,
			Sku:             sku,
			Status:          sug.Status,
			ApprovedQty:     sug.SuggestedQty,
			ApprovedBy:      actorID,
			AlreadyApproved: true,
		}, http.StatusOK, ""
	}
	qty := sug.SuggestedQty
	if qtyOverride != nil && *qtyOverride > 0 {
		qty = *qtyOverride
	}
	if qty <= 0 {
		return ApproveSuggestionResponse{}, http.StatusUnprocessableEntity, "suggested_qty_zero"
	}
	nextStatus := SuggestionStatusApproved
	statusCode := http.StatusOK
	if mode != "approve_only" && a.Orders != nil {
		nextStatus = SuggestionStatusConverted
		statusCode = http.StatusCreated
	}
	sug.SuggestedQty = qty
	sug.Status = nextStatus
	a.memStore.suggestions[key] = sug
	return ApproveSuggestionResponse{
		SuggestionID: retailerID + ":" + sku,
		RetailerID:   retailerID,
		Sku:          sku,
		Status:       nextStatus,
		ApprovedQty:  qty,
		ApprovedBy:   actorID,
	}, statusCode, ""
}

func resolveSuggestionIDParts(rawID, bodyRetailerID, bodySKU, callerRetailerID string) (retailerID, sku string) {
	if callerRetailerID != "" && strings.HasPrefix(rawID, callerRetailerID+":") {
		retailerID = callerRetailerID
		sku = strings.TrimPrefix(rawID, callerRetailerID+":")
	} else if idx := strings.Index(rawID, ":"); idx > 0 && idx < len(rawID)-1 {
		retailerID = strings.TrimSpace(rawID[:idx])
		sku = strings.TrimSpace(rawID[idx+1:])
	} else if idx := strings.Index(rawID, "|"); idx > 0 && idx < len(rawID)-1 {
		retailerID = strings.TrimSpace(rawID[:idx])
		sku = strings.TrimSpace(rawID[idx+1:])
	} else if rawID != "" {
		sku = rawID
		if callerRetailerID != "" {
			retailerID = callerRetailerID
		}
	}
	if bodyRetailerID != "" {
		retailerID = bodyRetailerID
	}
	if bodySKU != "" {
		sku = bodySKU
	}
	return strings.TrimSpace(retailerID), strings.TrimSpace(sku)
}

func (a *SuggestionsAPI) retailerLinkedToSupplierStrong(ctx context.Context, supplierID, retailerID string) bool {
	if strings.TrimSpace(retailerID) == "" || strings.TrimSpace(supplierID) == "" {
		return false
	}
	iter := a.Spanner.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT 1 FROM Orders WHERE SupplierId = @supplierId AND RetailerId = @retailerId LIMIT 1`,
		Params: map[string]any{
			"supplierId": supplierID,
			"retailerId": retailerID,
		},
	})
	defer iter.Stop()
	if _, err := iter.Next(); err == nil {
		return true
	}
	// Also check SupplierRetailerLinks or RetailerPaymentTerms for onboarding-stage retailers.
	if _, err := a.Spanner.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "SupplierRetailerLinks", spanner.Key{supplierID, retailerID}, []string{"RetailerId"}); err == nil {
		return true
	}
	if _, err := a.Spanner.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "RetailerPaymentTerms", spanner.Key{retailerID, supplierID}, []string{"RetailerId"}); err == nil {
		return true
	}
	return false
}

func (a *SuggestionsAPI) loadSuggestionStrong(ctx context.Context, retailerID, sku string) (ReorderSuggestion, error) {
	row, err := a.Spanner.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "ReorderSuggestions", spanner.Key{retailerID, sku},
		[]string{"RetailerId", "Sku", "SuggestedQty", "AdjustedDemand", "CurrentStock", "InFlightQty", "SafetyStock", "SuggestedByDate", "ComputedAt", "Status"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return ReorderSuggestion{}, errSuggestionNotFound
		}
		return ReorderSuggestion{}, err
	}
	var s ReorderSuggestion
	var suggestedBy civil.Date
	if err := row.Columns(&s.RetailerId, &s.Sku, &s.SuggestedQty, &s.AdjustedDemand, &s.CurrentStock, &s.InFlightQty, &s.SafetyStock, &suggestedBy, &s.ComputedAt, &s.Status); err != nil {
		return ReorderSuggestion{}, err
	}
	s.SuggestedByDate = suggestedBy
	return s, nil
}

func (a *SuggestionsAPI) loadRetailerLocationStrong(ctx context.Context, retailerID string) (lat, lng float64, h3 string, err error) {
	row, err := a.Spanner.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "Retailers", spanner.Key{retailerID}, []string{"Lat", "Lng", "H3Cell"})
	if err != nil {
		return 0, 0, "", err
	}
	var latCol, lngCol spanner.NullFloat64
	var h3Col spanner.NullString
	if err := row.Columns(&latCol, &lngCol, &h3Col); err != nil {
		return 0, 0, "", err
	}
	if !latCol.Valid || !lngCol.Valid {
		return 0, 0, "", errRetailerLocationMissing
	}
	return latCol.Float64, lngCol.Float64, strings.TrimSpace(h3Col.StringVal), nil
}

func firstNonEmptyRepl(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

// Keep unused fmt import check clean if needed.
var _ = fmt.Sprintf
