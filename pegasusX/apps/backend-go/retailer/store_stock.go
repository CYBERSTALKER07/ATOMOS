package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// Stock bins (store-side, not warehouse ATP).
const (
	BinBackroom   = "BACKROOM"
	BinFloor      = "FLOOR"
	BinQuarantine = "QUARANTINE"
)

// Movement types.
const (
	MoveReceive       = "RECEIVE"
	MovePutaway       = "PUTAWAY"
	MoveTransferBin   = "TRANSFER_BIN"
	MoveAdjust        = "ADJUST"
	MoveCountVariance = "COUNT_VARIANCE"
	MoveSale          = "SALE"
	MoveSaleVoid      = "SALE_VOID"
	MoveClaimHold     = "CLAIM_HOLD"    // sellable → QUARANTINE on claim file
	MoveClaimRelease  = "CLAIM_RELEASE" // leave QUARANTINE (return / waste)
	MoveClaimRestore  = "CLAIM_RESTORE" // QUARANTINE → FLOOR on reject
)

// Claim stock disposition for ResolveClaimStock.
const (
	ClaimStockReturn  = "RETURN"  // remove from quarantine (reverse logistics)
	ClaimStockRestore = "RESTORE" // back to floor
	ClaimStockWaste   = "WASTE"   // scrap out of quarantine
)

// StockBalanceDTO is one SKU/bin balance row.
type StockBalanceDTO struct {
	LocationID string `json:"location_id"`
	StockBin   string `json:"stock_bin"`
	Sku        string `json:"sku"`
	OnHand     int64  `json:"on_hand"`
	Reserved   int64  `json:"reserved"`
	Available  int64  `json:"available"`
}

// StockMovementDTO is a ledger row.
type StockMovementDTO struct {
	MovementID   string `json:"movement_id"`
	LocationID   string `json:"location_id"`
	StockBin     string `json:"stock_bin"`
	Sku          string `json:"sku"`
	Qty          int64  `json:"qty"`
	MovementType string `json:"movement_type"`
	RefType      string `json:"ref_type,omitempty"`
	RefID        string `json:"ref_id,omitempty"`
	ActorUserID  string `json:"actor_user_id,omitempty"`
	Note         string `json:"note,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// ReceiveLine is a putaway line from a Pegasus order.
type ReceiveLine struct {
	Sku         string `json:"sku"`
	ProductName string `json:"product_name,omitempty"`
	OrderedQty  int64  `json:"ordered_qty"`
	AcceptedQty int64  `json:"accepted_qty"`
}

// ReceiveSessionDTO wire shape.
type ReceiveSessionDTO struct {
	SessionID  string        `json:"session_id"`
	RetailerID string        `json:"retailer_id"`
	LocationID string        `json:"location_id"`
	OrderID    string        `json:"order_id"`
	Status     string        `json:"status"`
	Lines      []ReceiveLine `json:"lines"`
	CreatedAt  string        `json:"created_at,omitempty"`
	ConfirmedAt string       `json:"confirmed_at,omitempty"`
}

type stockBalanceKey struct {
	LocationID string
	StockBin   string
	Sku        string
}

type memStockBalance struct {
	RetailerID string
	OnHand     int64
	Reserved   int64
}

// HandleStockList serves GET /v1/retailer/stock?location_id=
func (s *Service) HandleStockList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockView) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStockView})
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
	if locID == "" {
		if primary, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = primary.LocationID
		}
	}
	if locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_id_required"})
		return
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	rows, err := s.listStockBalances(r.Context(), orgID, locID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_stock_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id": orgID,
		"location_id": locID,
		"items":       rows,
	})
}

// HandleStockSKU serves GET /v1/retailer/stock/{sku}?location_id=
func (s *Service) HandleStockSKU(w http.ResponseWriter, r *http.Request) {
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
	sku := strings.TrimSpace(chi.URLParam(r, "sku"))
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if sku == "" || locID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku_and_location_required"})
		return
	}
	rows, err := s.listStockBalances(r.Context(), orgID, locID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_stock_failed"})
		return
	}
	var filtered []StockBalanceDTO
	for _, row := range rows {
		if row.Sku == sku {
			filtered = append(filtered, row)
		}
	}
	movs, _ := s.listStockMovements(r.Context(), orgID, locID, sku, 50)
	writeJSON(w, http.StatusOK, map[string]any{
		"sku":         sku,
		"location_id": locID,
		"balances":    filtered,
		"movements":   movs,
	})
}

// HandleStockReceiveSession serves POST /v1/retailer/stock/receive-sessions
// body: { order_id, location_id?, confirm?: true, lines?: [{sku, accepted_qty}] }
func (s *Service) HandleStockReceiveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermDockReceive) && !auth.HasRetailerPerm(claims, auth.PermStockAdjust)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 64*1024)
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
		OrderID    string `json:"order_id"`
		LocationID string `json:"location_id"`
		Confirm    *bool  `json:"confirm"`
		StockBin   string `json:"stock_bin"`
		Lines      []struct {
			Sku         string `json:"sku"`
			AcceptedQty int64  `json:"accepted_qty"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		if primary, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = primary.LocationID
		}
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinBackroom
	}

	// Ensure STORE_STOCK pack.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackSTORESTOCK) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSTORESTOCK, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}

	// Idempotent: existing session for order.
	if existing, found, _ := s.getReceiveSessionByOrder(r.Context(), orderID); found {
		if existing.Status == "CONFIRMED" {
			respBytes, _ := json.Marshal(existing)
			idemCommitted = true
			s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
			writeJSONBytes(w, http.StatusOK, respBytes)
			return
		}
	}

	lines, err := s.loadOrderLinesForReceive(r.Context(), orgID, orderID)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	// Apply accepted qty overrides.
	if len(req.Lines) > 0 {
		bySKU := map[string]int64{}
		for _, l := range req.Lines {
			bySKU[strings.TrimSpace(l.Sku)] = l.AcceptedQty
		}
		for i := range lines {
			if q, ok := bySKU[lines[i].Sku]; ok {
				lines[i].AcceptedQty = q
			}
		}
	}

	session := ReceiveSessionDTO{
		SessionID:  s.newID(),
		RetailerID: orgID,
		LocationID: locID,
		OrderID:    orderID,
		Status:     "DRAFT",
		Lines:      lines,
		CreatedAt:  s.now().UTC().Format(time.RFC3339Nano),
	}
	confirm := true
	if req.Confirm != nil {
		confirm = *req.Confirm
	}
	if confirm {
		if err := s.confirmReceiveSession(r.Context(), session, bin, auth.ResolveRetailerUserID(claims)); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		session.Status = "CONFIRMED"
		session.ConfirmedAt = s.now().UTC().Format(time.RFC3339Nano)
	} else {
		if err := s.saveReceiveSession(r.Context(), session, auth.ResolveRetailerUserID(claims)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_session_failed"})
			return
		}
	}
	respBytes, _ := json.Marshal(session)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleStockReceiveConfirm serves POST /v1/retailer/stock/receive-sessions/{sessionID}/confirm
func (s *Service) HandleStockReceiveConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermDockReceive) && !auth.HasRetailerPerm(claims, auth.PermStockAdjust)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionID"))
	session, found, err := s.getReceiveSession(r.Context(), sessionID)
	if err != nil || !found || session.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session_not_found"})
		return
	}
	if session.Status == "CONFIRMED" {
		writeJSON(w, http.StatusOK, session)
		return
	}
	var req struct {
		StockBin string `json:"stock_bin"`
		Lines    []struct {
			Sku         string `json:"sku"`
			AcceptedQty int64  `json:"accepted_qty"`
		} `json:"lines"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if len(req.Lines) > 0 {
		bySKU := map[string]int64{}
		for _, l := range req.Lines {
			bySKU[strings.TrimSpace(l.Sku)] = l.AcceptedQty
		}
		for i := range session.Lines {
			if q, ok := bySKU[session.Lines[i].Sku]; ok {
				session.Lines[i].AcceptedQty = q
			}
		}
	}
	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinBackroom
	}
	if err := s.confirmReceiveSession(r.Context(), session, bin, auth.ResolveRetailerUserID(claims)); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	session.Status = "CONFIRMED"
	session.ConfirmedAt = s.now().UTC().Format(time.RFC3339Nano)
	writeJSON(w, http.StatusOK, session)
}

// HandleStockTransfer serves POST /v1/retailer/stock/transfer
// body: { location_id, sku, qty, from_bin, to_bin } or cross-location { from_location_id, to_location_id, ... }
func (s *Service) HandleStockTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockAdjust) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStockAdjust})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 16*1024)
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
		LocationID     string `json:"location_id"`
		FromLocationID string `json:"from_location_id"`
		ToLocationID   string `json:"to_location_id"`
		Sku            string `json:"sku"`
		Qty            int64  `json:"qty"`
		FromBin        string `json:"from_bin"`
		ToBin          string `json:"to_bin"`
		Note           string `json:"note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	sku := strings.TrimSpace(req.Sku)
	if sku == "" || req.Qty <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku_and_positive_qty_required"})
		return
	}
	fromLoc := strings.TrimSpace(req.FromLocationID)
	toLoc := strings.TrimSpace(req.ToLocationID)
	if fromLoc == "" {
		fromLoc = strings.TrimSpace(req.LocationID)
	}
	if toLoc == "" {
		toLoc = fromLoc
	}
	if fromLoc == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "location_id_required"})
		return
	}
	fromBin := normalizeBin(req.FromBin)
	toBin := normalizeBin(req.ToBin)
	if fromBin == "" {
		fromBin = BinBackroom
	}
	if toBin == "" {
		toBin = BinFloor
	}
	if fromLoc == toLoc && fromBin == toBin {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "transfer_noop"})
		return
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, fromLoc); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "from_location_not_found"})
		return
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, toLoc); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "to_location_not_found"})
		return
	}

	actor := auth.ResolveRetailerUserID(claims)
	if err := s.applyTransfer(r.Context(), orgID, fromLoc, toLoc, fromBin, toBin, sku, req.Qty, actor, req.Note); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	resp := map[string]any{"status": "ok", "sku": sku, "qty": req.Qty, "from": fromBin, "to": toBin}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleStockAdjust serves POST /v1/retailer/stock/adjust
func (s *Service) HandleStockAdjust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermStockAdjust) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermStockAdjust})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 16*1024)
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
		LocationID string `json:"location_id"`
		Sku        string `json:"sku"`
		QtyDelta   int64  `json:"qty_delta"` // can be negative
		StockBin   string `json:"stock_bin"`
		Note       string `json:"note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	sku := strings.TrimSpace(req.Sku)
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if sku == "" || locID == "" || req.QtyDelta == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku_location_qty_required"})
		return
	}
	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinBackroom
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	actor := auth.ResolveRetailerUserID(claims)
	if err := s.applyAdjust(r.Context(), orgID, locID, bin, sku, req.QtyDelta, actor, req.Note); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	// Refresh reorder on_hand for this sku.
	_ = s.syncReorderCurrentStock(r.Context(), orgID, sku)

	resp := map[string]any{"status": "ok", "sku": sku, "qty_delta": req.QtyDelta}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleStockCount serves POST /v1/retailer/stock/counts
// body: { location_id, stock_bin?, lines: [{sku, counted_qty}], commit?: true }
func (s *Service) HandleStockCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
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
		LocationID string `json:"location_id"`
		StockBin   string `json:"stock_bin"`
		Commit     *bool  `json:"commit"`
		Lines      []struct {
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

	type countLine struct {
		Sku        string `json:"sku"`
		SystemQty  int64  `json:"system_qty"`
		CountedQty int64  `json:"counted_qty"`
		Variance   int64  `json:"variance"`
	}
	var lines []countLine
	for _, l := range req.Lines {
		sku := strings.TrimSpace(l.Sku)
		if sku == "" {
			continue
		}
		sys, _ := s.getOnHand(r.Context(), locID, bin, sku)
		lines = append(lines, countLine{
			Sku: sku, SystemQty: sys, CountedQty: l.CountedQty, Variance: l.CountedQty - sys,
		})
	}
	countID := s.newID()
	commit := true
	if req.Commit != nil {
		commit = *req.Commit
	}
	status := "DRAFT"
	if commit {
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
		// Write count movement markers with COUNT_VARIANCE type for audit (adjust already wrote ADJUST).
		// Also persist count header.
		status = "COMMITTED"
	}
	linesJSON, _ := json.Marshal(lines)
	if err := s.saveStockCount(r.Context(), countID, orgID, locID, status, string(linesJSON), auth.ResolveRetailerUserID(claims), commit); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save_count_failed"})
		return
	}
	resp := map[string]any{
		"count_id":    countID,
		"location_id": locID,
		"stock_bin":   bin,
		"status":      status,
		"lines":       lines,
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	code := http.StatusCreated
	s.saveIdempotency(r.Context(), r, body, code, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(respBytes)
}

// HandleStockMovements serves GET /v1/retailer/stock/movements?location_id=&sku=
func (s *Service) HandleStockMovements(w http.ResponseWriter, r *http.Request) {
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
	sku := strings.TrimSpace(r.URL.Query().Get("sku"))
	movs, err := s.listStockMovements(r.Context(), orgID, locID, sku, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_movements_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": movs})
}

// ---- core stock math ----

func normalizeBin(b string) string {
	b = strings.ToUpper(strings.TrimSpace(b))
	switch b {
	case BinBackroom, BinFloor, BinQuarantine:
		return b
	default:
		return ""
	}
}

func (s *Service) assertLocationInOrg(ctx context.Context, orgID, locID string) error {
	loc, found, err := s.getLocation(ctx, locID)
	if err != nil {
		return err
	}
	if !found || loc.RetailerID != orgID {
		return errors.New("location_not_found")
	}
	return nil
}

func (s *Service) available(onHand, reserved int64) int64 {
	a := onHand - reserved
	if a < 0 {
		return 0
	}
	return a
}

func (s *Service) getOnHand(ctx context.Context, locationID, bin, sku string) (int64, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.stockBalances == nil {
			return 0, nil
		}
		b, ok := s.stockBalances[stockBalanceKey{locationID, bin, sku}]
		if !ok {
			return 0, nil
		}
		return b.OnHand, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerStockBalances",
		spanner.Key{locationID, bin, sku}, []string{"OnHand"})
	if err != nil {
		if isNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	var onHand int64
	if err := row.Columns(&onHand); err != nil {
		return 0, err
	}
	return onHand, nil
}

// SumOnHandForSKU across all bins/locations for a retailer (reorder AI).
func (s *Service) SumOnHandForSKU(ctx context.Context, retailerID, sku string) (int64, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var total int64
		for k, b := range s.stockBalances {
			if b.RetailerID == retailerID && k.Sku == sku {
				total += b.OnHand
			}
		}
		return total, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(OnHand), 0) FROM RetailerStockBalances
			WHERE RetailerId = @rid AND Sku = @sku`,
		Params: map[string]any{"rid": retailerID, "sku": sku},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var total int64
	if err := row.Column(0, &total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Service) listStockBalances(ctx context.Context, retailerID, locationID string) ([]StockBalanceDTO, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []StockBalanceDTO
		for k, b := range s.stockBalances {
			if b.RetailerID != retailerID || k.LocationID != locationID {
				continue
			}
			out = append(out, StockBalanceDTO{
				LocationID: k.LocationID, StockBin: k.StockBin, Sku: k.Sku,
				OnHand: b.OnHand, Reserved: b.Reserved, Available: s.available(b.OnHand, b.Reserved),
			})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Sku != out[j].Sku {
				return out[i].Sku < out[j].Sku
			}
			return out[i].StockBin < out[j].StockBin
		})
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT LocationId, StockBin, Sku, OnHand, Reserved FROM RetailerStockBalances
			WHERE RetailerId = @rid AND LocationId = @lid`,
		Params: map[string]any{"rid": retailerID, "lid": locationID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []StockBalanceDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var loc, bin, sku string
		var onHand, reserved int64
		if err := row.Columns(&loc, &bin, &sku, &onHand, &reserved); err != nil {
			return nil, err
		}
		out = append(out, StockBalanceDTO{
			LocationID: loc, StockBin: bin, Sku: sku,
			OnHand: onHand, Reserved: reserved, Available: s.available(onHand, reserved),
		})
	}
	return out, nil
}

func (s *Service) applyDelta(ctx context.Context, retailerID, locationID, bin, sku string, delta int64, moveType, refType, refID, actor, note string) error {
	if delta == 0 {
		return nil
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.stockBalances == nil {
			s.stockBalances = map[stockBalanceKey]memStockBalance{}
		}
		if s.stockMovements == nil {
			s.stockMovements = []StockMovementDTO{}
		}
		key := stockBalanceKey{locationID, bin, sku}
		cur := s.stockBalances[key]
		cur.RetailerID = retailerID
		next := cur.OnHand + delta
		if next < 0 {
			return errors.New("insufficient_stock")
		}
		cur.OnHand = next
		s.stockBalances[key] = cur
		s.stockMovements = append(s.stockMovements, StockMovementDTO{
			MovementID: s.newID(), LocationID: locationID, StockBin: bin, Sku: sku,
			Qty: delta, MovementType: moveType, RefType: refType, RefID: refID,
			ActorUserID: actor, Note: note, CreatedAt: s.now().UTC().Format(time.RFC3339Nano),
		})
		// C3.3: bump location inventory etag (caller holds s.mu)
		s.bumpStockLocationVersionLocked(retailerID, locationID, bin)
		return nil
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current
		var onHand, reserved int64
		row, err := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{locationID, bin, sku},
			[]string{"OnHand", "Reserved"})
		if err != nil && !isNotFound(err) {
			return err
		}
		if err == nil {
			_ = row.Columns(&onHand, &reserved)
		}
		next := onHand + delta
		if next < 0 {
			return errors.New("insufficient_stock")
		}
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
				"LocationId": locationID,
				"StockBin":   bin,
				"Sku":        sku,
				"RetailerId": retailerID,
				"OnHand":     next,
				"Reserved":   reserved,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
			spanner.InsertMap("RetailerStockMovements", map[string]any{
				"MovementId":   s.newID(),
				"RetailerId":   retailerID,
				"LocationId":   locationID,
				"StockBin":     bin,
				"Sku":          sku,
				"Qty":          delta,
				"MovementType": moveType,
				"RefType":      nullableStr(refType),
				"RefId":        nullableStr(refID),
				"ActorUserId":  nullableStr(actor),
				"Note":         nullableStr(note),
				"CreatedAt":    spanner.CommitTimestamp,
			}),
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		evType := events.EventStoreStockAdjusted
		switch moveType {
		case MoveReceive:
			evType = events.EventStoreStockReceived
		case MoveTransferBin:
			evType = events.EventStoreStockTransferred
		case MoveCountVariance:
			evType = events.EventStoreStockCounted
		}
		payload := map[string]any{
			"type":        evType,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": retailerID,
			"location_id": locationID,
			"stock_bin":   bin,
			"sku":         sku,
			"qty":         delta,
			"movement":    moveType,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err == nil {
		// C3.3: location inventory etag (best-effort separate write if version table present)
		s.bumpStockLocationVersion(ctx, retailerID, locationID, bin)
	}
	return err
}

func (s *Service) applyTransfer(ctx context.Context, retailerID, fromLoc, toLoc, fromBin, toBin, sku string, qty int64, actor, note string) error {
	return s.applyTransferTyped(ctx, retailerID, fromLoc, toLoc, fromBin, toBin, sku, qty, MoveTransferBin, "TRANSFER", actor, note)
}

func (s *Service) applyTransferTyped(ctx context.Context, retailerID, fromLoc, toLoc, fromBin, toBin, sku string, qty int64, moveType, refType, actor, note string) error {
	if err := s.applyDelta(ctx, retailerID, fromLoc, fromBin, sku, -qty, moveType, refType, toLoc+":"+toBin, actor, note); err != nil {
		return err
	}
	if err := s.applyDelta(ctx, retailerID, toLoc, toBin, sku, qty, moveType, refType, fromLoc+":"+fromBin, actor, note); err != nil {
		_ = s.applyDelta(ctx, retailerID, fromLoc, fromBin, sku, qty, MoveAdjust, "TRANSFER_ROLLBACK", "", actor, "rollback")
		return err
	}
	return nil
}

// ClaimHoldLine is one SKU qty to quarantine or resolve.
type ClaimHoldLine struct {
	SKU string
	Qty int64
}

// HoldForClaim moves sellable stock (FLOOR then BACKROOM) into QUARANTINE for a filed claim.
// Best-effort per line: insufficient stock is skipped (no receive / already sold) without failing the claim.
func (s *Service) HoldForClaim(ctx context.Context, retailerID, claimID, orderID string, lines []ClaimHoldLine, actor string) error {
	if s == nil || retailerID == "" || claimID == "" || len(lines) == 0 {
		return nil
	}
	loc, err := s.EnsurePrimaryLocation(ctx, retailerID)
	if err != nil {
		return err
	}
	locID := loc.LocationID
	note := "claim_hold:" + claimID
	if orderID != "" {
		note += " order:" + orderID
	}
	for _, ln := range lines {
		sku := strings.TrimSpace(ln.SKU)
		need := ln.Qty
		if sku == "" || need <= 0 {
			continue
		}
		// Prefer FLOOR, then BACKROOM.
		for _, bin := range []string{BinFloor, BinBackroom} {
			if need <= 0 {
				break
			}
			avail := s.onHandInBin(ctx, locID, bin, sku)
			if avail <= 0 {
				continue
			}
			take := need
			if take > avail {
				take = avail
			}
			if err := s.applyTransferTyped(ctx, retailerID, locID, locID, bin, BinQuarantine, sku, take, MoveClaimHold, "CLAIM", actor, note); err != nil {
				s.log.Warn("claim hold transfer failed", "sku", sku, "bin", bin, "qty", take, "err", err)
				continue
			}
			need -= take
		}
		_ = s.syncReorderCurrentStock(ctx, retailerID, sku)
	}
	_ = s.emitPosEvent(ctx, retailerID, events.EventStoreStockTransferred, map[string]any{
		"reason":   "CLAIM_HOLD",
		"claim_id": claimID,
		"order_id": orderID,
	})
	return nil
}

// ResolveClaimStock disposes quarantined units after claim adjudication.
// disposition: RETURN | RESTORE | WASTE
func (s *Service) ResolveClaimStock(ctx context.Context, retailerID, claimID string, lines []ClaimHoldLine, disposition, actor string) error {
	if s == nil || retailerID == "" || claimID == "" || len(lines) == 0 {
		return nil
	}
	disposition = strings.ToUpper(strings.TrimSpace(disposition))
	if disposition == "" {
		disposition = ClaimStockReturn
	}
	loc, err := s.EnsurePrimaryLocation(ctx, retailerID)
	if err != nil {
		return err
	}
	locID := loc.LocationID
	note := "claim_" + strings.ToLower(disposition) + ":" + claimID
	for _, ln := range lines {
		sku := strings.TrimSpace(ln.SKU)
		need := ln.Qty
		if sku == "" || need <= 0 {
			continue
		}
		avail := s.onHandInBin(ctx, locID, BinQuarantine, sku)
		if avail <= 0 {
			continue
		}
		take := need
		if take > avail {
			take = avail
		}
		switch disposition {
		case ClaimStockRestore:
			_ = s.applyTransferTyped(ctx, retailerID, locID, locID, BinQuarantine, BinFloor, sku, take, MoveClaimRestore, "CLAIM", actor, note)
		case ClaimStockWaste:
			_ = s.applyDelta(ctx, retailerID, locID, BinQuarantine, sku, -take, MoveClaimRelease, "CLAIM", claimID, actor, note+":waste")
		default: // RETURN — leave store via reverse logistics
			_ = s.applyDelta(ctx, retailerID, locID, BinQuarantine, sku, -take, MoveClaimRelease, "CLAIM", claimID, actor, note+":return")
		}
		_ = s.syncReorderCurrentStock(ctx, retailerID, sku)
	}
	return nil
}

func (s *Service) onHandInBin(ctx context.Context, locationID, bin, sku string) int64 {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.stockBalances == nil {
			return 0
		}
		b := s.stockBalances[stockBalanceKey{locationID, bin, sku}]
		return b.OnHand
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerStockBalances",
		spanner.Key{locationID, bin, sku}, []string{"OnHand"})
	if err != nil {
		return 0
	}
	var onHand int64
	_ = row.Columns(&onHand)
	return onHand
}

func (s *Service) applyAdjust(ctx context.Context, retailerID, locationID, bin, sku string, delta int64, actor, note string) error {
	moveType := MoveAdjust
	if strings.HasPrefix(note, "cycle_count:") {
		moveType = MoveCountVariance
	}
	return s.applyDelta(ctx, retailerID, locationID, bin, sku, delta, moveType, "ADJUST", "", actor, note)
}

func (s *Service) confirmReceiveSession(ctx context.Context, session ReceiveSessionDTO, bin, actor string) error {
	// Persist session as confirmed and apply RECEIVE deltas.
	for _, line := range session.Lines {
		if line.AcceptedQty <= 0 {
			continue
		}
		if err := s.applyDelta(ctx, session.RetailerID, session.LocationID, bin, line.Sku, line.AcceptedQty, MoveReceive, "ORDER", session.OrderID, actor, "receive"); err != nil {
			return err
		}
		_ = s.syncReorderCurrentStock(ctx, session.RetailerID, line.Sku)
	}
	session.Status = "CONFIRMED"
	return s.saveReceiveSession(ctx, session, actor)
}

func (s *Service) saveReceiveSession(ctx context.Context, session ReceiveSessionDTO, actor string) error {
	linesJSON, _ := json.Marshal(session.Lines)
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.receiveSessions == nil {
			s.receiveSessions = map[string]ReceiveSessionDTO{}
		}
		s.receiveSessions[session.SessionID] = session
		s.receiveByOrder[session.OrderID] = session.SessionID
		return nil
	}
	row := map[string]any{
		"SessionId":  session.SessionID,
		"RetailerId": session.RetailerID,
		"LocationId": session.LocationID,
		"OrderId":    session.OrderID,
		"Status":     session.Status,
		"LinesJson":  string(linesJSON),
		"CreatedBy":  nullableStr(actor),
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if session.Status == "CONFIRMED" {
		row["ConfirmedAt"] = spanner.CommitTimestamp
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerReceiveSessions", row)})
	return err
}

func (s *Service) getReceiveSession(ctx context.Context, sessionID string) (ReceiveSessionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		ss, ok := s.receiveSessions[sessionID]
		return ss, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerReceiveSessions", spanner.Key{sessionID},
		[]string{"SessionId", "RetailerId", "LocationId", "OrderId", "Status", "LinesJson", "CreatedAt", "ConfirmedAt"})
	if err != nil {
		if isNotFound(err) {
			return ReceiveSessionDTO{}, false, nil
		}
		return ReceiveSessionDTO{}, false, err
	}
	return decodeReceiveSessionRow(row)
}

func (s *Service) getReceiveSessionByOrder(ctx context.Context, orderID string) (ReceiveSessionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		sid, ok := s.receiveByOrder[orderID]
		if !ok {
			return ReceiveSessionDTO{}, false, nil
		}
		ss, ok2 := s.receiveSessions[sid]
		return ss, ok2, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, RetailerId, LocationId, OrderId, Status, LinesJson, CreatedAt, ConfirmedAt
			FROM RetailerReceiveSessions@{FORCE_INDEX=UQ_RetailerReceiveSessions_ByOrder}
			WHERE OrderId = @oid LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ReceiveSessionDTO{}, false, nil
	}
	if err != nil {
		return ReceiveSessionDTO{}, false, err
	}
	return decodeReceiveSessionRow(row)
}

func decodeReceiveSessionRow(row *spanner.Row) (ReceiveSessionDTO, bool, error) {
	var ss ReceiveSessionDTO
	var linesJSON string
	var created time.Time
	var confirmed spanner.NullTime
	if err := row.Columns(&ss.SessionID, &ss.RetailerID, &ss.LocationID, &ss.OrderID, &ss.Status, &linesJSON, &created, &confirmed); err != nil {
		return ReceiveSessionDTO{}, false, err
	}
	_ = json.Unmarshal([]byte(linesJSON), &ss.Lines)
	ss.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	if confirmed.Valid {
		ss.ConfirmedAt = confirmed.Time.UTC().Format(time.RFC3339Nano)
	}
	return ss, true, nil
}

func (s *Service) loadOrderLinesForReceive(ctx context.Context, retailerID, orderID string) ([]ReceiveLine, error) {
	if s.spannerClient == nil {
		// Memory: allow synthetic lines from test/manual inject via empty error
		return nil, errors.New("order_not_found")
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{orderID},
		[]string{"RetailerId", "Status", "LineItemsJson"})
	if err != nil {
		if isNotFound(err) {
			return nil, errors.New("order_not_found")
		}
		return nil, err
	}
	var rid, status string
	var lineBytes []byte
	if err := row.Columns(&rid, &status, &lineBytes); err != nil {
		return nil, err
	}
	if rid != retailerID {
		return nil, errors.New("order_not_owned")
	}
	// Allow receive when delivered-ish or completed.
	st := strings.ToUpper(status)
	switch st {
	case "COMPLETED", "ARRIVED", "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", "DELIVERED_ON_CREDIT", "FISCALIZING", "FISCAL_FAILED":
		// ok
	default:
		return nil, fmt.Errorf("order_status_not_receivable:%s", st)
	}
	// LineItemsJson is the order qty source of truth for receive.
	// Cap to delivered − remaining/offload − open logistics claim qty (G9).
	var items []struct {
		SKU         string `json:"sku"`
		Name        string `json:"name"`
		ProductName string `json:"product_name"`
		Quantity    int64  `json:"quantity"`
		Qty         int64  `json:"qty"`
		Delivered   int64  `json:"delivered_qty"`
		Remaining   int64  `json:"remaining_qty"`
		OffloadQty  int64  `json:"offload_qty"`
	}
	if err := json.Unmarshal(lineBytes, &items); err != nil {
		return nil, errors.New("order_lines_invalid")
	}
	claimedBySKU := s.openClaimQtyBySKU(ctx, orderID)
	var lines []ReceiveLine
	for _, it := range items {
		sku := strings.TrimSpace(it.SKU)
		if sku == "" {
			continue
		}
		ordered := it.Quantity
		if ordered == 0 {
			ordered = it.Qty
		}
		qty := ReceivableQty(ordered, it.Delivered, it.Remaining, it.OffloadQty, claimedBySKU[sku])
		name := it.Name
		if name == "" {
			name = it.ProductName
		}
		lines = append(lines, ReceiveLine{Sku: sku, ProductName: name, OrderedQty: qty, AcceptedQty: qty})
	}
	if len(lines) == 0 {
		return nil, errors.New("order_has_no_lines")
	}
	return lines, nil
}

// ReceivableQty is delivered (else ordered) minus driver-excepted residual and open claim qty.
// Never returns negative.
func ReceivableQty(ordered, delivered, remaining, offload, openClaimed int64) int64 {
	base := ordered
	if delivered > 0 {
		base = delivered
	}
	excepted := remaining
	if offload > excepted {
		excepted = offload
	}
	out := base - excepted - openClaimed
	if out < 0 {
		return 0
	}
	return out
}

// openClaimQtyBySKU sums OPEN/UNDER_REVIEW/APPROVED claim line qtys for an order (best-effort).
func (s *Service) openClaimQtyBySKU(ctx context.Context, orderID string) map[string]int64 {
	out := map[string]int64{}
	if s == nil || s.spannerClient == nil || strings.TrimSpace(orderID) == "" {
		return out
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT Status, LineItemsJSON FROM Claims@{FORCE_INDEX=Idx_Claims_ByOrderCreated}
			WHERE OrderId = @oid ORDER BY CreatedAt DESC LIMIT 50`,
		Params: map[string]any{"oid": strings.TrimSpace(orderID)},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.Warn("open claim qty lookup failed", "order_id", orderID, "err", err)
			return out
		}
		var status string
		var lineBytes []byte
		if err := row.Columns(&status, &lineBytes); err != nil {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "OPEN", "UNDER_REVIEW", "APPROVED", "RESOLVED":
		default:
			continue
		}
		var lines []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
			Qty      int64  `json:"qty"`
		}
		if err := json.Unmarshal(lineBytes, &lines); err != nil {
			continue
		}
		for _, ln := range lines {
			sku := strings.TrimSpace(ln.SKU)
			q := ln.Quantity
			if q == 0 {
				q = ln.Qty
			}
			if sku == "" || q <= 0 {
				continue
			}
			out[sku] += q
		}
	}
	return out
}

func (s *Service) listStockMovements(ctx context.Context, retailerID, locationID, sku string, limit int) ([]StockMovementDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []StockMovementDTO
		for i := len(s.stockMovements) - 1; i >= 0 && len(out) < limit; i-- {
			m := s.stockMovements[i]
			// no retailer on DTO in memory — filter loosely
			if locationID != "" && m.LocationID != locationID {
				continue
			}
			if sku != "" && m.Sku != sku {
				continue
			}
			out = append(out, m)
		}
		return out, nil
	}
	sql := `SELECT MovementId, LocationId, StockBin, Sku, Qty, MovementType, IFNULL(RefType,''), IFNULL(RefId,''), IFNULL(ActorUserId,''), IFNULL(Note,''), CreatedAt
		FROM RetailerStockMovements@{FORCE_INDEX=Idx_RetailerStockMovements_ByLocationCreated}
		WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	if sku != "" {
		sql += ` AND Sku = @sku`
		params["sku"] = sku
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`
	params["lim"] = int64(limit)
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []StockMovementDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m StockMovementDTO
		var created time.Time
		if err := row.Columns(&m.MovementID, &m.LocationID, &m.StockBin, &m.Sku, &m.Qty, &m.MovementType, &m.RefType, &m.RefID, &m.ActorUserID, &m.Note, &created); err != nil {
			return nil, err
		}
		m.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, m)
	}
	return out, nil
}

func (s *Service) saveStockCount(ctx context.Context, countID, retailerID, locationID, status, linesJSON, actor string, committed bool) error {
	if s.spannerClient == nil {
		return nil
	}
	row := map[string]any{
		"CountId":    countID,
		"RetailerId": retailerID,
		"LocationId": locationID,
		"Status":     status,
		"LinesJson":  linesJSON,
		"CreatedBy":  nullableStr(actor),
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if committed {
		row["CommittedAt"] = spanner.CommitTimestamp
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("RetailerStockCounts", row)})
	return err
}

// syncReorderCurrentStock writes store on-hand into ReorderSuggestions.CurrentStock when row exists.
func (s *Service) syncReorderCurrentStock(ctx context.Context, retailerID, sku string) error {
	if s.spannerClient == nil {
		return nil
	}
	total, err := s.SumOnHandForSKU(ctx, retailerID, sku)
	if err != nil {
		return err
	}
	// Best-effort update; ignore missing suggestion rows.
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE ReorderSuggestions SET CurrentStock = @qty, ComputedAt = CURRENT_TIMESTAMP()
				WHERE RetailerId = @rid AND Sku = @sku`,
			Params: map[string]any{"qty": total, "rid": retailerID, "sku": sku},
		})
		return err
	})
	return err
}

// InjectTestOrderLines seeds order line load for memory tests via receive without Spanner.
// Not used in production.
func (s *Service) injectMemoryReceive(orgID, locID, orderID string, lines []ReceiveLine) error {
	session := ReceiveSessionDTO{
		SessionID: s.newID(), RetailerID: orgID, LocationID: locID, OrderID: orderID,
		Status: "DRAFT", Lines: lines, CreatedAt: s.now().UTC().Format(time.RFC3339Nano),
	}
	return s.confirmReceiveSession(context.Background(), session, BinBackroom, "test")
}
