package order

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PaymentMethod string

const (
	MethodCard   PaymentMethod = "CARD"
	MethodCash   PaymentMethod = "CASH"
	MethodCredit PaymentMethod = "CREDIT"
	MethodRefund PaymentMethod = "REFUND"
)

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "PENDING"
	PaymentStatusCaptured PaymentStatus = "CAPTURED"
	PaymentStatusFailed   PaymentStatus = "FAILED"
	PaymentStatusReversed PaymentStatus = "REVERSED"
)

type PaymentLeg struct {
	OrderID        string             `spanner:"OrderId"`
	LegID          string             `spanner:"LegId"`
	Method         PaymentMethod      `spanner:"Method"`
	AmountMinor    int64              `spanner:"AmountMinor"`
	Status         PaymentStatus      `spanner:"Status"`
	IdempotencyKey string             `spanner:"IdempotencyKey"`
	ProviderRef    spanner.NullString `spanner:"ProviderRef"`
	CreatedAt      time.Time          `spanner:"CreatedAt"`
	CapturedAt     spanner.NullTime   `spanner:"CapturedAt"`
}

type SettlementException struct {
	OrderID     string             `spanner:"OrderId"`
	ExceptionID string             `spanner:"ExceptionId"`
	Type        string             `spanner:"Type"`
	AmountMinor int64              `spanner:"AmountMinor"`
	Status      string             `spanner:"Status"`
	Reason      spanner.NullString `spanner:"Reason"`
	CreatedBy   string             `spanner:"CreatedBy"`
	CreatedAt   time.Time          `spanner:"CreatedAt"`
}

// AssertMoneyCoversDelivery evaluates OrderPaymentLegs and exceptions vs order.DeliveredGross.
func (s *Service) AssertMoneyCoversDelivery(ctx context.Context, orderID string, proposedAmountMinor int64, proposedExceptionsMinor int64) error {
	delivered, err := s.getDeliveredGrossMinor(ctx, orderID)
	if err != nil {
		return err
	}

	paid, err := s.getCapturedPaymentMinor(ctx, orderID)
	if err != nil {
		return err
	}

	exceptions, err := s.getExceptionsTotalMinor(ctx, orderID)
	if err != nil {
		return err
	}

	totalCovered := paid + proposedAmountMinor + exceptions + proposedExceptionsMinor

	if totalCovered < delivered {
		return status.Errorf(codes.FailedPrecondition,
			"payment shortfall: delivered %d, covered %d", delivered, totalCovered)
	}
	return nil
}

// AssertMoneyCoversDeliveryTxn is the in-transaction variant (must not open Single() reads).
func (s *Service) AssertMoneyCoversDeliveryTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string, proposedAmountMinor int64, proposedExceptionsMinor int64) error {
	orderRecord, ok, err := s.repo.GetOrderTxn(ctx, txn, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}
	if !ok {
		return fmt.Errorf("order not found: %s", orderID)
	}
	delivered := deliveredGrossFromOrder(orderRecord)
	paid, err := s.getCapturedPaymentMinorTxn(ctx, txn, orderID)
	if err != nil {
		return err
	}
	exceptions, err := s.getExceptionsTotalMinorTxn(ctx, txn, orderID)
	if err != nil {
		return err
	}
	totalCovered := paid + proposedAmountMinor + exceptions + proposedExceptionsMinor
	if totalCovered < delivered {
		return status.Errorf(codes.FailedPrecondition,
			"payment shortfall: delivered %d, covered %d", delivered, totalCovered)
	}
	return nil
}

func sumAmountMinorRows(iter *spanner.RowIterator) (int64, error) {
	defer iter.Stop()
	var total int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		var amt int64
		if err := row.Columns(&amt); err != nil {
			return 0, err
		}
		total += amt
	}
	return total, nil
}

// getExceptionsTotalMinor sums all settlement exceptions for the order.
func (s *Service) getExceptionsTotalMinor(ctx context.Context, orderID string) (int64, error) {
	if s.spannerClient == nil {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderSettlementExceptions WHERE OrderId = @order_id`,
		Params: map[string]interface{}{
			"order_id": orderID,
		},
	}
	total, err := sumAmountMinorRows(s.spannerClient.Single().Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query exceptions: %w", err)
	}
	return total, nil
}

func (s *Service) getExceptionsTotalMinorTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderSettlementExceptions WHERE OrderId = @order_id`,
		Params: map[string]interface{}{
			"order_id": orderID,
		},
	}
	total, err := sumAmountMinorRows(txn.Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query exceptions: %w", err)
	}
	return total, nil
}

func deliveredGrossFromOrder(orderRecord Order) int64 {
	var gross int64
	for _, item := range orderRecord.LineItems {
		if item.OffloadStatus == "PARTIAL" || item.DeliveredQty > 0 {
			gross += item.DeliveredQty * item.UnitPrice
		} else if item.OffloadStatus == "RETURNED" || item.OffloadStatus == "FULL_RETURN" {
			// Delivered 0
		} else {
			gross += item.Quantity * item.UnitPrice
		}
	}
	return gross
}

// getDeliveredGrossMinor calculates the delivered gross using only lines with DeliveredQty > 0
// if there is a partial delivery. Otherwise it uses Quantity.
func (s *Service) getDeliveredGrossMinor(ctx context.Context, orderID string) (int64, error) {
	orderRecord, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return 0, fmt.Errorf("failed to get order: %w", err)
	}
	if !ok {
		return 0, fmt.Errorf("order not found: %s", orderID)
	}
	return deliveredGrossFromOrder(orderRecord), nil
}

// getCapturedPaymentMinor sums all CAPTURED payment legs for the order.
func (s *Service) getCapturedPaymentMinor(ctx context.Context, orderID string) (int64, error) {
	if s.spannerClient == nil {
		return 0, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderPaymentLegs WHERE OrderId = @order_id AND Status = @status`,
		Params: map[string]interface{}{
			"order_id": orderID,
			"status":   string(PaymentStatusCaptured),
		},
	}
	total, err := sumAmountMinorRows(s.spannerClient.Single().Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query payment legs: %w", err)
	}
	return total, nil
}

func (s *Service) getCapturedPaymentMinorTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AmountMinor FROM OrderPaymentLegs WHERE OrderId = @order_id AND Status = @status`,
		Params: map[string]interface{}{
			"order_id": orderID,
			"status":   string(PaymentStatusCaptured),
		},
	}
	total, err := sumAmountMinorRows(txn.Query(ctx, stmt))
	if err != nil {
		return 0, fmt.Errorf("query payment legs: %w", err)
	}
	return total, nil
}

// RecordPaymentLeg writes a payment leg to the database within a transaction.
func (s *Service) RecordPaymentLeg(ctx context.Context, txn *spanner.ReadWriteTransaction, leg PaymentLeg) error {
	m, err := spanner.InsertStruct("OrderPaymentLegs", leg)
	if err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// RecordSettlementException writes a settlement exception to the database within a transaction.
func (s *Service) RecordSettlementException(ctx context.Context, txn *spanner.ReadWriteTransaction, ex SettlementException) error {
	m, err := spanner.InsertStruct("OrderSettlementExceptions", ex)
	if err != nil {
		return err
	}
	return txn.BufferWrite([]*spanner.Mutation{m})
}
