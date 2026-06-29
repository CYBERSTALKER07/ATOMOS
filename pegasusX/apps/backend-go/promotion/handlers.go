package promotion

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

type createPromotionRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	DiscountBps         int64    `json:"discount_bps"`
	ScopeType           string   `json:"scope_type"`
	ScopeProductID      string   `json:"scope_product_id"`
	ScopeCategoryID     string   `json:"scope_category_id"`
	RetailerScope       string   `json:"retailer_scope"`
	RetailerIDs         []string `json:"retailer_ids"`
	MinLineQuantity     int64    `json:"min_line_quantity"`
	MinOrderAmountMinor int64    `json:"min_order_amount_minor"`
	StartsAt            *string  `json:"starts_at"`
	EndsAt              *string  `json:"ends_at"`
	Priority            int64    `json:"priority"`
}

type quoteRequest struct {
	SupplierID string      `json:"supplier_id"`
	Lines      []LineInput `json:"lines"`
}

// HandleListSupplierPromotions serves GET /v1/supplier/promotions.
func (s *Service) HandleListSupplierPromotions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	items, err := s.ListForSupplier(r.Context(), supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_promotions_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promotions": items})
}

// HandleCreateSupplierPromotion serves POST /v1/supplier/promotions.
func (s *Service) HandleCreateSupplierPromotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}

	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req createPromotionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	p, err := s.CreatePromotion(r.Context(), decodePromotionRequest(supplierID, "", req))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respBytes, _ := json.Marshal(p)
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleUpdateSupplierPromotion serves PATCH /v1/supplier/promotions/{promotionID}.
func (s *Service) HandleUpdateSupplierPromotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	promotionID := strings.TrimSpace(chi.URLParam(r, "promotionID"))

	body, ok := readMutationBody(w, r, 64*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req createPromotionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	p, err := s.UpdatePromotion(r.Context(), decodePromotionRequest(supplierID, promotionID, req))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respBytes, _ := json.Marshal(p)
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleDeactivateSupplierPromotion serves DELETE /v1/supplier/promotions/{promotionID}.
func (s *Service) HandleDeactivateSupplierPromotion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "supplier_scope_required"})
		return
	}
	promotionID := strings.TrimSpace(chi.URLParam(r, "promotionID"))

	body, ok := readMutationBody(w, r, 4*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	if err := s.DeactivatePromotion(r.Context(), supplierID, promotionID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respBytes, _ := json.Marshal(map[string]string{"status": "deactivated"})
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

// HandleCheckoutQuote serves POST /v1/retailer/checkout/quote.
func (s *Service) HandleCheckoutQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	defer r.Body.Close()
	if strings.TrimSpace(req.SupplierID) == "" || len(req.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_and_lines_required"})
		return
	}
	quote, err := s.QuoteCheckout(r.Context(), req.SupplierID, claims.Subject, req.Lines)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, quote)
}

func decodePromotionRequest(supplierID, promotionID string, req createPromotionRequest) Promotion {
	retailerScope := RetailerScopeAll
	if strings.EqualFold(req.RetailerScope, string(RetailerScopeAllowlist)) {
		retailerScope = RetailerScopeAllowlist
	}
	return Promotion{
		PromotionID:         promotionID,
		SupplierID:          supplierID,
		Name:                strings.TrimSpace(req.Name),
		Description:         strings.TrimSpace(req.Description),
		DiscountBps:         req.DiscountBps,
		ScopeType:           ScopeType(strings.ToUpper(strings.TrimSpace(req.ScopeType))),
		ScopeProductID:      strings.TrimSpace(req.ScopeProductID),
		ScopeCategoryID:     strings.TrimSpace(req.ScopeCategoryID),
		RetailerScope:       retailerScope,
		RetailerIDs:         req.RetailerIDs,
		MinLineQuantity:     req.MinLineQuantity,
		MinOrderAmountMinor: req.MinOrderAmountMinor,
		StartsAt:            parseTimePtr(req.StartsAt),
		EndsAt:              parseTimePtr(req.EndsAt),
		Priority:            req.Priority,
		IsActive:            true,
	}
}

func parseTimePtr(raw *string) *time.Time {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
	if err != nil {
		return nil
	}
	utc := t.UTC()
	return &utc
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
