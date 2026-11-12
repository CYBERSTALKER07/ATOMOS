package retailer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
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
	DamagedQty  int64  `json:"damaged_qty,omitempty"`
	MissingQty  int64  `json:"missing_qty,omitempty"`
}

// ReceiveSessionDTO wire shape.
type ReceiveSessionDTO struct {
	SessionID   string        `json:"session_id"`
	RetailerID  string        `json:"retailer_id"`
	LocationID  string        `json:"location_id"`
	OrderID     string        `json:"order_id"`
	Status      string        `json:"status"`
	Lines       []ReceiveLine `json:"lines"`
	CreatedAt   string        `json:"created_at,omitempty"`
	ConfirmedAt string        `json:"confirmed_at,omitempty"`
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
	if binFilter := normalizeBin(r.URL.Query().Get("stock_bin")); binFilter != "" {
		filtered := make([]StockBalanceDTO, 0, len(rows))
		for _, row := range rows {
			if row.StockBin == binFilter {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	if rows == nil {
		rows = []StockBalanceDTO{}
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
			DamagedQty  int64  `json:"damaged_qty"`
			MissingQty  int64  `json:"missing_qty"`
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
		bySKU := map[string]*struct {
			AcceptedQty int64
			DamagedQty  int64
			MissingQty  int64
		}{}
		for _, l := range req.Lines {
			lCopy := l
			bySKU[strings.TrimSpace(l.Sku)] = &struct {
				AcceptedQty int64
				DamagedQty  int64
				MissingQty  int64
			}{lCopy.AcceptedQty, lCopy.DamagedQty, lCopy.MissingQty}
		}
		for i := range lines {
			if data, ok := bySKU[lines[i].Sku]; ok {
				lines[i].AcceptedQty = data.AcceptedQty
				lines[i].DamagedQty = data.DamagedQty
				lines[i].MissingQty = data.MissingQty
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
			DamagedQty  int64  `json:"damaged_qty"`
			MissingQty  int64  `json:"missing_qty"`
		} `json:"lines"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if len(req.Lines) > 0 {
		bySKU := map[string]*struct {
			AcceptedQty int64
			DamagedQty  int64
			MissingQty  int64
		}{}
		for _, l := range req.Lines {
			lCopy := l
			bySKU[strings.TrimSpace(l.Sku)] = &struct {
				AcceptedQty int64
				DamagedQty  int64
				MissingQty  int64
			}{lCopy.AcceptedQty, lCopy.DamagedQty, lCopy.MissingQty}
		}
		for i := range session.Lines {
			if data, ok := bySKU[session.Lines[i].Sku]; ok {
				session.Lines[i].AcceptedQty = data.AcceptedQty
				session.Lines[i].DamagedQty = data.DamagedQty
				session.Lines[i].MissingQty = data.MissingQty
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
		if s.spannerClient == nil {
			for _, l := range lines {
				if l.Variance == 0 {
					continue
				}
				if err := s.applyAdjust(r.Context(), orgID, locID, bin, l.Sku, l.Variance, actor, "cycle_count:"+countID); err != nil {
					writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "sku": l.Sku})
					return
				}
			}
		} else {
			_, err := s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				for _, l := range lines {
					if l.Variance == 0 {
						continue
					}
					if err := s.applyDeltaInTxn(ctx, txn, orgID, locID, bin, l.Sku, l.Variance, MoveCountVariance, "ADJUST", "", actor, "cycle_count:"+countID); err != nil {
						return fmt.Errorf("sku %s: %w", l.Sku, err)
					}
				}
				return nil
			})
			if err != nil {
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
		}

		for _, l := range lines {
			if l.Variance != 0 {
				_ = s.syncReorderCurrentStock(r.Context(), orgID, l.Sku)
			}
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
	if movs == nil {
		movs = []StockMovementDTO{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": movs})
}

// ---- core stock math ----
