package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	Currency          string            `json:"currency"`
	OpenedAt          string            `json:"opened_at,omitempty"`
	ClosedAt          string            `json:"closed_at,omitempty"`
	AnomalyFlags      []string          `json:"anomaly_flags,omitempty"`
	AnomalyReview     *PosAnomalyReview `json:"anomaly_review,omitempty"`
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
	SaleID          string        `json:"sale_id"`
	SessionID       string        `json:"session_id"`
	RegisterID      string        `json:"register_id"`
	LocationID      string        `json:"location_id"`
	RetailerID      string        `json:"retailer_id"`
	CashierUserID   string        `json:"cashier_user_id"`
	Status          string        `json:"status"`
	TotalMinor      int64         `json:"total_minor"`
	Currency        string        `json:"currency"`
	ReceiptNumber   string        `json:"receipt_number"`
	Lines           []PosSaleLine `json:"lines"`
	Tenders         []PosTender   `json:"tenders"`
	StockBin        string        `json:"stock_bin"`
	CreatedAt       string        `json:"created_at,omitempty"`
	VoidedAt        string        `json:"voided_at,omitempty"`
	VoidedByUserID  string        `json:"voided_by_user_id,omitempty"`
	VoidReason      string        `json:"void_reason,omitempty"`
	ClientSaleID    string        `json:"client_sale_id,omitempty"`
	Origin          string            `json:"origin,omitempty"` // online | offline
	ClientCreatedAt string            `json:"client_created_at,omitempty"`
	AnomalyFlags    []string          `json:"anomaly_flags,omitempty"`
	AnomalyReview   *PosAnomalyReview `json:"anomaly_review,omitempty"`
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
	enabled, err := s.LoadEnabledPacks(r.Context(), orgID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_packs_failed"})
		return
	}
	if !enabled.Has(PackSTORESTOCK) {
		if err := s.SetPackEnabled(r.Context(), orgID, PackSTORESTOCK, auth.ResolveRetailerUserID(claims), true, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "enable_store_stock_failed"})
			return
		}
	}
	if !enabled.Has(PackPOS) {
		if err := s.SetPackEnabled(r.Context(), orgID, PackPOS, auth.ResolveRetailerUserID(claims), true, map[string]any{}); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "enable_pos_failed"})
			return
		}
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
	respBytes, err := json.Marshal(reg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_failed"})
		return
	}
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(respBytes); err != nil && s.log != nil {
		s.log.Warn("register response write failed", "err", err)
	}
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
	currency, err := stampPackCurrency(r.Context(), req.Currency)
	if err != nil {
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
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
	if err := s.emitPosEvent(r.Context(), orgID, events.EventPosSessionOpened, map[string]any{
		"session_id":  sess.SessionID,
		"register_id": regID,
		"location_id": reg.LocationID,
	}); err != nil && s.log != nil {
		s.log.Warn("pos session open event emit failed", "session_id", sess.SessionID, "err", err)
	}
	respBytes, err := json.Marshal(sess)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_failed"})
		return
	}
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(respBytes); err != nil && s.log != nil {
		s.log.Warn("pos session open response write failed", "err", err)
	}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

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
	enabled, err := s.LoadEnabledPacks(r.Context(), orgID)
	if err != nil && s.log != nil {
		s.log.Warn("load enabled packs failed on session close", "retailer_id", orgID, "err", err)
	}
	if err == nil && enabled.Has(PackSHIFTS) && abs64(variance) >= cfg.varianceAlertMinor {
		s.alertOwnersPosVariance(r.Context(), orgID, sess, variance)
	}
	if err := s.emitPosEvent(r.Context(), orgID, events.EventPosSessionClosed, map[string]any{
		"session_id":     sessionID,
		"variance_minor": variance,
		"expected_minor": expected,
		"closing_minor":  req.ClosingCashMinor,
	}); err != nil && s.log != nil {
		s.log.Warn("pos session close event emit failed", "session_id", sessionID, "err", err)
	}
	sessSales, salesErr := s.listSessionSales(r.Context(), sessionID, 500)
	if salesErr != nil && s.log != nil {
		s.log.Warn("list session sales for anomaly review failed", "session_id", sessionID, "err", salesErr)
	}
	if review, flagged := s.inspectSessionPosAnomaly(r.Context(), orgID, sess, sessSales); flagged {
		sess.AnomalyFlags = []string{PosAnomalyReviewFlag}
		sess.AnomalyReview = &review
	}
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
		SessionID       string        `json:"session_id"`
		StockBin        string        `json:"stock_bin"`
		Currency        string        `json:"currency"`
		Lines           []PosSaleLine `json:"lines"`
		Tenders         []PosTender   `json:"tenders"`
		ClientSaleID    string        `json:"client_sale_id"`
		ClientCreatedAt string        `json:"client_created_at"`
		Origin          string        `json:"origin"` // online | offline
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
	clientSaleID := strings.TrimSpace(req.ClientSaleID)
	origin := strings.ToLower(strings.TrimSpace(req.Origin))
	if origin == "" {
		origin = "online"
	}
	if origin != "online" && origin != "offline" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid_origin", "allowed": "online,offline"})
		return
	}

	// Offline origin must be cash-only (no card capture offline).
	if origin == "offline" {
		for _, t := range req.Tenders {
			method := strings.ToUpper(strings.TrimSpace(t.Method))
			if method != "" && method != TenderCash {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
					"error":  "offline_card_forbidden",
					"detail": "offline sales allow CASH tender only",
				})
				return
			}
		}
	}

	// Idempotent return by client_sale_id (survives beyond HTTP idempotency TTL).
	if clientSaleID != "" {
		if existing, ok := s.getPosSaleByClientID(r.Context(), orgID, clientSaleID); ok {
			respBytes, err := json.Marshal(existing)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_failed"})
				return
			}
			idemCommitted = true
			s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write(respBytes); err != nil && s.log != nil {
				s.log.Warn("pos sale replay response write failed", "err", err)
			}
			return
		}
	}

	sess, found, err := s.getPosSession(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session_lookup_failed"})
		return
	}
	if !found || sess.RetailerID != orgID {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session_not_open"})
		return
	}
	// Historical offline POS sync is allowed on a closed session owned by this tenant
	// (CASH-only + client_sale_id idempotency still enforced); online sales require OPEN.
	isClosedOfflineSync := origin == "offline" && sess.Status == PosSessionClosed
	if sess.Status != PosSessionOpen && !isClosedOfflineSync {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session_not_open"})
		return
	}

	// Normalize lines (local catalog → local: namespace; barcode → catalog id)
	var total int64
	lines := make([]PosSaleLine, 0, len(req.Lines))
	for _, l := range req.Lines {
		sku, name, errMsg := s.validatePosSaleSKU(r.Context(), orgID, l.Sku, l.Name, l.UnitPriceMinor)
		if errMsg == "local_skus_failed" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "local_skus_failed"})
			return
		}
		if errMsg != "" || l.Qty <= 0 || l.UnitPriceMinor < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_line", "sku": sku, "detail": errMsg})
			return
		}
		lineTotal := l.Qty * l.UnitPriceMinor
		lines = append(lines, PosSaleLine{
			Sku: sku, Name: name, Qty: l.Qty,
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
	// Re-check offline tenders after defaulting empty → cash.
	if origin == "offline" {
		for _, t := range tenders {
			if t.Method != TenderCash {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "offline_card_forbidden"})
				return
			}
		}
	}

	currency, err := stampPackCurrency(r.Context(), firstNonEmpty(req.Currency, sess.Currency))
	if err != nil {
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}

	bin := normalizeBin(req.StockBin)
	if bin == "" {
		bin = BinFloor
	}
	actor := auth.ResolveRetailerUserID(claims)
	saleID := s.newID()
	receipt := s.nextReceiptNumberWithSaleID(orgID, saleID)
	sale := PosSaleDTO{
		SaleID:          saleID,
		SessionID:       sessionID,
		RegisterID:      sess.RegisterID,
		LocationID:      sess.LocationID,
		RetailerID:      orgID,
		CashierUserID:   actor,
		Status:          PosSaleCompleted,
		TotalMinor:      total,
		Currency:        currency,
		ReceiptNumber:   receipt,
		Lines:           lines,
		Tenders:         tenders,
		StockBin:        bin,
		CreatedAt:       s.now().UTC().Format(time.RFC3339Nano),
		ClientSaleID:    clientSaleID,
		Origin:          origin,
		ClientCreatedAt: strings.TrimSpace(req.ClientCreatedAt),
	}

	committedSale, chosenBins, replayed, err := s.commitPosSaleAtomic(r.Context(), sale, bin, actor, origin == "offline")
	if err != nil {
		var stockErr *posStockConflictError
		if errors.As(err, &stockErr) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": stockErr.err.Error(), "sku": stockErr.sku})
			return
		}
		if errors.Is(err, errPosSessionNotOpen) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "session_not_open"})
			return
		}
		if s.log != nil {
			s.log.Warn("pos sale atomic commit failed", "retailer_id", orgID, "sale_id", saleID, "err", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist_sale_failed"})
		return
	}

	if !replayed {
		// Best-effort secondary projections after atomic sale+stock commit.
		for i, l := range committedSale.Lines {
			if err := s.syncReorderCurrentStock(r.Context(), orgID, l.Sku); err != nil && s.log != nil {
				s.log.Warn("pos sale reorder stock sync failed", "retailer_id", orgID, "sku", l.Sku, "err", err)
			}
			checkBin := bin
			if i < len(chosenBins) && chosenBins[i] != "" {
				checkBin = chosenBins[i]
			}
			onHand, ohErr := s.getOnHand(r.Context(), sess.LocationID, checkBin, l.Sku)
			if ohErr == nil {
				if err := s.CheckAndGenerateOOSAlerts(r.Context(), orgID, sess.LocationID, l.Sku, onHand, 10); err != nil && s.log != nil {
					s.log.Warn("pos sale OOS alert check failed", "retailer_id", orgID, "sku", l.Sku, "err", err)
				}
			}
		}
		s.refreshHqStockSnapshotsForSale(r.Context(), orgID, sess.LocationID, bin, committedSale.Lines)
		s.recordSellThroughSale(r.Context(), orgID, sess.LocationID, committedSale.Lines)
	}

	respBytes, err := json.Marshal(committedSale)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "marshal_failed"})
		return
	}
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(respBytes); err != nil && s.log != nil {
		s.log.Warn("pos sale response write failed", "err", err)
	}
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
	sess, sessOK, err := s.getPosSession(r.Context(), sale.SessionID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session_lookup_failed"})
		return
	}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	voidedSale, alreadyVoided, err := s.commitPosVoidAtomic(r.Context(), orgID, saleID, actor, strings.TrimSpace(req.Reason))
	if err != nil {
		if errors.Is(err, errPosSaleNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sale_not_found"})
			return
		}
		if s.log != nil {
			s.log.Warn("pos sale void atomic commit failed", "retailer_id", orgID, "sale_id", saleID, "err", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "void_failed"})
		return
	}
	if !alreadyVoided {
		for _, l := range voidedSale.Lines {
			if err := s.syncReorderCurrentStock(r.Context(), orgID, l.Sku); err != nil && s.log != nil {
				s.log.Warn("pos void reorder stock sync failed", "retailer_id", orgID, "sku", l.Sku, "err", err)
			}
		}
		s.refreshHqStockSnapshotsForSale(r.Context(), orgID, voidedSale.LocationID, voidedSale.StockBin, voidedSale.Lines)
		s.recordSellThroughVoid(r.Context(), orgID, voidedSale.LocationID, voidedSale.Lines)
	}
	sessForVoid, sessFound, sessErr := s.getPosSession(r.Context(), voidedSale.SessionID)
	if sessErr != nil && s.log != nil {
		s.log.Warn("load pos session for void anomaly review failed", "session_id", voidedSale.SessionID, "err", sessErr)
	}
	if !sessFound {
		sessForVoid = PosSessionDTO{SessionID: voidedSale.SessionID, RegisterID: voidedSale.RegisterID, LocationID: voidedSale.LocationID, RetailerID: orgID}
	}
	sessSales, salesErr := s.listSessionSales(r.Context(), voidedSale.SessionID, 500)
	if salesErr != nil && s.log != nil {
		s.log.Warn("list session sales for void anomaly review failed", "session_id", voidedSale.SessionID, "err", salesErr)
	}
	if review, flagged := s.inspectSessionPosAnomaly(r.Context(), orgID, sessForVoid, sessSales); flagged {
		voidedSale.AnomalyFlags = []string{PosAnomalyReviewFlag}
		voidedSale.AnomalyReview = &review
	}
	writeJSON(w, http.StatusOK, voidedSale)
}

var (
	errPosSessionNotOpen = errors.New("session_not_open")
	errPosSaleNotFound   = errors.New("sale_not_found")
)

type posStockConflictError struct {
	sku string
	err error
}

func (e *posStockConflictError) Error() string {
	return e.err.Error()
}

func (e *posStockConflictError) Unwrap() error {
	return e.err
}

func posSaleRowMap(sale PosSaleDTO, linesJSON, tendersJSON []byte) map[string]any {
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
	if sale.ClientSaleID != "" {
		row["ClientSaleId"] = sale.ClientSaleID
	}
	if sale.Origin != "" {
		row["Origin"] = sale.Origin
	}
	if sale.ClientCreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, sale.ClientCreatedAt); err == nil {
			row["ClientCreatedAt"] = t
		} else if t, err := time.Parse(time.RFC3339, sale.ClientCreatedAt); err == nil {
			row["ClientCreatedAt"] = t
		}
	}
	if sale.Status == PosSaleVoided {
		row["VoidedAt"] = spanner.CommitTimestamp
		row["VoidedByUserId"] = nullableStr(sale.VoidedByUserID)
		row["VoidReason"] = nullableStr(sale.VoidReason)
	}
	return row
}

// commitPosSaleAtomic deducts stock for all sale lines, persists RetailerPosSales,
// updates RetailerHqSalesDaily, and emits EventPosSaleCompleted in one transaction.
func (s *Service) commitPosSaleAtomic(ctx context.Context, sale PosSaleDTO, preferredBin, actor string, allowClosedOffline bool) (PosSaleDTO, []string, bool, error) {
	linesJSON, err := json.Marshal(sale.Lines)
	if err != nil {
		return PosSaleDTO{}, nil, false, err
	}
	tendersJSON, err := json.Marshal(sale.Tenders)
	if err != nil {
		return PosSaleDTO{}, nil, false, err
	}

	if s.spannerClient == nil {
		return s.commitPosSaleMemoryAtomic(sale, preferredBin, actor, allowClosedOffline)
	}

	var (
		committedSale PosSaleDTO
		chosenBins    []string
		replayed      bool
	)
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		committedSale = sale
		replayed = false
		chosenBins = make([]string, len(sale.Lines))

		// Re-check offline client_sale_id inside txn so concurrent retries cannot double-deduct stock.
		if sale.ClientSaleID != "" {
			if existing, ok, lookupErr := findPosSaleByClientInTxn(ctx, txn, sale.RetailerID, sale.ClientSaleID); lookupErr == nil && ok {
				committedSale = existing
				replayed = true
				return nil
			}
		}

		// Re-verify session ownership & status inside txn.
		sessRow, sessErr := txn.ReadRow(ctx, "RetailerPosSessions", spanner.Key{sale.SessionID}, []string{"RetailerId", "Status"})
		if sessErr != nil {
			if isNotFound(sessErr) {
				return errPosSessionNotOpen
			}
			return sessErr
		}
		var sessRetailerID, sessStatus string
		if err := sessRow.Columns(&sessRetailerID, &sessStatus); err != nil {
			return err
		}
		if sessRetailerID != sale.RetailerID || (sessStatus != PosSessionOpen && !(allowClosedOffline && sessStatus == PosSessionClosed)) {
			return errPosSessionNotOpen
		}

		type balState struct {
			onHand   int64
			reserved int64
			dirty    bool
		}
		balances := map[stockBalanceKey]balState{}
		balOrder := make([]stockBalanceKey, 0, len(sale.Lines))
		loadBal := func(bin, sku string) (balState, error) {
			k := stockBalanceKey{sale.LocationID, bin, sku}
			if st, ok := balances[k]; ok {
				return st, nil
			}
			row, err := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{sale.LocationID, bin, sku}, []string{"OnHand", "Reserved"})
			if err != nil && !isNotFound(err) {
				return balState{}, err
			}
			var st balState
			if err == nil {
				if err := row.Columns(&st.onHand, &st.reserved); err != nil {
					return balState{}, err
				}
			}
			balances[k] = st
			return st, nil
		}

		var muts []*spanner.Mutation
		binCandidates := []string{preferredBin, BinFloor, BinBackroom}
		for i, l := range sale.Lines {
			seen := map[string]bool{}
			selectedBin := ""
			for _, candidate := range binCandidates {
				b := normalizeBin(candidate)
				if b == "" || seen[b] {
					continue
				}
				seen[b] = true
				st, err := loadBal(b, l.Sku)
				if err != nil {
					return err
				}
				if st.onHand >= l.Qty {
					k := stockBalanceKey{sale.LocationID, b, l.Sku}
					if !st.dirty {
						balOrder = append(balOrder, k)
					}
					st.onHand -= l.Qty
					st.dirty = true
					balances[k] = st
					selectedBin = b
					break
				}
			}
			if selectedBin == "" {
				return &posStockConflictError{sku: l.Sku, err: errors.New("insufficient_stock")}
			}
			chosenBins[i] = selectedBin
			muts = append(muts, spanner.InsertMap("RetailerStockMovements", map[string]any{
				"MovementId":   s.newID(),
				"RetailerId":   sale.RetailerID,
				"LocationId":   sale.LocationID,
				"StockBin":     selectedBin,
				"Sku":          l.Sku,
				"Qty":          -l.Qty,
				"MovementType": MoveSale,
				"RefType":      nullableStr("SALE"),
				"RefId":        nullableStr(sale.SaleID),
				"ActorUserId":  nullableStr(actor),
				"Note":         nullableStr("pos_sale"),
				"CreatedAt":    spanner.CommitTimestamp,
			}))
		}

		for _, k := range balOrder {
			st := balances[k]
			muts = append(muts, spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
				"LocationId": k.LocationID,
				"StockBin":   k.StockBin,
				"Sku":        k.Sku,
				"RetailerId": sale.RetailerID,
				"OnHand":     st.onHand,
				"Reserved":   st.reserved,
				"UpdatedAt":  spanner.CommitTimestamp,
			}))
		}

		muts = append(muts, spanner.InsertOrUpdateMap("RetailerPosSales", posSaleRowMap(sale, linesJSON, tendersJSON)))
		if hqMuts, hqErr := s.hqSalesMutationsInTxn(ctx, txn, sale, false); hqErr == nil {
			muts = append(muts, hqMuts...)
		} else if s.log != nil {
			s.log.Warn("hq sales mutations skipped", "sale_id", sale.SaleID, "err", hqErr)
		}

		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":           events.EventPosSaleCompleted,
			"timestamp":      s.now().Format(time.RFC3339Nano),
			"retailer_id":    sale.RetailerID,
			"sale_id":        sale.SaleID,
			"session_id":     sale.SessionID,
			"total_minor":    sale.TotalMinor,
			"receipt":        sale.ReceiptNumber,
			"location_id":    sale.LocationID,
			"origin":         sale.Origin,
			"client_sale_id": sale.ClientSaleID,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, sale.RetailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err != nil {
		return PosSaleDTO{}, nil, false, err
	}
	return committedSale, chosenBins, replayed, nil
}

func (s *Service) commitPosSaleMemoryAtomic(sale PosSaleDTO, preferredBin, actor string, allowClosedOffline bool) (PosSaleDTO, []string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sale.ClientSaleID != "" && s.posSalesByClient != nil {
		if existingID, ok := s.posSalesByClient[sale.RetailerID+"|"+sale.ClientSaleID]; ok {
			if existing, okSale := s.posSales[existingID]; okSale {
				return existing, nil, true, nil
			}
		}
	}

	if s.posSessions != nil {
		sess, ok := s.posSessions[sale.SessionID]
		if !ok || sess.RetailerID != sale.RetailerID || (sess.Status != PosSessionOpen && !(allowClosedOffline && sess.Status == PosSessionClosed)) {
			return PosSaleDTO{}, nil, false, errPosSessionNotOpen
		}
	}

	// Stage balance decrements before mutating state so multi-line stock failures never leak.
	staged := map[stockBalanceKey]memStockBalance{}
	chosenBins := make([]string, len(sale.Lines))
	type stagedMove struct {
		bin string
		sku string
		qty int64
	}
	moves := make([]stagedMove, 0, len(sale.Lines))
	binCandidates := []string{preferredBin, BinFloor, BinBackroom}

	for i, l := range sale.Lines {
		seen := map[string]bool{}
		selectedBin := ""
		for _, candidate := range binCandidates {
			b := normalizeBin(candidate)
			if b == "" || seen[b] {
				continue
			}
			seen[b] = true
			k := stockBalanceKey{sale.LocationID, b, l.Sku}
			cur, ok := staged[k]
			if !ok && s.stockBalances != nil {
				cur = s.stockBalances[k]
			}
			if cur.OnHand >= l.Qty {
				cur.RetailerID = sale.RetailerID
				cur.OnHand -= l.Qty
				staged[k] = cur
				selectedBin = b
				break
			}
		}
		if selectedBin == "" {
			return PosSaleDTO{}, nil, false, &posStockConflictError{sku: l.Sku, err: errors.New("insufficient_stock")}
		}
		chosenBins[i] = selectedBin
		moves = append(moves, stagedMove{bin: selectedBin, sku: l.Sku, qty: l.Qty})
	}

	if s.stockBalances == nil {
		s.stockBalances = map[stockBalanceKey]memStockBalance{}
	}
	for k, v := range staged {
		s.stockBalances[k] = v
		s.bumpStockLocationVersionLocked(sale.RetailerID, k.LocationID, k.StockBin)
	}
	nowStr := s.now().UTC().Format(time.RFC3339Nano)
	for _, mv := range moves {
		s.stockMovements = append(s.stockMovements, StockMovementDTO{
			MovementID: s.newID(), LocationID: sale.LocationID, StockBin: mv.bin, Sku: mv.sku,
			Qty: -mv.qty, MovementType: MoveSale, RefType: "SALE", RefID: sale.SaleID,
			ActorUserID: actor, Note: "pos_sale", CreatedAt: nowStr,
		})
	}
	if s.posSales == nil {
		s.posSales = map[string]PosSaleDTO{}
	}
	if s.posSalesByClient == nil {
		s.posSalesByClient = map[string]string{}
	}
	s.posSales[sale.SaleID] = sale
	if sale.ClientSaleID != "" {
		s.posSalesByClient[sale.RetailerID+"|"+sale.ClientSaleID] = sale.SaleID
	}
	s.applyHqFromSaleMemoryLocked(sale)
	return sale, chosenBins, false, nil
}

func (s *Service) commitPosVoidAtomic(ctx context.Context, orgID, saleID, actor, reason string) (PosSaleDTO, bool, error) {
	now := s.now().UTC()
	nowStr := now.Format(time.RFC3339Nano)

	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		sale, ok := s.posSales[saleID]
		if !ok || sale.RetailerID != orgID {
			return PosSaleDTO{}, false, errPosSaleNotFound
		}
		if sale.Status == PosSaleVoided {
			return sale, true, nil
		}
		if s.stockBalances == nil {
			s.stockBalances = map[stockBalanceKey]memStockBalance{}
		}
		for _, l := range sale.Lines {
			if l.Qty <= 0 {
				continue
			}
			k := stockBalanceKey{sale.LocationID, sale.StockBin, l.Sku}
			cur := s.stockBalances[k]
			cur.RetailerID = orgID
			cur.OnHand += l.Qty
			s.stockBalances[k] = cur
			s.bumpStockLocationVersionLocked(orgID, sale.LocationID, sale.StockBin)
			s.stockMovements = append(s.stockMovements, StockMovementDTO{
				MovementID: s.newID(), LocationID: sale.LocationID, StockBin: sale.StockBin, Sku: l.Sku,
				Qty: l.Qty, MovementType: MoveSaleVoid, RefType: "SALE", RefID: saleID,
				ActorUserID: actor, Note: "void", CreatedAt: nowStr,
			})
		}
		sale.Status = PosSaleVoided
		sale.VoidedAt = nowStr
		sale.VoidedByUserID = actor
		sale.VoidReason = reason
		s.posSales[saleID] = sale
		s.applyHqFromVoidMemoryLocked(sale)
		return sale, false, nil
	}

	var (
		out           PosSaleDTO
		alreadyVoided bool
	)
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		alreadyVoided = false
		row, err := txn.ReadRow(ctx, "RetailerPosSales", spanner.Key{saleID},
			[]string{"SaleId", "SessionId", "RegisterId", "LocationId", "RetailerId", "CashierUserId",
				"Status", "TotalMinor", "Currency", "ReceiptNumber", "LinesJson", "TendersJson", "StockBin",
				"CreatedAt", "VoidedAt", "VoidedByUserId", "VoidReason"})
		if err != nil {
			if isNotFound(err) {
				return errPosSaleNotFound
			}
			return err
		}
		sale, ok, err := decodePosSaleRow(row)
		if err != nil || !ok || sale.RetailerID != orgID {
			return errPosSaleNotFound
		}
		if sale.Status == PosSaleVoided {
			out = sale
			alreadyVoided = true
			return nil
		}

		type balState struct {
			onHand   int64
			reserved int64
			dirty    bool
		}
		balances := map[stockBalanceKey]balState{}
		balOrder := make([]stockBalanceKey, 0, len(sale.Lines))
		var muts []*spanner.Mutation
		for _, l := range sale.Lines {
			if l.Qty <= 0 {
				continue
			}
			k := stockBalanceKey{sale.LocationID, sale.StockBin, l.Sku}
			st, seen := balances[k]
			if !seen {
				balRow, balErr := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{k.LocationID, k.StockBin, k.Sku}, []string{"OnHand", "Reserved"})
				if balErr != nil && !isNotFound(balErr) {
					return balErr
				}
				if balErr == nil {
					if err := balRow.Columns(&st.onHand, &st.reserved); err != nil {
						return err
					}
				}
				balOrder = append(balOrder, k)
			}
			st.onHand += l.Qty
			st.dirty = true
			balances[k] = st
			muts = append(muts, spanner.InsertMap("RetailerStockMovements", map[string]any{
				"MovementId":   s.newID(),
				"RetailerId":   orgID,
				"LocationId":   sale.LocationID,
				"StockBin":     sale.StockBin,
				"Sku":          l.Sku,
				"Qty":          l.Qty,
				"MovementType": MoveSaleVoid,
				"RefType":      nullableStr("SALE"),
				"RefId":        nullableStr(saleID),
				"ActorUserId":  nullableStr(actor),
				"Note":         nullableStr("void"),
				"CreatedAt":    spanner.CommitTimestamp,
			}))
		}
		for _, k := range balOrder {
			st := balances[k]
			if !st.dirty {
				continue
			}
			muts = append(muts, spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
				"LocationId": k.LocationID,
				"StockBin":   k.StockBin,
				"Sku":        k.Sku,
				"RetailerId": orgID,
				"OnHand":     st.onHand,
				"Reserved":   st.reserved,
				"UpdatedAt":  spanner.CommitTimestamp,
			}))
		}

		sale.Status = PosSaleVoided
		sale.VoidedAt = nowStr
		sale.VoidedByUserID = actor
		sale.VoidReason = reason
		muts = append(muts, spanner.UpdateMap("RetailerPosSales", map[string]any{
			"SaleId":         saleID,
			"Status":         PosSaleVoided,
			"VoidedAt":       spanner.CommitTimestamp,
			"VoidedByUserId": nullableStr(actor),
			"VoidReason":     nullableStr(reason),
		}))

		if hqMuts, hqErr := s.hqSalesMutationsInTxn(ctx, txn, sale, true); hqErr == nil {
			muts = append(muts, hqMuts...)
		} else if s.log != nil {
			s.log.Warn("hq sales void mutations skipped", "sale_id", saleID, "err", hqErr)
		}

		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":        events.EventPosSaleVoided,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": orgID,
			"sale_id":     saleID,
			"reason":      reason,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, orgID, events.TopicMain, payload); err != nil {
			return err
		}
		if err := buf.Flush(txn); err != nil {
			return err
		}
		out = sale
		return nil
	})
	if err != nil {
		return PosSaleDTO{}, false, err
	}
	return out, alreadyVoided, nil
}

func findPosSaleByClientInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, clientSaleID string) (PosSaleDTO, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT SaleId, SessionId, RegisterId, LocationId, RetailerId, CashierUserId,
			Status, TotalMinor, Currency, ReceiptNumber, LinesJson, TendersJson, StockBin,
			CreatedAt, VoidedAt, VoidedByUserId, VoidReason, ClientSaleId, Origin, ClientCreatedAt
			FROM RetailerPosSales@{FORCE_INDEX=UQ_RetailerPosSales_ClientSale}
			WHERE RetailerId = @rid AND ClientSaleId = @cid
			LIMIT 1`,
		Params: map[string]any{"rid": retailerID, "cid": clientSaleID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return PosSaleDTO{}, false, err
	}
	return decodePosSaleRowExtended(row)
}

// HandlePosSaleRefund is alias for void in v1 (same-day full refund).
func (s *Service) HandlePosSaleRefund(w http.ResponseWriter, r *http.Request) {
	s.HandlePosSaleVoid(w, r)
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
	iter := s.spannerClient.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, spanner.Statement{SQL: sql, Params: params})
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
	return s.savePosSaleWithHQ(ctx, sale, false)
}

// savePosSaleWithHQ persists the sale ledger and HQ sales daily deltas in one Apply.
// void=false → sale complete deltas; void=true → void deltas (same txn as ledger).
func (s *Service) savePosSaleWithHQ(ctx context.Context, sale PosSaleDTO, isVoid bool) error {
	linesJSON, err := json.Marshal(sale.Lines)
	if err != nil {
		return err
	}
	tendersJSON, err := json.Marshal(sale.Tenders)
	if err != nil {
		return err
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		if s.posSales == nil {
			s.posSales = map[string]PosSaleDTO{}
		}
		if s.posSalesByClient == nil {
			s.posSalesByClient = map[string]string{}
		}
		s.posSales[sale.SaleID] = sale
		if sale.ClientSaleID != "" {
			s.posSalesByClient[sale.RetailerID+"|"+sale.ClientSaleID] = sale.SaleID
		}
		if isVoid {
			s.applyHqFromVoidMemoryLocked(sale)
		} else {
			s.applyHqFromSaleMemoryLocked(sale)
		}
		s.mu.Unlock()
		return nil
	}
	row := posSaleRowMap(sale, linesJSON, tendersJSON)
	// C2.1: HQ sales daily in the SAME Apply as the sale ledger (not a goroutine).
	muts := []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerPosSales", row)}
	var hqMuts []*spanner.Mutation
	var hqErr error
	if isVoid {
		hqMuts, hqErr = s.buildHqSalesMutationsFromVoid(ctx, sale)
	} else {
		hqMuts, hqErr = s.buildHqSalesMutationsFromSale(ctx, sale)
	}
	if hqErr == nil {
		muts = append(muts, hqMuts...)
	} else if s.log != nil {
		s.log.Warn("hq sales mutations skipped", "sale_id", sale.SaleID, "err", hqErr)
	}
	_, err = s.spannerClient.Apply(ctx, muts)
	return err
}

// getPosSaleByClientID returns a sale previously created with client_sale_id (offline sync idempotency).
func (s *Service) getPosSaleByClientID(ctx context.Context, retailerID, clientSaleID string) (PosSaleDTO, bool) {
	clientSaleID = strings.TrimSpace(clientSaleID)
	if clientSaleID == "" {
		return PosSaleDTO{}, false
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.posSalesByClient == nil {
			return PosSaleDTO{}, false
		}
		saleID, ok := s.posSalesByClient[retailerID+"|"+clientSaleID]
		if !ok {
			return PosSaleDTO{}, false
		}
		sale, ok := s.posSales[saleID]
		return sale, ok
	}
	stmt := spanner.Statement{
		SQL: `SELECT SaleId, SessionId, RegisterId, LocationId, RetailerId, CashierUserId,
			Status, TotalMinor, Currency, ReceiptNumber, LinesJson, TendersJson, StockBin,
			CreatedAt, VoidedAt, VoidedByUserId, VoidReason, ClientSaleId, Origin, ClientCreatedAt
			FROM RetailerPosSales@{FORCE_INDEX=UQ_RetailerPosSales_ClientSale}
			WHERE RetailerId = @rid AND ClientSaleId = @cid
			LIMIT 1`,
		Params: map[string]any{"rid": retailerID, "cid": clientSaleID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		// Index/columns may not exist yet on older DBs — fail open to HTTP idempotency only.
		return PosSaleDTO{}, false
	}
	sale, ok, err := decodePosSaleRowExtended(row)
	if err != nil || !ok {
		return PosSaleDTO{}, false
	}
	return sale, true
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
	if err := json.Unmarshal([]byte(linesJSON), &sale.Lines); err != nil {
		return PosSaleDTO{}, false, fmt.Errorf("decode pos sale lines: %w", err)
	}
	if err := json.Unmarshal([]byte(tendersJSON), &sale.Tenders); err != nil {
		return PosSaleDTO{}, false, fmt.Errorf("decode pos sale tenders: %w", err)
	}
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

// decodePosSaleRowExtended includes offline audit columns (ClientSaleId, Origin, ClientCreatedAt).
func decodePosSaleRowExtended(row *spanner.Row) (PosSaleDTO, bool, error) {
	var sale PosSaleDTO
	var linesJSON, tendersJSON string
	var created time.Time
	var voided spanner.NullTime
	var voidedBy, voidReason, clientSaleID, origin spanner.NullString
	var clientCreated spanner.NullTime
	if err := row.Columns(
		&sale.SaleID, &sale.SessionID, &sale.RegisterID, &sale.LocationID, &sale.RetailerID, &sale.CashierUserID,
		&sale.Status, &sale.TotalMinor, &sale.Currency, &sale.ReceiptNumber, &linesJSON, &tendersJSON, &sale.StockBin,
		&created, &voided, &voidedBy, &voidReason, &clientSaleID, &origin, &clientCreated,
	); err != nil {
		return PosSaleDTO{}, false, err
	}
	if err := json.Unmarshal([]byte(linesJSON), &sale.Lines); err != nil {
		return PosSaleDTO{}, false, fmt.Errorf("decode pos sale lines: %w", err)
	}
	if err := json.Unmarshal([]byte(tendersJSON), &sale.Tenders); err != nil {
		return PosSaleDTO{}, false, fmt.Errorf("decode pos sale tenders: %w", err)
	}
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
	if clientSaleID.Valid {
		sale.ClientSaleID = clientSaleID.StringVal
	}
	if origin.Valid {
		sale.Origin = origin.StringVal
	}
	if clientCreated.Valid {
		sale.ClientCreatedAt = clientCreated.Time.UTC().Format(time.RFC3339Nano)
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
	iter := s.spannerClient.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
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

const retailerPosSalesReportLimit = 2000

// listRetailerPosSales reads POS ledger rows in [from, to) for reports/pulse.
// Spanner is the live SoT; memory only when the client is unset (tests / no-Spanner).
func (s *Service) listRetailerPosSales(ctx context.Context, retailerID, locationID string, from, to time.Time, limit int) ([]PosSaleDTO, error) {
	if limit <= 0 {
		limit = retailerPosSalesReportLimit
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []PosSaleDTO
		for _, sale := range s.posSales {
			if sale.RetailerID != retailerID {
				continue
			}
			if locationID != "" && sale.LocationID != locationID {
				continue
			}
			t, okT := parseReportTime(sale.CreatedAt)
			if !okT || t.Before(from) || !t.Before(to) {
				continue
			}
			out = append(out, sale)
		}
		return out, nil
	}
	sql := `SELECT SaleId, SessionId, RegisterId, LocationId, RetailerId, CashierUserId,
			Status, TotalMinor, Currency, ReceiptNumber, LinesJson, TendersJson, StockBin,
			CreatedAt, VoidedAt, VoidedByUserId, VoidReason
			FROM RetailerPosSales@{FORCE_INDEX=Idx_RetailerPosSales_ByRetailer}
			WHERE RetailerId = @rid AND CreatedAt >= @from AND CreatedAt < @to`
	params := map[string]any{
		"rid":  retailerID,
		"from": from,
		"to":   to,
		"lim":  int64(limit + 1),
	}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`
	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := s.spannerClient.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(ctx, stmt)
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
	if len(out) > limit {
		return nil, fmt.Errorf("pos sales window exceeds report limit")
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
	return s.nextReceiptNumberWithSaleID(retailerID, s.newID())
}

func (s *Service) nextReceiptNumberWithSaleID(_ string, saleID string) string {
	ts := s.now().UTC().Format("20060102150405.000000") // 21 chars
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, saleID)
	if len(clean) > 8 {
		clean = clean[len(clean)-8:]
	}
	if clean == "" {
		return "R" + ts
	}
	return "R" + ts + "-" + clean
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
