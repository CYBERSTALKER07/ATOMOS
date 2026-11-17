package supplier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"google.golang.org/api/iterator"
)

func (s *Service) listFinanceExceptionRows(ctx context.Context, supplierID string) ([]SupplierExceptionRow, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return nil, nil
	}
	rows := make([]SupplierExceptionRow, 0)

	if s.portalSpanner != nil {
		seenFiscalOrders := make(map[string]bool)

		fiscalIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT OrderId, RetailerId, COALESCE(DriverId, ''), Status, COALESCE(FiscalStatus, ''), TotalMinor, UpdatedAt
				FROM Orders
				WHERE SupplierId = @sid
				  AND (Status = 'FISCAL_FAILED' OR UPPER(COALESCE(FiscalStatus, '')) IN ('FAILED', 'FISCAL_FAILED', 'DEAD_LETTER'))
				ORDER BY UpdatedAt DESC LIMIT 50`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := fiscalIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fiscalIter.Stop()
				return nil, err
			}
			var orderID, retailerID, driverID, status, fiscalStatus string
			var totalMinor int64
			var updatedAt time.Time
			if err := row.Columns(&orderID, &retailerID, &driverID, &status, &fiscalStatus, &totalMinor, &updatedAt); err != nil {
				continue
			}
			kind := "FISCAL_FAILED"
			note := "fiscal_failed"
			if strings.EqualFold(strings.TrimSpace(fiscalStatus), "DEAD_LETTER") {
				kind = "FISCAL_DEAD_LETTER"
				note = "fiscal_dead_letter"
			}
			seenFiscalOrders[orderID] = true
			rows = append(rows, SupplierExceptionRow{
				OrderID:          orderID,
				Kind:             kind,
				Status:           status,
				RetailerID:       retailerID,
				DriverID:         driverID,
				Note:             note,
				MoneyAtRiskMinor: totalMinor,
				SeverityTier:     "CRITICAL",
				UpdatedAt:        updatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		fiscalIter.Stop()

		ticketIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT t.OrderId, o.RetailerId, COALESCE(o.DriverId, ''), t.Type, t.Status, COALESCE(t.Title, ''), COALESCE(o.TotalMinor, 0), t.CreatedAt
				FROM ExceptionTickets t
				JOIN Orders o ON o.OrderId = t.OrderId
				WHERE o.SupplierId = @sid
				  AND UPPER(t.Status) IN ('OPEN', 'PENDING')
				  AND UPPER(t.Type) IN ('FISCAL_DEAD_LETTER', 'FISCAL_FAILED', 'FISCAL_FAILURE', 'FISCAL_TIMEOUT')
				ORDER BY t.CreatedAt DESC LIMIT 50`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := ticketIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				ticketIter.Stop()
				return nil, err
			}
			var orderID, retailerID, driverID, tType, status, title string
			var totalMinor int64
			var createdAt time.Time
			if err := row.Columns(&orderID, &retailerID, &driverID, &tType, &status, &title, &totalMinor, &createdAt); err != nil {
				continue
			}
			if seenFiscalOrders[orderID] {
				continue
			}
			seenFiscalOrders[orderID] = true
			kind := "FISCAL_FAILED"
			if strings.EqualFold(strings.TrimSpace(tType), "FISCAL_DEAD_LETTER") {
				kind = "FISCAL_DEAD_LETTER"
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          orderID,
				Kind:             kind,
				Status:           status,
				RetailerID:       retailerID,
				DriverID:         driverID,
				Note:             firstNonEmptyExceptionStr(title, strings.ToLower(kind)),
				MoneyAtRiskMinor: totalMinor,
				SeverityTier:     "CRITICAL",
				UpdatedAt:        createdAt.UTC().Format(time.RFC3339Nano),
			})
		}
		ticketIter.Stop()

		buyerIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT OrderId, RetailerId, COALESCE(DriverId, ''), Status, TotalMinor, UpdatedAt
				FROM Orders
				WHERE SupplierId = @sid AND BuyerAcceptanceStatus = 'REJECTED'
				ORDER BY UpdatedAt DESC LIMIT 30`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := buyerIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				buyerIter.Stop()
				return nil, err
			}
			var orderID, retailerID, driverID, status string
			var totalMinor int64
			var updatedAt time.Time
			if err := row.Columns(&orderID, &retailerID, &driverID, &status, &totalMinor, &updatedAt); err != nil {
				continue
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          orderID,
				Kind:             "BUYER_REJECTED",
				Status:           status,
				RetailerID:       retailerID,
				DriverID:         driverID,
				Note:             "buyer_rejected_delivery",
				MoneyAtRiskMinor: totalMinor,
				UpdatedAt:        updatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		buyerIter.Stop()

		creditIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT RetailerId, Status, CurrentBalanceMinor, UpdatedAt
				FROM RetailerCreditProfiles
				WHERE SupplierId = @sid AND Status IN ('FROZEN','BLACKLISTED')
				ORDER BY UpdatedAt DESC LIMIT 30`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := creditIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				creditIter.Stop()
				return nil, err
			}
			var retailerID, status string
			var balMinor int64
			var updatedAt time.Time
			if err := row.Columns(&retailerID, &status, &balMinor, &updatedAt); err != nil {
				continue
			}
			if balMinor < 0 {
				balMinor = 0
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          retailerID,
				Kind:             "CREDIT_FREEZE",
				Status:           status,
				RetailerID:       retailerID,
				Note:             "credit_profile_attention",
				MoneyAtRiskMinor: balMinor,
				UpdatedAt:        updatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		creditIter.Stop()

		promiseIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT PromiseId, RetailerId, PromisedMinor, Status, COALESCE(UpdatedAt, CreatedAt)
				FROM ArPaymentPromises
				WHERE SupplierId = @sid
				  AND (UPPER(Status) = 'BREACHED' OR (UPPER(Status) IN ('ACTIVE', 'PENDING') AND DueAt < @now))
				ORDER BY DueAt ASC LIMIT 30`,
			Params: map[string]any{"sid": sid, "now": s.now().UTC()},
		})
		for {
			row, err := promiseIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				promiseIter.Stop()
				return nil, err
			}
			var promiseID, retailerID, status string
			var promisedMinor int64
			var updatedAt time.Time
			if err := row.Columns(&promiseID, &retailerID, &promisedMinor, &status, &updatedAt); err != nil {
				continue
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          promiseID,
				Kind:             "PROMISE_BREACH",
				Status:           status,
				RetailerID:       retailerID,
				Note:             "payment_promise_breached",
				MoneyAtRiskMinor: promisedMinor,
				UpdatedAt:        updatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		promiseIter.Stop()

		cashIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT cr.ReconciliationId, cr.DriverId, cr.Status, ABS(cr.DifferenceMinor), cr.CreatedAt
				FROM CashReconciliations cr
				WHERE cr.Status IN ('PENDING','DISPUTED')
				  AND (cr.SupplierId = @sid OR cr.DriverId IN (
				    SELECT DISTINCT DriverId FROM SupplierTruckManifests WHERE SupplierId = @sid
				  ))
				ORDER BY cr.CreatedAt DESC LIMIT 30`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := cashIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				cashIter.Stop()
				return nil, err
			}
			var reconID, driverID, status string
			var diffMinor int64
			var createdAt time.Time
			if err := row.Columns(&reconID, &driverID, &status, &diffMinor, &createdAt); err != nil {
				continue
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          reconID,
				Kind:             "CASH_DISCREPANCY",
				Status:           status,
				DriverID:         driverID,
				Note:             "driver:" + driverID,
				MoneyAtRiskMinor: diffMinor,
				UpdatedAt:        createdAt.UTC().Format(time.RFC3339Nano),
			})
		}
		cashIter.Stop()

		cnIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT cn.CreditNoteId, cn.OrderId, o.RetailerId, cn.Status, ABS(cn.TotalGrossMinor), cn.CreatedAt
				FROM CreditNotes cn
				JOIN Orders o ON o.OrderId = cn.OrderId
				WHERE o.SupplierId = @sid AND cn.Status = 'DRAFT'
				ORDER BY cn.CreatedAt DESC LIMIT 30`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := cnIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				cnIter.Stop()
				return nil, err
			}
			var cnID, orderID, retailerID, status string
			var totalGrossMinor int64
			var createdAt time.Time
			if err := row.Columns(&cnID, &orderID, &retailerID, &status, &totalGrossMinor, &createdAt); err != nil {
				continue
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          orderID,
				Kind:             "CREDIT_NOTE_DRAFT",
				Status:           status,
				RetailerID:       retailerID,
				Note:             cnID,
				MoneyAtRiskMinor: totalGrossMinor,
				UpdatedAt:        createdAt.UTC().Format(time.RFC3339Nano),
			})
		}
		cnIter.Stop()

		claimsIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT ClaimId, OrderId, RetailerId, Status, COALESCE(AmountMinor, 0), UpdatedAt
				FROM Claims
				WHERE SupplierId = @sid
				  AND UPPER(Status) IN ('OPEN', 'PENDING', 'UNDER_REVIEW', 'ESCALATED')
				ORDER BY UpdatedAt DESC LIMIT 30`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := claimsIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				claimsIter.Stop()
				return nil, err
			}
			var claimID, orderID, retailerID, status string
			var amountMinor int64
			var updatedAt time.Time
			if err := row.Columns(&claimID, &orderID, &retailerID, &status, &amountMinor, &updatedAt); err != nil {
				continue
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          firstNonEmptyExceptionStr(orderID, claimID),
				Kind:             "CLAIM_REVIEW",
				Status:           status,
				RetailerID:       retailerID,
				Note:             claimID,
				MoneyAtRiskMinor: amountMinor,
				UpdatedAt:        updatedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		claimsIter.Stop()

		shopClosedIter := s.portalSpanner.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, spanner.Statement{
			SQL: `
				SELECT sc.AttemptId, sc.OrderId, sc.DriverId, sc.RetailerId, COALESCE(sc.Resolution, 'PENDING'), COALESCE(o.TotalMinor, 0), sc.ReportedAt
				FROM ShopClosedAttempts sc
				JOIN Orders o ON o.OrderId = sc.OrderId
				WHERE o.SupplierId = @sid AND (sc.Resolution IS NULL OR sc.Resolution = '')
				ORDER BY sc.ReportedAt DESC LIMIT 30`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := shopClosedIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				shopClosedIter.Stop()
				return nil, err
			}
			var attemptID, orderID, driverID, retailerID, status string
			var totalMinor int64
			var reportedAt time.Time
			if err := row.Columns(&attemptID, &orderID, &driverID, &retailerID, &status, &totalMinor, &reportedAt); err != nil {
				continue
			}
			rows = append(rows, SupplierExceptionRow{
				OrderID:          orderID,
				Kind:             "SHOP_CLOSED",
				Status:           status,
				RetailerID:       retailerID,
				DriverID:         driverID,
				Note:             "attempt:" + attemptID,
				MoneyAtRiskMinor: totalMinor,
				UpdatedAt:        reportedAt.UTC().Format(time.RFC3339Nano),
			})
		}
		shopClosedIter.Stop()
	}

	exportRows, err := s.listExportRejectedExceptions(ctx, sid)
	if err != nil {
		return nil, err
	}
	rows = append(rows, exportRows...)

	dedupRows, err := s.listRetailerDedupExceptions(ctx, sid)
	if err != nil {
		return nil, err
	}
	rows = append(rows, dedupRows...)

	return rows, nil
}

func (s *Service) listExportRejectedExceptions(ctx context.Context, supplierID string) ([]SupplierExceptionRow, error) {
	repo := s.OnecPostings()
	if repo == nil {
		return nil, nil
	}
	postings, err := repo.ListBySupplier(ctx, supplierID, partner.OnecStatusRejected, 30)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierExceptionRow, 0, len(postings))
	for _, p := range postings {
		note := strings.TrimSpace(p.DocumentKey)
		if reason := strings.TrimSpace(p.RejectReason); reason != "" {
			if note != "" {
				note += ": " + reason
			} else {
				note = reason
			}
		}
		severity := "high"
		if p.ExposureMinor > 0 {
			severity = "critical"
		}
		out = append(out, SupplierExceptionRow{
			OrderID:          p.PostingID,
			Kind:             "EXPORT_REJECTED",
			Status:           p.Status,
			Note:             note,
			UpdatedAt:        p.UpdatedAt.UTC().Format(time.RFC3339Nano),
			MoneyAtRiskMinor: p.ExposureMinor,
			Severity:         severity,
			PostingID:        p.PostingID,
			DocumentKey:      p.DocumentKey,
		})
	}
	return out, nil
}

func (s *Service) listRetailerDedupExceptions(ctx context.Context, supplierID string) ([]SupplierExceptionRow, error) {
	if s.portalSpanner == nil && !s.hasMemoryRetailerLinks() {
		return nil, nil
	}
	repo := s.RetailerLinks()
	if repo == nil {
		return nil, nil
	}
	links, err := repo.List(ctx, supplierID, entityresolution.DedupStatusDuplicateCandidate, 30)
	if err != nil {
		return nil, err
	}
	out := make([]SupplierExceptionRow, 0, len(links))
	for _, link := range links {
		var details []string
		if link.TaxIDTIN != "" {
			details = append(details, "tin:"+link.TaxIDTIN)
		}
		if link.NormalizedPhone != "" {
			details = append(details, "phone:"+link.NormalizedPhone)
		}
		if link.DedupClusterID != "" {
			details = append(details, "cluster:"+link.DedupClusterID)
		}
		if link.LegalName != "" {
			details = append(details, "name:"+link.LegalName)
		}
		note := strings.Join(details, " ")
		if note == "" {
			note = fmt.Sprintf("duplicate_candidate:%s", link.RetailerID)
		}
		out = append(out, SupplierExceptionRow{
			OrderID:        link.RetailerID,
			Kind:           "TIN_DEDUP_CANDIDATE",
			Status:         link.DedupStatus,
			RetailerID:     link.RetailerID,
			Note:           note,
			UpdatedAt:      link.UpdatedAt.UTC().Format(time.RFC3339Nano),
			Severity:       "high",
			DedupClusterID: link.DedupClusterID,
		})
	}
	return out, nil
}
