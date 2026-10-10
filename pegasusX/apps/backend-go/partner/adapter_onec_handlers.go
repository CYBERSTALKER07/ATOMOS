package partner

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/pegasusx/pegasusx/apps/backend-go/partner/adapters/onec"
)

var (
	onecDocMu sync.Mutex
	onecSeen  = map[string]bool{} // tenant|doc
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
	key := p.TenantType + "|" + p.TenantID + "|" + extDoc
	onecDocMu.Lock()
	if onecSeen[key] {
		onecDocMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "duplicate", "external_doc_id": extDoc, "adapter": onec.AdapterName,
		})
		return
	}
	onecSeen[key] = true
	onecDocMu.Unlock()

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
	if len(prices) > 0 {
		priceResults, _ = h.Svc.UpsertPrices(r.Context(), p, prices)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "accepted",
		"adapter":         onec.AdapterName,
		"external_doc_id": extDoc,
		"products":        results,
		"prices":          priceResults,
		"journal_dialect": onec.JournalDialect,
		"note":            "Journals export remains POST /partner/v1/exports resource=journals format=xml",
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
