package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"google.golang.org/api/iterator"
)

// Wave C3.1 parked carts (holds).
// Invariants:
//   - Never write stock / OnHand on park, resume, or void.
//   - Resume only when request LocationId matches hold LocationId.
//   - Default TTL 24h.
//   - Flag POS_HOLDS_ENABLED (default off).

const (
	PosHoldHELD     = "HELD"
	PosHoldRESUMED  = "RESUMED"
	PosHoldEXPIRED  = "EXPIRED"
	PosHoldVOIDED   = "VOIDED"
	posHoldTTLHours = 24
)

// PosHoldDTO is a parked cart snapshot.
type PosHoldDTO struct {
	HoldID     string          `json:"hold_id"`
	RetailerID string          `json:"retailer_id"`
	LocationID string          `json:"location_id"`
	RegisterID string          `json:"register_id,omitempty"`
	UserID     string          `json:"user_id"`
	Status     string          `json:"status"`
	Cart       json.RawMessage `json:"cart"`
	Note       string          `json:"note,omitempty"`
	ExpiresAt  string          `json:"expires_at"`
	CreatedAt  string          `json:"created_at,omitempty"`
	UpdatedAt  string          `json:"updated_at,omitempty"`
	ResumedAt  string          `json:"resumed_at,omitempty"`
	VoidedAt   string          `json:"voided_at,omitempty"`
}

func (s *Service) posHoldsEnabled() bool {
	if s != nil && s.posHoldsOverride != nil {
		return *s.posHoldsOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("POS_HOLDS_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (s *Service) requirePosHolds(w http.ResponseWriter) bool {
	if s.posHoldsEnabled() {
		return true
	}
	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "not_found",
		"code":  "POS_HOLDS_DISABLED",
	})
	return false
}

// HandlePosHolds serves GET/POST /v1/retailer/pos/holds
func (s *Service) HandlePosHolds(w http.ResponseWriter, r *http.Request) {
	if !s.requirePosHolds(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handlePosHoldsList(w, r)
	case http.MethodPost:
		s.handlePosHoldsPark(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandlePosHoldResume serves POST /v1/retailer/pos/holds/{holdID}/resume
func (s *Service) HandlePosHoldResume(w http.ResponseWriter, r *http.Request) {
	if !s.requirePosHolds(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.handlePosHoldResume(w, r)
}

// HandlePosHoldVoid serves POST /v1/retailer/pos/holds/{holdID}/void
func (s *Service) HandlePosHoldVoid(w http.ResponseWriter, r *http.Request) {
	if !s.requirePosHolds(w) {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	s.handlePosHoldVoid(w, r)
}

func (s *Service) handlePosHoldsPark(w http.ResponseWriter, r *http.Request) {
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
		LocationID string          `json:"location_id"`
		RegisterID string          `json:"register_id"`
		Cart       json.RawMessage `json:"cart"`
		Note       string          `json:"note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		// Prefer active location claim
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	if locID == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "location_id_required"})
		return
	}
	if len(req.Cart) == 0 || string(req.Cart) == "null" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "cart_required"})
		return
	}
	// Validate cart is JSON object or array (snapshot only — no stock)
	var cartProbe any
	if err := json.Unmarshal(req.Cart, &cartProbe); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "cart_invalid_json"})
		return
	}

	now := s.now().UTC()
	hold := PosHoldDTO{
		HoldID:     s.newID(),
		RetailerID: orgID,
		LocationID: locID,
		RegisterID: strings.TrimSpace(req.RegisterID),
		UserID:     auth.ResolveRetailerUserID(claims),
		Status:     PosHoldHELD,
		Cart:       req.Cart,
		Note:       strings.TrimSpace(req.Note),
		ExpiresAt:  now.Add(posHoldTTLHours * time.Hour).Format(time.RFC3339Nano),
		CreatedAt:  now.Format(time.RFC3339Nano),
		UpdatedAt:  now.Format(time.RFC3339Nano),
	}
	// IMPORTANT: no stock / OnHand mutation on park.
	if err := s.savePosHold(r.Context(), hold); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "park_hold_failed"})
		return
	}
	respBytes, _ := json.Marshal(hold)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	writeJSONBytes(w, http.StatusCreated, respBytes)
}

func (s *Service) handlePosHoldsList(w http.ResponseWriter, r *http.Request) {
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
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}
	// Lazily expire past-due HELD rows in list path (best-effort).
	_ = s.expirePosHolds(r.Context(), orgID, locID)

	items, err := s.listPosHolds(r.Context(), orgID, locID, PosHoldHELD)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_holds_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"retailer_id": orgID,
		"location_id": locID,
		"items":       items,
	})
}

func (s *Service) handlePosHoldResume(w http.ResponseWriter, r *http.Request) {
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
	holdID := strings.TrimSpace(chi.URLParam(r, "holdID"))
	if holdID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hold_id_required"})
		return
	}
	var req struct {
		LocationID string `json:"location_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	reqLoc := strings.TrimSpace(req.LocationID)
	if reqLoc == "" {
		reqLoc = strings.TrimSpace(claims.ActiveLocationID)
	}

	hold, found, err := s.getPosHold(r.Context(), orgID, holdID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_hold_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "hold_not_found"})
		return
	}
	if hold.Status != PosHoldHELD {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "hold_not_held", "status": hold.Status})
		return
	}
	// Expire check
	if exp, err := time.Parse(time.RFC3339Nano, hold.ExpiresAt); err == nil && s.now().UTC().After(exp) {
		hold.Status = PosHoldEXPIRED
		hold.UpdatedAt = s.now().UTC().Format(time.RFC3339Nano)
		_ = s.savePosHold(r.Context(), hold)
		writeJSON(w, http.StatusGone, map[string]string{"error": "hold_expired", "code": "HOLD_EXPIRED"})
		return
	}
	// Same location only (cross-register OK; cross-store forbidden)
	if reqLoc == "" || hold.LocationID != reqLoc {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":             "location_mismatch",
			"code":              "HOLD_LOCATION_MISMATCH",
			"hold_location_id":  hold.LocationID,
			"request_location_id": reqLoc,
		})
		return
	}

	now := s.now().UTC().Format(time.RFC3339Nano)
	hold.Status = PosHoldRESUMED
	hold.ResumedAt = now
	hold.UpdatedAt = now
	// IMPORTANT: no stock mutation on resume — client rehydrates cart only.
	if err := s.savePosHold(r.Context(), hold); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "resume_hold_failed"})
		return
	}
	writeJSON(w, http.StatusOK, hold)
}

func (s *Service) handlePosHoldVoid(w http.ResponseWriter, r *http.Request) {
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
	holdID := strings.TrimSpace(chi.URLParam(r, "holdID"))
	if holdID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hold_id_required"})
		return
	}
	hold, found, err := s.getPosHold(r.Context(), orgID, holdID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_hold_failed"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "hold_not_found"})
		return
	}
	if hold.Status != PosHoldHELD {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "hold_not_held", "status": hold.Status})
		return
	}
	now := s.now().UTC().Format(time.RFC3339Nano)
	hold.Status = PosHoldVOIDED
	hold.VoidedAt = now
	hold.UpdatedAt = now
	// IMPORTANT: no stock mutation on void.
	if err := s.savePosHold(r.Context(), hold); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "void_hold_failed"})
		return
	}
	writeJSON(w, http.StatusOK, hold)
}

// ---- persistence (no stock tables) ----

func (s *Service) savePosHold(ctx context.Context, h PosHoldDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.posHolds == nil {
			s.posHolds = map[string]PosHoldDTO{}
		}
		key := h.RetailerID + "|" + h.HoldID
		s.posHolds[key] = h
		return nil
	}
	// Table missing pre-migration: swallow not-found style errors so POS still works.
	row := map[string]any{
		"HoldId":     h.HoldID,
		"RetailerId": h.RetailerID,
		"LocationId": h.LocationID,
		"UserId":     h.UserID,
		"Status":     h.Status,
		"CartJson":   string(h.Cart),
		"ExpiresAt":  mustParseTime(h.ExpiresAt, s.now().UTC().Add(posHoldTTLHours*time.Hour)),
		"UpdatedAt":  spanner.CommitTimestamp,
	}
	if h.RegisterID != "" {
		row["RegisterId"] = h.RegisterID
	}
	if h.Note != "" {
		row["Note"] = h.Note
	}
	if h.CreatedAt != "" {
		// Insert path uses commit ts; updates keep original via InsertOrUpdate without CreatedAt
		// We use InsertOrUpdate and set CreatedAt only when empty was new — always set CommitTimestamp on first write.
		row["CreatedAt"] = spanner.CommitTimestamp
	} else {
		row["CreatedAt"] = spanner.CommitTimestamp
	}
	if h.ResumedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, h.ResumedAt); err == nil {
			row["ResumedAt"] = t
		}
	}
	if h.VoidedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, h.VoidedAt); err == nil {
			row["VoidedAt"] = t
		}
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerPosHolds", row),
	})
	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerPosHolds")) {
		return nil
	}
	return err
}

func (s *Service) getPosHold(ctx context.Context, retailerID, holdID string) (PosHoldDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		h, ok := s.posHolds[retailerID+"|"+holdID]
		return h, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerPosHolds", spanner.Key{retailerID, holdID},
		[]string{"HoldId", "RetailerId", "LocationId", "RegisterId", "UserId", "Status", "CartJson", "Note", "ExpiresAt", "CreatedAt", "UpdatedAt", "ResumedAt", "VoidedAt"})
	if err != nil {
		if isNotFound(err) || strings.Contains(err.Error(), "RetailerPosHolds") {
			return PosHoldDTO{}, false, nil
		}
		return PosHoldDTO{}, false, err
	}
	h, err := decodePosHoldRow(row)
	if err != nil {
		return PosHoldDTO{}, false, err
	}
	return h, true, nil
}

func (s *Service) listPosHolds(ctx context.Context, retailerID, locationID, status string) ([]PosHoldDTO, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []PosHoldDTO
		for _, h := range s.posHolds {
			if h.RetailerID != retailerID {
				continue
			}
			if locationID != "" && h.LocationID != locationID {
				continue
			}
			if status != "" && h.Status != status {
				continue
			}
			out = append(out, h)
		}
		return out, nil
	}
	sql := `SELECT HoldId, RetailerId, LocationId, IFNULL(RegisterId, ''), UserId, Status, CartJson,
		IFNULL(Note, ''), ExpiresAt, CreatedAt, UpdatedAt, ResumedAt, VoidedAt
		FROM RetailerPosHolds WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	if status != "" {
		sql += ` AND Status = @st`
		params["st"] = status
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT 100`
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []PosHoldDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerPosHolds") {
				return nil, nil
			}
			return nil, err
		}
		h, err := decodePosHoldRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

func (s *Service) expirePosHolds(ctx context.Context, retailerID, locationID string) error {
	items, err := s.listPosHolds(ctx, retailerID, locationID, PosHoldHELD)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	for _, h := range items {
		exp, err := time.Parse(time.RFC3339Nano, h.ExpiresAt)
		if err != nil {
			continue
		}
		if now.After(exp) {
			h.Status = PosHoldEXPIRED
			h.UpdatedAt = now.Format(time.RFC3339Nano)
			_ = s.savePosHold(ctx, h)
		}
	}
	return nil
}

func decodePosHoldRow(row *spanner.Row) (PosHoldDTO, error) {
	var h PosHoldDTO
	var reg, note, cart string
	var exp, created, updated time.Time
	var resumed, voided spanner.NullTime
	// Flexible: RegisterId/Note may be string from IFNULL query or NullString from ReadRow.
	var regNS, noteNS spanner.NullString
	// Prefer NullString decode path used by ReadRow
	if err := row.Columns(
		&h.HoldID, &h.RetailerID, &h.LocationID, &regNS, &h.UserID, &h.Status, &cart,
		&noteNS, &exp, &created, &updated, &resumed, &voided,
	); err != nil {
		// Fallback IFNULL string columns
		if err2 := row.Columns(
			&h.HoldID, &h.RetailerID, &h.LocationID, &reg, &h.UserID, &h.Status, &cart,
			&note, &exp, &created, &updated, &resumed, &voided,
		); err2 != nil {
			return PosHoldDTO{}, err
		}
	} else {
		if regNS.Valid {
			reg = regNS.StringVal
		}
		if noteNS.Valid {
			note = noteNS.StringVal
		}
	}
	h.RegisterID = reg
	h.Note = note
	h.Cart = json.RawMessage(cart)
	h.ExpiresAt = exp.UTC().Format(time.RFC3339Nano)
	h.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	h.UpdatedAt = updated.UTC().Format(time.RFC3339Nano)
	if resumed.Valid {
		h.ResumedAt = resumed.Time.UTC().Format(time.RFC3339Nano)
	}
	if voided.Valid {
		h.VoidedAt = voided.Time.UTC().Format(time.RFC3339Nano)
	}
	return h, nil
}

func mustParseTime(s string, fallback time.Time) time.Time {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return fallback
}

// SweepExpiredPosHolds marks HELD rows past ExpiresAt as EXPIRED.
// Idempotent; never touches stock. Returns number of holds expired.
// C3.2 worker + ops helper.
func (s *Service) SweepExpiredPosHolds(ctx context.Context) (int, error) {
	if s == nil || !s.posHoldsEnabled() {
		return 0, nil
	}
	now := s.now().UTC()
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		n := 0
		for k, h := range s.posHolds {
			if h.Status != PosHoldHELD {
				continue
			}
			exp, err := time.Parse(time.RFC3339Nano, h.ExpiresAt)
			if err != nil {
				continue
			}
			if now.After(exp) {
				h.Status = PosHoldEXPIRED
				h.UpdatedAt = now.Format(time.RFC3339Nano)
				s.posHolds[k] = h
				n++
			}
		}
		return n, nil
	}

	// Index: Idx_RetailerPosHolds_ByExpires (Status, ExpiresAt)
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT HoldId, RetailerId, LocationId, IFNULL(RegisterId, ''), UserId, Status, CartJson,
			IFNULL(Note, ''), ExpiresAt, CreatedAt, UpdatedAt, ResumedAt, VoidedAt
			FROM RetailerPosHolds@{FORCE_INDEX=Idx_RetailerPosHolds_ByExpires}
			WHERE Status = @st AND ExpiresAt < @now
			LIMIT 500`,
		Params: map[string]any{"st": PosHoldHELD, "now": now},
	})
	defer iter.Stop()
	n := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerPosHolds") {
				return n, nil
			}
			return n, err
		}
		h, err := decodePosHoldRow(row)
		if err != nil {
			return n, err
		}
		h.Status = PosHoldEXPIRED
		h.UpdatedAt = now.Format(time.RFC3339Nano)
		if err := s.savePosHold(ctx, h); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// RunPosHoldsSweeper periodically expires past-due HELD carts.
// Interval default 15m. No-op when POS_HOLDS_ENABLED is off.
func (s *Service) RunPosHoldsSweeper(ctx context.Context, interval time.Duration) {
	if s == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	// One pass at start when enabled.
	if s.posHoldsEnabled() {
		if n, err := s.SweepExpiredPosHolds(ctx); err != nil {
			if s.log != nil {
				s.log.Warn("pos holds sweeper initial pass failed", "err", err)
			}
		} else if s.log != nil && n > 0 {
			s.log.Info("pos holds sweeper expired", "count", n)
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.posHoldsEnabled() {
				continue
			}
			n, err := s.SweepExpiredPosHolds(ctx)
			if err != nil {
				if s.log != nil {
					s.log.Warn("pos holds sweeper failed", "err", err)
				}
				continue
			}
			if s.log != nil && n > 0 {
				s.log.Info("pos holds sweeper expired", "count", n)
			}
		}
	}
}
