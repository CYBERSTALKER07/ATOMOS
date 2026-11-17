package roi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// SpannerRepository persists RoiBaselines and RoiEvents in Cloud Spanner and reads
// live AR / Inventory / BillingMeterEvents metrics using strong snapshots.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed ROI repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Client exposes the underlying *spanner.Client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

type entityStampedOutboxBuf struct {
	inner    outbox.TxnBuffer
	entityID string
}

func (b *entityStampedOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.entityID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.entityID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}

func (r *SpannerRepository) UpsertBaseline(ctx context.Context, b Baseline) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(_ context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertOrUpdateMap("RoiBaselines", map[string]any{
				"TenantId":               b.TenantID,
				"BaselineId":             b.BaselineID,
				"WindowDays":             b.WindowDays,
				"BaselineArMinor":        b.BaselineArMinor,
				"BaselineBadDebtMinor":   b.BaselineBadDebtMinor,
				"BaselineExpiredMinor":   b.BaselineExpiredMinor,
				"BaselineStockoutMinor":  b.BaselineStockoutMinor,
				"BaselineLogisticsMinor": b.BaselineLogisticsMinor,
				"BaselineLabourHours":    b.BaselineLabourHours,
				"SurveyHourlyWageMinor":  b.SurveyHourlyWageMinor,
				"SurveyNotes":            nullIfEmpty(b.SurveyNotes),
				"SurveyJson":             nullIfEmpty(b.SurveyJSON),
				"Currency":               b.Currency,
				"CapturedBy":             b.CapturedBy,
				"PeriodStart":            b.PeriodStart.UTC(),
				"PeriodEnd":              b.PeriodEnd.UTC(),
				"CreatedAt":              spanner.CommitTimestamp,
				"UpdatedAt":              spanner.CommitTimestamp,
			}),
		})
	})
}

func (r *SpannerRepository) GetLatestBaseline(ctx context.Context, tenantID string) (Baseline, bool, error) {
	if r == nil || r.client == nil {
		return Baseline{}, false, ErrRepoUnavailable
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, BaselineId, WindowDays, BaselineArMinor, BaselineBadDebtMinor,
		             BaselineExpiredMinor, BaselineStockoutMinor, BaselineLogisticsMinor,
		             BaselineLabourHours, SurveyHourlyWageMinor, SurveyNotes, SurveyJson,
		             Currency, CapturedBy, PeriodStart, PeriodEnd, CreatedAt, UpdatedAt
		      FROM RoiBaselines
		      WHERE TenantId = @tid
		      ORDER BY CreatedAt DESC, BaselineId DESC
		      LIMIT 1`,
		Params: map[string]any{"tid": strings.TrimSpace(tenantID)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return Baseline{}, false, nil
	}
	if err != nil {
		return Baseline{}, false, fmt.Errorf("read RoiBaselines: %w", err)
	}
	b, err := scanBaselineRow(row)
	if err != nil {
		return Baseline{}, false, err
	}
	return b, true, nil
}

func (r *SpannerRepository) FindEventByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (RoiEvent, bool, error) {
	if r == nil || r.client == nil {
		return RoiEvent{}, false, ErrRepoUnavailable
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return RoiEvent{}, false, nil
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL: `SELECT TenantId, EventId, Category, SavingsMinor, Currency, Methodology,
		             ConfidenceBps, IsEstimate, HoldoutControl, ProposalId,
		             EvidenceRefType, EvidenceRefId, Notes, RecordedBy,
		             IdempotencyKey, OccurredAt, CreatedAt
		      FROM RoiEvents@{FORCE_INDEX=UQ_RoiEvents_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
		      WHERE TenantId = @tid AND IdempotencyKey = @idem AND IdempotencyKey IS NOT NULL
		      LIMIT 1`,
		Params: map[string]any{
			"tid":  strings.TrimSpace(tenantID),
			"idem": strings.TrimSpace(idempotencyKey),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return RoiEvent{}, false, nil
	}
	if err != nil {
		return RoiEvent{}, false, fmt.Errorf("lookup RoiEvents idempotency: %w", err)
	}
	ev, err := scanEventRow(row)
	if err != nil {
		return RoiEvent{}, false, err
	}
	return ev, true, nil
}

func (r *SpannerRepository) CreateEvent(ctx context.Context, ev RoiEvent) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if ev.IdempotencyKey != "" {
			iter := txn.Query(txCtx, spanner.Statement{
				SQL: `SELECT EventId
				      FROM RoiEvents@{FORCE_INDEX=UQ_RoiEvents_Idempotency,spanner_emulator.disable_query_null_filtered_index_check=true}
				      WHERE TenantId = @tid AND IdempotencyKey = @idem AND IdempotencyKey IS NOT NULL
				      LIMIT 1`,
				Params: map[string]any{
					"tid":  ev.TenantID,
					"idem": ev.IdempotencyKey,
				},
			})
			_, err := iter.Next()
			iter.Stop()
			if err == nil {
				return nil
			}
			if err != iterator.Done {
				return fmt.Errorf("check RoiEvents idempotency in txn: %w", err)
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("RoiEvents", map[string]any{
				"TenantId":        ev.TenantID,
				"EventId":         ev.EventID,
				"Category":        ev.Category,
				"SavingsMinor":    ev.SavingsMinor,
				"Currency":        ev.Currency,
				"Methodology":     ev.Methodology,
				"ConfidenceBps":   ev.ConfidenceBps,
				"IsEstimate":      ev.IsEstimate,
				"HoldoutControl":  ev.HoldoutControl,
				"ProposalId":      nullIfEmpty(ev.ProposalID),
				"EvidenceRefType": nullIfEmpty(ev.EvidenceRefType),
				"EvidenceRefId":   nullIfEmpty(ev.EvidenceRefID),
				"Notes":           nullIfEmpty(ev.Notes),
				"RecordedBy":      ev.RecordedBy,
				"IdempotencyKey":  nullIfEmpty(ev.IdempotencyKey),
				"OccurredAt":      ev.OccurredAt.UTC(),
				"CreatedAt":       spanner.CommitTimestamp,
			}),
		}); err != nil {
			return fmt.Errorf("insert RoiEvents: %w", err)
		}

		outboxInner := outbox.NewSpannerTxnBuffer(txn)
		stampedBuf := &entityStampedOutboxBuf{inner: outboxInner, entityID: ev.TenantID}
		payload := events.RoiEventRecordedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventRoiEventRecorded,
				Version:   1,
				Timestamp: ev.CreatedAt.UTC().Format(time.RFC3339Nano),
			},
			TenantID:        ev.TenantID,
			EventID:         ev.EventID,
			Category:        ev.Category,
			SavingsMinor:    ev.SavingsMinor,
			Currency:        ev.Currency,
			Methodology:     ev.Methodology,
			ConfidenceBps:   ev.ConfidenceBps,
			IsEstimate:      ev.IsEstimate,
			HoldoutControl:  ev.HoldoutControl,
			ProposalID:      ev.ProposalID,
			EvidenceRefType: ev.EvidenceRefType,
			EvidenceRefID:   ev.EvidenceRefID,
			OccurredAt:      ev.OccurredAt.UTC().Format(time.RFC3339Nano),
		}
		if err := outbox.EmitJSON(txCtx, stampedBuf, events.AggregateRoiEvent, ev.EventID, events.TopicMain, payload); err != nil {
			return fmt.Errorf("emit roi.event.recorded: %w", err)
		}
		return outboxInner.Flush(txCtx)
	})
}

func (r *SpannerRepository) ListEvents(ctx context.Context, f EventsFilter) ([]RoiEvent, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	params := map[string]any{"tid": strings.TrimSpace(f.TenantID)}
	sql := `SELECT TenantId, EventId, Category, SavingsMinor, Currency, Methodology,
	               ConfidenceBps, IsEstimate, HoldoutControl, ProposalId,
	               EvidenceRefType, EvidenceRefId, Notes, RecordedBy,
	               IdempotencyKey, OccurredAt, CreatedAt
	        FROM RoiEvents
	        WHERE TenantId = @tid`
	if cat := strings.TrimSpace(f.Category); cat != "" {
		sql += ` AND Category = @cat`
		params["cat"] = strings.ToUpper(cat)
	}
	if f.Since != nil {
		sql += ` AND OccurredAt >= @since`
		params["since"] = f.Since.UTC()
	}
	if f.Until != nil {
		sql += ` AND OccurredAt <= @until`
		params["until"] = f.Until.UTC()
	}
	limit := f.Limit
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	sql += ` ORDER BY OccurredAt DESC, EventId DESC LIMIT @lim`
	params["lim"] = int64(limit)

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []RoiEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list RoiEvents: %w", err)
		}
		ev, err := scanEventRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, nil
}

// ReadHistoricalMetrics computes 90-day baseline AR, overdue bad-debt exposure, and expired/shrink stock value.
func (r *SpannerRepository) ReadHistoricalMetrics(ctx context.Context, tenantID string, start, end time.Time) (HistoricalMetrics, error) {
	if r == nil || r.client == nil {
		return HistoricalMetrics{}, ErrRepoUnavailable
	}
	out := HistoricalMetrics{Currency: "UZS"}
	tid := strings.TrimSpace(tenantID)
	badDebtCutoff := end.Add(-30 * 24 * time.Hour)
	today := civil.DateOf(end.UTC())

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(roCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		arIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT COALESCE(SUM(PrincipalMinor), 0),
			             COALESCE(SUM(CASE WHEN (Status IN ('OPEN','PARTIAL') AND DueAt < @badCutoff) OR (Status = 'VOID' AND BalanceMinor > 0) THEN BalanceMinor ELSE 0 END), 0)
			      FROM ArInvoices
			      WHERE SupplierId = @tid AND CreatedAt >= @start`,
			Params: map[string]any{
				"tid":       tid,
				"start":     start.UTC(),
				"badCutoff": badDebtCutoff.UTC(),
			},
		})
		if row, rErr := arIter.Next(); rErr == nil {
			_ = row.Columns(&out.ArMinor, &out.BadDebtMinor)
		} else if rErr != iterator.Done {
			arIter.Stop()
			return fmt.Errorf("query baseline ArInvoices: %w", rErr)
		}
		arIter.Stop()

		var shrinkMinor int64
		movIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT COALESCE(SUM(TotalCostMinor), 0)
			      FROM InventoryMovements
			      WHERE EntityId = @tid AND MovementType = 'ADJUSTMENT_SHRINK' AND CreatedAt >= @start`,
			Params: map[string]any{"tid": tid, "start": start.UTC()},
		})
		if row, mErr := movIter.Next(); mErr == nil {
			_ = row.Columns(&shrinkMinor)
		} else if mErr != iterator.Done {
			movIter.Stop()
			return fmt.Errorf("query baseline InventoryMovements: %w", mErr)
		}
		movIter.Stop()

		var expiredLotMinor int64
		lotIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT COALESCE(SUM(sl.QuantityOnHand * COALESCE(p.PriceMinor, 0)), 0)
			      FROM StockLots sl
			      LEFT JOIN Products p ON sl.ProductId = p.ProductId
			      WHERE sl.SupplierId = @tid
			        AND (sl.Status = 'EXPIRED' OR (sl.ExpiryDate IS NOT NULL AND sl.ExpiryDate < @today AND sl.QuantityOnHand > 0))`,
			Params: map[string]any{"tid": tid, "today": today},
		})
		if row, lErr := lotIter.Next(); lErr == nil {
			_ = row.Columns(&expiredLotMinor)
		} else if lErr != iterator.Done {
			lotIter.Stop()
			return fmt.Errorf("query baseline StockLots: %w", lErr)
		}
		lotIter.Stop()

		out.ExpiredMinor = shrinkMinor + expiredLotMinor
		return nil
	})
	return out, err
}

// ReadPlatformFeesMinor computes actual platform fees from BillingMeterEvents + BillingFeeSchedules + PLATFORM AR invoices.
func (r *SpannerRepository) ReadPlatformFeesMinor(ctx context.Context, tenantID string, start, end time.Time) (int64, string, error) {
	if r == nil || r.client == nil {
		return 0, "", ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	var feesMinor int64
	currency := "UZS"

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(roCtx context.Context, ro *spanner.ReadOnlyTransaction) error {
		// 1. Check billed PLATFORM AR invoices in window.
		var billedInvoiceMinor int64
		invIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT COALESCE(SUM(PrincipalMinor), 0)
			      FROM ArInvoices
			      WHERE SupplierId = 'PLATFORM' AND RetailerId = @tid AND Status != 'VOID'
			        AND CreatedAt >= @start AND CreatedAt <= @end`,
			Params: map[string]any{"tid": tid, "start": start.UTC(), "end": end.UTC()},
		})
		if row, iErr := invIter.Next(); iErr == nil {
			_ = row.Columns(&billedInvoiceMinor)
		} else if iErr != iterator.Done {
			invIter.Stop()
			return fmt.Errorf("query platform AR invoices: %w", iErr)
		}
		invIter.Stop()

		// 2. Query BillingMeterEvents in window.
		var (
			completedGmv     int64
			refundedGmv      int64
			commissionMinor  int64
			explicitFeeMinor int64
		)
		orders := make(map[string]struct{})
		bmIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT MeterType, Amount, OrderId
			      FROM BillingMeterEvents
			      WHERE SupplierId = @tid AND ProcessedAt >= @start AND ProcessedAt <= @end`,
			Params: map[string]any{"tid": tid, "start": start.UTC(), "end": end.UTC()},
		})
		for {
			row, bErr := bmIter.Next()
			if bErr == iterator.Done {
				break
			}
			if bErr != nil {
				bmIter.Stop()
				return fmt.Errorf("query BillingMeterEvents: %w", bErr)
			}
			var mType, orderID string
			var amt int64
			if err := row.Columns(&mType, &amt, &orderID); err != nil {
				bmIter.Stop()
				return err
			}
			switch strings.ToUpper(strings.TrimSpace(mType)) {
			case "ORDER_COMPLETED", "ORDER_GMV":
				completedGmv += amt
				if strings.TrimSpace(orderID) != "" {
					orders[orderID] = struct{}{}
				}
			case "ORDER_REFUNDED":
				refundedGmv += amt
			case "COMMISSION_SETTLED":
				commissionMinor += amt
			case "PLATFORM_FEE", "SUBSCRIPTION_FEE":
				explicitFeeMinor += amt
			}
		}
		bmIter.Stop()

		// 3. Resolve fee schedule if GMV/orders exist.
		var perOrderMinor, gmvBps, subMinor int64
		schedIter := ro.Query(roCtx, spanner.Statement{
			SQL: `SELECT PerOrderMinor, GmvBps, MonthlySubscriptionMinor, Currency
			      FROM BillingFeeSchedules
			      WHERE (SupplierId = @tid OR SupplierId = '')
			        AND EffectiveFrom <= @end AND (EffectiveTo IS NULL OR EffectiveTo > @start)
			      ORDER BY CASE WHEN SupplierId = @tid THEN 0 ELSE 1 END, EffectiveFrom DESC
			      LIMIT 1`,
			Params: map[string]any{"tid": tid, "start": start.UTC(), "end": end.UTC()},
		})
		if row, sErr := schedIter.Next(); sErr == nil {
			var cur spanner.NullString
			if err := row.Columns(&perOrderMinor, &gmvBps, &subMinor, &cur); err == nil && strings.TrimSpace(cur.StringVal) != "" {
				currency = strings.ToUpper(strings.TrimSpace(cur.StringVal))
			}
		} else if sErr != iterator.Done {
			schedIter.Stop()
			return fmt.Errorf("query BillingFeeSchedules: %w", sErr)
		}
		schedIter.Stop()

		netGmv := completedGmv - refundedGmv
		if netGmv < 0 {
			netGmv = 0
		}
		orderCount := int64(len(orders))
		meteredScheduleFee := perOrderMinor*orderCount + (netGmv*gmvBps)/10000
		if subMinor > 0 && (orderCount > 0 || netGmv > 0) {
			months := int64(end.Sub(start) / (30 * 24 * time.Hour))
			if months < 1 {
				months = 1
			}
			meteredScheduleFee += subMinor * months
		}

		// Use the larger of invoiced platform fee or metered calculation + explicit fee events without double-counting.
		calculated := meteredScheduleFee + commissionMinor + explicitFeeMinor
		if billedInvoiceMinor > calculated {
			feesMinor = billedInvoiceMinor
		} else {
			feesMinor = calculated
		}
		return nil
	})
	return feesMinor, currency, err
}

func scanBaselineRow(row *spanner.Row) (Baseline, error) {
	var (
		b           Baseline
		surveyNotes spanner.NullString
		surveyJSON  spanner.NullString
		periodStart spanner.NullTime
		periodEnd   spanner.NullTime
	)
	if err := row.Columns(
		&b.TenantID, &b.BaselineID, &b.WindowDays, &b.BaselineArMinor, &b.BaselineBadDebtMinor,
		&b.BaselineExpiredMinor, &b.BaselineStockoutMinor, &b.BaselineLogisticsMinor,
		&b.BaselineLabourHours, &b.SurveyHourlyWageMinor, &surveyNotes, &surveyJSON,
		&b.Currency, &b.CapturedBy, &periodStart, &periodEnd, &b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return Baseline{}, fmt.Errorf("scan RoiBaselines row: %w", err)
	}
	b.SurveyNotes = surveyNotes.StringVal
	b.SurveyJSON = surveyJSON.StringVal
	if periodStart.Valid {
		b.PeriodStart = periodStart.Time.UTC()
	}
	if periodEnd.Valid {
		b.PeriodEnd = periodEnd.Time.UTC()
	}
	b.CreatedAt = b.CreatedAt.UTC()
	b.UpdatedAt = b.UpdatedAt.UTC()
	if b.SurveyJSON != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(b.SurveyJSON), &m); err == nil {
			b.Survey = m
		}
	}
	return b, nil
}

func scanEventRow(row *spanner.Row) (RoiEvent, error) {
	var (
		ev      RoiEvent
		propID  spanner.NullString
		refType spanner.NullString
		refID   spanner.NullString
		notes   spanner.NullString
		idemKey spanner.NullString
	)
	if err := row.Columns(
		&ev.TenantID, &ev.EventID, &ev.Category, &ev.SavingsMinor, &ev.Currency, &ev.Methodology,
		&ev.ConfidenceBps, &ev.IsEstimate, &ev.HoldoutControl, &propID,
		&refType, &refID, &notes, &ev.RecordedBy,
		&idemKey, &ev.OccurredAt, &ev.CreatedAt,
	); err != nil {
		return RoiEvent{}, fmt.Errorf("scan RoiEvents row: %w", err)
	}
	ev.ProposalID = propID.StringVal
	ev.EvidenceRefType = refType.StringVal
	ev.EvidenceRefID = refID.StringVal
	ev.Notes = notes.StringVal
	ev.IdempotencyKey = idemKey.StringVal
	ev.OccurredAt = ev.OccurredAt.UTC()
	ev.CreatedAt = ev.CreatedAt.UTC()
	return ev, nil
}

func nullIfEmpty(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}
