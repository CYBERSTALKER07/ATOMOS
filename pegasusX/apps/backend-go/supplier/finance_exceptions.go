package supplier

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (s *Service) listFinanceExceptionRows(ctx context.Context, supplierID string) ([]SupplierExceptionRow, error) {
	if s.portalSpanner == nil || strings.TrimSpace(supplierID) == "" {
		return nil, nil
	}
	rows := make([]SupplierExceptionRow, 0)

	fiscalIter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT OrderId, RetailerId, Status, UpdatedAt
			FROM Orders
			WHERE SupplierId = @sid AND FiscalStatus = 'FISCAL_FAILED'
			ORDER BY UpdatedAt DESC LIMIT 30`,
		Params: map[string]any{"sid": supplierID},
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
		var orderID, retailerID, status string
		var updatedAt time.Time
		if err := row.Columns(&orderID, &retailerID, &status, &updatedAt); err != nil {
			continue
		}
		rows = append(rows, SupplierExceptionRow{
			OrderID:    orderID,
			Kind:       "FISCAL_FAILED",
			Status:     status,
			RetailerID: retailerID,
			Note:       "fiscal_failed",
			UpdatedAt:  updatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	fiscalIter.Stop()

	buyerIter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT OrderId, RetailerId, Status, UpdatedAt
			FROM Orders
			WHERE SupplierId = @sid AND BuyerAcceptanceStatus = 'REJECTED'
			ORDER BY UpdatedAt DESC LIMIT 30`,
		Params: map[string]any{"sid": supplierID},
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
		var orderID, retailerID, status string
		var updatedAt time.Time
		if err := row.Columns(&orderID, &retailerID, &status, &updatedAt); err != nil {
			continue
		}
		rows = append(rows, SupplierExceptionRow{
			OrderID:    orderID,
			Kind:       "BUYER_REJECTED",
			Status:     status,
			RetailerID: retailerID,
			Note:       "buyer_rejected_delivery",
			UpdatedAt:  updatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	buyerIter.Stop()

	creditIter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT RetailerId, Status, UpdatedAt
			FROM RetailerCreditProfiles
			WHERE SupplierId = @sid AND Status IN ('FROZEN','BLACKLISTED')
			ORDER BY UpdatedAt DESC LIMIT 30`,
		Params: map[string]any{"sid": supplierID},
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
		var updatedAt time.Time
		if err := row.Columns(&retailerID, &status, &updatedAt); err != nil {
			continue
		}
		rows = append(rows, SupplierExceptionRow{
			OrderID:    retailerID,
			Kind:       "CREDIT_FREEZE",
			Status:     status,
			RetailerID: retailerID,
			Note:       "credit_profile_attention",
			UpdatedAt:  updatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	creditIter.Stop()

	cashIter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT cr.ReconciliationId, cr.DriverId, cr.Status, cr.CreatedAt
			FROM CashReconciliations cr
			WHERE cr.Status IN ('PENDING','DISPUTED')
			  AND cr.DriverId IN (
			    SELECT DISTINCT DriverId FROM SupplierTruckManifests WHERE SupplierId = @sid
			  )
			ORDER BY cr.CreatedAt DESC LIMIT 30`,
		Params: map[string]any{"sid": supplierID},
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
		var createdAt time.Time
		if err := row.Columns(&reconID, &driverID, &status, &createdAt); err != nil {
			continue
		}
		rows = append(rows, SupplierExceptionRow{
			OrderID:    reconID,
			Kind:       "CASH_DISCREPANCY",
			Status:     status,
			Note:       "driver:" + driverID,
			UpdatedAt:  createdAt.UTC().Format(time.RFC3339Nano),
		})
	}
	cashIter.Stop()

	cnIter := s.portalSpanner.Single().Query(ctx, spanner.Statement{
		SQL: `
			SELECT cn.CreditNoteId, cn.OrderId, cn.Status, cn.CreatedAt
			FROM CreditNotes cn
			JOIN Orders o ON o.OrderId = cn.OrderId
			WHERE o.SupplierId = @sid AND cn.Status = 'DRAFT'
			ORDER BY cn.CreatedAt DESC LIMIT 30`,
		Params: map[string]any{"sid": supplierID},
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
		var cnID, orderID, status string
		var createdAt time.Time
		if err := row.Columns(&cnID, &orderID, &status, &createdAt); err != nil {
			continue
		}
		rows = append(rows, SupplierExceptionRow{
			OrderID:    orderID,
			Kind:       "CREDIT_NOTE_DRAFT",
			Status:     status,
			Note:       cnID,
			UpdatedAt:  createdAt.UTC().Format(time.RFC3339Nano),
		})
	}
	cnIter.Stop()

	return rows, nil
}
