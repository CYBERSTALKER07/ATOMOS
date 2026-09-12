package globalproducts

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/internal/web"
)

// HandleGetGlobal serves GET /v1/global-products/{id}
func (s *Service) HandleGetGlobal(w http.ResponseWriter, r *http.Request) {
	if !Enabled() {
		web.JSONError(w, "global products disabled", http.StatusNotFound)
		return
	}
	id := chi.URLParam(r, "id")
	gp, err := s.GetGlobal(r.Context(), id)
	if err != nil {
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if gp == nil {
		web.JSONError(w, "not found", http.StatusNotFound)
		return
	}
	web.JSONResponse(w, http.StatusOK, gp)
}

// HandleListOffers serves GET /v1/global-products/{id}/offers
func (s *Service) HandleListOffers(w http.ResponseWriter, r *http.Request) {
	if !Enabled() {
		web.JSONError(w, "global products disabled", http.StatusNotFound)
		return
	}
	id := chi.URLParam(r, "id")
	offers, err := s.ListOffers(r.Context(), id)
	if err != nil {
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if offers == nil {
		offers = []Offer{}
	}
	web.JSONResponse(w, http.StatusOK, map[string]any{"global_product_id": id, "offers": offers})
}

// HandleLinkProduct serves POST /v1/supplier/products/{productId}/link-global
func (s *Service) HandleLinkProduct(w http.ResponseWriter, r *http.Request) {
	if !Enabled() {
		web.JSONError(w, "global products disabled", http.StatusNotFound)
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || supplierID == "" {
		web.JSONError(w, "supplier_scope_required", http.StatusForbidden)
		return
	}
	productID := chi.URLParam(r, "productId")
	var body struct {
		GlobalProductID string `json:"global_product_id"`
		Name            string `json:"name"`
		Brand           string `json:"brand"`
		Barcode         string `json:"barcode"`
		PriceMinor      int64  `json:"price_minor"`
		Currency        string `json:"currency"`
		UnitsPerPack    int64  `json:"units_per_pack"`
		UomCode         string `json:"uom_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		web.JSONError(w, "invalid body", http.StatusBadRequest)
		return
	}
	in := ProductInput{
		ProductID:    productID,
		SupplierID:   supplierID,
		Name:         body.Name,
		Brand:        body.Brand,
		Barcode:      body.Barcode,
		PriceMinor:   body.PriceMinor,
		Currency:     body.Currency,
		UnitsPerPack: body.UnitsPerPack,
		UomCode:      body.UomCode,
	}
	if strings.TrimSpace(in.Name) == "" {
		in.Name = productID
	}
	res, err := s.LinkExplicit(r.Context(), in, body.GlobalProductID)
	if err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	web.JSONResponse(w, http.StatusOK, res)
}

// HandleListMatchQueue serves GET /v1/admin/product-match-queue
func (s *Service) HandleListMatchQueue(w http.ResponseWriter, r *http.Request) {
	if !Enabled() {
		web.JSONError(w, "global products disabled", http.StatusNotFound)
		return
	}
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.ListMatchQueue(r.Context(), status, limit)
	if err != nil {
		web.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []MatchQueueItem{}
	}
	web.JSONResponse(w, http.StatusOK, map[string]any{"items": items})
}

// HandleResolveMatch serves POST /v1/admin/product-match-queue/{id}/resolve
func (s *Service) HandleResolveMatch(w http.ResponseWriter, r *http.Request) {
	if !Enabled() {
		web.JSONError(w, "global products disabled", http.StatusNotFound)
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		web.JSONError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	queueID := chi.URLParam(r, "id")
	var body struct {
		Decision           string `json:"decision"`
		GlobalProductID    string `json:"global_product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		web.JSONError(w, "invalid body", http.StatusBadRequest)
		return
	}
	actorSupplier := ""
	if claims.Role != auth.RoleAdmin {
		if sid, ok := auth.ResolveSupplierID(r.Context()); ok {
			actorSupplier = sid
		}
	}
	if err := s.ResolveMatch(r.Context(), queueID, body.Decision, actorSupplier, body.GlobalProductID); err != nil {
		web.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	web.JSONResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}
