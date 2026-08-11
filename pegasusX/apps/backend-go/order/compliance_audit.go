package order

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
	"google.golang.org/api/iterator"
)

// Compliance audit read models (Phase 1 — regulatory/compliance-audit-dashboard.md).

// ComplianceFiscalOpenRow is one order stuck in the fiscal hard-gate.
type ComplianceFiscalOpenRow struct {
	OrderID       string    `json:"order_id"`
	RetailerID    string    `json:"retailer_id"`
	DriverID      string    `json:"driver_id,omitempty"`
	Status        string    `json:"status"`
	FiscalStatus  string    `json:"fiscal_status,omitempty"`
	TotalMinor    int64     `json:"total_minor"`
	Currency      string    `json:"currency"`
	AttemptID     string    `json:"latest_fiscal_attempt_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ComplianceForceCompleteRow is a force-completed fiscal skip audit row.
type ComplianceForceCompleteRow struct {
	OrderID      string    `json:"order_id"`
	RetailerID   string    `json:"retailer_id"`
	DriverID     string    `json:"driver_id,omitempty"`
	Status       string    `json:"status"`
	FiscalStatus string    `json:"fiscal_status"`
	ReasonCode   string    `json:"reason_code,omitempty"`
	ActorID      string    `json:"actor_id,omitempty"`
	AttemptID    string    `json:"attempt_id,omitempty"`
	TotalMinor   int64     `json:"total_minor"`
	Currency     string    `json:"currency"`
	CompletedAt  time.Time `json:"completed_at"`
}

// ComplianceClaimMismatchRow flags claim vs order residual inconsistency.
type ComplianceClaimMismatchRow struct {
	ClaimID          string `json:"claim_id"`
	OrderID          string `json:"order_id"`
	RetailerID       string `json:"retailer_id"`
	ClaimStatus      string `json:"claim_status"`
	ClaimAmountMinor int64  `json:"claim_amount_minor"`
	OrderTotalMinor  int64  `json:"order_total_minor"`
	OrderStatus      string `json:"order_status"`
	Currency         string `json:"currency"`
	MismatchReason   string `json:"mismatch_reason"`
	CreatedAt        time.Time `json:"created_at"`
}

// ComplianceCreditFreezeRow is an active frozen/blacklisted credit profile.
type ComplianceCreditFreezeRow struct {
	RetailerID           string    `json:"retailer_id"`
	Status               string    `json:"status"`
	RiskTier             string    `json:"risk_tier,omitempty"`
	CreditLimitMinor     int64     `json:"credit_limit_minor"`
	CurrentBalanceMinor  int64     `json:"current_balance_minor"`
	AvailableCreditMinor int64     `json:"available_credit_minor"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ComplianceSummary is the dashboard KPI strip.
type ComplianceSummary struct {
	OpenFiscalCount      int `json:"open_fiscal_count"`
	ForceCompleteCount   int `json:"force_complete_count"`
	ClaimMismatchCount   int `json:"claim_mismatch_count"`
	CreditFreezeCount    int `json:"credit_freeze_count"`
	GeneratedAt          string `json:"generated_at"`
}

// ComplianceDashboardResponse is GET /v1/compliance/dashboard (combined).
type ComplianceDashboardResponse struct {
	Summary         ComplianceSummary             `json:"summary"`
	OpenFiscal      []ComplianceFiscalOpenRow     `json:"open_fiscal"`
	ForceCompletes  []ComplianceForceCompleteRow  `json:"force_completes"`
	ClaimMismatches []ComplianceClaimMismatchRow  `json:"claim_mismatches"`
	CreditFreezes   []ComplianceCreditFreezeRow   `json:"credit_freezes"`
}

func (s *Service) resolveComplianceSupplier(r *http.Request) (string, bool) {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		return "", false
	}
	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok || strings.TrimSpace(supplierID) == "" {
		supplierID = strings.TrimSpace(claims.SupplierID)
	}
	if supplierID == "" {
		supplierID = strings.TrimSpace(s.resolveSupplierScope(r.Context()))
	}
	return supplierID, supplierID != ""
}

func complianceLimit(r *http.Request, def, max int) int {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > max {
		return max
	}
	return n
}

// HandleComplianceDashboard is GET /v1/compliance/dashboard.
func (s *Service) HandleComplianceDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := s.resolveComplianceSupplier(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "compliance_unavailable"})
		return
	}
	ctx := r.Context()
	limit := complianceLimit(r, 50, 200)

	openFiscal, err := s.listOpenFiscal(ctx, supplierID, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "compliance open fiscal failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "open_fiscal_query_failed"})
		return
	}
	forceRows, err := s.listForceCompletes(ctx, supplierID, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "compliance force-completes failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "force_complete_query_failed"})
		return
	}
	mismatches, err := s.listClaimMismatches(ctx, supplierID, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "compliance claim mismatches failed", "err", err)
		// Soft: claims table may not exist on older DBs.
		mismatches = []ComplianceClaimMismatchRow{}
	}
	freezes, err := s.listCreditFreezes(ctx, supplierID, limit)
	if err != nil {
		s.log.ErrorContext(ctx, "compliance credit freezes failed", "err", err)
		freezes = []ComplianceCreditFreezeRow{}
	}

	resp := ComplianceDashboardResponse{
		Summary: ComplianceSummary{
			OpenFiscalCount:    len(openFiscal),
			ForceCompleteCount: len(forceRows),
			ClaimMismatchCount: len(mismatches),
			CreditFreezeCount:  len(freezes),
			GeneratedAt:        s.now().UTC().Format(time.RFC3339Nano),
		},
		OpenFiscal:      openFiscal,
		ForceCompletes:  forceRows,
		ClaimMismatches: mismatches,
		CreditFreezes:   freezes,
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleComplianceFiscalOpen is GET /v1/compliance/fiscal-open.
func (s *Service) HandleComplianceFiscalOpen(w http.ResponseWriter, r *http.Request) {
	s.handleComplianceList(w, r, "open_fiscal", func(ctx context.Context, supplierID string, limit int) (any, error) {
		rows, err := s.listOpenFiscal(ctx, supplierID, limit)
		return map[string]any{"data": rows, "count": len(rows)}, err
	})
}

// HandleComplianceForceCompletes is GET /v1/compliance/force-completes.
func (s *Service) HandleComplianceForceCompletes(w http.ResponseWriter, r *http.Request) {
	s.handleComplianceList(w, r, "force_completes", func(ctx context.Context, supplierID string, limit int) (any, error) {
		rows, err := s.listForceCompletes(ctx, supplierID, limit)
		return map[string]any{"data": rows, "count": len(rows)}, err
	})
}

// HandleComplianceClaimMismatches is GET /v1/compliance/claim-mismatches.
func (s *Service) HandleComplianceClaimMismatches(w http.ResponseWriter, r *http.Request) {
	s.handleComplianceList(w, r, "claim_mismatches", func(ctx context.Context, supplierID string, limit int) (any, error) {
		rows, err := s.listClaimMismatches(ctx, supplierID, limit)
		return map[string]any{"data": rows, "count": len(rows)}, err
	})
}

// HandleComplianceCreditFreezes is GET /v1/compliance/credit-freezes.
func (s *Service) HandleComplianceCreditFreezes(w http.ResponseWriter, r *http.Request) {
	s.handleComplianceList(w, r, "credit_freezes", func(ctx context.Context, supplierID string, limit int) (any, error) {
		rows, err := s.listCreditFreezes(ctx, supplierID, limit)
		return map[string]any{"data": rows, "count": len(rows)}, err
	})
}

// HandleComplianceExport is GET /v1/compliance/export?format=json|csv&bucket=all|open_fiscal|force_completes|claim_mismatches|credit_freezes
func (s *Service) HandleComplianceExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := s.resolveComplianceSupplier(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "compliance_unavailable"})
		return
	}
	ctx := r.Context()
	limit := complianceLimit(r, 200, 500)
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	bucket := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("bucket")))
	if bucket == "" {
		bucket = "all"
	}

	dash, err := s.buildDashboard(ctx, supplierID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="compliance-export.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"bucket", "id", "order_id", "retailer_id", "status", "amount_minor", "currency", "detail", "updated_at"})
		writeCSVBucket := func(b string, rows [][]string) {
			for _, row := range rows {
				_ = cw.Write(append([]string{b}, row...))
			}
		}
		if bucket == "all" || bucket == "open_fiscal" {
			for _, r := range dash.OpenFiscal {
				writeCSVBucket("open_fiscal", [][]string{{
					r.OrderID, r.OrderID, r.RetailerID, r.Status,
					strconv.FormatInt(r.TotalMinor, 10), r.Currency, r.FiscalStatus, r.UpdatedAt.UTC().Format(time.RFC3339),
				}})
			}
		}
		if bucket == "all" || bucket == "force_completes" {
			for _, r := range dash.ForceCompletes {
				writeCSVBucket("force_completes", [][]string{{
					r.OrderID, r.OrderID, r.RetailerID, r.FiscalStatus,
					strconv.FormatInt(r.TotalMinor, 10), r.Currency, r.ReasonCode + "|" + r.ActorID, r.CompletedAt.UTC().Format(time.RFC3339),
				}})
			}
		}
		if bucket == "all" || bucket == "claim_mismatches" {
			for _, r := range dash.ClaimMismatches {
				writeCSVBucket("claim_mismatches", [][]string{{
					r.ClaimID, r.OrderID, r.RetailerID, r.ClaimStatus,
					strconv.FormatInt(r.ClaimAmountMinor, 10), r.Currency, r.MismatchReason, r.CreatedAt.UTC().Format(time.RFC3339),
				}})
			}
		}
		if bucket == "all" || bucket == "credit_freezes" {
			for _, r := range dash.CreditFreezes {
				writeCSVBucket("credit_freezes", [][]string{{
					r.RetailerID, "", r.RetailerID, r.Status,
					strconv.FormatInt(r.CurrentBalanceMinor, 10), "UZS", r.RiskTier, r.UpdatedAt.UTC().Format(time.RFC3339),
				}})
			}
		}
		cw.Flush()
		return
	default:
		// JSON
		payload := any(dash)
		switch bucket {
		case "open_fiscal":
			payload = map[string]any{"data": dash.OpenFiscal, "count": len(dash.OpenFiscal)}
		case "force_completes":
			payload = map[string]any{"data": dash.ForceCompletes, "count": len(dash.ForceCompletes)}
		case "claim_mismatches":
			payload = map[string]any{"data": dash.ClaimMismatches, "count": len(dash.ClaimMismatches)}
		case "credit_freezes":
			payload = map[string]any{"data": dash.CreditFreezes, "count": len(dash.CreditFreezes)}
		}
		w.Header().Set("Content-Disposition", `attachment; filename="compliance-export.json"`)
		writeJSON(w, http.StatusOK, payload)
	}
}

func (s *Service) handleComplianceList(w http.ResponseWriter, r *http.Request, label string, fn func(context.Context, string, int) (any, error)) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID, ok := s.resolveComplianceSupplier(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "compliance_unavailable"})
		return
	}
	limit := complianceLimit(r, 50, 200)
	out, err := fn(r.Context(), supplierID, limit)
	if err != nil {
		s.log.ErrorContext(r.Context(), "compliance list failed", "bucket", label, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": label + "_query_failed"})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) buildDashboard(ctx context.Context, supplierID string, limit int) (ComplianceDashboardResponse, error) {
	openFiscal, err := s.listOpenFiscal(ctx, supplierID, limit)
	if err != nil {
		return ComplianceDashboardResponse{}, err
	}
	forceRows, err := s.listForceCompletes(ctx, supplierID, limit)
	if err != nil {
		return ComplianceDashboardResponse{}, err
	}
	mismatches, _ := s.listClaimMismatches(ctx, supplierID, limit)
	if mismatches == nil {
		mismatches = []ComplianceClaimMismatchRow{}
	}
	freezes, _ := s.listCreditFreezes(ctx, supplierID, limit)
	if freezes == nil {
		freezes = []ComplianceCreditFreezeRow{}
	}
	return ComplianceDashboardResponse{
		Summary: ComplianceSummary{
			OpenFiscalCount:    len(openFiscal),
			ForceCompleteCount: len(forceRows),
			ClaimMismatchCount: len(mismatches),
			CreditFreezeCount:  len(freezes),
			GeneratedAt:        s.now().UTC().Format(time.RFC3339Nano),
		},
		OpenFiscal:      openFiscal,
		ForceCompletes:  forceRows,
		ClaimMismatches: mismatches,
		CreditFreezes:   freezes,
	}, nil
}

func (s *Service) listOpenFiscal(ctx context.Context, supplierID string, limit int) ([]ComplianceFiscalOpenRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, IFNULL(DriverId, ''), Status, IFNULL(FiscalStatus, ''),
		             TotalMinor, Currency, IFNULL(LatestFiscalAttemptId, ''), UpdatedAt
		      FROM Orders
		      WHERE SupplierId = @sid
		        AND Status IN UNNEST(@statuses)
		      ORDER BY UpdatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{
			"sid":      supplierID,
			"statuses": []string{string(StatusFiscalizing), string(StatusFiscalFailed)},
			"lim":      int64(limit),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ComplianceFiscalOpenRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Fallback without FiscalStatus column.
			return s.listOpenFiscalLegacy(ctx, supplierID, limit)
		}
		var r ComplianceFiscalOpenRow
		if err := row.Columns(
			&r.OrderID, &r.RetailerID, &r.DriverID, &r.Status, &r.FiscalStatus,
			&r.TotalMinor, &r.Currency, &r.AttemptID, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) listOpenFiscalLegacy(ctx context.Context, supplierID string, limit int) ([]ComplianceFiscalOpenRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, IFNULL(DriverId, ''), Status, TotalMinor, Currency, UpdatedAt
		      FROM Orders
		      WHERE SupplierId = @sid AND Status IN UNNEST(@statuses)
		      ORDER BY UpdatedAt DESC LIMIT @lim`,
		Params: map[string]any{
			"sid":      supplierID,
			"statuses": []string{string(StatusFiscalizing), string(StatusFiscalFailed)},
			"lim":      int64(limit),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ComplianceFiscalOpenRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r ComplianceFiscalOpenRow
		if err := row.Columns(&r.OrderID, &r.RetailerID, &r.DriverID, &r.Status, &r.TotalMinor, &r.Currency, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) listForceCompletes(ctx context.Context, supplierID string, limit int) ([]ComplianceForceCompleteRow, error) {
	// Prefer OrderFiscalReceipts FORCE_SKIPPED joined to Orders.
	stmt := spanner.Statement{
		SQL: `SELECT o.OrderId, o.RetailerId, IFNULL(o.DriverId, ''), o.Status, IFNULL(o.FiscalStatus, ''),
		             IFNULL(f.ReasonCode, ''), IFNULL(f.ActorId, ''), f.AttemptId,
		             o.TotalMinor, o.Currency, f.UpdatedAt
		      FROM OrderFiscalReceipts f
		      JOIN Orders o ON o.OrderId = f.OrderId
		      WHERE o.SupplierId = @sid AND f.Status = @st
		      ORDER BY f.UpdatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{
			"sid": supplierID,
			"st":  FiscalAttemptForceSkipped,
			"lim": int64(limit),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ComplianceForceCompleteRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Fallback: Orders with FiscalStatus = FORCE_SKIPPED
			return s.listForceCompletesFromOrders(ctx, supplierID, limit)
		}
		var r ComplianceForceCompleteRow
		if err := row.Columns(
			&r.OrderID, &r.RetailerID, &r.DriverID, &r.Status, &r.FiscalStatus,
			&r.ReasonCode, &r.ActorID, &r.AttemptID, &r.TotalMinor, &r.Currency, &r.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) listForceCompletesFromOrders(ctx context.Context, supplierID string, limit int) ([]ComplianceForceCompleteRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, RetailerId, IFNULL(DriverId, ''), Status, IFNULL(FiscalStatus, ''),
		             TotalMinor, Currency, UpdatedAt
		      FROM Orders
		      WHERE SupplierId = @sid AND FiscalStatus = @fs
		      ORDER BY UpdatedAt DESC LIMIT @lim`,
		Params: map[string]any{
			"sid": supplierID,
			"fs":  FiscalStatusForceSkipped,
			"lim": int64(limit),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ComplianceForceCompleteRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r ComplianceForceCompleteRow
		if err := row.Columns(
			&r.OrderID, &r.RetailerID, &r.DriverID, &r.Status, &r.FiscalStatus,
			&r.TotalMinor, &r.Currency, &r.CompletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Service) listClaimMismatches(ctx context.Context, supplierID string, limit int) ([]ComplianceClaimMismatchRow, error) {
	// Query open/approved claims for this supplier. We fetch more than limit because we filter in memory.
	stmt := spanner.Statement{
		SQL: `SELECT c.ClaimId, c.OrderId, c.RetailerId, c.Status, IFNULL(c.AmountMinor, 0),
		             IFNULL(c.Currency, ''), c.CreatedAt,
		             o.TotalMinor, o.Status
		      FROM Claims c
		      JOIN Orders o ON o.OrderId = c.OrderId
		      WHERE c.SupplierId = @sid
		        AND c.Status IN UNNEST(@statuses)
		      ORDER BY c.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{
			"sid":      supplierID,
			"statuses": []string{"OPEN", "FILED", "UNDER_REVIEW", "APPROVED", "PENDING"},
			"lim":      int64(limit * 3), // Fetch more to allow in-memory filtering
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ComplianceClaimMismatchRow, 0)

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var r ComplianceClaimMismatchRow
		var orderStatus string
		if err := row.Columns(
			&r.ClaimID, &r.OrderID, &r.RetailerID, &r.ClaimStatus, &r.ClaimAmountMinor,
			&r.Currency, &r.CreatedAt, &r.OrderTotalMinor, &orderStatus,
		); err != nil {
			return nil, err
		}
		r.OrderStatus = orderStatus
		if r.Currency == "" {
			r.Currency = "UZS"
		}

		// 1. Terminal order mismatch
		if r.OrderStatus == string(StatusCancelled) || r.OrderStatus == string(StatusReconciliationRequired) {
			r.MismatchReason = "open_claim_on_terminal_order_status"
			out = append(out, r)
			if len(out) >= limit {
				break
			}
			continue
		}

		// 2. Value mismatch using canonical canonical GetRemainingClaimable
		if s.claimsBridge != nil {
			remMinor, deliveredGross, err := s.claimsBridge.GetRemainingClaimable(ctx, r.OrderID)
			if err == nil {
				r.OrderTotalMinor = deliveredGross // overwrite with canonical value
				// Note: If this claim is already APPROVED, it is already deducted from remMinor by the canonical function.
				// But for compliance dashboard, if the claim amount itself exceeds the residual that was available before it,
				// it's a mismatch. For now, since this runs async, if it's open, remMinor is the true remaining.
				// If it's already approved, this simple check might flag it if it took up all the residual.
				// We assume here that ClaimAmountMinor > remMinor for OPEN claims is the main mismatch we want.
				if r.ClaimAmountMinor > remMinor && r.ClaimStatus != "APPROVED" {
					r.MismatchReason = "claim_amount_exceeds_residual"
					out = append(out, r)
					if len(out) >= limit {
						break
					}
					continue
				} else if r.ClaimAmountMinor > r.OrderTotalMinor {
					// Fallback for approved claims: shouldn't exceed total order value anyway.
					r.MismatchReason = "claim_amount_exceeds_order_total"
					out = append(out, r)
					if len(out) >= limit {
						break
					}
					continue
				}
			}
		} else {
			// Fallback if bridge is nil
			if r.ClaimAmountMinor > r.OrderTotalMinor {
				r.MismatchReason = "claim_amount_exceeds_order_total"
				out = append(out, r)
				if len(out) >= limit {
					break
				}
			}
		}
	}
	return out, nil
}

func (s *Service) listCreditFreezes(ctx context.Context, supplierID string, limit int) ([]ComplianceCreditFreezeRow, error) {
	if s.credit != nil {
		profiles, err := s.credit.ListSupplierProfiles(ctx, supplierID, string(credit.StatusFrozen), limit)
		if err != nil {
			// Also pull blacklisted.
			return s.listCreditFreezesSpanner(ctx, supplierID, limit)
		}
		out := make([]ComplianceCreditFreezeRow, 0, len(profiles))
		for _, p := range profiles {
			out = append(out, ComplianceCreditFreezeRow{
				RetailerID:           p.RetailerID,
				Status:               string(p.Status),
				RiskTier:             string(p.RiskTier),
				CreditLimitMinor:     p.CreditLimitMinor,
				CurrentBalanceMinor:  p.CurrentBalanceMinor,
				AvailableCreditMinor: p.AvailableCreditMinor,
				UpdatedAt:            p.UpdatedAt,
			})
		}
		// Append blacklisted
		bl, blErr := s.credit.ListSupplierProfiles(ctx, supplierID, string(credit.StatusBlacklisted), limit)
		if blErr == nil {
			for _, p := range bl {
				out = append(out, ComplianceCreditFreezeRow{
					RetailerID:           p.RetailerID,
					Status:               string(p.Status),
					RiskTier:             string(p.RiskTier),
					CreditLimitMinor:     p.CreditLimitMinor,
					CurrentBalanceMinor:  p.CurrentBalanceMinor,
					AvailableCreditMinor: p.AvailableCreditMinor,
					UpdatedAt:            p.UpdatedAt,
				})
			}
		}
		if len(out) > limit {
			out = out[:limit]
		}
		return out, nil
	}
	return s.listCreditFreezesSpanner(ctx, supplierID, limit)
}

func (s *Service) listCreditFreezesSpanner(ctx context.Context, supplierID string, limit int) ([]ComplianceCreditFreezeRow, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RetailerId, Status, CreditLimitMinor, CurrentBalanceMinor,
		             AvailableCreditMinor, UpdatedAt
		      FROM RetailerCreditProfiles
		      WHERE SupplierId = @sid AND Status IN UNNEST(@statuses)
		      ORDER BY UpdatedAt DESC LIMIT @lim`,
		Params: map[string]any{
			"sid":      supplierID,
			"statuses": []string{string(credit.StatusFrozen), string(credit.StatusBlacklisted)},
			"lim":      int64(limit),
		},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]ComplianceCreditFreezeRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("credit freeze query: %w", err)
		}
		var r ComplianceCreditFreezeRow
		if err := row.Columns(
			&r.RetailerID, &r.Status, &r.CreditLimitMinor,
			&r.CurrentBalanceMinor, &r.AvailableCreditMinor, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
