package retailer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

const (
	AssistOpen      = "OPEN"
	AssistClaimed   = "CLAIMED"
	AssistDone      = "DONE"
	AssistCancelled = "CANCELLED"
	defaultAssistSLAMinutes = 15
)

// AssistTicketDTO is a floor help ticket.
type AssistTicketDTO struct {
	TicketID          string `json:"ticket_id"`
	RetailerID        string `json:"retailer_id"`
	LocationID        string `json:"location_id"`
	SectionID         string `json:"section_id"`
	Note              string `json:"note"`
	Status            string `json:"status"`
	CreatedByUserID   string `json:"created_by_user_id"`
	ClaimedByUserID   string `json:"claimed_by_user_id,omitempty"`
	CompletedByUserID string `json:"completed_by_user_id,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	ClaimedAt         string `json:"claimed_at,omitempty"`
	CompletedAt       string `json:"completed_at,omitempty"`
	SlaDueAt          string `json:"sla_due_at,omitempty"`
}

// HandleAssistTickets serves GET/POST /v1/retailer/assist/tickets
func (s *Service) HandleAssistTickets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAssistList(w, r)
	case http.MethodPost:
		s.handleAssistCreate(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleAssistList(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || (!auth.HasRetailerPerm(claims, auth.PermAssistRespond) && !auth.HasRetailerPerm(claims, auth.PermStockView)) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	locID := strings.TrimSpace(r.URL.Query().Get("location_id"))
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	items, err := s.listAssistTickets(r.Context(), orgID, locID, status, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_tickets_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) handleAssistCreate(w http.ResponseWriter, r *http.Request) {
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
		SectionID  string `json:"section_id"`
		LocationID string `json:"location_id"`
		Note       string `json:"note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	sectionID := strings.TrimSpace(req.SectionID)
	note := strings.TrimSpace(req.Note)
	if sectionID == "" || note == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "section_id_and_note_required"})
		return
	}
	sec, found, err := s.getSection(r.Context(), sectionID)
	if err != nil || !found || sec.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "section_not_found"})
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	// CUSTOMER_ASSIST hard-deps SECTIONS + TEAM
	enabled, _ := s.LoadEnabledPacks(r.Context(), orgID)
	if !enabled.Has(PackTEAM) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackTEAM, userID, true, map[string]any{})
	}
	if !enabled.Has(PackSTORESTOCK) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSTORESTOCK, userID, true, map[string]any{})
	}
	if !enabled.Has(PackSECTIONS) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackSECTIONS, userID, true, map[string]any{})
	}
	if !enabled.Has(PackCUSTOMERASSIST) {
		_ = s.SetPackEnabled(r.Context(), orgID, PackCUSTOMERASSIST, userID, true, map[string]any{
			"sla_minutes": defaultAssistSLAMinutes,
		})
	}
	slaMin := s.assistSLAMinutes(r.Context(), orgID)
	now := s.now().UTC()
	ticket := AssistTicketDTO{
		TicketID:        s.newID(),
		RetailerID:      orgID,
		LocationID:      sec.LocationID,
		SectionID:       sectionID,
		Note:            note,
		Status:          AssistOpen,
		CreatedByUserID: userID,
		CreatedAt:       now.Format(time.RFC3339Nano),
		SlaDueAt:        now.Add(time.Duration(slaMin) * time.Minute).Format(time.RFC3339Nano),
	}
	if err := s.saveAssistTicket(r.Context(), ticket); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_failed"})
		return
	}
	s.notifyAssistStaff(r.Context(), orgID, ticket, "New floor assist ticket")
	_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerAssistTicketOpened, map[string]any{
		"ticket_id":   ticket.TicketID,
		"section_id":  sectionID,
		"location_id": ticket.LocationID,
	})
	respBytes, _ := json.Marshal(ticket)
	idemCommitted = true
	s.saveIdempotency(r.Context(), r, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleAssistClaim serves POST /v1/retailer/assist/tickets/{ticketID}/claim
func (s *Service) HandleAssistClaim(w http.ResponseWriter, r *http.Request) {
	s.transitionAssist(w, r, AssistClaimed)
}

// HandleAssistComplete serves POST /v1/retailer/assist/tickets/{ticketID}/complete
func (s *Service) HandleAssistComplete(w http.ResponseWriter, r *http.Request) {
	s.transitionAssist(w, r, AssistDone)
}

// HandleAssistCancel serves POST /v1/retailer/assist/tickets/{ticketID}/cancel
func (s *Service) HandleAssistCancel(w http.ResponseWriter, r *http.Request) {
	s.transitionAssist(w, r, AssistCancelled)
}

func (s *Service) transitionAssist(w http.ResponseWriter, r *http.Request, toStatus string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	// claim/complete need assist.respond; cancel also allowed with stock.view (creator/manager path)
	canRespond := auth.HasRetailerPerm(claims, auth.PermAssistRespond)
	canCancel := canRespond || auth.HasRetailerPerm(claims, auth.PermStockView)
	if toStatus == AssistCancelled {
		if !canCancel {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	} else if !canRespond {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermAssistRespond})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	ticketID := strings.TrimSpace(chi.URLParam(r, "ticketID"))
	ticket, found, err := s.getAssistTicket(r.Context(), ticketID)
	if err != nil || !found || ticket.RetailerID != orgID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticket_not_found"})
		return
	}
	userID := auth.ResolveRetailerUserID(claims)
	now := s.now().UTC().Format(time.RFC3339Nano)
	switch toStatus {
	case AssistClaimed:
		if ticket.Status != AssistOpen {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "not_open"})
			return
		}
		ticket.Status = AssistClaimed
		ticket.ClaimedByUserID = userID
		ticket.ClaimedAt = now
		_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerAssistTicketClaimed, map[string]any{"ticket_id": ticketID})
	case AssistDone:
		if ticket.Status != AssistOpen && ticket.Status != AssistClaimed {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "already_closed"})
			return
		}
		ticket.Status = AssistDone
		ticket.CompletedByUserID = userID
		ticket.CompletedAt = now
		if ticket.ClaimedByUserID == "" {
			ticket.ClaimedByUserID = userID
			ticket.ClaimedAt = now
		}
		_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerAssistTicketCompleted, map[string]any{"ticket_id": ticketID})
	case AssistCancelled:
		if ticket.Status == AssistDone || ticket.Status == AssistCancelled {
			writeJSON(w, http.StatusOK, ticket)
			return
		}
		ticket.Status = AssistCancelled
		ticket.CompletedByUserID = userID
		ticket.CompletedAt = now
		_ = s.emitPosEvent(r.Context(), orgID, events.EventRetailerAssistTicketCancelled, map[string]any{"ticket_id": ticketID})
	}
	if err := s.saveAssistTicket(r.Context(), ticket); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update_failed"})
		return
	}
	writeJSON(w, http.StatusOK, ticket)
}

func (s *Service) assistSLAMinutes(ctx context.Context, retailerID string) int {
	raw, err := s.LoadPackConfig(ctx, retailerID, PackCUSTOMERASSIST)
	if err != nil || raw == nil {
		return defaultAssistSLAMinutes
	}
	if v, ok := raw["sla_minutes"].(float64); ok && v > 0 {
		return int(v)
	}
	if v, ok := raw["sla_minutes"].(int); ok && v > 0 {
		return v
	}
	return defaultAssistSLAMinutes
}

func (s *Service) notifyAssistStaff(ctx context.Context, orgID string, ticket AssistTicketDTO, title string) {
	body := fmt.Sprintf("Section %s: %s", ticket.SectionID, ticket.Note)
	deep := "/assist"
	if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
		_ = w.CreateNotification(ctx, orgID, "RETAILER", events.EventRetailerAssistTicketOpened, title, body, deep)
		staff, _ := s.listSectionStaff(ctx, ticket.SectionID)
		for _, uid := range staff {
			_ = w.CreateNotification(ctx, uid, "RETAILER", events.EventRetailerAssistTicketOpened, title, body, deep)
		}
		// Owners/managers as fallback
		if members, err := s.listOrgMembers(ctx, orgID); err == nil {
			for _, m := range members {
				if m.IsActive && (m.IsOwner || m.RetailerRole == "MANAGER" || m.RetailerRole == "SECTION_LEAD") {
					_ = w.CreateNotification(ctx, m.UserID, "RETAILER", events.EventRetailerAssistTicketOpened, title, body, deep)
				}
			}
		}
	}
}

func (s *Service) saveAssistTicket(ctx context.Context, t AssistTicketDTO) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.assistTickets == nil {
			s.assistTickets = map[string]AssistTicketDTO{}
		}
		s.assistTickets[t.TicketID] = t
		return nil
	}
	row := map[string]any{
		"TicketId":        t.TicketID,
		"RetailerId":      t.RetailerID,
		"LocationId":      t.LocationID,
		"SectionId":       t.SectionID,
		"Note":            t.Note,
		"Status":          t.Status,
		"CreatedByUserId": t.CreatedByUserID,
		"CreatedAt":       spanner.CommitTimestamp,
	}
	if t.ClaimedByUserID != "" {
		row["ClaimedByUserId"] = t.ClaimedByUserID
	}
	if t.CompletedByUserID != "" {
		row["CompletedByUserId"] = t.CompletedByUserID
	}
	if t.SlaDueAt != "" {
		if ts, err := time.Parse(time.RFC3339Nano, t.SlaDueAt); err == nil {
			row["SlaDueAt"] = ts
		}
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerAssistanceTickets", row)})
	return err
}

func (s *Service) getAssistTicket(ctx context.Context, ticketID string) (AssistTicketDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		t, ok := s.assistTickets[ticketID]
		return t, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerAssistanceTickets", spanner.Key{ticketID},
		[]string{"TicketId", "RetailerId", "LocationId", "SectionId", "Note", "Status",
			"CreatedByUserId", "ClaimedByUserId", "CompletedByUserId", "CreatedAt", "ClaimedAt", "CompletedAt", "SlaDueAt"})
	if err != nil {
		if isNotFound(err) {
			return AssistTicketDTO{}, false, nil
		}
		return AssistTicketDTO{}, false, err
	}
	return decodeAssistRow(row)
}

func decodeAssistRow(row *spanner.Row) (AssistTicketDTO, bool, error) {
	var t AssistTicketDTO
	var claimedBy, completedBy spanner.NullString
	var created time.Time
	var claimedAt, completedAt, slaDue spanner.NullTime
	if err := row.Columns(&t.TicketID, &t.RetailerID, &t.LocationID, &t.SectionID, &t.Note, &t.Status,
		&t.CreatedByUserID, &claimedBy, &completedBy, &created, &claimedAt, &completedAt, &slaDue); err != nil {
		return AssistTicketDTO{}, false, err
	}
	if claimedBy.Valid {
		t.ClaimedByUserID = claimedBy.StringVal
	}
	if completedBy.Valid {
		t.CompletedByUserID = completedBy.StringVal
	}
	t.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	if claimedAt.Valid {
		t.ClaimedAt = claimedAt.Time.UTC().Format(time.RFC3339Nano)
	}
	if completedAt.Valid {
		t.CompletedAt = completedAt.Time.UTC().Format(time.RFC3339Nano)
	}
	if slaDue.Valid {
		t.SlaDueAt = slaDue.Time.UTC().Format(time.RFC3339Nano)
	}
	return t, true, nil
}

func (s *Service) listAssistTickets(ctx context.Context, retailerID, locationID, status string, limit int) ([]AssistTicketDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []AssistTicketDTO
		for _, t := range s.assistTickets {
			if t.RetailerID != retailerID {
				continue
			}
			if locationID != "" && t.LocationID != locationID {
				continue
			}
			if status != "" && t.Status != status {
				continue
			}
			out = append(out, t)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
		if len(out) > limit {
			out = out[:limit]
		}
		return out, nil
	}
	sql := `SELECT TicketId, RetailerId, LocationId, SectionId, Note, Status,
		CreatedByUserId, ClaimedByUserId, CompletedByUserId, CreatedAt, ClaimedAt, CompletedAt, SlaDueAt
		FROM RetailerAssistanceTickets WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	if status != "" {
		sql += ` AND Status = @st`
		params["st"] = status
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`
	params["lim"] = int64(limit)
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []AssistTicketDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, _, err := decodeAssistRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}
