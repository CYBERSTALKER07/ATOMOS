package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Wave C3.3 offline count commit with base_version conflict protocol.
// Flag OFFLINE_COUNT_ENABLED (default off → 404 on commit; legacy POST /stock/counts still works).

func (s *Service) offlineCountEnabled() bool {
	if s != nil && s.offlineCountOverride != nil {
		return *s.offlineCountOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("OFFLINE_COUNT_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

type stockVersionKey struct {
	RetailerID string
	LocationID string
	StockBin   string
}

// GetStockLocationVersion returns the inventory etag for (org, location, bin).
// Starts at 0 until first stock mutation bumps it.
func (s *Service) GetStockLocationVersion(ctx context.Context, retailerID, locationID, bin string) (int64, error) {
	retailerID = strings.TrimSpace(retailerID)
	locationID = strings.TrimSpace(locationID)
	bin = normalizeBin(bin)
	if bin == "" {
		bin = BinFloor
	}
	if retailerID == "" || locationID == "" {
		return 0, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.stockVersionByLoc == nil {
			return 0, nil
		}
		return s.stockVersionByLoc[stockVersionKey{retailerID, locationID, bin}], nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerStockLocationVersions",
		spanner.Key{retailerID, locationID, bin}, []string{"Version"})
	if err != nil {
		if isNotFound(err) || strings.Contains(err.Error(), "RetailerStockLocationVersions") {
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

// bumpStockLocationVersion increments the etag after a stock mutation.
// Memory path must be called while s.mu is held if from applyDelta memory branch.
func (s *Service) bumpStockLocationVersionLocked(retailerID, locationID, bin string) {
	if s.stockVersionByLoc == nil {
		s.stockVersionByLoc = map[stockVersionKey]int64{}
	}
	k := stockVersionKey{retailerID, locationID, normalizeBin(bin)}
	if k.StockBin == "" {
		k.StockBin = BinFloor
	}
	s.stockVersionByLoc[k]++
}

func (s *Service) bumpStockLocationVersion(ctx context.Context, retailerID, locationID, bin string) {
	bin = normalizeBin(bin)
	if bin == "" {
		bin = BinFloor
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		s.bumpStockLocationVersionLocked(retailerID, locationID, bin)
		s.mu.Unlock()
		return
	}
	// Best-effort; missing table is OK pre-migration.
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var ver int64
		row, err := txn.ReadRow(ctx, "RetailerStockLocationVersions",
			spanner.Key{retailerID, locationID, bin}, []string{"Version"})
		if err != nil && !isNotFound(err) {
			if strings.Contains(err.Error(), "RetailerStockLocationVersions") {
				return nil
			}
			return err
		}
		if err == nil {
			_ = row.Columns(&ver)
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerStockLocationVersions", map[string]any{
				"RetailerId": retailerID,
				"LocationId": locationID,
				"StockBin":   bin,
				"Version":    ver + 1,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil && s.log != nil && !strings.Contains(err.Error(), "RetailerStockLocationVersions") {
		s.log.Warn("bump stock location version failed", "err", err)
	}
}

// HandleStockCountVersion serves GET /v1/retailer/stock/counts/version?location_id=&stock_bin=
func (s *Service) HandleStockCountVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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
	ver, err := s.GetStockLocationVersion(r.Context(), orgID, locID, bin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "version_lookup_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id": orgID,
		"location_id": locID,
		"stock_bin":   bin,
		"version":     ver,
	})
}

// HandleStockCountCommit serves POST /v1/retailer/stock/counts/commit
// body: { location_id, stock_bin?, base_version, lines: [{sku_id|sku, counted_qty}], force?, force_reason? }
func (s *Service) HandleStockCountCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if !s.offlineCountEnabled() {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
			"code":  "OFFLINE_COUNT_DISABLED",
		})
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
		ForceReason string `json:"force_reason"`
		Lines       []struct {
			SkuID      string `json:"sku_id"`
			Sku        string `json:"sku"`
			CountedQty int64  `json:"counted_qty"`
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
	if locID == "" || len(req.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_and_lines_required"})
		return
	}
	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinFloor
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}

	serverVer, err := s.GetStockLocationVersion(r.Context(), orgID, locID, bin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "version_lookup_failed"})
		return
	}

	// Build server snapshot for conflict body
	type serverLine struct {
		SkuID      string `json:"sku_id"`
		CountedQty int64  `json:"counted_qty"` // last known system = on_hand for draft-vs-current
		OnHand     int64  `json:"on_hand"`
	}
	serverLines := make([]serverLine, 0, len(req.Lines))
	for _, l := range req.Lines {
		sku := strings.TrimSpace(l.SkuID)
		if sku == "" {
			sku = strings.TrimSpace(l.Sku)
		}
		if sku == "" {
			continue
		}
		onHand, _ := s.getOnHand(r.Context(), locID, bin, sku)
		serverLines = append(serverLines, serverLine{
			SkuID: sku, CountedQty: onHand, OnHand: onHand,
		})
	}

	if req.BaseVersion != serverVer {
		if !req.Force {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":          "COUNT_VERSION_CONFLICT",
				"server_version": serverVer,
				"server_lines":   serverLines,
				"message":        "Draft base_version is stale",
			})
			return
		}
		// Force: MANAGER or OWNER only (ADMIN also allowed as operator)
		role := auth.EffectiveRetailerRole(claims)
		if role != "MANAGER" && role != "OWNER" && role != "ADMIN" {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error":  "force_requires_manager",
				"detail": "COUNT force is MANAGER or OWNER only",
			})
			return
		}
	}

	// Apply count (same as legacy commit path)
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
	countID := s.newID()
	actor := auth.ResolveRetailerUserID(claims)
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
	// applyAdjust already bumps version via applyDelta; force audit if forced
	linesJSON, _ := json.Marshal(lines)
	if err := s.saveStockCount(r.Context(), countID, orgID, locID, "COMMITTED", string(linesJSON), actor, true); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_count_failed"})
		return
	}
	if req.Force && req.BaseVersion != serverVer {
		_ = s.saveCountForceAudit(r.Context(), countID, orgID, locID, bin, actor, req.BaseVersion, serverVer, req.ForceReason)
	}
	newVer, _ := s.GetStockLocationVersion(r.Context(), orgID, locID, bin)
	resp := map[string]any{
		"count_id":       countID,
		"location_id":    locID,
		"stock_bin":      bin,
		"status":         "COMMITTED",
		"base_version":   req.BaseVersion,
		"server_version": newVer,
		"forced":         req.Force && req.BaseVersion != serverVer,
		"lines":          lines,
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	writeJSONBytes(w, http.StatusCreated, respBytes)
}

func (s *Service) saveCountForceAudit(ctx context.Context, countID, retailerID, locationID, bin, actor string, baseVer, serverVer int64, reason string) error {
	auditID := s.newID()
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.countForceAudits == nil {
			s.countForceAudits = []map[string]any{}
		}
		s.countForceAudits = append(s.countForceAudits, map[string]any{
			"audit_id": auditID, "count_id": countID, "retailer_id": retailerID,
			"location_id": locationID, "stock_bin": bin, "actor_user_id": actor,
			"base_version": baseVer, "server_version": serverVer, "reason": reason,
		})
		return nil
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("RetailerStockCountForceAudit", map[string]any{
			"AuditId":       auditID,
			"RetailerId":    retailerID,
			"LocationId":    locationID,
			"StockBin":      bin,
			"CountId":       countID,
			"ActorUserId":   actor,
			"BaseVersion":   baseVer,
			"ServerVersion": serverVer,
			"Reason":        nullableStr(strings.TrimSpace(reason)),
			"CreatedAt":     spanner.CommitTimestamp,
		}),
	})
	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerStockCountForceAudit")) {
		return nil
	}
	return err
}
