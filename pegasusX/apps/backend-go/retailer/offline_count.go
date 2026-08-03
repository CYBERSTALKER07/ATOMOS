package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// Wave C3.3 offline count commit with base_version conflict protocol.
// Flag OFFLINE_COUNT_ENABLED (default off).

type locationBinKey struct {
	LocationID string
	StockBin   string
}

func (s *Service) offlineCountEnabled() bool {
	if s != nil && s.offlineCountOverride != nil {
		return *s.offlineCountOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("OFFLINE_COUNT_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (s *Service) requireOfflineCount(w http.ResponseWriter) bool {
	if s.offlineCountEnabled() {
		return true
	}
	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "not_found",
		"code":  "OFFLINE_COUNT_DISABLED",
	})
	return false
}

// HandleStockCountVersion serves GET /v1/retailer/stock/counts/version
func (s *Service) HandleStockCountVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !s.requireOfflineCount(w) {
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockView) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	bin := normalizeBin(r.URL.Query().Get("stock_bin"))
	if bin == "" {
		bin = BinFloor
	}
	if locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_id_required"})
		return
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	ver, err := s.getStockLocationVersion(r.Context(), locID, bin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "version_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"location_id": locID,
		"stock_bin":   bin,
		"version":     ver,
	})
}

// HandleStockCountCommit serves POST /v1/retailer/stock/counts/commit
func (s *Service) HandleStockCountCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !s.requireOfflineCount(w) {
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockCount) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStockCount})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 256*1024)
	if !okBody {
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		LocationID  string `json:"location_id"`
		StockBin    string `json:"stock_bin"`
		BaseVersion int64  `json:"base_version"`
		Force       bool   `json:"force"`
		Lines       []struct {
			SkuID       string `json:"sku_id"`
			Sku         string `json:"sku"`
			CountedQty  int64  `json:"counted_qty"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinFloor
	}
	if locID == "" || len(req.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_and_lines_required"})
		return
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}

	serverVersion, err := s.getStockLocationVersion(r.Context(), locID, bin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "version_read_failed"})
		return
	}
	if !req.Force && req.BaseVersion != serverVersion {
		serverLines, err := s.buildCountConflictLines(r.Context(), locID, bin, req.Lines)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "conflict_lines_failed"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":          "COUNT_VERSION_CONFLICT",
			"server_version": serverVersion,
			"server_lines":   serverLines,
			"message":        "Draft base_version is stale",
		})
		return
	}
	if req.Force {
		role := auth.EffectiveRetailerRole(claims)
		if role != "MANAGER" && role != "OWNER" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "force_requires_manager_or_owner"})
			return
		}
	}

	type countLine struct {
		Sku        string `json:"sku"`
		SystemQty  int64  `json:"system_qty"`
		CountedQty int64  `json:"counted_qty"`
		Variance   int64  `json:"variance"`
	}
	var lines []countLine
	for _, l := range req.Lines {
		sku := strings.TrimSpace(l.SkuID)
		if sku == "" {
			sku = strings.TrimSpace(l.Sku)
		}
		if sku == "" {
			continue
		}
		sys, _ := s.getOnHand(r.Context(), locID, bin, sku)
		lines = append(lines, countLine{
			Sku: sku, SystemQty: sys, CountedQty: l.CountedQty, Variance: l.CountedQty - sys,
		})
	}
	if len(lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no_valid_lines"})
		return
	}

	countID := s.newID()
	actor := auth.ResolveRetailerUserID(claims)
	actorRole := auth.EffectiveRetailerRole(claims)

	if req.Force {
		linesJSON, _ := json.Marshal(lines)
		if err := s.saveCountForceAudit(r.Context(), countID, orgID, locID, bin, req.BaseVersion, serverVersion, actor, actorRole, linesJSON); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "force_audit_failed"})
			return
		}
	}

	for _, l := range lines {
		if l.Variance == 0 {
			continue
		}
		if err := s.applyAdjust(r.Context(), orgID, locID, bin, l.Sku, l.Variance, actor, "cycle_count:"+countID); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "sku": l.Sku})
			return
		}
		_ = s.syncReorderCurrentStock(r.Context(), orgID, l.Sku)
	}

	linesJSON, _ := json.Marshal(lines)
	if err := s.saveStockCount(r.Context(), countID, orgID, locID, "COMMITTED", string(linesJSON), actor, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_count_failed"})
		return
	}
	_ = s.emitPosEvent(r.Context(), orgID, events.EventStoreStockCounted, map[string]any{
		"count_id":    countID,
		"location_id": locID,
		"stock_bin":   bin,
		"offline":     true,
	})

	newVersion, _ := s.getStockLocationVersion(r.Context(), locID, bin)
	resp := map[string]any{
		"count_id":    countID,
		"location_id": locID,
		"stock_bin":   bin,
		"status":      "COMMITTED",
		"new_version": newVersion,
		"lines":       lines,
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func (s *Service) buildCountConflictLines(ctx context.Context, locID, bin string, reqLines []struct {
	SkuID      string `json:"sku_id"`
	Sku        string `json:"sku"`
	CountedQty int64  `json:"counted_qty"`
}) ([]map[string]any, error) {
	var out []map[string]any
	for _, l := range reqLines {
		sku := strings.TrimSpace(l.SkuID)
		if sku == "" {
			sku = strings.TrimSpace(l.Sku)
		}
		if sku == "" {
			continue
		}
		onHand, err := s.getOnHand(ctx, locID, bin, sku)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"sku_id":       sku,
			"counted_qty":  onHand,
			"on_hand":      onHand,
		})
	}
	return out, nil
}

func (s *Service) getStockLocationVersion(ctx context.Context, locationID, bin string) (int64, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.stockLocationVersions == nil {
			return 0, nil
		}
		return s.stockLocationVersions[locationBinKey{locationID, bin}], nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerStockLocationVersions",
		spanner.Key{locationID, bin}, []string{"Version"})
	if err != nil {
		if isNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	var ver int64
	if err := row.Columns(&ver); err != nil {
		return 0, err
	}
	return ver, nil
}

func (s *Service) bumpStockLocationVersionMemory(locationID, bin string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stockLocationVersions == nil {
		s.stockLocationVersions = map[locationBinKey]int64{}
	}
	key := locationBinKey{locationID, bin}
	s.stockLocationVersions[key]++
}

func (s *Service) bumpStockLocationVersionInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, locationID, bin string) error {
	var ver int64
	row, err := txn.ReadRow(ctx, "RetailerStockLocationVersions", spanner.Key{locationID, bin}, []string{"Version"})
	if err != nil && !isNotFound(err) {
		return err
	}
	if err == nil {
		_ = row.Columns(&ver)
	}
	ver++
	return txn.BufferWrite([]*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerStockLocationVersions", map[string]any{
			"LocationId": locationID,
			"StockBin":   bin,
			"Version":    ver,
			"UpdatedAt":  spanner.CommitTimestamp,
		}),
	})
}

func (s *Service) saveCountForceAudit(ctx context.Context, countID, retailerID, locationID, bin string, baseVer, serverVer int64, actor, role string, linesJSON []byte) error {
	if s.spannerClient == nil {
		return nil
	}
	auditID := s.newID()
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("RetailerStockCountForceAudits", map[string]any{
			"AuditId":       auditID,
			"CountId":       countID,
			"RetailerId":    retailerID,
			"LocationId":    locationID,
			"StockBin":      bin,
			"BaseVersion":   baseVer,
			"ServerVersion": serverVer,
			"ActorUserId":   actor,
			"ActorRole":     role,
			"LinesJson":     spanner.NullJSON{Value: json.RawMessage(linesJSON), Valid: true},
			"CreatedAt":     spanner.CommitTimestamp,
		}),
	})
	return err
}

// setStockLocationVersionForTest sets stock version in memory mode (tests only).
func (s *Service) setStockLocationVersionForTest(locationID, bin string, ver int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stockLocationVersions == nil {
		s.stockLocationVersions = map[locationBinKey]int64{}
	}
	s.stockLocationVersions[locationBinKey{locationID, bin}] = ver
}

func (s *Service) afterStockBalanceMutationInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, locationID, bin string) error {
	if s == nil || s.spannerClient == nil {
		return nil
	}
	return s.bumpStockLocationVersionInTxn(ctx, txn, locationID, bin)
}
