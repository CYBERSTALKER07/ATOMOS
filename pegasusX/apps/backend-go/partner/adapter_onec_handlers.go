package partner

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/partner/adapters/onec"
)

// HandleOneCImport POST /partner/v1/adapters/onec/import
// Accepts JSON ImportBatch or raw CommerceML-like XML (Content-Type application/xml).
func (h *Handlers) HandleOneCImport(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !requireDialect(w, r, p.TenantType, p.TenantID, DialectOneC) {
		return
	}
	if h == nil || h.Svc == nil || h.Svc.OnecPostings() == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "onec_postings_unavailable")
		return
	}
	body, err := readMasterDataBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	var batch onec.ImportBatch
	if strings.Contains(ct, "xml") || strings.HasPrefix(strings.TrimSpace(string(body)), "<?xml") || strings.Contains(string(body), "<Каталог") {
		batch, err = onec.ParseCommerceMLCatalog(body)
		if err != nil {
			writePartnerError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		if err := json.Unmarshal(body, &batch); err != nil {
			writePartnerError(w, http.StatusBadRequest, "invalid_json")
			return
		}
	}
	extDoc := strings.TrimSpace(batch.ExternalDocID)
	if extDoc == "" {
		extDoc = "onec-import"
	}

	postings := h.Svc.OnecPostings()
	now := time.Now().UTC()
	payloadBytes, marshalErr := json.Marshal(map[string]any{
		"external_doc_id": extDoc,
		"product_count":   len(batch.Products),
		"tenant_type":     p.TenantType,
	})
	if marshalErr != nil {
		payloadBytes = []byte(`{}`)
	}
	docKey := "import:" + extDoc
	posting, created, err := postings.CreateIfAbsent(r.Context(), OnecPosting{
		SupplierID:   p.TenantID,
		DocumentKey:  docKey,
		DocumentType: OnecDocImportBatch,
		PayloadJSON:  string(payloadBytes),
		Status:       OnecStatusImported,
		OnecRef:      extDoc,
		AckedAt:      &now,
	})
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "onec_idempotency_failed")
		return
	}
	if !created {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":          "duplicate",
			"external_doc_id": extDoc,
			"posting_id":      posting.PostingID,
			"adapter":         onec.AdapterName,
		})
		return
	}

	market := marketCodeForPartner(r, p.TenantType, p.TenantID)
	batch.Products = applyPackCurrency(batch.Products, market)
	items := make([]ProductUpsertItem, 0, len(batch.Products))
	for _, pr := range batch.Products {
		active := true
		items = append(items, ProductUpsertItem{
			ExternalID: pr.ExternalID,
			Name:       pr.Name,
			Barcode:    pr.Barcode,
			PriceMinor: pr.PriceMinor,
			Currency:   pr.Currency,
			IsActive:   &active,
			Unit:       "EA",
			CategoryID: "imported",
		})
	}
	results, err := h.Svc.UpsertProducts(r.Context(), p, items)
	if err != nil {
		if delErr := postings.Delete(r.Context(), p.TenantID, posting.PostingID); delErr != nil && h.Svc.log != nil {
			h.Svc.log.Warn("cleanup failed onec import posting", "posting_id", posting.PostingID, "err", delErr)
		}
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	// Optional price pass
	prices := make([]PriceUpsertItem, 0)
	for _, pr := range batch.Products {
		if pr.PriceMinor > 0 {
			prices = append(prices, PriceUpsertItem{
				ExternalID: pr.ExternalID,
				PriceMinor: pr.PriceMinor,
				Currency:   pr.Currency,
			})
		}
	}
	var priceResults []PriceUpsertResult
	var priceErrMsg string
	if len(prices) > 0 {
		var priceErr error
		priceResults, priceErr = h.Svc.UpsertPrices(r.Context(), p, prices)
		if priceErr != nil {
			priceErrMsg = priceErr.Error()
			if h.Svc.log != nil {
				h.Svc.log.Warn("1c import price pass failed", "supplier_id", p.TenantID, "err", priceErr)
			}
		}
	}
	resp := map[string]any{
		"status":          "accepted",
		"adapter":         onec.AdapterName,
		"external_doc_id": extDoc,
		"posting_id":      posting.PostingID,
		"products":        results,
		"prices":          priceResults,
		"journal_dialect": onec.JournalDialect,
		"note":            "Journals export remains POST /partner/v1/exports resource=journals format=xml",
	}
	if priceErrMsg != "" {
		resp["price_warning"] = priceErrMsg
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleOneCPending handles GET /partner/v1/adapters/onec/pending and GET /v1/supplier/partner-onec/pending.
func (h *Handlers) HandleOneCPending(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	if tt != TenantSupplier || strings.TrimSpace(tid) == "" {
		writePartnerError(w, http.StatusForbidden, "supplier_tenant_required")
		return
	}
	if h == nil || h.Svc == nil || h.Svc.OnecPostings() == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "onec_postings_unavailable")
		return
	}
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = OnecStatusPending
	}
	switch status {
	case OnecStatusPending, OnecStatusAcked, OnecStatusRejected, OnecStatusImported, "ALL":
	default:
		writePartnerError(w, http.StatusBadRequest, "invalid_status")
		return
	}
	limit := defaultOnecListLim
	if rawLim := strings.TrimSpace(r.URL.Query().Get("limit")); rawLim != "" {
		parsed, err := strconv.Atoi(rawLim)
		if err != nil || parsed <= 0 {
			writePartnerError(w, http.StatusBadRequest, "invalid_limit")
			return
		}
		if parsed > maxOnecListLim {
			parsed = maxOnecListLim
		}
		limit = parsed
	}
	postings, err := h.Svc.OnecPostings().ListBySupplier(r.Context(), tid, status, limit)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "onec_postings_list_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"postings": postings,
		"count":    len(postings),
	})
}

// HandleOneCAck handles POST /partner/v1/adapters/onec/ack and POST /v1/supplier/partner-onec/ack.
func (h *Handlers) HandleOneCAck(w http.ResponseWriter, r *http.Request) {
	if rejectPartnerAuditorOrAgent(w, r) {
		return
	}
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	if tt != TenantSupplier || strings.TrimSpace(tid) == "" {
		writePartnerError(w, http.StatusForbidden, "supplier_tenant_required")
		return
	}
	if h == nil || h.Svc == nil || h.Svc.OnecPostings() == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "onec_postings_unavailable")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		PostingID   string `json:"posting_id"`
		DocumentKey string `json:"document_key"`
		OnecRef     string `json:"onec_ref"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	target := firstNonEmpty(req.PostingID, req.DocumentKey)
	if target == "" {
		writePartnerError(w, http.StatusBadRequest, "posting_id_or_document_key_required")
		return
	}
	posting, err := h.Svc.OnecPostings().Ack(r.Context(), tid, target, req.OnecRef, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, ErrOnecPostingNotFound):
			writePartnerError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidOnecInput):
			writePartnerError(w, http.StatusBadRequest, err.Error())
		default:
			writePartnerError(w, http.StatusInternalServerError, "onec_ack_failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "acked",
		"posting": posting,
	})
}

// HandleOneCReject handles POST /partner/v1/adapters/onec/reject and POST /v1/supplier/partner-onec/reject.
func (h *Handlers) HandleOneCReject(w http.ResponseWriter, r *http.Request) {
	if rejectPartnerAuditorOrAgent(w, r) {
		return
	}
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	if tt != TenantSupplier || strings.TrimSpace(tid) == "" {
		writePartnerError(w, http.StatusForbidden, "supplier_tenant_required")
		return
	}
	if h == nil || h.Svc == nil || h.Svc.OnecPostings() == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "onec_postings_unavailable")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		PostingID     string `json:"posting_id"`
		DocumentKey   string `json:"document_key"`
		Reason        string `json:"reason"`
		RejectReason  string `json:"reject_reason"`
		ExposureMinor *int64 `json:"exposure_minor"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	target := firstNonEmpty(req.PostingID, req.DocumentKey)
	if target == "" {
		writePartnerError(w, http.StatusBadRequest, "posting_id_or_document_key_required")
		return
	}
	if req.ExposureMinor != nil && *req.ExposureMinor < 0 {
		writePartnerError(w, http.StatusBadRequest, "invalid_exposure_minor")
		return
	}
	reason := firstNonEmpty(req.RejectReason, req.Reason)
	posting, err := h.Svc.OnecPostings().Reject(r.Context(), tid, target, reason, req.ExposureMinor, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, ErrOnecPostingNotFound):
			writePartnerError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrInvalidOnecInput):
			writePartnerError(w, http.StatusBadRequest, err.Error())
		default:
			writePartnerError(w, http.StatusInternalServerError, "onec_reject_failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "rejected",
		"posting": posting,
	})
}

func applyPackCurrency(products []onec.ImportProduct, market string) []onec.ImportProduct {
	packCcy := packCurrencyOrEmpty(market)
	out := make([]onec.ImportProduct, 0, len(products))
	for _, pr := range products {
		if strings.TrimSpace(pr.Currency) == "" {
			pr.Currency = packCcy
		}
		out = append(out, pr)
	}
	return out
}
