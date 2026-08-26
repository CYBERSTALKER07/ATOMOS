package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// CashBagSummaryResponse is the shift-level cash summary.
type CashBagSummaryResponse struct {
	DriverID             string              `json:"driver_id"`
	ShiftDate            string              `json:"shift_date"`
	ExpectedCashMinor    int64               `json:"expected_cash_minor"`
	CollectedCashMinor   int64               `json:"collected_cash_minor"`
	DeclaredCashMinor    int64               `json:"declared_cash_minor"`
	DifferenceMinor      int64               `json:"difference_minor"`
	ReconciliationID     string              `json:"reconciliation_id,omitempty"`
	ReconciliationStatus string              `json:"reconciliation_status"`
	DriverNote           string              `json:"driver_note,omitempty"`
	FinanceNote          string              `json:"finance_note,omitempty"`
	PendingOrders        []PendingCollection `json:"pending_orders"`
}

// CashBagTurnInRequest is submitted by the driver upon returning to the warehouse.
type CashBagTurnInRequest struct {
	DeclaredCashMinor int64  `json:"declared_cash_minor"`
	DriverNote        string `json:"driver_note,omitempty"`
	RouteID           string `json:"route_id,omitempty"`
}

// CashReconciliation is the database model.
type CashReconciliation struct {
	ReconciliationID  string    `json:"reconciliation_id"`
	DriverID          string    `json:"driver_id"`
	RouteID           string    `json:"route_id,omitempty"`
	ShiftDate         string    `json:"shift_date"`
	ExpectedCashMinor int64     `json:"expected_cash_minor"`
	DeclaredCashMinor int64     `json:"declared_cash_minor"`
	DifferenceMinor   int64     `json:"difference_minor"`
	Status            string    `json:"status"` // PENDING, ACCEPTED, DISPUTED
	DriverNote        string    `json:"driver_note,omitempty"`
	FinanceNote       string    `json:"finance_note,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	ResolvedAt        *time.Time`json:"resolved_at,omitempty"`
	ResolvedBy        string    `json:"resolved_by,omitempty"`
}

// GetCashBagSummary loads shift cash status.
func (s *Service) GetCashBagSummary(ctx context.Context, driverID string) (*CashBagSummaryResponse, error) {
	now := s.now().UTC()
	shiftDate := civil.DateOf(now)
	todayStr := shiftDate.String()

	var pending []PendingCollection
	if s.pendingQuery != nil {
		pending = s.pendingQuery(driverID)
	}

	var expected int64
	for _, p := range pending {
		expected += p.Amount
	}

	resp := &CashBagSummaryResponse{
		DriverID:             driverID,
		ShiftDate:            todayStr,
		ExpectedCashMinor:    expected,
		CollectedCashMinor:   expected,
		ReconciliationStatus: "PENDING_TURN_IN",
		PendingOrders:        pending,
	}

	if s.spanner == nil {
		return resp, nil
	}

	// Check if there is an existing reconciliation row for today
	iter := s.spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ReconciliationId, RouteId, ShiftDate, ExpectedCashMinor, DeclaredCashMinor,
		             DifferenceMinor, Status, DriverNote, FinanceNote
		      FROM CashReconciliations
		      WHERE DriverId = @did AND ShiftDate = @sdate
		      ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{
			"did":   driverID,
			"sdate": shiftDate,
		},
	})
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return resp, nil
	}
	if err != nil {
		return nil, err
	}

	var reconID, status string
	var routeID, dNote, fNote spanner.NullString
	var sDate civil.Date
	var expMinor, decMinor, diffMinor int64
	if err := row.Columns(&reconID, &routeID, &sDate, &expMinor, &decMinor, &diffMinor, &status, &dNote, &fNote); err != nil {
		return nil, err
	}

	resp.ReconciliationID = reconID
	resp.DeclaredCashMinor = decMinor
	resp.DifferenceMinor = diffMinor
	resp.ReconciliationStatus = status
	if dNote.Valid {
		resp.DriverNote = dNote.StringVal
	}
	if fNote.Valid {
		resp.FinanceNote = fNote.StringVal
	}

	return resp, nil
}

// TurnInCashBag creates or updates the driver cash reconciliation entry for today.
func (s *Service) TurnInCashBag(ctx context.Context, driverID string, req CashBagTurnInRequest) (*CashReconciliation, error) {
	if s.spanner == nil {
		return nil, fmt.Errorf("spanner required")
	}
	driverID = strings.TrimSpace(driverID)
	if driverID == "" {
		return nil, fmt.Errorf("driver_id required")
	}

	now := s.now().UTC()
	shiftDate := civil.DateOf(now)

	var pending []PendingCollection
	if s.pendingQuery != nil {
		pending = s.pendingQuery(driverID)
	}
	var expected int64
	for _, p := range pending {
		expected += p.Amount
	}

	diff := req.DeclaredCashMinor - expected
	status := "SUBMITTED"
	if diff == 0 {
		status = "BALANCED"
	}

	reconID := uuid.NewString()
	cols := map[string]any{
		"ReconciliationId":  reconID,
		"DriverId":          driverID,
		"ShiftDate":         shiftDate,
		"ExpectedCashMinor": expected,
		"DeclaredCashMinor": req.DeclaredCashMinor,
		"DifferenceMinor":   diff,
		"Status":            status,
		"CreatedAt":         spanner.CommitTimestamp,
	}
	if req.RouteID != "" {
		cols["RouteId"] = req.RouteID
	}
	if req.DriverNote != "" {
		cols["DriverNote"] = req.DriverNote
	}

	err := spannerutils.RunReadWriteTransaction(ctx, s.spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("CashReconciliations", cols)}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		// If there is a shortfall or overage, emit event
		if diff < 0 {
			_ = outbox.EmitJSON(ctx, buf, events.AggregateDriver, driverID, events.TopicExceptions, map[string]any{
				"type":            events.EventCashShortfall,
				"driver_id":       driverID,
				"shortfall_minor": -diff,
				"declared_minor":  req.DeclaredCashMinor,
				"expected_minor":  expected,
				"shift_date":      shiftDate.String(),
			})
		} else if diff > 0 {
			_ = outbox.EmitJSON(ctx, buf, events.AggregateDriver, driverID, events.TopicExceptions, map[string]any{
				"type":           events.EventCashOverage,
				"driver_id":      driverID,
				"overage_minor":  diff,
				"declared_minor": req.DeclaredCashMinor,
				"expected_minor": expected,
				"shift_date":     shiftDate.String(),
			})
		}
		return buf.Flush(ctx)
	})
	if err != nil {
		return nil, err
	}

	return &CashReconciliation{
		ReconciliationID:  reconID,
		DriverID:          driverID,
		RouteID:           req.RouteID,
		ShiftDate:         shiftDate.String(),
		ExpectedCashMinor: expected,
		DeclaredCashMinor: req.DeclaredCashMinor,
		DifferenceMinor:   diff,
		Status:            status,
		DriverNote:        req.DriverNote,
		CreatedAt:         now,
	}, nil
}

// ReconcileCashBag allows finance/cashier to accept or dispute a turn-in.
func (s *Service) ReconcileCashBag(ctx context.Context, reconID, actor, action, financeNote string) (*CashReconciliation, error) {
	if s.spanner == nil {
		return nil, fmt.Errorf("spanner required")
	}
	reconID = strings.TrimSpace(reconID)
	if reconID == "" {
		return nil, fmt.Errorf("reconciliation_id required")
	}

	newStatus := "ACCEPTED"
	if strings.EqualFold(action, "DISPUTE") {
		newStatus = "DISPUTED"
	}

	now := s.now().UTC()
	var recon CashReconciliation

	err := spannerutils.RunReadWriteTransaction(ctx, s.spanner, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "CashReconciliations", spanner.Key{reconID},
			[]string{"ReconciliationId", "DriverId", "RouteId", "ShiftDate", "ExpectedCashMinor", "DeclaredCashMinor", "DifferenceMinor", "Status", "DriverNote"})
		if err != nil {
			return err
		}
		var rID, dID, st string
		var rtID, dNote spanner.NullString
		var sDate civil.Date
		var exp, dec, diff int64
		if err := row.Columns(&rID, &dID, &rtID, &sDate, &exp, &dec, &diff, &st, &dNote); err != nil {
			return err
		}

		recon = CashReconciliation{
			ReconciliationID:  rID,
			DriverID:          dID,
			ShiftDate:         sDate.String(),
			ExpectedCashMinor: exp,
			DeclaredCashMinor: dec,
			DifferenceMinor:   diff,
			Status:            newStatus,
			ResolvedBy:        actor,
		}
		if rtID.Valid {
			recon.RouteID = rtID.StringVal
		}
		if dNote.Valid {
			recon.DriverNote = dNote.StringVal
		}
		recon.FinanceNote = financeNote
		resolvedTime := now
		recon.ResolvedAt = &resolvedTime

		updates := map[string]any{
			"ReconciliationId": reconID,
			"Status":           newStatus,
			"ResolvedAt":       spanner.CommitTimestamp,
			"ResolvedBy":       actor,
		}
		if financeNote != "" {
			updates["FinanceNote"] = financeNote
		}

		return txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("CashReconciliations", updates)})
	})
	if err != nil {
		return nil, err
	}

	return &recon, nil
}

// HandleCashBagSummary serves GET /v1/fleet/driver/cash-bag/summary.
func (s *Service) HandleCashBagSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	summary, err := s.GetCashBagSummary(r.Context(), driverID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// HandleCashBagTurnIn serves POST /v1/fleet/driver/cash-bag/turn-in.
func (s *Service) HandleCashBagTurnIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	driverID := driverIDFromRequest(r)
	if driverID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req CashBagTurnInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	res, err := s.TurnInCashBag(r.Context(), driverID, req)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleCashReconciliationAccept serves POST /v1/warehouse/ops/cash-reconciliations/{id}/accept.
func (s *Service) HandleCashReconciliationAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	reconID := chi.URLParam(r, "id")
	if reconID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reconciliation_id_required"})
		return
	}
	actor := auth.ActorFromContext(r.Context())

	var body struct {
		FinanceNote string `json:"finance_note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	res, err := s.ReconcileCashBag(r.Context(), reconID, actor, "ACCEPT", body.FinanceNote)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleCashReconciliationDispute serves POST /v1/warehouse/ops/cash-reconciliations/{id}/dispute.
func (s *Service) HandleCashReconciliationDispute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	reconID := chi.URLParam(r, "id")
	if reconID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reconciliation_id_required"})
		return
	}
	actor := auth.ActorFromContext(r.Context())

	var body struct {
		FinanceNote string `json:"finance_note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	res, err := s.ReconcileCashBag(r.Context(), reconID, actor, "DISPUTE", body.FinanceNote)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// HandleListCashReconciliations serves GET /v1/warehouse/ops/cash-reconciliations.
func (s *Service) HandleListCashReconciliations(w http.ResponseWriter, r *http.Request) {
	if s.spanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"reconciliations": []CashReconciliation{}})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	sql := `SELECT ReconciliationId, DriverId, RouteId, ShiftDate, ExpectedCashMinor,
	               DeclaredCashMinor, DifferenceMinor, Status, DriverNote, FinanceNote,
	               CreatedAt, ResolvedAt, ResolvedBy
	        FROM CashReconciliations WHERE 1=1`
	params := map[string]any{}
	if status != "" {
		sql += ` AND Status = @status`
		params["status"] = strings.ToUpper(status)
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT 100`

	iter := s.spanner.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var list []CashReconciliation
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		var rID, dID, st string
		var rtID, dNote, fNote, rBy spanner.NullString
		var sDate civil.Date
		var exp, dec, diff int64
		var created time.Time
		var resolved spanner.NullTime
		if err := row.Columns(&rID, &dID, &rtID, &sDate, &exp, &dec, &diff, &st, &dNote, &fNote, &created, &resolved, &rBy); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		item := CashReconciliation{
			ReconciliationID:  rID,
			DriverID:          dID,
			ShiftDate:         sDate.String(),
			ExpectedCashMinor: exp,
			DeclaredCashMinor: dec,
			DifferenceMinor:   diff,
			Status:            st,
			CreatedAt:         created,
		}
		if rtID.Valid {
			item.RouteID = rtID.StringVal
		}
		if dNote.Valid {
			item.DriverNote = dNote.StringVal
		}
		if fNote.Valid {
			item.FinanceNote = fNote.StringVal
		}
		if rBy.Valid {
			item.ResolvedBy = rBy.StringVal
		}
		if resolved.Valid {
			t := resolved.Time
			item.ResolvedAt = &t
		}
		list = append(list, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"reconciliations": list})
}
