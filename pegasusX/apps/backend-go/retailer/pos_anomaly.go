package retailer

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

const (
	// PosAnomalyReviewFlag surfaces suspicious POS behavior for owner/audit review
	// without auto-rejecting or blocking legitimate shop operations.
	PosAnomalyReviewFlag = "POS_ANOMALY_REVIEW"

	posAnomalyRepeatedVoidCount     = 3
	posAnomalyMinSalesForRatio      = 3
	posAnomalyVoidRatioBpsThreshold = 3000 // 30.00% integer basis points
)

// PosAnomalyReview describes a non-blocking POS anomaly detected for review.
type PosAnomalyReview struct {
	Flag              string   `json:"flag"`
	SessionID         string   `json:"session_id,omitempty"`
	RegisterID        string   `json:"register_id,omitempty"`
	LocationID        string   `json:"location_id,omitempty"`
	RetailerID        string   `json:"retailer_id,omitempty"`
	Reasons           []string `json:"reasons,omitempty"`
	VoidCount         int      `json:"void_count"`
	TotalSalesCount   int      `json:"total_sales_count"`
	VoidTotalMinor    int64    `json:"void_total_minor"`
	VoidRatioBps      int64    `json:"void_ratio_bps"`
	CashVarianceMinor int64    `json:"cash_variance_minor"`
	ThresholdMinor    int64    `json:"threshold_minor"`
	AutoRejected      bool     `json:"auto_rejected"`
	DetectedAt        string   `json:"detected_at,omitempty"`
}

// evaluateSessionPosAnomaly detects repeated voids, high void-to-sale ratio,
// or material cash variance using integer math only. It never blocks or auto-rejects.
func evaluateSessionPosAnomaly(sess PosSessionDTO, sales []PosSaleDTO, varianceThresholdMinor int64, now time.Time) (PosAnomalyReview, bool) {
	if varianceThresholdMinor <= 0 {
		varianceThresholdMinor = defaultVarianceAlertMinor
	}
	var voidCount int
	var voidTotalMinor int64
	totalSales := len(sales)
	for _, sale := range sales {
		if strings.EqualFold(strings.TrimSpace(sale.Status), "VOIDED") || strings.TrimSpace(sale.VoidedAt) != "" {
			voidCount++
			voidTotalMinor += sale.TotalMinor
		}
	}

	var voidRatioBps int64
	if totalSales > 0 {
		voidRatioBps = int64(voidCount) * 10000 / int64(totalSales)
	}

	var absVariance int64
	var signedVariance int64
	if sess.VarianceMinor != nil {
		signedVariance = *sess.VarianceMinor
		absVariance = abs64(signedVariance)
	}

	var reasons []string
	if voidCount >= posAnomalyRepeatedVoidCount {
		reasons = append(reasons, "REPEATED_VOIDS")
	}
	if totalSales >= posAnomalyMinSalesForRatio && int64(voidCount)*10000 >= int64(totalSales)*posAnomalyVoidRatioBpsThreshold {
		reasons = append(reasons, "HIGH_VOID_RATIO")
	}
	if sess.VarianceMinor != nil && absVariance >= varianceThresholdMinor {
		reasons = append(reasons, "CASH_MISMATCH")
	}
	if len(reasons) == 0 {
		return PosAnomalyReview{}, false
	}

	return PosAnomalyReview{
		Flag:              PosAnomalyReviewFlag,
		SessionID:         sess.SessionID,
		RegisterID:        sess.RegisterID,
		LocationID:        sess.LocationID,
		RetailerID:        sess.RetailerID,
		Reasons:           reasons,
		VoidCount:         voidCount,
		TotalSalesCount:   totalSales,
		VoidTotalMinor:    voidTotalMinor,
		VoidRatioBps:      voidRatioBps,
		CashVarianceMinor: signedVariance,
		ThresholdMinor:    varianceThresholdMinor,
		AutoRejected:      false,
		DetectedAt:        now.UTC().Format(time.RFC3339Nano),
	}, true
}

func (s *Service) inspectSessionPosAnomaly(ctx context.Context, orgID string, sess PosSessionDTO, sales []PosSaleDTO) (PosAnomalyReview, bool) {
	if s == nil {
		return PosAnomalyReview{}, false
	}
	if sess.RetailerID == "" {
		sess.RetailerID = orgID
	}
	cfg := s.shiftsConfig(ctx, orgID)
	review, flagged := evaluateSessionPosAnomaly(sess, sales, cfg.varianceAlertMinor, s.currentTime())
	if !flagged {
		return PosAnomalyReview{}, false
	}
	s.recordAndNotifyPosAnomaly(ctx, orgID, review)
	return review, true
}

func (s *Service) recordAndNotifyPosAnomaly(ctx context.Context, orgID string, review PosAnomalyReview) {
	if s == nil {
		return
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		if s.posAnomaliesByOrg == nil {
			s.posAnomaliesByOrg = make(map[string]map[string]PosAnomalyReview)
		}
		if s.posAnomaliesByOrg[orgID] == nil {
			s.posAnomaliesByOrg[orgID] = make(map[string]PosAnomalyReview)
		}
		key := review.SessionID
		if key == "" {
			key = "pos-session-unknown"
		}
		s.posAnomaliesByOrg[orgID][key] = review
		s.mu.Unlock()
	} else {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
			buf := &spannerTxnBuffer{}
			if aErr := outbox.WriteAudit(txCtx, buf, "", orgID, "SYSTEM", PosAnomalyReviewFlag, "PosSession", review.SessionID, review); aErr != nil {
				return aErr
			}
			var muts []*spanner.Mutation
			for _, a := range buf.audits {
				muts = append(muts, spanner.InsertMap("AuditLog", a.AuditRowMap()))
			}
			if len(muts) > 0 {
				return txn.BufferWrite(muts)
			}
			return nil
		})
		if err != nil && s.log != nil {
			s.log.Warn("record POS_ANOMALY_REVIEW audit failed", "retailer_id", orgID, "session_id", review.SessionID, "err", err)
		}
	}

	if s.log != nil {
		s.log.Warn("POS anomaly detected (non-blocking review)",
			"flag", review.Flag,
			"retailer_id", orgID,
			"session_id", review.SessionID,
			"reasons", strings.Join(review.Reasons, ","),
			"void_count", review.VoidCount,
			"total_sales", review.TotalSalesCount,
			"cash_variance_minor", review.CashVarianceMinor,
		)
	}
	if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
		title := "POS anomaly flagged for review"
		body := fmt.Sprintf("Session %s flagged (%s): %d/%d voids, cash variance %d minor",
			review.SessionID, strings.Join(review.Reasons, ", "), review.VoidCount, review.TotalSalesCount, review.CashVarianceMinor)
		if err := w.CreateNotification(ctx, orgID, "RETAILER", PosAnomalyReviewFlag, title, body, "/pos"); err != nil && s.log != nil {
			s.log.Warn("notify POS anomaly failed", "retailer_id", orgID, "err", err)
		}
		members, err := s.listOrgMembers(ctx, orgID)
		if err != nil {
			if s.log != nil {
				s.log.Warn("list org members for POS anomaly notification failed", "retailer_id", orgID, "err", err)
			}
		} else {
			for _, m := range members {
				if m.IsOwner && m.IsActive {
					if nErr := w.CreateNotification(ctx, m.UserID, "RETAILER", PosAnomalyReviewFlag, title, body, "/pos"); nErr != nil && s.log != nil {
						s.log.Warn("notify owner POS anomaly failed", "user_id", m.UserID, "err", nErr)
					}
				}
			}
		}
	}
	if err := s.emitPosEvent(ctx, orgID, events.EventRetailerShiftVariance, map[string]any{
		"flag":                PosAnomalyReviewFlag,
		"session_id":          review.SessionID,
		"register_id":         review.RegisterID,
		"location_id":         review.LocationID,
		"reasons":             review.Reasons,
		"void_count":          review.VoidCount,
		"total_sales_count":   review.TotalSalesCount,
		"cash_variance_minor": review.CashVarianceMinor,
		"auto_rejected":       false,
	}); err != nil && s.log != nil {
		s.log.Warn("emit POS anomaly event failed", "retailer_id", orgID, "session_id", review.SessionID, "err", err)
	}
}

func (s *Service) listRetailerPosSessionsWindow(ctx context.Context, orgID string, since time.Time) ([]PosSessionDTO, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []PosSessionDTO
		for _, sess := range s.posSessions {
			if sess.RetailerID != orgID {
				continue
			}
			if !since.IsZero() && sess.OpenedAt != "" {
				if ts, err := time.Parse(time.RFC3339Nano, sess.OpenedAt); err == nil && ts.Before(since) {
					continue
				}
			}
			out = append(out, sess)
		}
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, RegisterId, LocationId, RetailerId, OpenedByUserId, ClosedByUserId,
			Status, OpeningFloatMinor, ClosingCashMinor, ExpectedCashMinor, VarianceMinor,
			Currency, OpenedAt, ClosedAt
			FROM RetailerPosSessions
			WHERE RetailerId = @rid AND OpenedAt >= @since
			ORDER BY OpenedAt DESC LIMIT 200`,
		Params: map[string]any{"rid": orgID, "since": since},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []PosSessionDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query pos sessions for anomaly review: %w", err)
		}
		sess, _, decErr := decodePosSessionRow(row)
		if decErr != nil {
			return nil, decErr
		}
		out = append(out, sess)
	}
	return out, nil
}

func (s *Service) collectControlTowerPosAnomalies(ctx context.Context, orgID string, from7, now time.Time) ([]PosAnomalyReview, error) {
	cfg := s.shiftsConfig(ctx, orgID)
	sales, err := s.reportsPosSales(ctx, orgID, "", from7, now)
	if err != nil {
		return nil, err
	}
	sessions, err := s.listRetailerPosSessionsWindow(ctx, orgID, from7)
	if err != nil {
		return nil, err
	}

	salesBySession := make(map[string][]PosSaleDTO)
	for _, sale := range sales {
		sid := strings.TrimSpace(sale.SessionID)
		if sid == "" {
			sid = "pos-window"
		}
		salesBySession[sid] = append(salesBySession[sid], sale)
	}

	sessByID := make(map[string]PosSessionDTO, len(sessions))
	for _, sess := range sessions {
		sessByID[sess.SessionID] = sess
	}

	bySession := make(map[string]PosAnomalyReview)
	if s.spannerClient == nil {
		s.mu.RLock()
		if s.posAnomaliesByOrg != nil {
			for sid, rec := range s.posAnomaliesByOrg[orgID] {
				bySession[sid] = rec
			}
		}
		s.mu.RUnlock()
	}

	for sid, sess := range sessByID {
		if rev, flagged := evaluateSessionPosAnomaly(sess, salesBySession[sid], cfg.varianceAlertMinor, now); flagged {
			bySession[sid] = rev
		}
	}
	for sid, sessSales := range salesBySession {
		if _, seen := sessByID[sid]; seen {
			continue
		}
		syn := PosSessionDTO{SessionID: sid, RetailerID: orgID}
		if len(sessSales) > 0 {
			syn.RegisterID = sessSales[0].RegisterID
			syn.LocationID = sessSales[0].LocationID
		}
		if rev, flagged := evaluateSessionPosAnomaly(syn, sessSales, cfg.varianceAlertMinor, now); flagged {
			bySession[sid] = rev
		}
	}

	if len(bySession) == 0 {
		return nil, nil
	}
	out := make([]PosAnomalyReview, 0, len(bySession))
	for _, rev := range bySession {
		rev.Flag = PosAnomalyReviewFlag
		rev.AutoRejected = false
		out = append(out, rev)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].SessionID < out[j].SessionID
	})
	return out, nil
}
