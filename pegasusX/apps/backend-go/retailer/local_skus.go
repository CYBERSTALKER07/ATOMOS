package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

const localSKUPrefix = "local:"

// LocalSKU is a retailer-owned POS catalog row (non-Pegasus goods).
type LocalSKU struct {
	LocalSkuID        string `json:"local_sku_id"`
	Barcode           string `json:"barcode,omitempty"`
	Name              string `json:"name"`
	Unit              string `json:"unit,omitempty"`
	DefaultPriceMinor int64  `json:"default_price_minor"`
	Currency          string `json:"currency,omitempty"`
	SectionID         string `json:"section_id,omitempty"`
	IsActive          bool   `json:"is_active"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

// NormalizeLocalSKUID ensures local: prefix for demand isolation.
func NormalizeLocalSKUID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(id), localSKUPrefix) {
		return localSKUPrefix + strings.TrimSpace(id[len(localSKUPrefix):])
	}
	return localSKUPrefix + id
}

// IsLocalSKU reports whether sku is retailer-local (never supplier reorder).
func IsLocalSKU(sku string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(sku)), localSKUPrefix)
}

// HandleLocalSKUs serves GET/POST /v1/retailer/local-skus
func (s *Service) HandleLocalSKUs(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !auth.HasRetailerPerm(claims, auth.PermStockView) &&
		!auth.HasRetailerPerm(claims, auth.PermPosSell) &&
		!auth.HasRetailerPerm(claims, auth.PermCapManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		activeOnly := r.URL.Query().Get("active") != "0" && r.URL.Query().Get("active") != "false"
		items, err := s.listLocalSKUs(r.Context(), orgID, q, activeOnly)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed", "detail": err.Error()})
			return
		}
		if items == nil {
			items = []LocalSKU{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !auth.HasRetailerPerm(claims, auth.PermCapManage) &&
			!auth.HasRetailerPerm(claims, auth.PermStockView) {
			// Allow stock.view + pos for quick-add; prefer cap.manage for catalog admin
		}
		var req struct {
			LocalSkuID        string `json:"local_sku_id"`
			Barcode           string `json:"barcode"`
			Name              string `json:"name"`
			Unit              string `json:"unit"`
			DefaultPriceMinor int64  `json:"default_price_minor"`
			Currency          string `json:"currency"`
			SectionID         string `json:"section_id"`
			IsActive          *bool  `json:"is_active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "name_required"})
			return
		}
		id := strings.TrimSpace(req.LocalSkuID)
		if id == "" {
			id = s.newID()
		}
		id = NormalizeLocalSKUID(id)
		active := true
		if req.IsActive != nil {
			active = *req.IsActive
		}
		cur := strings.TrimSpace(req.Currency)
		if cur == "" {
			cur = "UZS"
		}
		row := LocalSKU{
			LocalSkuID:        id,
			Barcode:           strings.TrimSpace(req.Barcode),
			Name:              name,
			Unit:              strings.TrimSpace(req.Unit),
			DefaultPriceMinor: req.DefaultPriceMinor,
			Currency:          cur,
			SectionID:         strings.TrimSpace(req.SectionID),
			IsActive:          active,
		}
		if err := s.saveLocalSKU(r.Context(), orgID, row); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_failed", "detail": err.Error()})
			return
		}
		// Re-read for timestamps
		if got, ok, _ := s.getLocalSKU(r.Context(), orgID, id); ok {
			row = got
		}
		writeJSON(w, http.StatusCreated, row)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleLocalSKUByID serves PATCH /v1/retailer/local-skus/{localSkuID}
func (s *Service) HandleLocalSKUByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if !auth.HasRetailerPerm(claims, auth.PermStockView) &&
		!auth.HasRetailerPerm(claims, auth.PermCapManage) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	id := NormalizeLocalSKUID(chi.URLParam(r, "localSkuID"))
	if id == localSKUPrefix {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "local_sku_id_required"})
		return
	}
	cur, found, err := s.getLocalSKU(r.Context(), orgID, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	var req struct {
		Barcode           *string `json:"barcode"`
		Name              *string `json:"name"`
		Unit              *string `json:"unit"`
		DefaultPriceMinor *int64  `json:"default_price_minor"`
		Currency          *string `json:"currency"`
		SectionID         *string `json:"section_id"`
		IsActive          *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if req.Barcode != nil {
		cur.Barcode = strings.TrimSpace(*req.Barcode)
	}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "name_required"})
			return
		}
		cur.Name = n
	}
	if req.Unit != nil {
		cur.Unit = strings.TrimSpace(*req.Unit)
	}
	if req.DefaultPriceMinor != nil {
		cur.DefaultPriceMinor = *req.DefaultPriceMinor
	}
	if req.Currency != nil {
		cur.Currency = strings.TrimSpace(*req.Currency)
	}
	if req.SectionID != nil {
		cur.SectionID = strings.TrimSpace(*req.SectionID)
	}
	if req.IsActive != nil {
		cur.IsActive = *req.IsActive
	}
	if err := s.saveLocalSKU(r.Context(), orgID, cur); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_failed"})
		return
	}
	if got, ok, _ := s.getLocalSKU(r.Context(), orgID, id); ok {
		cur = got
	}
	writeJSON(w, http.StatusOK, cur)
}

// HandlePOSCatalogSearch serves GET /v1/retailer/pos/catalog?q=
// Returns local SKUs (and optionally stock-matched names) for POS search.
func (s *Service) HandlePOSCatalogSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermPosSell) && !auth.HasRetailerPerm(claims, auth.PermStockView)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	locals, _ := s.listLocalSKUs(r.Context(), orgID, q, true)
	type hit struct {
		SKU               string `json:"sku"`
		Name              string `json:"name"`
		Source            string `json:"source"` // local | store_stock
		DefaultPriceMinor int64  `json:"default_price_minor,omitempty"`
		Barcode           string `json:"barcode,omitempty"`
	}
	out := make([]hit, 0, len(locals)+8)
	for _, l := range locals {
		out = append(out, hit{
			SKU: l.LocalSkuID, Name: l.Name, Source: "local",
			DefaultPriceMinor: l.DefaultPriceMinor, Barcode: l.Barcode,
		})
	}
	// Union: store stock SKUs matching query (Pegasus-received)
	if loc, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
		balances, _ := s.listStockBalances(r.Context(), orgID, loc.LocationID)
		ql := strings.ToLower(q)
		seen := map[string]bool{}
		for _, h := range out {
			seen[h.SKU] = true
		}
		for _, b := range balances {
			if b.OnHand <= 0 || IsLocalSKU(b.Sku) {
				continue
			}
			if ql != "" && !strings.Contains(strings.ToLower(b.Sku), ql) {
				continue
			}
			if seen[b.Sku] {
				continue
			}
			seen[b.Sku] = true
			out = append(out, hit{SKU: b.Sku, Name: b.Sku, Source: "store_stock"})
			if len(out) >= 50 {
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

// validatePosSaleSKU returns normalized sku or error reason.
// Allows: local catalog (active), free-form with local: prefix after catalog upsert, or any non-empty for store stock sales.
func (s *Service) validatePosSaleSKU(ctx context.Context, orgID, sku, name string, unitPrice int64) (normalized string, lineName string, errMsg string) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return "", "", "sku_required"
	}
	// Explicit local catalog match by id or barcode
	if IsLocalSKU(sku) {
		row, ok, _ := s.getLocalSKU(ctx, orgID, NormalizeLocalSKUID(sku))
		if ok && row.IsActive {
			n := row.Name
			if strings.TrimSpace(name) != "" {
				n = strings.TrimSpace(name)
			}
			return row.LocalSkuID, n, ""
		}
		// Allow selling local: even if not in catalog (opening stock path) — still namespaced
		n := strings.TrimSpace(name)
		if n == "" {
			n = sku
		}
		return NormalizeLocalSKUID(sku), n, ""
	}
	// Barcode lookup in local catalog
	if row, ok := s.findLocalByBarcode(ctx, orgID, sku); ok && row.IsActive {
		n := row.Name
		if strings.TrimSpace(name) != "" {
			n = strings.TrimSpace(name)
		}
		return row.LocalSkuID, n, ""
	}
	// Pegasus / free-form store SKU (existing behavior)
	n := strings.TrimSpace(name)
	if n == "" {
		n = sku
	}
	return sku, n, ""
}

func (s *Service) listLocalSKUs(ctx context.Context, orgID, q string, activeOnly bool) ([]LocalSKU, error) {
	if s.spannerClient == nil {
		return s.listLocalSKUsMem(orgID, q, activeOnly), nil
	}
	sql := `SELECT LocalSkuId, COALESCE(Barcode, ''), Name, COALESCE(Unit, ''), DefaultPriceMinor,
		COALESCE(Currency, 'UZS'), COALESCE(SectionId, ''), IsActive,
		CAST(CreatedAt AS STRING), CAST(UpdatedAt AS STRING)
		FROM RetailerLocalCatalog WHERE RetailerId = @rid`
	params := map[string]any{"rid": orgID}
	if activeOnly {
		sql += ` AND IsActive = true`
	}
	sql += ` ORDER BY Name LIMIT 200`
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []LocalSKU
	ql := strings.ToLower(strings.TrimSpace(q))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Pre-migration
			return s.listLocalSKUsMem(orgID, q, activeOnly), nil
		}
		var item LocalSKU
		if err := row.Columns(
			&item.LocalSkuID, &item.Barcode, &item.Name, &item.Unit, &item.DefaultPriceMinor,
			&item.Currency, &item.SectionID, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if ql != "" {
			if !strings.Contains(strings.ToLower(item.Name), ql) &&
				!strings.Contains(strings.ToLower(item.LocalSkuID), ql) &&
				!strings.Contains(strings.ToLower(item.Barcode), ql) {
				continue
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) listLocalSKUsMem(orgID, q string, activeOnly bool) []LocalSKU {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.localCatalog == nil {
		return nil
	}
	rows := s.localCatalog[orgID]
	ql := strings.ToLower(strings.TrimSpace(q))
	var out []LocalSKU
	for _, item := range rows {
		if activeOnly && !item.IsActive {
			continue
		}
		if ql != "" {
			if !strings.Contains(strings.ToLower(item.Name), ql) &&
				!strings.Contains(strings.ToLower(item.LocalSkuID), ql) &&
				!strings.Contains(strings.ToLower(item.Barcode), ql) {
				continue
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) getLocalSKU(ctx context.Context, orgID, id string) (LocalSKU, bool, error) {
	id = NormalizeLocalSKUID(id)
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, r := range s.localCatalog[orgID] {
			if r.LocalSkuID == id {
				return r, true, nil
			}
		}
		return LocalSKU{}, false, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerLocalCatalog",
		spanner.Key{orgID, id},
		[]string{"LocalSkuId", "Barcode", "Name", "Unit", "DefaultPriceMinor", "Currency", "SectionId", "IsActive", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if isNotFound(err) {
			return LocalSKU{}, false, nil
		}
		return LocalSKU{}, false, err
	}
	var item LocalSKU
	var barcode, unit, currency, section spanner.NullString
	var created, updated spanner.NullTime
	if err := row.Columns(
		&item.LocalSkuID, &barcode, &item.Name, &unit, &item.DefaultPriceMinor,
		&currency, &section, &item.IsActive, &created, &updated,
	); err != nil {
		return LocalSKU{}, false, err
	}
	if barcode.Valid {
		item.Barcode = barcode.StringVal
	}
	if unit.Valid {
		item.Unit = unit.StringVal
	}
	if currency.Valid {
		item.Currency = currency.StringVal
	}
	if section.Valid {
		item.SectionID = section.StringVal
	}
	if created.Valid {
		item.CreatedAt = created.Time.UTC().Format(time.RFC3339Nano)
	}
	if updated.Valid {
		item.UpdatedAt = updated.Time.UTC().Format(time.RFC3339Nano)
	}
	return item, true, nil
}

func (s *Service) findLocalByBarcode(ctx context.Context, orgID, barcode string) (LocalSKU, bool) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		return LocalSKU{}, false
	}
	items, _ := s.listLocalSKUs(ctx, orgID, barcode, true)
	for _, it := range items {
		if strings.EqualFold(it.Barcode, barcode) {
			return it, true
		}
	}
	return LocalSKU{}, false
}

func (s *Service) saveLocalSKU(ctx context.Context, orgID string, row LocalSKU) error {
	row.LocalSkuID = NormalizeLocalSKUID(row.LocalSkuID)
	now := s.now().UTC().Format(time.RFC3339Nano)
	if row.CreatedAt == "" {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.localCatalog == nil {
			s.localCatalog = map[string][]LocalSKU{}
		}
		list := s.localCatalog[orgID]
		found := false
		for i := range list {
			if list[i].LocalSkuID == row.LocalSkuID {
				if list[i].CreatedAt != "" {
					row.CreatedAt = list[i].CreatedAt
				}
				list[i] = row
				found = true
				break
			}
		}
		if !found {
			list = append(list, row)
		}
		s.localCatalog[orgID] = list
		return nil
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerLocalCatalog", map[string]any{
			"RetailerId":        orgID,
			"LocalSkuId":        row.LocalSkuID,
			"Barcode":           nullableStr(row.Barcode),
			"Name":              row.Name,
			"Unit":              nullableStr(row.Unit),
			"DefaultPriceMinor": row.DefaultPriceMinor,
			"Currency":          nullableStr(row.Currency),
			"SectionId":         nullableStr(row.SectionID),
			"IsActive":          row.IsActive,
			"CreatedAt":         spanner.CommitTimestamp,
			"UpdatedAt":         spanner.CommitTimestamp,
		}),
	})
	return err
}
