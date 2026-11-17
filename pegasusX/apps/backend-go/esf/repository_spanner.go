package esf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SpannerRepository implements Repository over Cloud Spanner using strong reads
// and atomic ReadWriteTransaction mutations across EsfDocuments, EsfStatusHistory,
// GlEntries/GlBalances, and OutboxEvents.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed ESF Repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

func (r *SpannerRepository) LoadOrderFiscalSource(ctx context.Context, entityID, orderID string) (OrderFiscalSource, error) {
	if r == nil || r.client == nil {
		return OrderFiscalSource{}, ErrRepoUnavailable
	}
	oid := strings.TrimSpace(orderID)
	if oid == "" {
		return OrderFiscalSource{}, ErrInvalidInput
	}

	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	var src OrderFiscalSource
	orderIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT o.OrderId, o.SupplierId, o.RetailerId, o.TotalMinor, o.Currency, o.CreatedAt,
		             COALESCE(sp.TaxId, ''),
		             COALESCE(NULLIF(rt.TaxId, ''), NULLIF(srl.TaxIdTIN, ''), ''),
		             COALESCE(NULLIF(srl.LegalName, ''), NULLIF(rt.Name, ''), '')
		      FROM Orders o
		      LEFT JOIN SupplierProfiles sp ON sp.SupplierId = o.SupplierId
		      LEFT JOIN Retailers rt ON rt.RetailerId = o.RetailerId
		      LEFT JOIN SupplierRetailerLinks srl ON srl.SupplierId = o.SupplierId AND srl.RetailerId = o.RetailerId
		      WHERE o.OrderId = @orderId`,
		Params: map[string]any{"orderId": oid},
	})
	defer orderIter.Stop()

	orderRow, err := orderIter.Next()
	if errors.Is(err, iterator.Done) {
		return OrderFiscalSource{}, ErrOrderNotFound
	}
	if err != nil {
		return OrderFiscalSource{}, fmt.Errorf("query order fiscal header: %w", err)
	}
	if err := orderRow.Columns(
		&src.OrderID, &src.SupplierID, &src.RetailerID,
		&src.TotalMinor, &src.Currency, &src.CreatedAt,
		&src.SupplierTIN, &src.CounterpartyTIN, &src.CounterpartyName,
	); err != nil {
		return OrderFiscalSource{}, fmt.Errorf("scan order fiscal header: %w", err)
	}

	if entityID != "" && src.SupplierID != "" && src.SupplierID != entityID {
		return OrderFiscalSource{}, ErrOrderForbidden
	}
	if src.Currency == "" {
		src.Currency = "UZS"
	}

	// Primary tax source: immutable OrderLineFiscalSnapshots joined with TaxRegimeVersions.
	snapIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT s.OrderLineId, s.RegimeId, s.VatRateBps, s.NetMinor, s.VatMinor, s.GrossMinor,
		             COALESCE(r.Currency, '')
		      FROM OrderLineFiscalSnapshots s
		      LEFT JOIN TaxRegimeVersions r ON r.Id = s.RegimeId
		      WHERE s.OrderId = @orderId
		      ORDER BY s.OrderLineId ASC`,
		Params: map[string]any{"orderId": oid},
	})
	defer snapIter.Stop()

	vatTotalsByRate := make(map[int64]int64)
	for {
		row, err := snapIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return OrderFiscalSource{}, fmt.Errorf("query OrderLineFiscalSnapshots: %w", err)
		}
		var line EsfCanonicalLine
		var regCur string
		if err := row.Columns(&line.OrderLineID, &line.RegimeID, &line.VatRateBps, &line.NetMinor, &line.VatMinor, &line.GrossMinor, &regCur); err != nil {
			return OrderFiscalSource{}, fmt.Errorf("scan OrderLineFiscalSnapshots: %w", err)
		}
		if src.Currency == "" && regCur != "" {
			src.Currency = regCur
		}
		src.Lines = append(src.Lines, line)
		src.TotalNetMinor += line.NetMinor
		src.TotalVatMinor += line.VatMinor
		src.TotalGrossMinor += line.GrossMinor
		vatTotalsByRate[line.VatRateBps] += line.GrossMinor
	}

	if len(src.Lines) > 0 {
		var bestBps int64
		var bestGross int64 = -1
		for bps, gross := range vatTotalsByRate {
			if gross > bestGross || (gross == bestGross && bps > bestBps) {
				bestGross = gross
				bestBps = bps
			}
		}
		src.DominantVatBps = bestBps
		return src, nil
	}

	// Fallback when an order does not yet have OrderLineFiscalSnapshots: resolve active TaxRegimeVersions.
	var regimeID string
	var defaultBps int64 = 1200 // Standard UZ VAT 12% fallback if no TaxRegimeVersions row seeded.
	asOf := src.CreatedAt
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	regIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT Id, VatRatesBps, Currency
		      FROM TaxRegimeVersions
		      WHERE CountryCode = 'UZ'
		        AND EffectiveFrom <= @asOf
		        AND (EffectiveTo IS NULL OR EffectiveTo > @asOf)
		      ORDER BY EffectiveFrom DESC
		      LIMIT 1`,
		Params: map[string]any{"asOf": asOf},
	})
	defer regIter.Stop()

	if regRow, rErr := regIter.Next(); rErr == nil {
		var rates []int64
		var regCur string
		if cErr := regRow.Columns(&regimeID, &rates, &regCur); cErr == nil {
			if src.Currency == "" && regCur != "" {
				src.Currency = regCur
			}
			for _, bps := range rates {
				if bps > defaultBps || (defaultBps == 1200 && len(rates) > 0 && bps > 0) {
					defaultBps = bps
				}
			}
			if len(rates) > 0 {
				// Prefer highest non-zero statutory rate, or rates[0] when explicitly 0%.
				chosen := rates[0]
				for _, bps := range rates {
					if bps > chosen {
						chosen = bps
					}
				}
				defaultBps = chosen
			}
		}
	} else if !errors.Is(rErr, iterator.Done) {
		return OrderFiscalSource{}, fmt.Errorf("query TaxRegimeVersions: %w", rErr)
	}

	var vatMinor int64
	if defaultBps > 0 && src.TotalMinor > 0 {
		denom := int64(10000) + defaultBps
		vatMinor = (src.TotalMinor*defaultBps + denom/2) / denom
	}
	netMinor := src.TotalMinor - vatMinor
	src.TotalNetMinor = netMinor
	src.TotalVatMinor = vatMinor
	src.TotalGrossMinor = src.TotalMinor
	src.DominantVatBps = defaultBps
	src.Lines = []EsfCanonicalLine{{
		OrderLineID: "order-total",
		RegimeID:    regimeID,
		VatRateBps:  defaultBps,
		NetMinor:    netMinor,
		VatMinor:    vatMinor,
		GrossMinor:  src.TotalMinor,
	}}
	return src, nil
}

func (r *SpannerRepository) SaveDocumentWithHistory(
	ctx context.Context,
	doc EsfDocument,
	hist EsfStatusHistoryEntry,
	eventType string,
	glVatReversal *VATReversalSpec,
) error {
	if r == nil || r.client == nil {
		return ErrRepoUnavailable
	}

	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if _, err := txn.ReadRow(txCtx, "EsfDocuments", spanner.Key{doc.EntityID, doc.EsfID}, []string{"EsfId"}); err == nil {
			return ErrAlreadyExists
		} else if status.Code(err) != codes.NotFound {
			return fmt.Errorf("check EsfDocuments PK: %w", err)
		}

		var issuedAtVal any = spanner.NullTime{}
		if doc.IssuedAt != nil && !doc.IssuedAt.IsZero() {
			issuedAtVal = doc.IssuedAt.UTC()
		}
		var acceptedAtVal any = spanner.NullTime{}
		if doc.AcceptedAt != nil && !doc.AcceptedAt.IsZero() {
			acceptedAtVal = doc.AcceptedAt.UTC()
		}
		createdAt := doc.CreatedAt.UTC()
		if createdAt.IsZero() {
			createdAt = time.Now().UTC()
		}
		updatedAt := doc.UpdatedAt.UTC()
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}

		docMut := spanner.InsertMap("EsfDocuments", map[string]any{
			"EntityId":         doc.EntityID,
			"EsfId":            doc.EsfID,
			"Direction":        doc.Direction,
			"OrderId":          nullIfEmpty(doc.OrderID),
			"PoId":             nullIfEmpty(doc.PoID),
			"CounterpartyTin":  doc.CounterpartyTIN,
			"CounterpartyName": nullIfEmpty(doc.CounterpartyName),
			"TotalMinor":       doc.TotalMinor,
			"VatMinor":         doc.VatMinor,
			"VatRateBps":       doc.VatRateBps,
			"Status":           doc.Status,
			"Operator":         doc.Operator,
			"OperatorRef":      nullIfEmpty(doc.OperatorRef),
			"SignedPayloadUri": nullIfEmpty(doc.SignedPayloadURI),
			"PayloadHash":      nullIfEmpty(doc.PayloadHash),
			"FailureReason":    nullIfEmpty(doc.FailureReason),
			"PeriodId":         doc.PeriodID,
			"IssuedAt":         issuedAtVal,
			"AcceptedAt":       acceptedAtVal,
			"CreatedAt":        createdAt,
			"UpdatedAt":        updatedAt,
		})

		histCreatedAt := hist.CreatedAt.UTC()
		if histCreatedAt.IsZero() {
			histCreatedAt = updatedAt
		}
		histMut := spanner.InsertMap("EsfStatusHistory", map[string]any{
			"EntityId":            hist.EntityID,
			"EsfId":               hist.EsfID,
			"HistoryId":           hist.HistoryID,
			"FromStatus":          nullIfEmpty(hist.FromStatus),
			"ToStatus":            hist.ToStatus,
			"ActorId":             hist.ActorID,
			"Reason":              nullIfEmpty(hist.Reason),
			"OperatorPayloadJson": nullIfEmpty(hist.OperatorPayloadJSON),
			"CreatedAt":           histCreatedAt,
		})

		buf := outbox.NewSpannerTxnBuffer(txn)
		if glVatReversal != nil && glVatReversal.VatMinor > 0 {
			if err := bufferESFVATReversalInTxn(txCtx, txn, buf, glVatReversal); err != nil {
				return err
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{docMut, histMut}); err != nil {
			return fmt.Errorf("buffer EsfDocument and EsfStatusHistory: %w", err)
		}

		if eventType != "" {
			if err := emitESFEventInTxn(txCtx, buf, doc, eventType, hist.ActorID, hist.Reason, histCreatedAt); err != nil {
				return err
			}
		}

		return buf.Flush(txCtx)
	})
}

func (r *SpannerRepository) UpdateDocumentStatusWithHistory(
	ctx context.Context,
	entityID, esfID, expectedPrevStatus, nextStatus string,
	updates StatusUpdateFields,
	hist EsfStatusHistoryEntry,
	eventType string,
	glVatReversal *VATReversalSpec,
) (EsfDocument, error) {
	if r == nil || r.client == nil {
		return EsfDocument{}, ErrRepoUnavailable
	}

	var updated EsfDocument
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "EsfDocuments", spanner.Key{entityID, esfID}, esfDocumentColumns())
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return fmt.Errorf("read EsfDocument for transition: %w", err)
		}
		doc, err := scanEsfDocument(row)
		if err != nil {
			return err
		}
		if expectedPrevStatus != "" && doc.Status != expectedPrevStatus {
			return fmt.Errorf("%w: status is %s, expected %s", ErrInvalidTransition, doc.Status, expectedPrevStatus)
		}

		updatedAt := updates.UpdatedAt.UTC()
		if updatedAt.IsZero() {
			updatedAt = time.Now().UTC()
		}

		updateFields := map[string]any{
			"EntityId":  entityID,
			"EsfId":     esfID,
			"Status":    nextStatus,
			"UpdatedAt": updatedAt,
		}
		doc.Status = nextStatus
		doc.UpdatedAt = updatedAt

		if updates.Operator != "" {
			doc.Operator = updates.Operator
			updateFields["Operator"] = updates.Operator
		}
		if updates.OperatorRef != "" {
			doc.OperatorRef = updates.OperatorRef
			updateFields["OperatorRef"] = updates.OperatorRef
		}
		if updates.SignedPayloadURI != "" {
			doc.SignedPayloadURI = updates.SignedPayloadURI
			updateFields["SignedPayloadUri"] = updates.SignedPayloadURI
		}
		if updates.PayloadHash != "" {
			doc.PayloadHash = updates.PayloadHash
			updateFields["PayloadHash"] = updates.PayloadHash
		}
		if updates.FailureReason != nil {
			doc.FailureReason = *updates.FailureReason
			updateFields["FailureReason"] = nullIfEmpty(*updates.FailureReason)
		}
		if updates.IssuedAt != nil {
			ts := updates.IssuedAt.UTC()
			doc.IssuedAt = &ts
			updateFields["IssuedAt"] = ts
		}
		if updates.AcceptedAt != nil {
			ts := updates.AcceptedAt.UTC()
			doc.AcceptedAt = &ts
			updateFields["AcceptedAt"] = ts
		}

		histCreatedAt := hist.CreatedAt.UTC()
		if histCreatedAt.IsZero() {
			histCreatedAt = updatedAt
		}
		histMut := spanner.InsertMap("EsfStatusHistory", map[string]any{
			"EntityId":            hist.EntityID,
			"EsfId":               hist.EsfID,
			"HistoryId":           hist.HistoryID,
			"FromStatus":          nullIfEmpty(hist.FromStatus),
			"ToStatus":            hist.ToStatus,
			"ActorId":             hist.ActorID,
			"Reason":              nullIfEmpty(hist.Reason),
			"OperatorPayloadJson": nullIfEmpty(hist.OperatorPayloadJSON),
			"CreatedAt":           histCreatedAt,
		})

		buf := outbox.NewSpannerTxnBuffer(txn)
		if glVatReversal != nil && glVatReversal.VatMinor > 0 {
			if err := bufferESFVATReversalInTxn(txCtx, txn, buf, glVatReversal); err != nil {
				return err
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("EsfDocuments", updateFields), histMut}); err != nil {
			return fmt.Errorf("buffer EsfDocument transition: %w", err)
		}

		if eventType != "" {
			if err := emitESFEventInTxn(txCtx, buf, doc, eventType, hist.ActorID, hist.Reason, histCreatedAt); err != nil {
				return err
			}
		}

		if err := buf.Flush(txCtx); err != nil {
			return err
		}
		updated = doc
		return nil
	})
	if err != nil {
		return EsfDocument{}, err
	}
	return updated, nil
}

func bufferESFVATReversalInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	buf outbox.TxnBuffer,
	spec *VATReversalSpec,
) error {
	if spec == nil || spec.VatMinor <= 0 {
		return nil
	}
	cur := strings.TrimSpace(spec.Currency)
	if cur == "" {
		cur = "UZS"
	}
	creditAcct := gl.AccountCreditNoteAdjustment
	var partyType, partyID string
	if strings.TrimSpace(spec.RetailerID) != "" {
		creditAcct = gl.AccountAccountsReceivable
		partyType = "RETAILER"
		partyID = strings.TrimSpace(spec.RetailerID)
	}
	occurredAt := spec.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	res, err := gl.BufferJournalEntryInTxn(ctx, txn, buf, gl.JournalEntryInput{
		EntityID:       spec.EntityID,
		SourceType:     "ESF_REJECTED_VAT_REVERSAL",
		SourceID:       spec.EsfID,
		Memo:           fmt.Sprintf("ESF %s rejected VAT reversal (order=%s)", spec.EsfID, spec.OrderID),
		Currency:       cur,
		CountryPack:    "UZ",
		IdempotencyKey: spec.IdempotencyKey,
		PostedBy:       spec.ActorID,
		OccurredAt:     occurredAt,
		Postings: []gl.PostingInput{
			{
				AccountCode: gl.AccountVatOutput,
				Direction:   gl.DirectionDebit,
				AmountMinor: spec.VatMinor,
				Currency:    cur,
				Description: fmt.Sprintf("Reverse output VAT on rejected ESF %s (order=%s)", spec.EsfID, spec.OrderID),
			},
			{
				AccountCode: creditAcct,
				Direction:   gl.DirectionCredit,
				AmountMinor: spec.VatMinor,
				Currency:    cur,
				PartyType:   partyType,
				PartyID:     partyID,
				Description: fmt.Sprintf("Offset rejected ESF VAT %s (order=%s)", spec.EsfID, spec.OrderID),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("post ESF rejection GL VAT reversal: %w", err)
	}
	if spec.GlEntryIDOut != nil {
		*spec.GlEntryIDOut = res.EntryID
	}
	return nil
}

func emitESFEventInTxn(
	ctx context.Context,
	buf outbox.TxnBuffer,
	doc EsfDocument,
	eventType, actorID, reason string,
	ts time.Time,
) error {
	tsStr := ts.UTC().Format(time.RFC3339Nano)
	stamped := &esfEntityOutboxBuf{entityID: doc.EntityID, inner: buf}
	return outbox.EmitJSON(ctx, stamped, events.AggregateEsfDocument, doc.EsfID, events.TopicLedger, events.EsfDocumentEvent{
		BaseEvent: events.BaseEvent{
			Type:      eventType,
			Version:   1,
			Timestamp: tsStr,
		},
		EntityID:        doc.EntityID,
		EsfID:           doc.EsfID,
		Direction:       doc.Direction,
		OrderID:         doc.OrderID,
		PoID:            doc.PoID,
		CounterpartyTIN: doc.CounterpartyTIN,
		TotalMinor:      doc.TotalMinor,
		VatMinor:        doc.VatMinor,
		VatRateBps:      doc.VatRateBps,
		Status:          doc.Status,
		Operator:        doc.Operator,
		OperatorRef:     doc.OperatorRef,
		PeriodID:        doc.PeriodID,
		PayloadHash:     doc.PayloadHash,
		ActorID:         actorID,
		Reason:          reason,
		OccurredAt:      tsStr,
	})
}

func (r *SpannerRepository) GetDocument(ctx context.Context, entityID, esfID string) (EsfDocument, []EsfStatusHistoryEntry, error) {
	if r == nil || r.client == nil {
		return EsfDocument{}, nil, ErrRepoUnavailable
	}

	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	row, err := roTxn.ReadRow(ctx, "EsfDocuments", spanner.Key{entityID, esfID}, esfDocumentColumns())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return EsfDocument{}, nil, ErrNotFound
		}
		return EsfDocument{}, nil, fmt.Errorf("read EsfDocument: %w", err)
	}
	doc, err := scanEsfDocument(row)
	if err != nil {
		return EsfDocument{}, nil, err
	}

	histIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT EntityId, EsfId, HistoryId, FromStatus, ToStatus, ActorId, Reason, OperatorPayloadJson, CreatedAt
		      FROM EsfStatusHistory
		      WHERE EntityId = @entityId AND EsfId = @esfId
		      ORDER BY CreatedAt ASC`,
		Params: map[string]any{"entityId": entityID, "esfId": esfID},
	})
	defer histIter.Stop()

	var hist []EsfStatusHistoryEntry
	for {
		hRow, err := histIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return EsfDocument{}, nil, fmt.Errorf("query EsfStatusHistory: %w", err)
		}
		var entry EsfStatusHistoryEntry
		var fromStatus, reason, opJSON spanner.NullString
		if err := hRow.Columns(&entry.EntityID, &entry.EsfID, &entry.HistoryID, &fromStatus, &entry.ToStatus, &entry.ActorID, &reason, &opJSON, &entry.CreatedAt); err != nil {
			return EsfDocument{}, nil, fmt.Errorf("scan EsfStatusHistory: %w", err)
		}
		if fromStatus.Valid {
			entry.FromStatus = fromStatus.StringVal
		}
		if reason.Valid {
			entry.Reason = reason.StringVal
		}
		if opJSON.Valid {
			entry.OperatorPayloadJSON = opJSON.StringVal
		}
		hist = append(hist, entry)
	}
	if hist == nil {
		hist = []EsfStatusHistoryEntry{}
	}
	return doc, hist, nil
}

func (r *SpannerRepository) FindLatestByOrderOrOperatorRef(ctx context.Context, entityID, orderID, operatorRef string) (*EsfDocument, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}

	var conds []string
	params := map[string]any{}
	if strings.TrimSpace(orderID) != "" {
		conds = append(conds, "OrderId = @orderId")
		params["orderId"] = strings.TrimSpace(orderID)
	}
	if strings.TrimSpace(operatorRef) != "" {
		conds = append(conds, "OperatorRef = @operatorRef")
		params["operatorRef"] = strings.TrimSpace(operatorRef)
	}
	if len(conds) == 0 {
		return nil, ErrInvalidInput
	}

	where := "(" + strings.Join(conds, " OR ") + ")"
	if strings.TrimSpace(entityID) != "" {
		where = "EntityId = @entityId AND " + where
		params["entityId"] = strings.TrimSpace(entityID)
	}

	sql := fmt.Sprintf(
		"SELECT %s FROM EsfDocuments WHERE %s ORDER BY UpdatedAt DESC LIMIT 1",
		strings.Join(esfDocumentColumns(), ", "), where,
	)
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find ESF document: %w", err)
	}
	doc, err := scanEsfDocument(row)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *SpannerRepository) ListDocuments(ctx context.Context, filter ListFilter) ([]EsfDocument, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	where := []string{"EntityId = @entityId"}
	params := map[string]any{
		"entityId": strings.TrimSpace(filter.EntityID),
		"limit":    int64(limit),
	}
	if d := strings.ToUpper(strings.TrimSpace(filter.Direction)); d != "" {
		where = append(where, "Direction = @direction")
		params["direction"] = d
	}
	if st := strings.ToUpper(strings.TrimSpace(filter.Status)); st != "" {
		where = append(where, "Status = @status")
		params["status"] = st
	}
	if p := strings.TrimSpace(filter.PeriodID); p != "" {
		where = append(where, "PeriodId = @periodId")
		params["periodId"] = p
	}
	if o := strings.TrimSpace(filter.OrderID); o != "" {
		where = append(where, "OrderId = @orderId")
		params["orderId"] = o
	}

	sql := fmt.Sprintf(
		"SELECT %s FROM EsfDocuments WHERE %s ORDER BY CreatedAt DESC LIMIT @limit",
		strings.Join(esfDocumentColumns(), ", "),
		strings.Join(where, " AND "),
	)
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	out := make([]EsfDocument, 0)
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list EsfDocuments: %w", err)
		}
		doc, err := scanEsfDocument(row)
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func (r *SpannerRepository) GetVATSummary(ctx context.Context, entityID, periodID string) (VATSummaryReport, error) {
	if r == nil || r.client == nil {
		return VATSummaryReport{}, ErrRepoUnavailable
	}

	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	report := VATSummaryReport{
		EntityID: strings.TrimSpace(entityID),
		PeriodID: strings.TrimSpace(periodID),
		Currency: "UZS",
		ByRate:   []VATRateBreakdown{},
	}
	byRate := make(map[int64]*VATRateBreakdown)
	ensureRate := func(bps int64) *VATRateBreakdown {
		if b, ok := byRate[bps]; ok {
			return b
		}
		b := &VATRateBreakdown{VatRateBps: bps}
		byRate[bps] = b
		return b
	}

	esfIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT Direction, Status, TotalMinor, VatMinor, VatRateBps
		      FROM EsfDocuments
		      WHERE EntityId = @entityId AND PeriodId = @periodId`,
		Params: map[string]any{"entityId": report.EntityID, "periodId": report.PeriodID},
	})
	defer esfIter.Stop()

	for {
		row, err := esfIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return VATSummaryReport{}, fmt.Errorf("query EsfDocuments for VAT summary: %w", err)
		}
		var dir, st string
		var totalMinor, vatMinor, rateBps int64
		if err := row.Columns(&dir, &st, &totalMinor, &vatMinor, &rateBps); err != nil {
			return VATSummaryReport{}, fmt.Errorf("scan EsfDocuments VAT summary: %w", err)
		}
		if st == StatusRejected {
			report.RejectedCount++
			continue
		}
		if st != StatusSubmitted && st != StatusAccepted {
			continue
		}
		baseMinor := totalMinor - vatMinor
		if baseMinor < 0 {
			baseMinor = 0
		}
		rb := ensureRate(rateBps)
		rb.DocCount++
		if dir == DirectionOutbound {
			report.OutboundCount++
			report.EsfOutputVatMinor += vatMinor
			rb.OutboundBaseMinor += baseMinor
			rb.OutboundVatMinor += vatMinor
		} else if dir == DirectionInbound {
			report.InboundCount++
			report.EsfInputVatMinor += vatMinor
			rb.InboundBaseMinor += baseMinor
			rb.InboundVatMinor += vatMinor
		}
	}

	glIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT AccountCode, COALESCE(SUM(DebitMinor), 0), COALESCE(SUM(CreditMinor), 0), COALESCE(ANY_VALUE(Currency), 'UZS')
		      FROM GlBalances
		      WHERE EntityId = @entityId AND PeriodId = @periodId AND AccountCode IN (@outAcct, @inAcct)
		      GROUP BY AccountCode`,
		Params: map[string]any{
			"entityId": report.EntityID,
			"periodId": report.PeriodID,
			"outAcct":  gl.AccountVatOutput,
			"inAcct":   gl.AccountVatInput,
		},
	})
	defer glIter.Stop()

	for {
		row, err := glIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return VATSummaryReport{}, fmt.Errorf("query GlBalances for VAT summary: %w", err)
		}
		var acct, cur string
		var dr, cr int64
		if err := row.Columns(&acct, &dr, &cr, &cur); err != nil {
			return VATSummaryReport{}, fmt.Errorf("scan GlBalances VAT summary: %w", err)
		}
		if cur != "" {
			report.Currency = cur
		}
		switch acct {
		case gl.AccountVatOutput:
			// TAX:VAT:OUTPUT is a liability (normal credit): net credit balance = output VAT payable.
			report.GlOutputVatMinor += cr - dr
		case gl.AccountVatInput:
			// TAX:VAT:INPUT is an asset (normal debit): net debit balance = input VAT receivable.
			report.GlInputVatMinor += dr - cr
		}
	}

	report.EsfNetVatMinor = report.EsfOutputVatMinor - report.EsfInputVatMinor
	report.GlNetVatMinor = report.GlOutputVatMinor - report.GlInputVatMinor
	report.OutputDiffMinor = report.EsfOutputVatMinor - report.GlOutputVatMinor
	report.InputDiffMinor = report.EsfInputVatMinor - report.GlInputVatMinor
	report.DiffMinor = report.EsfNetVatMinor - report.GlNetVatMinor
	report.Reconciled = report.OutputDiffMinor == 0 && report.InputDiffMinor == 0 && report.DiffMinor == 0

	var rates []int64
	for bps := range byRate {
		rates = append(rates, bps)
	}
	sort.Slice(rates, func(i, j int) bool { return rates[i] < rates[j] })
	for _, bps := range rates {
		report.ByRate = append(report.ByRate, *byRate[bps])
	}
	return report, nil
}

func (r *SpannerRepository) ListMissingIKPU(ctx context.Context, supplierID string) ([]MissingIKPUItem, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}

	roTxn := r.client.ReadOnlyTransaction()
	defer roTxn.Close()

	// Check whether Products carries direct IKPU/Package/Marking columns in this deployment.
	colPresent := map[string]bool{}
	colIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT COLUMN_NAME
		      FROM INFORMATION_SCHEMA.COLUMNS
		      WHERE TABLE_NAME = 'Products'
		        AND COLUMN_NAME IN ('IkpuCode', 'PackageCode', 'MarkingCode')`,
	})
	defer colIter.Stop()
	for {
		cRow, err := colIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			break
		}
		var colName string
		if cRow.Column(0, &colName) == nil {
			colPresent[colName] = true
		}
	}

	selectCols := []string{"ProductId", "Name", "CategoryId", "COALESCE(Barcode, '')"}
	if colPresent["IkpuCode"] {
		selectCols = append(selectCols, "COALESCE(IkpuCode, '')")
	}
	if colPresent["PackageCode"] {
		selectCols = append(selectCols, "COALESCE(PackageCode, '')")
	}
	if colPresent["MarkingCode"] {
		selectCols = append(selectCols, "COALESCE(MarkingCode, '')")
	}

	type skuRecord struct {
		item MissingIKPUItem
	}
	records := make(map[string]*skuRecord)

	prodSQL := fmt.Sprintf(
		"SELECT %s FROM Products WHERE SupplierId = @supplierId AND IsActive = TRUE ORDER BY Name ASC",
		strings.Join(selectCols, ", "),
	)
	pIter := roTxn.Query(ctx, spanner.Statement{
		SQL:    prodSQL,
		Params: map[string]any{"supplierId": strings.TrimSpace(supplierID)},
	})
	defer pIter.Stop()

	for {
		pRow, err := pIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query Products for missing IKPU: %w", err)
		}
		var pid, name, catID, barcode, ikpu, pkg, marking string
		ptrs := []any{&pid, &name, &catID, &barcode}
		if colPresent["IkpuCode"] {
			ptrs = append(ptrs, &ikpu)
		}
		if colPresent["PackageCode"] {
			ptrs = append(ptrs, &pkg)
		}
		if colPresent["MarkingCode"] {
			ptrs = append(ptrs, &marking)
		}
		if err := pRow.Columns(ptrs...); err != nil {
			return nil, fmt.Errorf("scan Products for missing IKPU: %w", err)
		}
		records[pid] = &skuRecord{
			item: MissingIKPUItem{
				ProductID:   pid,
				Name:        name,
				CategoryID:  catID,
				Barcode:     barcode,
				IKPUCode:    ikpu,
				PackageCode: pkg,
				MarkingCode: marking,
			},
		}
	}

	// Enrich/audit from OntologyObjects(ObjectType='SKU') where ikpu_code/package_code live.
	skuIter := roTxn.Query(ctx, spanner.Statement{
		SQL: `SELECT ObjectId, PropertiesJson
		      FROM OntologyObjects
		      WHERE SupplierId = @supplierId AND ObjectType = 'SKU'`,
		Params: map[string]any{"supplierId": strings.TrimSpace(supplierID)},
	})
	defer skuIter.Stop()

	for {
		sRow, err := skuIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query OntologyObjects SKU compliance: %w", err)
		}
		var objID, propsJSON string
		if err := sRow.Columns(&objID, &propsJSON); err != nil {
			return nil, fmt.Errorf("scan OntologyObjects SKU compliance: %w", err)
		}
		var props struct {
			ProductID   string `json:"product_id"`
			Name        string `json:"name"`
			CategoryID  string `json:"category_id"`
			Barcode     string `json:"barcode"`
			IKPUCode    string `json:"ikpu_code"`
			PackageCode string `json:"package_code"`
			MarkingCode string `json:"marking_code"`
			IsActive    *bool  `json:"is_active"`
		}
		if uErr := json.Unmarshal([]byte(propsJSON), &props); uErr != nil {
			continue
		}
		if props.IsActive != nil && !*props.IsActive {
			continue
		}
		pid := strings.TrimSpace(props.ProductID)
		if pid == "" {
			pid = strings.TrimSpace(objID)
		}
		rec, exists := records[pid]
		if !exists {
			// If Products table doesn't carry this SKU (or SKU exists only in ontology), track it directly.
			if props.Name == "" && props.IKPUCode == "" && props.Barcode == "" {
				continue
			}
			rec = &skuRecord{
				item: MissingIKPUItem{
					ProductID:  pid,
					Name:       props.Name,
					CategoryID: props.CategoryID,
				},
			}
			records[pid] = rec
		}
		if rec.item.Barcode == "" && props.Barcode != "" {
			rec.item.Barcode = props.Barcode
		}
		if rec.item.IKPUCode == "" && props.IKPUCode != "" {
			rec.item.IKPUCode = props.IKPUCode
		}
		if rec.item.PackageCode == "" && props.PackageCode != "" {
			rec.item.PackageCode = props.PackageCode
		}
		if rec.item.MarkingCode == "" && props.MarkingCode != "" {
			rec.item.MarkingCode = props.MarkingCode
		}
	}

	out := make([]MissingIKPUItem, 0)
	for _, rec := range records {
		missing := detectMissingComplianceFields(rec.item.IKPUCode, rec.item.PackageCode, rec.item.MarkingCode, rec.item.Barcode)
		if len(missing) == 0 {
			continue
		}
		rec.item.MissingFields = missing
		out = append(out, rec.item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].ProductID < out[j].ProductID
	})
	return out, nil
}

func esfDocumentColumns() []string {
	return []string{
		"EntityId", "EsfId", "Direction", "OrderId", "PoId",
		"CounterpartyTin", "CounterpartyName", "TotalMinor", "VatMinor", "VatRateBps",
		"Status", "Operator", "OperatorRef", "SignedPayloadUri", "PayloadHash",
		"FailureReason", "PeriodId", "IssuedAt", "AcceptedAt", "CreatedAt", "UpdatedAt",
	}
}

func scanEsfDocument(row *spanner.Row) (EsfDocument, error) {
	var doc EsfDocument
	var orderID, poID, cpName, opRef, signedURI, payloadHash, failReason spanner.NullString
	var issuedAt, acceptedAt spanner.NullTime
	if err := row.Columns(
		&doc.EntityID, &doc.EsfID, &doc.Direction, &orderID, &poID,
		&doc.CounterpartyTIN, &cpName, &doc.TotalMinor, &doc.VatMinor, &doc.VatRateBps,
		&doc.Status, &doc.Operator, &opRef, &signedURI, &payloadHash,
		&failReason, &doc.PeriodID, &issuedAt, &acceptedAt, &doc.CreatedAt, &doc.UpdatedAt,
	); err != nil {
		return EsfDocument{}, fmt.Errorf("scan EsfDocument: %w", err)
	}
	if orderID.Valid {
		doc.OrderID = orderID.StringVal
	}
	if poID.Valid {
		doc.PoID = poID.StringVal
	}
	if cpName.Valid {
		doc.CounterpartyName = cpName.StringVal
	}
	if opRef.Valid {
		doc.OperatorRef = opRef.StringVal
	}
	if signedURI.Valid {
		doc.SignedPayloadURI = signedURI.StringVal
	}
	if payloadHash.Valid {
		doc.PayloadHash = payloadHash.StringVal
	}
	if failReason.Valid {
		doc.FailureReason = failReason.StringVal
	}
	if issuedAt.Valid {
		ts := issuedAt.Time.UTC()
		doc.IssuedAt = &ts
	}
	if acceptedAt.Valid {
		ts := acceptedAt.Time.UTC()
		doc.AcceptedAt = &ts
	}
	return doc, nil
}

func nullIfEmpty(s string) spanner.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}

type esfEntityOutboxBuf struct {
	inner    outbox.TxnBuffer
	entityID string
}

func (b *esfEntityOutboxBuf) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(b.entityID) != "" {
		e.SupplierID = outbox.ResolveSupplierID(b.entityID, e.Payload)
	}
	return b.inner.BufferOutbox(ctx, e)
}
