package order

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SplitTenderPlan defines the multi-rail payment allocation.
type SplitTenderPlan struct {
	WalletMinor   int64  `json:"wallet_minor"`
	CreditMinor   int64  `json:"credit_minor"`
	CardMinor     int64  `json:"card_minor"`
	CashMinor     int64  `json:"cash_minor"`
	ProviderToken string `json:"provider_token,omitempty"`
}

// ValidateSplitTenderPlan checks that split legs strictly sum to the expected gross minor.
func ValidateSplitTenderPlan(expectedGrossMinor int64, plan SplitTenderPlan) error {
	if expectedGrossMinor <= 0 {
		return status.Error(codes.InvalidArgument, "gross_total_must_be_positive")
	}
	if plan.WalletMinor < 0 || plan.CreditMinor < 0 || plan.CardMinor < 0 || plan.CashMinor < 0 {
		return status.Error(codes.InvalidArgument, "negative_tender_amount")
	}
	total := plan.WalletMinor + plan.CreditMinor + plan.CardMinor + plan.CashMinor
	if total != expectedGrossMinor {
		return status.Errorf(codes.InvalidArgument, "tender_split_mismatch: sum %d != expected %d", total, expectedGrossMinor)
	}
	return nil
}

// BuildSplitTenderLegs creates the set of PaymentLeg structs for an order.
func BuildSplitTenderLegs(orderID string, plan SplitTenderPlan, now time.Time) []PaymentLeg {
	now = now.UTC()
	var legs []PaymentLeg

	if plan.WalletMinor > 0 {
		legs = append(legs, PaymentLeg{
			OrderID:        orderID,
			LegID:          uuid.NewString(),
			Method:         MethodWallet,
			AmountMinor:    plan.WalletMinor,
			Status:         PaymentStatusCaptured,
			IdempotencyKey: fmt.Sprintf("split-%s-wallet-%d", orderID, now.UnixNano()),
			CreatedAt:      now,
			CapturedAt:     spanner.NullTime{Time: now, Valid: true},
		})
	}
	if plan.CreditMinor > 0 {
		legs = append(legs, PaymentLeg{
			OrderID:        orderID,
			LegID:          uuid.NewString(),
			Method:         MethodCredit,
			AmountMinor:    plan.CreditMinor,
			Status:         PaymentStatusAuthorized,
			IdempotencyKey: fmt.Sprintf("split-%s-credit-%d", orderID, now.UnixNano()),
			CreatedAt:      now,
		})
	}
	if plan.CardMinor > 0 {
		var pRef spanner.NullString
		if plan.ProviderToken != "" {
			pRef = spanner.NullString{StringVal: plan.ProviderToken, Valid: true}
		}
		legs = append(legs, PaymentLeg{
			OrderID:        orderID,
			LegID:          uuid.NewString(),
			Method:         MethodCard,
			AmountMinor:    plan.CardMinor,
			Status:         PaymentStatusAuthorized,
			ProviderRef:    pRef,
			IdempotencyKey: fmt.Sprintf("split-%s-card-%d", orderID, now.UnixNano()),
			CreatedAt:      now,
		})
	}
	if plan.CashMinor > 0 {
		legs = append(legs, PaymentLeg{
			OrderID:        orderID,
			LegID:          uuid.NewString(),
			Method:         MethodCash,
			AmountMinor:    plan.CashMinor,
			Status:         PaymentStatusPending,
			IdempotencyKey: fmt.Sprintf("split-%s-cash-%d", orderID, now.UnixNano()),
			CreatedAt:      now,
		})
	}

	return legs
}

// RecordSplitTenderTxn inserts split legs and emits outbox events inside a Spanner RW transaction.
func (s *Service) RecordSplitTenderTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, order Order, plan SplitTenderPlan) error {
	now := s.now().UTC()
	legs := BuildSplitTenderLegs(order.OrderID, plan, now)

	for _, leg := range legs {
		if err := s.RecordPaymentLeg(ctx, txn, leg); err != nil {
			return fmt.Errorf("record leg %s (%s): %w", leg.LegID, leg.Method, err)
		}
	}

	buf := &spannerTxnBuffer{}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, order.OrderID, events.TopicMain, map[string]any{
		"type":         events.EventSplitPaymentCreated,
		"order_id":     order.OrderID,
		"supplier_id":  order.SupplierID,
		"retailer_id":  order.RetailerID,
		"wallet_minor": plan.WalletMinor,
		"credit_minor": plan.CreditMinor,
		"card_minor":   plan.CardMinor,
		"cash_minor":   plan.CashMinor,
		"currency":     order.Currency,
		"timestamp":    now.Format(time.RFC3339Nano),
	}); err != nil {
		return err
	}

	for _, m := range bufferedOutboxMutations(buf, now) {
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
	}
	return nil
}
