package order

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

const systemColdChainReporter = "wms-cold-chain"

// SystemTemperatureBreachArgs is used by WMS cold-chain auto-raise (same Spanner txn).
type SystemTemperatureBreachArgs struct {
	ManifestID string
	ReadingID  string
	TempC      float64
	MinC       float64
	MaxC       float64
	OrderIDs   []string
}

// RaiseSystemTemperatureBreachInTxn inserts OPEN TEMPERATURE_BREACH reports + outbox for reportable orders.
// Idempotent: skips an order that already has an open SYSTEM cold-chain breach.
func (s *Service) RaiseSystemTemperatureBreachInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	args SystemTemperatureBreachArgs,
) error {
	if s == nil || txn == nil {
		return fmt.Errorf("unavailable")
	}
	note := fmt.Sprintf(
		"WMS cold-chain excursion: temp=%.2fC band=[%.2f,%.2f] manifest=%s reading=%s",
		args.TempC, args.MinC, args.MaxC, strings.TrimSpace(args.ManifestID), strings.TrimSpace(args.ReadingID),
	)
	for _, orderID := range args.OrderIDs {
		orderID = strings.TrimSpace(orderID)
		if orderID == "" {
			continue
		}
		current, found, err := s.loadOrderInTxn(ctx, txn, orderID)
		if err != nil {
			return err
		}
		if !found || !conditionReportable(current.Status) {
			continue
		}
		exists, err := hasOpenSystemColdChainBreachInTxn(ctx, txn, orderID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		report := ConditionReport{
			ReportID:         s.newID(),
			OrderID:          current.OrderID,
			SupplierID:       current.SupplierID,
			RetailerID:       current.RetailerID,
			ConditionType:    ConditionTypeTemperatureBreach,
			Severity:         SeverityHigh,
			Description:      note,
			PhotoURLs:        nil,
			ProofIDs:         nil,
			ReportedBy:       systemColdChainReporter,
			ReportedByRole:   "SYSTEM",
			ResolutionStatus: ResolutionStatusOpen,
			CreatedAt:        s.now(),
		}
		if err := bufferConditionReportInTxn(ctx, txn, report); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadOrderInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (Order, bool, error) {
	if repo, ok := s.repo.(*SpannerRepository); ok {
		return repo.GetOrderTxn(ctx, txn, orderID)
	}
	// Non-Spanner repos (tests): fall back to GetOrder outside txn semantics.
	return s.repo.GetOrder(ctx, orderID)
}

func hasOpenSystemColdChainBreachInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT ReportId FROM OrderConditionReports
		      WHERE OrderId = @oid
		        AND ConditionType = @ctype
		        AND ResolutionStatus = @st
		        AND ReportedBy = @by
		      LIMIT 1`,
		Params: map[string]any{
			"oid":   orderID,
			"ctype": string(ConditionTypeTemperatureBreach),
			"st":    string(ResolutionStatusOpen),
			"by":    systemColdChainReporter,
		},
	})
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func bufferConditionReportInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, report ConditionReport) error {
	photoURLs, err := json.Marshal(report.PhotoURLs)
	if err != nil {
		return fmt.Errorf("marshal condition report photo urls: %w", err)
	}
	proofIDs, err := json.Marshal(report.ProofIDs)
	if err != nil {
		return fmt.Errorf("marshal condition report proof ids: %w", err)
	}
	buf := &spannerTxnBuffer{}
	payload := events.ConditionEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventOrderConditionReported,
			Timestamp: report.CreatedAt.UTC().Format(time.RFC3339Nano),
		},
		ReportID:      report.ReportID,
		OrderID:       report.OrderID,
		SupplierID:    report.SupplierID,
		RetailerID:    report.RetailerID,
		ReporterID:    report.ReportedBy,
		ReporterRole:  report.ReportedByRole,
		ConditionType: string(report.ConditionType),
		SKU:           report.SKU,
		Quantity:      0,
		GCSPaths:      report.PhotoURLs,
		Notes:         report.Description,
	}
	if err := outbox.EmitJSON(ctx, buf, events.AggregateConditionReport, report.ReportID, events.TopicMain, payload); err != nil {
		return err
	}
	mutations := []*spanner.Mutation{
		spanner.InsertMap("OrderConditionReports", map[string]any{
			"ReportId":         report.ReportID,
			"OrderId":          report.OrderID,
			"SupplierId":       report.SupplierID,
			"RetailerId":       report.RetailerID,
			"LineItemIndex":    nullableInt64(report.LineItemIndex),
			"SKU":              nullableString(report.SKU),
			"ConditionType":    string(report.ConditionType),
			"Severity":         string(report.Severity),
			"Description":      nullableString(report.Description),
			"PhotoURLsJson":    photoURLs,
			"ProofIdsJson":     proofIDs,
			"ReportedBy":       report.ReportedBy,
			"ReportedByRole":   report.ReportedByRole,
			"ResolutionStatus": string(report.ResolutionStatus),
			"ResolvedBy":       nullableString(report.ResolvedBy),
			"ResolvedAt":       nullableTime(report.ResolvedAt),
			"ResolutionNotes":  nullableString(report.ResolutionNotes),
			"CreatedAt":        report.CreatedAt.UTC(),
		}),
	}
	for _, e := range buf.events {
		createdAt := e.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
			"EventId":       e.EventID,
			"AggregateType": e.AggregateType,
			"AggregateId":   e.AggregateID,
			"TopicName":     e.TopicName,
			"Payload":       e.Payload,
			"CreatedAt":     createdAt,
			"PublishedAt":   nil,
		}))
	}
	return txn.BufferWrite(mutations)
}
