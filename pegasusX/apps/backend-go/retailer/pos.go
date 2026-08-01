package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// POS DTOs and constants.

const (
	RegisterStatusActive   = "ACTIVE"
	RegisterStatusInactive = "INACTIVE"
	PosSessionOpen         = "OPEN"
	PosSessionClosed       = "CLOSED"
	PosSaleCompleted       = "COMPLETED"
	PosSaleVoided          = "VOIDED"
	TenderCash             = "CASH"
	TenderCard             = "CARD"
	TenderOther            = "OTHER"
)

// RegisterDTO is a till/register.
type RegisterDTO struct {
	RegisterID string `json:"register_id"`
	RetailerID string `json:"retailer_id"`
	LocationID string `json:"location_id"`
	Label      string `json:"label"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// PosSessionDTO is an open/closed register session.
type PosSessionDTO struct {
	SessionID         string `json:"session_id"`
	RegisterID        string `json:"register_id"`
	LocationID        string `json:"location_id"`
	RetailerID        string `json:"retailer_id"`
	OpenedByUserID    string `json:"opened_by_user_id"`
	ClosedByUserID    string `json:"closed_by_user_id,omitempty"`
	Status            string `json:"status"`
	OpeningFloatMinor int64  `json:"opening_float_minor"`
	ClosingCashMinor  *int64 `json:"closing_cash_minor,omitempty"`
	ExpectedCashMinor *int64 `json:"expected_cash_minor,omitempty"`
	VarianceMinor     *int64 `json:"variance_minor,omitempty"`
	Currency          string `json:"currency"`
	OpenedAt          string `json:"opened_at,omitempty"`
	ClosedAt          string `json:"closed_at,omitempty"`
}

// PosSaleLine is one sold SKU line.
type PosSaleLine struct {
	Sku            string `json:"sku"`
	Name           string `json:"name,omitempty"`
	Qty            int64  `json:"qty"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	LineTotalMinor int64  `json:"line_total_minor"`
}

// PosTender is a payment tender.
type PosTender struct {
	Method      string `json:"method"` // CASH | CARD | OTHER
	AmountMinor int64  `json:"amount_minor"`
}

// PosSaleDTO is a completed or voided sale.
type PosSaleDTO struct {
	SaleID         string        `json:"sale_id"`
	SessionID      string        `json:"session_id"`
	RegisterID     string        `json:"register_id"`
	LocationID     string        `json:"location_id"`
	RetailerID     string        `json:"retailer_id"`
	CashierUserID  string        `json:"cashier_user_id"`
	Status         string        `json:"status"`
	TotalMinor     int64         `json:"total_minor"`
	Currency       string        `json:"currency"`
	ReceiptNumber  string        `json:"receipt_number"`
	Lines          []PosSaleLine `json:"lines"`
	Tenders        []PosTender   `json:"tenders"`
	StockBin       string        `json:"stock_bin"`
	CreatedAt      string        `json:"created_at,omitempty"`
	VoidedAt       string        `json:"voided_at,omitempty"`
	VoidedByUserID string        `json:"voided_by_user_id,omitempty"`
	VoidReason     string        `json:"void_reason,omitempty"`
}

// HandleRegisters serves GET/POST /v1/retailer/registers
func (s *Service) HandleRegisters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleRegistersGet(w, r)
	case http.MethodPost:
		s.handleRegistersPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleRegistersGet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermPosSell) && !auth.HasRetailerPerm(claims, auth.PermCapManage)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	items, err := s.listRegisters(r.Context(), orgID, locID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_registers_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) handleRegistersPost(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermCapManage) && !auth.HasRetailerPerm(claims, auth.PermLocationManage)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
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
		Label      string `json:"label"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		if p, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = p.LocationID
		}
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = "Register 1"
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	// POS hard-deps STORE_STOCK — auto-enable both packs for owner path.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackSTORESTOCK) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSTORESTOCK, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}
	if !enabled.Has(PackPOS) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackPOS, auth.ResolveRetailerUserID(claims), true, map[string]any{})
	}

	reg := RegisterDTO{
		RegisterID: s.newID(),
		RetailerID: orgID,
		LocationID: locID,
		Label:      label,
		Status:     RegisterStatusActive,
		CreatedAt:  s.now().UTC().Format(time.RFC3339Nano),
	}
	if err := s.saveRegister(r.Context(), reg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_register_failed"})
		return
	}
	respBytes, _ := json.Marshal(reg)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandlePosSessionOpen serves POST /v1/retailer/pos/sessions/open
func (s *Service) HandlePosSessionOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermPosSell) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermPosSell})
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
		RegisterID        string `json:"register_id"`
		OpeningFloatMinor int64  `json:"opening_float_minor"`
		Currency          string `json:"currency"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	regID := strings.TrimSpace(req.RegisterID)
	if regID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "register_id_required"})
		return
	}
	reg, found, err := s.getRegister(r.Context(), regID)
	if err != nil || !found || reg.RetailerID != orgID || reg.Status != RegisterStatusActive {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "register_not_found"})
		return
	}
	// One open session per register.
	if open, okOpen, _ := s.getOpenSessionForRegister(r.Context(), regID); okOpen {
		respBytes, _ := json.Marshal(open)
		idemCommitted = true
		s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
		writeJSONBytes(w, http.StatusOK, respBytes)
		return
	}
	// Phase 5: SHIFTS pack + require_shift_to_open_register → must be clocked in.
	userID := auth.ResolveRetailerUserID(claims)
	if err := s.requireClockedInForPOS(r.Context(), orgID, userID, reg.LocationID); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
	if req.OpeningFloatMinor < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "opening_float_invalid"})
		return
	}
	sess := PosSessionDTO{
		SessionID:         s.newID(),
		RegisterID:        regID,
		LocationID:        reg.LocationID,
		RetailerID:        orgID,
		OpenedByUserID:    userID,
		Status:            PosSessionOpen,
		OpeningFloatMinor: req.OpeningFloatMinor,
		Currency:          currency,
		OpenedAt:          s.now().UTC().Format(time.RFC3339Nano),
	}
	if err := s.savePosSession(r.Context(), sess); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "open_session_failed"})
		return
	}
	// Link open shift (if any) for cash recon on shift close.
	s.linkShiftToPosSession(r.Context(), regID, sess.SessionID)
	_ = s.emitPosEvent(r.Context(), orgID, events.EventPosSessionOpened, map[string]any{
		"session_id":  sess.SessionID,
		"register_id": regID,
		"location_id": reg.LocationID,
	})
	respBytes, _ := json.Marshal(sess)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandlePosSessionClose serves POST /v1/retailer/pos/sessions/{sessionID}/close
func (s *Service) HandlePosSessionClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermPosSell) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionID"))
	sess, found, err := s.getPosSession(r.Context(), sessionID)
	if err != nil || !found || sess.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session_not_found"})
		return
	}
	if sess.Status == PosSessionClosed {
		writeJSON(w, http.StatusOK, sess)
		return
	}
	var req struct {
		ClosingCashMinor int64 `json:"closing_cash_minor"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// expected = opening + cash sales - cash voids (simplified: sum CASH tenders on COMPLETED - VOIDED)
	cashSales, err := s.sumSessionCashTenders(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cash_sum_failed"})
		return
	}
	expected := sess.OpeningFloatMinor + cashSales
	variance := req.ClosingCashMinor - expected
	now := s.now().UTC().Format(time.RFC3339Nano)
	sess.Status = PosSessionClosed
	sess.ClosedByUserID = auth.ResolveRetailerUserID(claims)
	sess.ClosingCashMinor = &req.ClosingCashMinor
	sess.ExpectedCashMinor = &expected
	sess.VarianceMinor = &variance
	sess.ClosedAt = now
	if err := s.savePosSession(r.Context(), sess); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "close_session_failed"})
		return
	}
	// Phase 5: variance alert when SHIFTS pack on and threshold breached.
	cfg := s.shiftsConfig(r.Context(), orgID)
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if enabled.Has(PackSHIFTS) && abs64(variance) >= cfg.varianceAlertMinor {
		s.alertOwnersPosVariance(r.Context(), orgID, sess, variance)
	}
	_ = s.emitPosEvent(r.Context(), orgID, events.EventPosSessionClosed, map[string]any{
		"session_id":     sessionID,
		"variance_minor": variance,
		"expected_minor": expected,
		"closing_minor":  req.ClosingCashMinor,
	})
	writeJSON(w, http.StatusOK, sess)
}

// HandlePosSessionGet serves GET /v1/retailer/pos/sessions/{sessionID}
func (s *Service) HandlePosSessionGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermPosSell) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	sessionID := strings.TrimSpace(chi.URLParam(r, "sessionID"))
	sess, found, err := s.getPosSession(r.Context(), sessionID)
	if err != nil || !found || sess.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session_not_found"})
		return
	}
	sales, _ := s.listSessionSales(r.Context(), sessionID, 100)
	writeJSON(w, http.StatusOK, map[string]any{"session": sess, "sales": sales})
}

// HandlePosSale serves POST /v1/retailer/pos/sales
func (s *Service) HandlePosSale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermPosSell) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermPosSell})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, okBody := readLimitedBody(w, r, 128*1024)
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
		SessionID string        `json:"session_id"`
		StockBin  string        `json:"stock_bin"`
		Currency  string        `json:"currency"`
		Lines     []PosSaleLine `json:"lines"`
		Tenders   []PosTender   `json:"tenders"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" || len(req.Lines) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session_and_lines_required"})
		return
	}
	sess, found, err := s.getPosSession(r.Context(), sessionID)
	if err != nil || !found || sess.RetailerID != orgID || sess.Status != PosSessionOpen {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session_not_open"})
		return
	}

	// Normalize lines
	var total int64
	lines := make([]PosSaleLine, 0, len(req.Lines))
	for _, l := range req.Lines {
		sku := strings.TrimSpace(l.Sku)
		if sku == "" || l.Qty <= 0 || l.UnitPriceMinor < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_line", "sku": sku})
			return
		}
		lineTotal := l.Qty * l.UnitPriceMinor
		lines = append(lines, PosSaleLine{
			Sku: sku, Name: strings.TrimSpace(l.Name), Qty: l.Qty,
			UnitPriceMinor: l.UnitPriceMinor, LineTotalMinor: lineTotal,
		})
		total += lineTotal
	}
	// Tenders must sum to total
	var tenderSum int64
	tenders := make([]PosTender, 0, len(req.Tenders))
	if len(req.Tenders) == 0 {
		// default cash
		tenders = append(tenders, PosTender{Method: TenderCash, AmountMinor: total})
		tenderSum = total
	} else {
		for _, t := range req.Tenders {
			method := strings.ToUpper(strings.TrimSpace(t.Method))
			if method == "" {
				method = TenderCash
			}
			if t.AmountMinor <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_tender"})
				return
			}
			tenders = append(tenders, PosTender{Method: method, AmountMinor: t.AmountMinor})
			tenderSum += t.AmountMinor
		}
	}
	if tenderSum != total {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":       "tender_total_mismatch",
			"total_minor": fmt.Sprintf("%d", total),
			"tender_sum":  fmt.Sprintf("%d", tenderSum),
		})
		return
	}

	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinFloor
	}
	// Prefer FLOOR; if insufficient, try BACKROOM then fail.
	actor := auth.ResolveRetailerUserID(claims)
	saleID := s.newID()
	receipt := s.nextReceiptNumber(orgID)

	// Decrement stock for each line
	for _, l := range lines {
		if err := s.decrementForSale(r.Context(), orgID, sess.LocationID, bin, l.Sku, l.Qty, saleID, actor); err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "sku": l.Sku})
			return
		}
		_ = s.syncReorderCurrentStock(r.Context(), orgID, l.Sku)
	}

	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = sess.Currency
	}
	if currency == "" {
		currency = "UZS"
	}
	sale := PosSaleDTO{
		SaleID:        saleID,
		SessionID:     sessionID,
		RegisterID:    sess.RegisterID,
		LocationID:    sess.LocationID,
		RetailerID:    orgID,
		CashierUserID: actor,
		Status:        PosSaleCompleted,
		TotalMinor:    total,
		Currency:      currency,
		ReceiptNumber: receipt,
		Lines:         lines,
		Tenders:       tenders,
		StockBin:      bin,
		CreatedAt:     s.now().UTC().Format(time.RFC3339Nano),
	}
	if err := s.savePosSale(r.Context(), sale); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_sale_failed"})
		return
	}
	_ = s.emitPosEvent(r.Context(), orgID, events.EventPosSaleCompleted, map[string]any{
		"sale_id":       saleID,
		"session_id":    sessionID,
		"total_minor":   total,
		"receipt":       receipt,
		"location_id":   sess.LocationID,
	})
	respBytes, _ := json.Marshal(sale)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandlePosSaleVoid serves POST /v1/retailer/pos/sales/{saleID}/void
func (s *Service) HandlePosSaleVoid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	// Void: pos.void full, or cashier limited to same session same day via role MANAGER+
	role := auth.EffectiveRetailerRole(claims)
	canVoid := auth.HasRetailerPerm(claims, auth.PermPosVoid) ||
		role == "MANAGER" || role == "ADMIN" || role == "OWNER"
	// Cashiers have limited void in matrix - HasRetailerPerm for CASHIER does not include pos.void fully
	// Allow CASHIER void if same cashier and sale total under threshold
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	saleID := strings.TrimSpace(chi.URLParam(r, "saleID"))
	sale, found, err := s.getPosSale(r.Context(), saleID)
	if err != nil || !found || sale.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "sale_not_found"})
		return
	}
	if sale.Status == PosSaleVoided {
		writeJSON(w, http.StatusOK, sale)
		return
	}
	// Session must still be open for cashier self-void; managers can void closed session same day.
	sess, sessOK, _ := s.getPosSession(r.Context(), sale.SessionID)
	actor := auth.ResolveRetailerUserID(claims)
	const cashierVoidMaxMinor int64 = 500_000 // 5000.00 if 2-decimal
	if !canVoid {
		if role == "CASHIER" && sale.CashierUserID == actor && sessOK && sess.Status == PosSessionOpen && sale.TotalMinor <= cashierVoidMaxMinor {
			// limited cashier void
		} else {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "void_requires_manager", "permission": auth.PermPosVoid})
			return
		}
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Restock
	for _, l := range sale.Lines {
		_ = s.applyDelta(r.Context(), orgID, sale.LocationID, sale.StockBin, l.Sku, l.Qty, MoveSaleVoid, "SALE", saleID, actor, "void")
		_ = s.syncReorderCurrentStock(r.Context(), orgID, l.Sku)
	}
	sale.Status = PosSaleVoided
	sale.VoidedAt = s.now().UTC().Format(time.RFC3339Nano)
	sale.VoidedByUserID = actor
	sale.VoidReason = strings.TrimSpace(req.Reason)
	if err := s.savePosSale(r.Context(), sale); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "void_failed"})
		return
	}
	_ = s.emitPosEvent(r.Context(), orgID, events.EventPosSaleVoided, map[string]any{
		"sale_id": saleID, "reason": sale.VoidReason,
	})
	writeJSON(w, http.StatusOK, sale)
}

// HandlePosSaleRefund is alias for void in v1 (same-day full refund).
func (s *Service) HandlePosSaleRefund(w http.ResponseWriter, r *http.Request) {
	s.HandlePosSaleVoid(w, r)
}

// ---- stock helper for POS ----

func (s *Service) decrementForSale(ctx context.Context, retailerID, locationID, preferredBin, sku string, qty int64, saleID, actor string) error {
	// Try preferred bin first, then FLOOR, then BACKROOM.
	order := []string{preferredBin, BinFloor, BinBackroom}
	seen := map[string]bool{}
	var lastErr error
	for _, bin := range order {
		bin = normalizeBin(bin)
		if bin == "" || seen[bin] {
			continue
		}
		seen[bin] = true
		onHand, _ := s.getOnHand(ctx, locationID, bin, sku)
		if onHand >= qty {
			return s.applyDelta(ctx, retailerID, locationID, bin, sku, -qty, MoveSale, "SALE", saleID, actor, "pos_sale")
		}
		lastErr = errors.New("insufficient_stock")
	}
	if lastErr == nil {
		lastErr = errors.New("insufficient_stock")
	}
	return lastErr
}

// ---- persistence ----

func (s *Service) listRegisters(ctx context.Context, retailerID, locationID string) ([]RegisterDTO, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []RegisterDTO
		for _, reg := range s.posRegisters {
			if reg.RetailerID != retailerID {
				continue
			}
			if locationID != "" && reg.LocationID != locationID {
				continue
			}
			out = append(out, reg)
		}
		return out, nil
	}
	sql := `SELECT RegisterId, RetailerId, LocationId, Label, Status, CreatedAt
		FROM RetailerRegisters WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	sql += ` ORDER BY UpdatedAt DESC`
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []RegisterDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var reg RegisterDTO
		var created time.Time
		if err := row.Columns(&reg.RegisterID, &reg.RetailerID, &reg.LocationID, &reg.Label, &reg.Status, &created); err != nil {
			return nil, err
		}
		reg.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, reg)
	}
	return out, nil
}

func (s *Service) saveRegister(ctx context.Context, reg RegisterDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.posRegisters == nil {
			s.posRegisters = map[string]RegisterDTO{}
		}
		s.posRegisters[reg.RegisterID] = reg
		return nil
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("RetailerRegisters", map[string]any{
			"RegisterId": reg.RegisterID,
			"RetailerId": reg.RetailerID,
			"LocationId": reg.LocationID,
			"Label":      reg.Label,
			"Status":     reg.Status,
			"CreatedAt":  spanner.CommitTimestamp,
			"UpdatedAt":  spanner.CommitTimestamp,
		}),
	})
	return err
}

func (s *Service) getRegister(ctx context.Context, registerID string) (RegisterDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		reg, ok := s.posRegisters[registerID]
		return reg, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerRegisters", spanner.Key{registerID},
		[]string{"RegisterId", "RetailerId", "LocationId", "Label", "Status", "CreatedAt"})
	if err != nil {
		if isNotFound(err) {
			return RegisterDTO{}, false, nil
		}
		return RegisterDTO{}, false, err
	}
	var reg RegisterDTO
	var created time.Time
	if err := row.Columns(&reg.RegisterID, &reg.RetailerID, &reg.LocationID, &reg.Label, &reg.Status, &created); err != nil {
		return RegisterDTO{}, false, err
	}
	reg.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	return reg, true, nil
}

func (s *Service) savePosSession(ctx context.Context, sess PosSessionDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.posSessions == nil {
			s.posSessions = map[string]PosSessionDTO{}
		}
		s.posSessions[sess.SessionID] = sess
		return nil
	}
	row := map[string]any{
		"SessionId":         sess.SessionID,
		"RegisterId":        sess.RegisterID,
		"LocationId":        sess.LocationID,
		"RetailerId":        sess.RetailerID,
		"OpenedByUserId":    sess.OpenedByUserID,
		"Status":            sess.Status,
		"OpeningFloatMinor": sess.OpeningFloatMinor,
		"Currency":          sess.Currency,
		"OpenedAt":          spanner.CommitTimestamp,
	}
	if sess.ClosedByUserID != "" {
		row["ClosedByUserId"] = sess.ClosedByUserID
	}
	if sess.ClosingCashMinor != nil {
		row["ClosingCashMinor"] = *sess.ClosingCashMinor
	}
	if sess.ExpectedCashMinor != nil {
		row["ExpectedCashMinor"] = *sess.ExpectedCashMinor
	}
	if sess.VarianceMinor != nil {
		row["VarianceMinor"] = *sess.VarianceMinor
	}
	if sess.Status == PosSessionClosed {
		row["ClosedAt"] = spanner.CommitTimestamp
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerPosSessions", row)})
	return err
}

func (s *Service) getPosSession(ctx context.Context, sessionID string) (PosSessionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		sess, ok := s.posSessions[sessionID]
		return sess, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerPosSessions", spanner.Key{sessionID},
		[]string{"SessionId", "RegisterId", "LocationId", "RetailerId", "OpenedByUserId", "ClosedByUserId",
			"Status", "OpeningFloatMinor", "ClosingCashMinor", "ExpectedCashMinor", "VarianceMinor",
			"Currency", "OpenedAt", "ClosedAt"})
	if err != nil {
		if isNotFound(err) {
			return PosSessionDTO{}, false, nil
		}
		return PosSessionDTO{}, false, err
	}
	return decodePosSessionRow(row)
}

func decodePosSessionRow(row *spanner.Row) (PosSessionDTO, bool, error) {
	var sess PosSessionDTO
	var closedBy spanner.NullString
	var closeCash, expected, variance spanner.NullInt64
	var opened, closed spanner.NullTime
	if err := row.Columns(
		&sess.SessionID, &sess.RegisterID, &sess.LocationID, &sess.RetailerID, &sess.OpenedByUserID, &closedBy,
		&sess.Status, &sess.OpeningFloatMinor, &closeCash, &expected, &variance,
		&sess.Currency, &opened, &closed,
	); err != nil {
		return PosSessionDTO{}, false, err
	}
	if closedBy.Valid {
		sess.ClosedByUserID = closedBy.StringVal
	}
	if closeCash.Valid {
		v := closeCash.Int64
		sess.ClosingCashMinor = &v
	}
	if expected.Valid {
		v := expected.Int64
		sess.ExpectedCashMinor = &v
	}
	if variance.Valid {
		v := variance.Int64
		sess.VarianceMinor = &v
	}
	if opened.Valid {
		sess.OpenedAt = opened.Time.UTC().Format(time.RFC3339Nano)
	}
	if closed.Valid {
		sess.ClosedAt = closed.Time.UTC().Format(time.RFC3339Nano)
	}
	return sess, true, nil
}

func (s *Service) getOpenSessionForRegister(ctx context.Context, registerID string) (PosSessionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, sess := range s.posSessions {
			if sess.RegisterID == registerID && sess.Status == PosSessionOpen {
				return sess, true, nil
			}
		}
		return PosSessionDTO{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, RegisterId, LocationId, RetailerId, OpenedByUserId, ClosedByUserId,
			Status, OpeningFloatMinor, ClosingCashMinor, ExpectedCashMinor, VarianceMinor,
			Currency, OpenedAt, ClosedAt
			FROM RetailerPosSessions@{FORCE_INDEX=Idx_RetailerPosSessions_ByRegister}
			WHERE RegisterId = @rid AND Status = @st
			ORDER BY OpenedAt DESC LIMIT 1`,
		Params: map[string]any{"rid": registerID, "st": PosSessionOpen},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return PosSessionDTO{}, false, nil
	}
	if err != nil {
		return PosSessionDTO{}, false, err
	}
	return decodePosSessionRow(row)
}

func (s *Service) savePosSale(ctx context.Context, sale PosSaleDTO) error {
	linesJSON, _ := json.Marshal(sale.Lines)
	tendersJSON, _ := json.Marshal(sale.Tenders)
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.posSales == nil {
			s.posSales = map[string]PosSaleDTO{}
		}
		s.posSales[sale.SaleID] = sale
		return nil
	}
	row := map[string]any{
		"SaleId":        sale.SaleID,
		"SessionId":     sale.SessionID,
		"RegisterId":    sale.RegisterID,
		"LocationId":    sale.LocationID,
		"RetailerId":    sale.RetailerID,
		"CashierUserId": sale.CashierUserID,
		"Status":        sale.Status,
		"TotalMinor":    sale.TotalMinor,
		"Currency":      sale.Currency,
		"ReceiptNumber": sale.ReceiptNumber,
		"LinesJson":     string(linesJSON),
		"TendersJson":   string(tendersJSON),
		"StockBin":      sale.StockBin,
		"CreatedAt":     spanner.CommitTimestamp,
	}
	if sale.Status == PosSaleVoided {
		row["VoidedAt"] = spanner.CommitTimestamp
		row["VoidedByUserId"] = nullableStr(sale.VoidedByUserID)
		row["VoidReason"] = nullableStr(sale.VoidReason)
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerPosSales", row)})
	return err
}

func (s *Service) getPosSale(ctx context.Context, saleID string) (PosSaleDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		sale, ok := s.posSales[saleID]
		return sale, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerPosSales", spanner.Key{saleID},
		[]string{"SaleId", "SessionId", "RegisterId", "LocationId", "RetailerId", "CashierUserId",
			"Status", "TotalMinor", "Currency", "ReceiptNumber", "LinesJson", "TendersJson", "StockBin",
			"CreatedAt", "VoidedAt", "VoidedByUserId", "VoidReason"})
	if err != nil {
		if isNotFound(err) {
			return PosSaleDTO{}, false, nil
		}
		return PosSaleDTO{}, false, err
	}
	return decodePosSaleRow(row)
}

func decodePosSaleRow(row *spanner.Row) (PosSaleDTO, bool, error) {
	var sale PosSaleDTO
	var linesJSON, tendersJSON string
	var created time.Time
	var voided spanner.NullTime
	var voidedBy, voidReason spanner.NullString
	if err := row.Columns(
		&sale.SaleID, &sale.SessionID, &sale.RegisterID, &sale.LocationID, &sale.RetailerID, &sale.CashierUserID,
		&sale.Status, &sale.TotalMinor, &sale.Currency, &sale.ReceiptNumber, &linesJSON, &tendersJSON, &sale.StockBin,
		&created, &voided, &voidedBy, &voidReason,
	); err != nil {
		return PosSaleDTO{}, false, err
	}
	_ = json.Unmarshal([]byte(linesJSON), &sale.Lines)
	_ = json.Unmarshal([]byte(tendersJSON), &sale.Tenders)
	sale.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	if voided.Valid {
		sale.VoidedAt = voided.Time.UTC().Format(time.RFC3339Nano)
	}
	if voidedBy.Valid {
		sale.VoidedByUserID = voidedBy.StringVal
	}
	if voidReason.Valid {
		sale.VoidReason = voidReason.StringVal
	}
	return sale, true, nil
}

func (s *Service) listSessionSales(ctx context.Context, sessionID string, limit int) ([]PosSaleDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []PosSaleDTO
		for _, sale := range s.posSales {
			if sale.SessionID == sessionID {
				out = append(out, sale)
			}
		}
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT SaleId, SessionId, RegisterId, LocationId, RetailerId, CashierUserId,
			Status, TotalMinor, Currency, ReceiptNumber, LinesJson, TendersJson, StockBin,
			CreatedAt, VoidedAt, VoidedByUserId, VoidReason
			FROM RetailerPosSales@{FORCE_INDEX=Idx_RetailerPosSales_BySession}
			WHERE SessionId = @sid ORDER BY CreatedAt DESC LIMIT @lim`,
		Params: map[string]any{"sid": sessionID, "lim": int64(limit)},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []PosSaleDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		sale, _, err := decodePosSaleRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, sale)
	}
	return out, nil
}

func (s *Service) sumSessionCashTenders(ctx context.Context, sessionID string) (int64, error) {
	sales, err := s.listSessionSales(ctx, sessionID, 500)
	if err != nil {
		return 0, err
	}
	var cash int64
	for _, sale := range sales {
		if sale.Status != PosSaleCompleted {
			continue
		}
		for _, t := range sale.Tenders {
			if strings.EqualFold(t.Method, TenderCash) {
				cash += t.AmountMinor
			}
		}
	}
	return cash, nil
}

func (s *Service) nextReceiptNumber(retailerID string) string {
	// Simple monotonic-ish receipt: time-based (unique index handles rare collisions).
	return fmt.Sprintf("R%s", s.now().UTC().Format("20060102150405.000000"))
}

func (s *Service) emitPosEvent(ctx context.Context, retailerID, eventType string, fields map[string]any) error {
	if s.spannerClient == nil {
		return nil
	}
	payload := map[string]any{
		"type":        eventType,
		"timestamp":   s.now().Format(time.RFC3339Nano),
		"retailer_id": retailerID,
	}
	for k, v := range fields {
		payload[k] = v
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	return err
}
