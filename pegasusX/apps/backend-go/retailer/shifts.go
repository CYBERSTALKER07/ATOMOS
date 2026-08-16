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

const (
	TimeEntryOpen   = "OPEN"
	TimeEntryClosed = "CLOSED"
	ShiftOpen       = "OPEN"
	ShiftClosed     = "CLOSED"

	// Default POS/shift config when pack is on.
	defaultRequireShiftToOpenRegister       = true
	defaultVarianceAlertMinor         int64 = 10_000 // 100.00 if 2-decimal currency
)

func defaultMaxShiftHours() int {
	h, err := auth.LaborMaxShiftHoursFromContext(context.Background(), "")
	if err != nil || h <= 0 {
		return 0
	}
	return int(h)
}

// TimeEntryDTO is a personal clock-in/out record.
type TimeEntryDTO struct {
	EntryID    string `json:"entry_id"`
	RetailerID string `json:"retailer_id"`
	UserID     string `json:"user_id"`
	LocationID string `json:"location_id"`
	Status     string `json:"status"`
	ClockInAt  string `json:"clock_in_at,omitempty"`
	ClockOutAt string `json:"clock_out_at,omitempty"`
	AutoClosed bool   `json:"auto_closed"`
	Note       string `json:"note,omitempty"`
}

// ShiftDTO is a location/register labor+cash period.
type ShiftDTO struct {
	ShiftID            string `json:"shift_id"`
	RetailerID         string `json:"retailer_id"`
	LocationID         string `json:"location_id"`
	RegisterID         string `json:"register_id,omitempty"`
	OpenedByUserID     string `json:"opened_by_user_id"`
	ClosedByUserID     string `json:"closed_by_user_id,omitempty"`
	Status             string `json:"status"`
	OpeningFloatMinor  int64  `json:"opening_float_minor"`
	ClosingCashMinor   *int64 `json:"closing_cash_minor,omitempty"`
	ExpectedCashMinor  *int64 `json:"expected_cash_minor,omitempty"`
	VarianceMinor      *int64 `json:"variance_minor,omitempty"`
	Currency           string `json:"currency"`
	LinkedPosSessionID string `json:"linked_pos_session_id,omitempty"`
	OpenedAt           string `json:"opened_at,omitempty"`
	ClosedAt           string `json:"closed_at,omitempty"`
}

// HandleClockIn serves POST /v1/retailer/time/clock-in
func (s *Service) HandleClockIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermShiftOpen) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermShiftOpen})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	var req struct {
		LocationID string `json:"location_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		if p, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = p.LocationID
		}
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	// Auto-enable SHIFTS pack.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackSHIFTS) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSHIFTS, userID, true, map[string]any{
			"require_clock_in":               true,
			"require_shift_to_open_register": true,
			"max_shift_hours":                defaultMaxShiftHours(),
			"variance_alert_minor":           defaultVarianceAlertMinor,
		})
	}
	// Reject if already clocked in.
	if open, okOpen, _ := s.getOpenTimeEntry(r.Context(), userID); okOpen {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "already_clocked_in",
			"entry": open,
		})
		return
	}
	entry := TimeEntryDTO{
		EntryID:    s.newID(),
		RetailerID: orgID,
		UserID:     userID,
		LocationID: locID,
		Status:     TimeEntryOpen,
		ClockInAt:  s.now().UTC().Format(time.RFC3339Nano),
		AutoClosed: false,
	}
	if err := s.saveTimeEntry(r.Context(), entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clock_in_failed"})
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

// HandleClockOut serves POST /v1/retailer/time/clock-out
func (s *Service) HandleClockOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	entry, found, err := s.getOpenTimeEntry(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_entry_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "not_clocked_in"})
		return
	}
	entry.Status = TimeEntryClosed
	entry.ClockOutAt = s.now().UTC().Format(time.RFC3339Nano)
	if err := s.saveTimeEntry(r.Context(), entry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clock_out_failed"})
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

// HandleTimeEntries serves GET /v1/retailer/time/entries
func (s *Service) HandleTimeEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	userFilter := strings.TrimSpace(r.URL.Query().Get("user_id"))
	// Non-managers only see self.
	role := auth.EffectiveRetailerRole(claims)
	self := auth.ResolveRetailerUserID(claims)
	if role != "OWNER" && role != "ADMIN" && role != "MANAGER" {
		userFilter = self
	}
	items, err := s.listTimeEntries(r.Context(), orgID, userFilter, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_entries_failed"})
		return
	}
	open, _, _ := s.getOpenTimeEntry(r.Context(), self)
	writeJSON(w, http.StatusOK, map[string]any{
		"items":      items,
		"open_entry": open,
		"clocked_in": open.EntryID != "",
	})
}

// HandleShifts serves GET /v1/retailer/shifts and POST open
func (s *Service) HandleShifts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleShiftsGet(w, r)
	case http.MethodPost:
		s.handleShiftOpen(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleShiftsGet(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermShiftOpen) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	items, err := s.listShifts(r.Context(), orgID, locID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_shifts_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) handleShiftOpen(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermShiftOpen) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermShiftOpen})
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
		LocationID        string `json:"location_id"`
		RegisterID        string `json:"register_id"`
		OpeningFloatMinor int64  `json:"opening_float_minor"`
		Currency          string `json:"currency"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		if p, e := s.EnsurePrimaryLocation(r.Context(), orgID); e == nil {
			locID = p.LocationID
		}
	}
	if err := s.assertLocationInOrg(r.Context(), orgID, locID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "location_not_found"})
		return
	}
	// Require clock-in when config says so.
	cfg := s.shiftsConfig(r.Context(), orgID)
	if cfg.requireClockIn {
		if open, okOpen, _ := s.getOpenTimeEntry(r.Context(), userID); !okOpen {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "clock_in_required"})
			return
		} else if open.LocationID != locID {
			// Allow different location? Fail closed for safety.
			writeJSON(w, http.StatusConflict, map[string]string{"error": "clock_in_location_mismatch"})
			return
		}
	}
	// Auto-enable pack.
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackSHIFTS) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSHIFTS, userID, true, map[string]any{
			"require_clock_in":               true,
			"require_shift_to_open_register": true,
			"max_shift_hours":                defaultMaxShiftHours(),
			"variance_alert_minor":           defaultVarianceAlertMinor,
		})
	}
	// One open shift per register if register set; else per location.
	regID := strings.TrimSpace(req.RegisterID)
	if regID != "" {
		if open, okOpen, _ := s.getOpenShiftForRegister(r.Context(), regID); okOpen {
			respBytes, _ := json.Marshal(open)
			idemCommitted = true
			s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
			writeJSONBytes(w, http.StatusOK, respBytes)
			return
		}
	}
	currency := strings.TrimSpace(req.Currency)
	if currency == "" {
		currency = "UZS"
	}
	if req.OpeningFloatMinor < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "opening_float_invalid"})
		return
	}
	shift := ShiftDTO{
		ShiftID:           s.newID(),
		RetailerID:        orgID,
		LocationID:        locID,
		RegisterID:        regID,
		OpenedByUserID:    userID,
		Status:            ShiftOpen,
		OpeningFloatMinor: req.OpeningFloatMinor,
		Currency:          currency,
		OpenedAt:          s.now().UTC().Format(time.RFC3339Nano),
	}
	if err := s.saveShift(r.Context(), shift); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "open_shift_failed"})
		return
	}
	respBytes, _ := json.Marshal(shift)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleShiftClose serves POST /v1/retailer/shifts/{shiftID}/close
func (s *Service) HandleShiftClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	// Openers (shift.open) may close; managers/owners also have shift.close.
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermShiftClose) && !auth.HasRetailerPerm(claims, auth.PermShiftOpen)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	shiftID := strings.TrimSpace(chi.URLParam(r, "shiftID"))
	shift, found, err := s.getShift(r.Context(), shiftID)
	if err != nil || !found || shift.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "shift_not_found"})
		return
	}
	if shift.Status == ShiftClosed {
		writeJSON(w, http.StatusOK, shift)
		return
	}
	var req struct {
		ClosingCashMinor int64 `json:"closing_cash_minor"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Expected cash: opening float + cash from linked POS session if any.
	var cashSales int64
	if shift.LinkedPosSessionID != "" {
		cashSales, _ = s.sumSessionCashTenders(r.Context(), shift.LinkedPosSessionID)
	} else if shift.RegisterID != "" {
		// Sum cash from all open/closed POS sessions opened during this shift window for register — simplified: use linked only.
		// If POS session currently open on register, use it.
		if pos, okPos, _ := s.getOpenSessionForRegister(r.Context(), shift.RegisterID); okPos {
			cashSales, _ = s.sumSessionCashTenders(r.Context(), pos.SessionID)
			shift.LinkedPosSessionID = pos.SessionID
		}
	}
	expected := shift.OpeningFloatMinor + cashSales
	variance := req.ClosingCashMinor - expected
	now := s.now().UTC().Format(time.RFC3339Nano)
	shift.Status = ShiftClosed
	shift.ClosedByUserID = auth.ResolveRetailerUserID(claims)
	shift.ClosingCashMinor = &req.ClosingCashMinor
	shift.ExpectedCashMinor = &expected
	shift.VarianceMinor = &variance
	shift.ClosedAt = now
	if err := s.saveShift(r.Context(), shift); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "close_shift_failed"})
		return
	}
	// Variance alert
	cfg := s.shiftsConfig(r.Context(), orgID)
	if abs64(variance) >= cfg.varianceAlertMinor {
		s.alertOwnersVariance(r.Context(), orgID, shift, variance)
	}
	writeJSON(w, http.StatusOK, shift)
}

// shiftsConfig holds pack config with defaults.
type shiftsConfig struct {
	requireClockIn             bool
	requireShiftToOpenRegister bool
	maxShiftHours              int
	varianceAlertMinor         int64
}

func (s *Service) shiftsConfig(ctx context.Context, retailerID string) shiftsConfig {
	cfg := shiftsConfig{
		requireClockIn:             true,
		requireShiftToOpenRegister: defaultRequireShiftToOpenRegister,
		maxShiftHours:              defaultMaxShiftHours(),
		varianceAlertMinor:         defaultVarianceAlertMinor,
	}
	enabled, _ := s.LoadEnabledPacks(ctx, retailerID)
	if !enabled.Has(PackSHIFTS) {
		// Pack off: do not require shift for POS
		cfg.requireShiftToOpenRegister = false
		cfg.requireClockIn = false
		return cfg
	}
	raw, err := s.LoadPackConfig(ctx, retailerID, PackSHIFTS)
	if err != nil || raw == nil {
		return cfg
	}
	if v, ok := raw["require_clock_in"].(bool); ok {
		cfg.requireClockIn = v
	}
	if v, ok := raw["require_shift_to_open_register"].(bool); ok {
		cfg.requireShiftToOpenRegister = v
	}
	if v, ok := raw["max_shift_hours"].(float64); ok && v > 0 {
		cfg.maxShiftHours = int(v)
	}
	if v, ok := raw["variance_alert_minor"].(float64); ok && v >= 0 {
		cfg.varianceAlertMinor = int64(v)
	}
	// JSON numbers sometimes int
	if v, ok := raw["max_shift_hours"].(int); ok && v > 0 {
		cfg.maxShiftHours = v
	}
	if v, ok := raw["variance_alert_minor"].(int64); ok {
		cfg.varianceAlertMinor = v
	}
	if v, ok := raw["variance_alert_minor"].(int); ok {
		cfg.varianceAlertMinor = int64(v)
	}
	return cfg
}

// requireClockedInForPOS returns error message if POS open should be blocked.
func (s *Service) requireClockedInForPOS(ctx context.Context, orgID, userID, locationID string) error {
	cfg := s.shiftsConfig(ctx, orgID)
	if !cfg.requireShiftToOpenRegister {
		return nil
	}
	open, ok, err := s.getOpenTimeEntry(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("clock_in_required")
	}
	if locationID != "" && open.LocationID != "" && open.LocationID != locationID {
		return errors.New("clock_in_location_mismatch")
	}
	// Auto-close if max hours exceeded
	if cfg.maxShiftHours > 0 && open.ClockInAt != "" {
		if t, e := time.Parse(time.RFC3339Nano, open.ClockInAt); e == nil {
			if s.now().Sub(t) > time.Duration(cfg.maxShiftHours)*time.Hour {
				open.Status = TimeEntryClosed
				open.ClockOutAt = s.now().UTC().Format(time.RFC3339Nano)
				open.AutoClosed = true
				open.Note = "auto_closed_max_shift_hours"
				_ = s.saveTimeEntry(ctx, open)
				return errors.New("clock_in_expired_reclock_required")
			}
		}
	}
	return nil
}

func (s *Service) alertOwnersVariance(ctx context.Context, orgID string, shift ShiftDTO, variance int64) {
	title := "Cash variance on shift close"
	body := fmt.Sprintf("Shift %s variance %d minor (location %s)", shift.ShiftID, variance, shift.LocationID)
	s.notifyCashVariance(ctx, orgID, events.EventRetailerShiftVariance, title, body, "/shifts", map[string]any{
		"shift_id":       shift.ShiftID,
		"location_id":    shift.LocationID,
		"variance_minor": variance,
	})
}

func (s *Service) alertOwnersPosVariance(ctx context.Context, orgID string, sess PosSessionDTO, variance int64) {
	title := "Cash variance on POS session close"
	body := fmt.Sprintf("POS session %s variance %d minor (location %s)", sess.SessionID, variance, sess.LocationID)
	s.notifyCashVariance(ctx, orgID, events.EventRetailerShiftVariance, title, body, "/pos", map[string]any{
		"session_id":     sess.SessionID,
		"register_id":    sess.RegisterID,
		"location_id":    sess.LocationID,
		"variance_minor": variance,
		"source":         "pos_session",
	})
}

func (s *Service) notifyCashVariance(ctx context.Context, orgID, eventType, title, body, deep string, fields map[string]any) {
	if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
		_ = w.CreateNotification(ctx, orgID, "RETAILER", eventType, title, body, deep)
		if members, err := s.listOrgMembers(ctx, orgID); err == nil {
			for _, m := range members {
				if m.IsOwner && m.IsActive {
					_ = w.CreateNotification(ctx, m.UserID, "RETAILER", eventType, title, body, deep)
				}
			}
		}
	}
	if s.spannerClient != nil {
		_, _ = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := &spannerTxnBuffer{}
			payload := map[string]any{
				"type":        eventType,
				"timestamp":   s.now().Format(time.RFC3339Nano),
				"retailer_id": orgID,
			}
			for k, v := range fields {
				payload[k] = v
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, orgID, events.TopicMain, payload); err != nil {
				return err
			}
			return buf.Flush(txn)
		})
	}
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ---- persistence ----

func (s *Service) saveTimeEntry(ctx context.Context, e TimeEntryDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.timeEntries == nil {
			s.timeEntries = map[string]TimeEntryDTO{}
		}
		s.timeEntries[e.EntryID] = e
		return nil
	}
	row := map[string]any{
		"EntryId":    e.EntryID,
		"RetailerId": e.RetailerID,
		"UserId":     e.UserID,
		"LocationId": e.LocationID,
		"Status":     e.Status,
		"AutoClosed": e.AutoClosed,
		"ClockInAt":  spanner.CommitTimestamp,
	}
	if e.Note != "" {
		row["Note"] = e.Note
	}
	if e.Status == TimeEntryClosed {
		row["ClockOutAt"] = spanner.CommitTimestamp
	}
	// For updates preserve ClockInAt — use InsertOrUpdate carefully.
	// Prefer DML for close.
	if e.Status == TimeEntryClosed && e.ClockInAt != "" {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, err := txn.Update(ctx, spanner.Statement{
				SQL: `UPDATE RetailerTimeEntries SET Status = @st, ClockOutAt = PENDING_COMMIT_TIMESTAMP(),
					AutoClosed = @auto, Note = @note WHERE EntryId = @id`,
				Params: map[string]any{
					"st": e.Status, "auto": e.AutoClosed, "note": nullableStr(e.Note), "id": e.EntryID,
				},
			})
			return err
		})
		return err
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("RetailerTimeEntries", row)})
	return err
}

func (s *Service) getOpenTimeEntry(ctx context.Context, userID string) (TimeEntryDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, e := range s.timeEntries {
			if e.UserID == userID && e.Status == TimeEntryOpen {
				return e, true, nil
			}
		}
		return TimeEntryDTO{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT EntryId, RetailerId, UserId, LocationId, Status, ClockInAt, ClockOutAt, AutoClosed, IFNULL(Note,'')
			FROM RetailerTimeEntries@{FORCE_INDEX=Idx_RetailerTimeEntries_ByUserStatus}
			WHERE UserId = @uid AND Status = @st
			ORDER BY ClockInAt DESC LIMIT 1`,
		Params: map[string]any{"uid": userID, "st": TimeEntryOpen},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return TimeEntryDTO{}, false, nil
	}
	if err != nil {
		return TimeEntryDTO{}, false, err
	}
	return decodeTimeEntryRow(row)
}

func decodeTimeEntryRow(row *spanner.Row) (TimeEntryDTO, bool, error) {
	var e TimeEntryDTO
	var inAt time.Time
	var outAt spanner.NullTime
	var note string
	if err := row.Columns(&e.EntryID, &e.RetailerID, &e.UserID, &e.LocationID, &e.Status, &inAt, &outAt, &e.AutoClosed, &note); err != nil {
		return TimeEntryDTO{}, false, err
	}
	e.ClockInAt = inAt.UTC().Format(time.RFC3339Nano)
	if outAt.Valid {
		e.ClockOutAt = outAt.Time.UTC().Format(time.RFC3339Nano)
	}
	e.Note = note
	return e, true, nil
}

func (s *Service) listTimeEntries(ctx context.Context, retailerID, userID string, limit int) ([]TimeEntryDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []TimeEntryDTO
		for _, e := range s.timeEntries {
			if e.RetailerID != retailerID {
				continue
			}
			if userID != "" && e.UserID != userID {
				continue
			}
			out = append(out, e)
		}
		return out, nil
	}
	sql := `SELECT EntryId, RetailerId, UserId, LocationId, Status, ClockInAt, ClockOutAt, AutoClosed, IFNULL(Note,'')
		FROM RetailerTimeEntries WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID}
	if userID != "" {
		sql += ` AND UserId = @uid`
		params["uid"] = userID
	}
	sql += ` ORDER BY ClockInAt DESC LIMIT @lim`
	params["lim"] = int64(limit)
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []TimeEntryDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		e, _, err := decodeTimeEntryRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Service) saveShift(ctx context.Context, sh ShiftDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.shifts == nil {
			s.shifts = map[string]ShiftDTO{}
		}
		s.shifts[sh.ShiftID] = sh
		return nil
	}
	row := map[string]any{
		"ShiftId":           sh.ShiftID,
		"RetailerId":        sh.RetailerID,
		"LocationId":        sh.LocationID,
		"OpenedByUserId":    sh.OpenedByUserID,
		"Status":            sh.Status,
		"OpeningFloatMinor": sh.OpeningFloatMinor,
		"Currency":          sh.Currency,
		"OpenedAt":          spanner.CommitTimestamp,
	}
	if sh.RegisterID != "" {
		row["RegisterId"] = sh.RegisterID
	}
	if sh.LinkedPosSessionID != "" {
		row["LinkedPosSessionId"] = sh.LinkedPosSessionID
	}
	if sh.Status == ShiftClosed {
		row["ClosedByUserId"] = nullableStr(sh.ClosedByUserID)
		if sh.ClosingCashMinor != nil {
			row["ClosingCashMinor"] = *sh.ClosingCashMinor
		}
		if sh.ExpectedCashMinor != nil {
			row["ExpectedCashMinor"] = *sh.ExpectedCashMinor
		}
		if sh.VarianceMinor != nil {
			row["VarianceMinor"] = *sh.VarianceMinor
		}
		row["ClosedAt"] = spanner.CommitTimestamp
		// Update path
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			_, err := txn.Update(ctx, spanner.Statement{
				SQL: `UPDATE RetailerShifts SET Status = @st, ClosedByUserId = @by,
					ClosingCashMinor = @close, ExpectedCashMinor = @exp, VarianceMinor = @var,
					LinkedPosSessionId = @pos, ClosedAt = PENDING_COMMIT_TIMESTAMP()
					WHERE ShiftId = @id`,
				Params: map[string]any{
					"st": sh.Status, "by": nullableStr(sh.ClosedByUserID),
					"close": nullInt(sh.ClosingCashMinor), "exp": nullInt(sh.ExpectedCashMinor),
					"var": nullInt(sh.VarianceMinor), "pos": nullableStr(sh.LinkedPosSessionID),
					"id": sh.ShiftID,
				},
			})
			return err
		})
		return err
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("RetailerShifts", row)})
	return err
}

func nullInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func (s *Service) getShift(ctx context.Context, shiftID string) (ShiftDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		sh, ok := s.shifts[shiftID]
		return sh, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerShifts", spanner.Key{shiftID},
		[]string{"ShiftId", "RetailerId", "LocationId", "RegisterId", "OpenedByUserId", "ClosedByUserId",
			"Status", "OpeningFloatMinor", "ClosingCashMinor", "ExpectedCashMinor", "VarianceMinor",
			"Currency", "LinkedPosSessionId", "OpenedAt", "ClosedAt"})
	if err != nil {
		if isNotFound(err) {
			return ShiftDTO{}, false, nil
		}
		return ShiftDTO{}, false, err
	}
	return decodeShiftRow(row)
}

func decodeShiftRow(row *spanner.Row) (ShiftDTO, bool, error) {
	var sh ShiftDTO
	var reg, closedBy, linked spanner.NullString
	var closeC, exp, variance spanner.NullInt64
	var opened, closed spanner.NullTime
	if err := row.Columns(
		&sh.ShiftID, &sh.RetailerID, &sh.LocationID, &reg, &sh.OpenedByUserID, &closedBy,
		&sh.Status, &sh.OpeningFloatMinor, &closeC, &exp, &variance,
		&sh.Currency, &linked, &opened, &closed,
	); err != nil {
		return ShiftDTO{}, false, err
	}
	if reg.Valid {
		sh.RegisterID = reg.StringVal
	}
	if closedBy.Valid {
		sh.ClosedByUserID = closedBy.StringVal
	}
	if linked.Valid {
		sh.LinkedPosSessionID = linked.StringVal
	}
	if closeC.Valid {
		v := closeC.Int64
		sh.ClosingCashMinor = &v
	}
	if exp.Valid {
		v := exp.Int64
		sh.ExpectedCashMinor = &v
	}
	if variance.Valid {
		v := variance.Int64
		sh.VarianceMinor = &v
	}
	if opened.Valid {
		sh.OpenedAt = opened.Time.UTC().Format(time.RFC3339Nano)
	}
	if closed.Valid {
		sh.ClosedAt = closed.Time.UTC().Format(time.RFC3339Nano)
	}
	return sh, true, nil
}

func (s *Service) listShifts(ctx context.Context, retailerID, locationID string, limit int) ([]ShiftDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []ShiftDTO
		for _, sh := range s.shifts {
			if sh.RetailerID != retailerID {
				continue
			}
			if locationID != "" && sh.LocationID != locationID {
				continue
			}
			out = append(out, sh)
		}
		return out, nil
	}
	sql := `SELECT ShiftId, RetailerId, LocationId, RegisterId, OpenedByUserId, ClosedByUserId,
		Status, OpeningFloatMinor, ClosingCashMinor, ExpectedCashMinor, VarianceMinor,
		Currency, LinkedPosSessionId, OpenedAt, ClosedAt
		FROM RetailerShifts WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	sql += ` ORDER BY OpenedAt DESC LIMIT @lim`
	params["lim"] = int64(limit)
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []ShiftDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		sh, _, err := decodeShiftRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, nil
}

func (s *Service) getOpenShiftForRegister(ctx context.Context, registerID string) (ShiftDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, sh := range s.shifts {
			if sh.RegisterID == registerID && sh.Status == ShiftOpen {
				return sh, true, nil
			}
		}
		return ShiftDTO{}, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT ShiftId, RetailerId, LocationId, RegisterId, OpenedByUserId, ClosedByUserId,
			Status, OpeningFloatMinor, ClosingCashMinor, ExpectedCashMinor, VarianceMinor,
			Currency, LinkedPosSessionId, OpenedAt, ClosedAt
			FROM RetailerShifts WHERE RegisterId = @rid AND Status = @st
			ORDER BY OpenedAt DESC LIMIT 1`,
		Params: map[string]any{"rid": registerID, "st": ShiftOpen},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ShiftDTO{}, false, nil
	}
	if err != nil {
		return ShiftDTO{}, false, err
	}
	return decodeShiftRow(row)
}

// linkShiftToPosSession attaches POS session id to open shift for the register.
func (s *Service) linkShiftToPosSession(ctx context.Context, registerID, posSessionID string) {
	sh, ok, err := s.getOpenShiftForRegister(ctx, registerID)
	if err != nil || !ok {
		return
	}
	sh.LinkedPosSessionID = posSessionID
	_ = s.saveShift(ctx, sh)
}
