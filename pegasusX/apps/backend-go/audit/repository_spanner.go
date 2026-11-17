package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SpannerRepository implements Repository against Cloud Spanner using strictly
// append-only InsertMap("AuditEvents", ...) and explicit read-only / StrongRead() queries.
type SpannerRepository struct {
	client *spanner.Client
	now    func() time.Time
}

// NewSpannerRepository constructs a Cloud Spanner-backed audit repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{
		client: client,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Client exposes the underlying Spanner client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

type dayChainTip struct {
	SeqNum    int64
	EntryHash string
}

// BatchRecorder chains one or more AuditEvents inside a single Spanner ReadWriteTransaction.
// It caches the per-(TenantId, DayBucket) tip across buffered writes within the transaction
// so multiple events recorded in one transaction form an unbroken monotonic hash chain.
type BatchRecorder struct {
	txn        *spanner.ReadWriteTransaction
	now        func() time.Time
	diffWriter DiffWriter
	tips       map[string]dayChainTip
	holds      map[string]*LegalHold
	holdMiss   map[string]bool
}

// NewBatchRecorder constructs an intra-transaction AuditEvents batch recorder.
func NewBatchRecorder(txn *spanner.ReadWriteTransaction) *BatchRecorder {
	return &BatchRecorder{
		txn:      txn,
		now:      func() time.Time { return time.Now().UTC() },
		tips:     make(map[string]dayChainTip),
		holds:    make(map[string]*LegalHold),
		holdMiss: make(map[string]bool),
	}
}

// WithDiffWriter configures an optional WORM DiffWriter for >32KB diffs buffered via BatchRecorder.
func (b *BatchRecorder) WithDiffWriter(w DiffWriter) *BatchRecorder {
	if b != nil {
		b.diffWriter = w
	}
	return b
}

// BufferEventInTxn appends a single AuditEvents row inside txn.
// For multiple audit events inside the same RW transaction, use NewBatchRecorder(txn).Record(...) instead.
func BufferEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, in EventInput) (AuditEvent, error) {
	return NewBatchRecorder(txn).Record(ctx, in)
}

// Record appends one hash-chained AuditEvents row into b.txn using spanner.InsertMap only.
func (b *BatchRecorder) Record(ctx context.Context, in EventInput) (AuditEvent, error) {
	if b == nil || b.txn == nil {
		return AuditEvent{}, fmt.Errorf("%w: nil spanner read-write transaction", ErrInvalidAuditInput)
	}
	in.TenantID = strings.TrimSpace(in.TenantID)
	in.ActorID = strings.TrimSpace(in.ActorID)
	in.Action = strings.TrimSpace(in.Action)
	if in.TenantID == "" {
		return AuditEvent{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidAuditInput)
	}
	if in.Action == "" {
		return AuditEvent{}, fmt.Errorf("%w: action is required", ErrInvalidAuditInput)
	}
	if in.ActorID == "" {
		in.ActorID = "system"
	}
	in.ActorType = NormalizeActorType(string(in.ActorType))

	dayBucket := strings.TrimSpace(in.DayBucket)
	if dayBucket == "" {
		dayBucket = b.now().UTC().Format("2006-01-02")
	} else if !IsValidDayBucket(dayBucket) {
		return AuditEvent{}, ErrInvalidDayBucket
	}

	eventID := strings.TrimSpace(in.EventID)
	if eventID == "" {
		eventID = "aud_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}

	tipKey := in.TenantID + "|" + dayBucket
	tip, ok := b.tips[tipKey]
	if !ok {
		var err error
		tip, err = readDayChainTipInTxn(ctx, b.txn, in.TenantID, dayBucket)
		if err != nil {
			return AuditEvent{}, err
		}
	}

	legalHold := in.LegalHold
	legalHoldReason := strings.TrimSpace(in.LegalHoldReason)
	objType := strings.TrimSpace(in.ObjectType)
	objID := strings.TrimSpace(in.ObjectID)
	if !legalHold && objType != "" && objID != "" {
		hold, err := b.lookupActiveLegalHoldInTxn(ctx, in.TenantID, objType, objID)
		if err != nil {
			return AuditEvent{}, err
		}
		if hold != nil {
			legalHold = true
			legalHoldReason = hold.Reason
		}
	}

	ev := AuditEvent{
		TenantID:        in.TenantID,
		DayBucket:       dayBucket,
		EventID:         eventID,
		SeqNum:          tip.SeqNum + 1,
		ActorID:         in.ActorID,
		ActorType:       in.ActorType,
		Role:            strings.TrimSpace(in.Role),
		Action:          in.Action,
		ObjectType:      objType,
		ObjectID:        objID,
		BeforeJSON:      in.BeforeJSON,
		AfterJSON:       in.AfterJSON,
		DiffGcsURI:      strings.TrimSpace(in.DiffGcsURI),
		ReasonCode:      strings.TrimSpace(in.ReasonCode),
		ApprovalID:      strings.TrimSpace(in.ApprovalID),
		TraceID:         strings.TrimSpace(in.TraceID),
		IP:              strings.TrimSpace(in.IP),
		DeviceID:        strings.TrimSpace(in.DeviceID),
		PrevHash:        tip.EntryHash,
		LegalHold:       legalHold,
		LegalHoldReason: legalHoldReason,
		RetentionUntil:  in.RetentionUntil,
		CreatedAt:       b.now().UTC(),
		Source:          "audit_events",
	}

	if len(ev.BeforeJSON) > MaxInlineDiffBytes || len(ev.AfterJSON) > MaxInlineDiffBytes {
		writer := b.diffWriter
		if writer == nil {
			envWriter, err := ResolveDiffWriterFromEnv(ctx)
			if err != nil && auth.IsProduction() {
				return AuditEvent{}, err
			}
			writer = envWriter
		}
		if err := ApplyLargeDiffOffload(ctx, &ev, writer); err != nil {
			return AuditEvent{}, err
		}
	}

	canonical, err := CanonicalAuditBytes(ev)
	if err != nil {
		return AuditEvent{}, err
	}
	ev.EntryHash = ComputeEntryHash(ev.PrevHash, canonical)

	mut := spanner.InsertMap("AuditEvents", auditEventRowMap(ev))
	if err := b.txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
		return AuditEvent{}, fmt.Errorf("buffer AuditEvents insert: %w", err)
	}
	b.tips[tipKey] = dayChainTip{
		SeqNum:    ev.SeqNum,
		EntryHash: ev.EntryHash,
	}
	return ev, nil
}

func (b *BatchRecorder) lookupActiveLegalHoldInTxn(ctx context.Context, tenantID, objectType, objectID string) (*LegalHold, error) {
	k := tenantID + "|" + strings.ToUpper(objectType) + "|" + objectID
	if hold, ok := b.holds[k]; ok {
		return hold, nil
	}
	if b.holdMiss[k] {
		return nil, nil
	}
	iter := b.txn.Query(ctx, spanner.Statement{
		SQL: `SELECT HoldId, ObjectType, ObjectId, Reason, Status, CreatedBy, ExpiresAt, CreatedAt
		      FROM AuditLegalHolds
		      WHERE TenantId = @tid
		        AND ObjectType = @otype
		        AND ObjectId = @oid
		        AND Status = @status
		        AND (ExpiresAt IS NULL OR ExpiresAt > @now)
		      ORDER BY CreatedAt DESC
		      LIMIT 1`,
		Params: map[string]any{
			"tid":    tenantID,
			"otype":  objectType,
			"oid":    objectID,
			"status": LegalHoldStatusActive,
			"now":    b.now().UTC(),
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		b.holdMiss[k] = true
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lookup active legal hold: %w", err)
	}
	h, err := scanLegalHoldRow(tenantID, row)
	if err != nil {
		return nil, err
	}
	b.holds[k] = &h
	return &h, nil
}

func readDayChainTipInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, tenantID, dayBucket string) (dayChainTip, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT SeqNum, EntryHash
		      FROM AuditEvents@{FORCE_INDEX=Idx_AuditEvents_ByDaySeq}
		      WHERE TenantId = @tid AND DayBucket = @day
		      ORDER BY SeqNum DESC
		      LIMIT 1`,
		Params: map[string]any{
			"tid": tenantID,
			"day": dayBucket,
		},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return dayChainTip{SeqNum: 0, EntryHash: GenesisHash}, nil
	}
	if err != nil {
		return dayChainTip{}, fmt.Errorf("read audit chain tip: %w", err)
	}
	var seqNum int64
	var entryHash string
	if err := row.Columns(&seqNum, &entryHash); err != nil {
		return dayChainTip{}, fmt.Errorf("scan audit chain tip: %w", err)
	}
	return dayChainTip{SeqNum: seqNum, EntryHash: entryHash}, nil
}

func auditEventRowMap(ev AuditEvent) map[string]any {
	m := map[string]any{
		"TenantId":        ev.TenantID,
		"DayBucket":       ev.DayBucket,
		"EventId":         ev.EventID,
		"SeqNum":          ev.SeqNum,
		"ActorId":         ev.ActorID,
		"ActorType":       string(ev.ActorType),
		"Role":            nullStr(ev.Role),
		"Action":          ev.Action,
		"ObjectType":      nullStr(ev.ObjectType),
		"ObjectId":        nullStr(ev.ObjectID),
		"BeforeJson":      nullStr(ev.BeforeJSON),
		"AfterJson":       nullStr(ev.AfterJSON),
		"DiffGcsUri":      nullStr(ev.DiffGcsURI),
		"ReasonCode":      nullStr(ev.ReasonCode),
		"ApprovalId":      nullStr(ev.ApprovalID),
		"TraceId":         nullStr(ev.TraceID),
		"Ip":              nullStr(ev.IP),
		"DeviceId":        nullStr(ev.DeviceID),
		"PrevHash":        ev.PrevHash,
		"EntryHash":       ev.EntryHash,
		"LegalHold":       ev.LegalHold,
		"LegalHoldReason": nullStr(ev.LegalHoldReason),
		"CreatedAt":       spanner.CommitTimestamp,
	}
	if ev.RetentionUntil != nil && !ev.RetentionUntil.IsZero() {
		m["RetentionUntil"] = ev.RetentionUntil.UTC()
	} else {
		m["RetentionUntil"] = spanner.NullTime{}
	}
	return m
}

// Record appends one AuditEvents row in its own Spanner ReadWriteTransaction.
func (r *SpannerRepository) Record(ctx context.Context, in EventInput) (AuditEvent, error) {
	if r == nil || r.client == nil {
		return AuditEvent{}, ErrRepoUnavailable
	}
	var out AuditEvent
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		ev, err := r.RecordInTxn(ctx, txn, in)
		if err != nil {
			return err
		}
		out = ev
		return nil
	})
	if err != nil {
		return AuditEvent{}, err
	}
	return out, nil
}

// RecordInTxn appends one AuditEvents row inside txn.
func (r *SpannerRepository) RecordInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, in EventInput) (AuditEvent, error) {
	return NewBatchRecorder(txn).Record(ctx, in)
}

// ListEvents queries AuditEvents (plus readable AuditLog fallback when IncludeLegacy or object filtering is set)
// inside an explicit ReadOnlyTransaction.
func (r *SpannerRepository) ListEvents(ctx context.Context, filter Filter) ([]AuditEvent, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	var results []AuditEvent
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		sql, params := buildAuditEventsQuery(filter, limit)
		iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("query AuditEvents: %w", nextErr)
			}
			ev, scanErr := scanAuditEventRow(row)
			if scanErr != nil {
				return scanErr
			}
			results = append(results, ev)
		}

		if (filter.IncludeLegacy || (filter.ObjectType != "" && filter.ObjectID != "")) && len(results) < limit {
			legacyRows, legErr := queryLegacyAuditLogInTxn(ctx, txn, filter, limit-len(results))
			if legErr != nil {
				return legErr
			}
			results = append(results, legacyRows...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if !results[i].CreatedAt.Equal(results[j].CreatedAt) {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		}
		if results[i].DayBucket != results[j].DayBucket {
			return results[i].DayBucket > results[j].DayBucket
		}
		return results[i].SeqNum > results[j].SeqNum
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// GetObjectHistory returns chronological-descending AuditEvents + AuditLog fallback for one object.
func (r *SpannerRepository) GetObjectHistory(ctx context.Context, tenantID, objectType, objectID, viewerRole string, limit int) ([]AuditEvent, error) {
	return r.ListEvents(ctx, Filter{
		TenantID:      tenantID,
		ObjectType:    objectType,
		ObjectID:      objectID,
		Limit:         limit,
		IncludeLegacy: true,
		ViewerRole:    viewerRole,
	})
}

func buildAuditEventsQuery(f Filter, limit int) (string, map[string]any) {
	params := map[string]any{"lim": int64(limit)}
	clauses := make([]string, 0, 8)
	if tid := strings.TrimSpace(f.TenantID); tid != "" {
		clauses = append(clauses, "TenantId = @tid")
		params["tid"] = tid
	}
	if day := strings.TrimSpace(f.DayBucket); day != "" {
		clauses = append(clauses, "DayBucket = @day")
		params["day"] = day
	}
	if ot := strings.TrimSpace(f.ObjectType); ot != "" {
		clauses = append(clauses, "LOWER(ObjectType) = LOWER(@otype)")
		params["otype"] = ot
	}
	if oid := strings.TrimSpace(f.ObjectID); oid != "" {
		clauses = append(clauses, "ObjectId = @oid")
		params["oid"] = oid
	}
	if act := strings.TrimSpace(f.ActorID); act != "" {
		clauses = append(clauses, "ActorId = @actor")
		params["actor"] = act
	}
	if f.ActorType != "" {
		clauses = append(clauses, "ActorType = @atype")
		params["atype"] = string(f.ActorType)
	}
	if action := strings.TrimSpace(f.Action); action != "" {
		clauses = append(clauses, "Action = @action")
		params["action"] = action
	}
	if f.From != nil && !f.From.IsZero() {
		clauses = append(clauses, "CreatedAt >= @from")
		params["from"] = f.From.UTC()
	}
	if f.To != nil && !f.To.IsZero() {
		clauses = append(clauses, "CreatedAt <= @to")
		params["to"] = f.To.UTC()
	}

	tableExpr := "AuditEvents"
	if f.TenantID != "" {
		switch {
		case f.ObjectType != "" && f.ObjectID != "":
			// Lowercase filter still benefits from ByObject only when exact case matches; keep base/ByCreated safe.
			tableExpr = "AuditEvents@{FORCE_INDEX=Idx_AuditEvents_ByCreated}"
		case f.ActorID != "":
			tableExpr = "AuditEvents@{FORCE_INDEX=Idx_AuditEvents_ByActor}"
		default:
			tableExpr = "AuditEvents@{FORCE_INDEX=Idx_AuditEvents_ByCreated}"
		}
	}

	sql := `SELECT TenantId, DayBucket, EventId, SeqNum, ActorId, ActorType, Role, Action,
		       ObjectType, ObjectId, BeforeJson, AfterJson, DiffGcsUri, ReasonCode,
		       ApprovalId, TraceId, Ip, DeviceId, PrevHash, EntryHash,
		       LegalHold, LegalHoldReason, RetentionUntil, CreatedAt
		FROM ` + tableExpr
	if len(clauses) > 0 {
		sql += " WHERE " + strings.Join(clauses, " AND ")
	}
	sql += " ORDER BY CreatedAt DESC, SeqNum DESC LIMIT @lim"
	return sql, params
}

func queryLegacyAuditLogInTxn(ctx context.Context, txn *spanner.ReadOnlyTransaction, f Filter, limit int) ([]AuditEvent, error) {
	if limit <= 0 {
		return nil, nil
	}
	params := map[string]any{"lim": int64(limit)}
	clauses := make([]string, 0, 6)
	if tid := strings.TrimSpace(f.TenantID); tid != "" {
		clauses = append(clauses, "SupplierId = @tid")
		params["tid"] = tid
	}
	if ot := strings.TrimSpace(f.ObjectType); ot != "" {
		clauses = append(clauses, "LOWER(AggregateType) = LOWER(@otype)")
		params["otype"] = ot
	}
	if oid := strings.TrimSpace(f.ObjectID); oid != "" {
		clauses = append(clauses, "AggregateId = @oid")
		params["oid"] = oid
	}
	if act := strings.TrimSpace(f.ActorID); act != "" {
		clauses = append(clauses, "ActorId = @actor")
		params["actor"] = act
	}
	if f.From != nil && !f.From.IsZero() {
		clauses = append(clauses, "CreatedAt >= @from")
		params["from"] = f.From.UTC()
	}
	if f.To != nil && !f.To.IsZero() {
		clauses = append(clauses, "CreatedAt <= @to")
		params["to"] = f.To.UTC()
	}

	sql := `SELECT AuditId, SupplierId, ActorId, ActorRole, Action, AggregateType, AggregateId, DetailsJson, TraceId, CreatedAt
		FROM AuditLog`
	if len(clauses) > 0 {
		sql += " WHERE " + strings.Join(clauses, " AND ")
	}
	sql += " ORDER BY CreatedAt DESC LIMIT @lim"

	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []AuditEvent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// If AuditLog isn't present in a stripped test DB, ignore gracefully; otherwise report.
			if spanner.ErrCode(err) == codes.NotFound {
				return nil, nil
			}
			return nil, fmt.Errorf("query legacy AuditLog: %w", err)
		}
		var auditID, supplierID, actorID, actorRole, action, aggType, aggID string
		var details []byte
		var traceID spanner.NullString
		var createdAt time.Time
		if err := row.Columns(&auditID, &supplierID, &actorID, &actorRole, &action, &aggType, &aggID, &details, &traceID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan legacy AuditLog: %w", err)
		}
		ev := AuditEvent{
			TenantID:   supplierID,
			DayBucket:  createdAt.UTC().Format("2006-01-02"),
			EventID:    auditID,
			ActorID:    actorID,
			ActorType:  ActorTypeHuman,
			Role:       actorRole,
			Action:     action,
			ObjectType: aggType,
			ObjectID:   aggID,
			AfterJSON:  string(details),
			TraceID:    traceID.StringVal,
			CreatedAt:  createdAt.UTC(),
			Source:     "legacy_audit_log",
		}
		out = append(out, ev)
	}
	return out, nil
}

// VerifyDayChain reads all AuditEvents for (tenantID, dayBucket) and any matching AuditChainAnchors
// in a consistent ReadOnlyTransaction and verifies every PrevHash -> EntryHash link.
func (r *SpannerRepository) VerifyDayChain(ctx context.Context, tenantID, dayBucket string) (ChainVerifyResult, error) {
	if r == nil || r.client == nil {
		return ChainVerifyResult{}, ErrRepoUnavailable
	}
	var dayEvents []AuditEvent
	var anchor Anchor
	var hasAnchor bool

	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT TenantId, DayBucket, EventId, SeqNum, ActorId, ActorType, Role, Action,
			             ObjectType, ObjectId, BeforeJson, AfterJson, DiffGcsUri, ReasonCode,
			             ApprovalId, TraceId, Ip, DeviceId, PrevHash, EntryHash,
			             LegalHold, LegalHoldReason, RetentionUntil, CreatedAt
			      FROM AuditEvents
			      WHERE TenantId = @tid AND DayBucket = @day
			      ORDER BY SeqNum ASC, EventId ASC`,
			Params: map[string]any{"tid": tenantID, "day": dayBucket},
		})
		defer iter.Stop()
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("verify query AuditEvents: %w", nextErr)
			}
			ev, scanErr := scanAuditEventRow(row)
			if scanErr != nil {
				return scanErr
			}
			dayEvents = append(dayEvents, ev)
		}

		aRow, aErr := txn.ReadRow(ctx, "AuditChainAnchors", spanner.Key{tenantID, dayBucket}, []string{
			"TenantId", "DayBucket", "HeadEventId", "EventCount", "RootHash", "WormUri", "SignerKeyId", "AnchoredBy", "AnchoredAt",
		})
		if aErr == nil {
			var scanErr error
			anchor, scanErr = scanAnchorRow(aRow)
			if scanErr != nil {
				return scanErr
			}
			hasAnchor = true
		} else if spanner.ErrCode(aErr) != codes.NotFound {
			return fmt.Errorf("verify read anchor: %w", aErr)
		}
		return nil
	})
	if err != nil {
		return ChainVerifyResult{}, err
	}

	return verifyOrderedEvents(tenantID, dayBucket, dayEvents, anchor, hasAnchor), nil
}

func verifyOrderedEvents(tenantID, dayBucket string, dayEvents []AuditEvent, anchor Anchor, hasAnchor bool) ChainVerifyResult {
	sort.Slice(dayEvents, func(i, j int) bool {
		if dayEvents[i].SeqNum != dayEvents[j].SeqNum {
			return dayEvents[i].SeqNum < dayEvents[j].SeqNum
		}
		return dayEvents[i].EventID < dayEvents[j].EventID
	})

	res := ChainVerifyResult{
		TenantID:   tenantID,
		DayBucket:  dayBucket,
		Valid:      true,
		EventCount: int64(len(dayEvents)),
		RootHash:   GenesisHash,
	}

	expectedPrev := GenesisHash
	var expectedSeq int64 = 1
	for _, ev := range dayEvents {
		if ev.SeqNum != expectedSeq {
			res.Valid = false
			res.BrokenEventID = ev.EventID
			res.FailureReason = fmt.Sprintf("sequence gap at event %s: got SeqNum=%d want %d", ev.EventID, ev.SeqNum, expectedSeq)
			return res
		}
		if ev.PrevHash != expectedPrev {
			res.Valid = false
			res.BrokenEventID = ev.EventID
			res.FailureReason = fmt.Sprintf("prev_hash mismatch at event %s (seq %d): got %s want %s", ev.EventID, ev.SeqNum, ev.PrevHash, expectedPrev)
			return res
		}
		canonical, err := CanonicalAuditBytes(ev)
		if err != nil {
			res.Valid = false
			res.BrokenEventID = ev.EventID
			res.FailureReason = fmt.Sprintf("canonical marshal error at %s: %v", ev.EventID, err)
			return res
		}
		computed := ComputeEntryHash(ev.PrevHash, canonical)
		if ev.EntryHash != computed {
			res.Valid = false
			res.BrokenEventID = ev.EventID
			res.FailureReason = fmt.Sprintf("entry_hash tamper detected at event %s (seq %d): stored=%s computed=%s", ev.EventID, ev.SeqNum, ev.EntryHash, computed)
			return res
		}
		expectedPrev = ev.EntryHash
		res.HeadEventID = ev.EventID
		res.RootHash = ev.EntryHash
		expectedSeq++
	}

	if hasAnchor {
		res.AnchoredRootHash = anchor.RootHash
		res.AnchoredEventCount = anchor.EventCount
		var matched bool
		switch {
		case anchor.EventCount < 0 || anchor.EventCount > res.EventCount:
			matched = false
			res.FailureReason = fmt.Sprintf("anchored event count (%d) exceeds observed chain length (%d)", anchor.EventCount, res.EventCount)
		case anchor.EventCount == 0:
			matched = anchor.RootHash == GenesisHash
			if !matched {
				res.FailureReason = fmt.Sprintf("anchored empty-day root mismatch: anchor=%s want %s", anchor.RootHash, GenesisHash)
			}
		default:
			prefixHash := dayEvents[anchor.EventCount-1].EntryHash
			matched = anchor.RootHash == prefixHash
			if !matched {
				res.FailureReason = fmt.Sprintf("anchored root mismatch at seq %d: anchor=%s vs chain=%s",
					anchor.EventCount, anchor.RootHash, prefixHash)
			}
		}
		res.AnchorMatched = &matched
		if !matched {
			res.Valid = false
		}
	}
	return res
}

// WriteDailyAnchor verifies the chain for (tenantID, dayBucket), writes a signed canonical root JSON
// to sink (when configured), upserts AuditChainAnchors, and emits events.EventAuditAnchorWritten atomically.
func (r *SpannerRepository) WriteDailyAnchor(ctx context.Context, tenantID, dayBucket, anchoredBy string, sink WormAnchorSink) (Anchor, error) {
	if r == nil || r.client == nil {
		return Anchor{}, ErrRepoUnavailable
	}
	verifyRes, err := r.VerifyDayChain(ctx, tenantID, dayBucket)
	if err != nil {
		return Anchor{}, err
	}
	if !verifyRes.Valid {
		return Anchor{}, fmt.Errorf("%w: %s", ErrChainBroken, verifyRes.FailureReason)
	}

	// Check if already anchored for identical RootHash & EventCount (idempotent return before touching WORM).
	var existingAnchor Anchor
	var exists bool
	if err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		row, rErr := txn.ReadRow(ctx, "AuditChainAnchors", spanner.Key{tenantID, dayBucket}, []string{
			"TenantId", "DayBucket", "HeadEventId", "EventCount", "RootHash", "WormUri", "SignerKeyId", "AnchoredBy", "AnchoredAt",
		})
		if spanner.ErrCode(rErr) == codes.NotFound {
			return nil
		}
		if rErr != nil {
			return rErr
		}
		var scanErr error
		existingAnchor, scanErr = scanAnchorRow(row)
		if scanErr != nil {
			return scanErr
		}
		exists = true
		return nil
	}); err != nil {
		return Anchor{}, fmt.Errorf("read existing anchor: %w", err)
	}
	if exists && existingAnchor.RootHash == verifyRes.RootHash && existingAnchor.EventCount == verifyRes.EventCount {
		return existingAnchor, nil
	}

	now := r.now().UTC()
	headID := verifyRes.HeadEventID
	if headID == "" {
		headID = "genesis"
	}
	anchor := Anchor{
		TenantID:    tenantID,
		DayBucket:   dayBucket,
		HeadEventID: headID,
		EventCount:  verifyRes.EventCount,
		RootHash:    verifyRes.RootHash,
		AnchoredBy:  anchoredBy,
		AnchoredAt:  now,
	}

	blob, raw, err := SignAnchorBlob(anchor)
	if err != nil {
		return Anchor{}, err
	}
	anchor.SignerKeyID = blob.SignerKeyID
	if sink != nil {
		wormURI, err := sink.WriteAnchor(ctx, blob, raw)
		if err != nil {
			return Anchor{}, err
		}
		anchor.WormURI = wormURI
	}

	err = spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, rErr := txn.ReadRow(ctx, "AuditChainAnchors", spanner.Key{tenantID, dayBucket}, []string{
			"TenantId", "DayBucket", "HeadEventId", "EventCount", "RootHash", "WormUri", "SignerKeyId", "AnchoredBy", "AnchoredAt",
		})
		if rErr == nil {
			cur, scanErr := scanAnchorRow(row)
			if scanErr != nil {
				return scanErr
			}
			if cur.RootHash == anchor.RootHash && cur.EventCount == anchor.EventCount {
				anchor = cur
				return nil
			}
		} else if spanner.ErrCode(rErr) != codes.NotFound {
			return rErr
		}

		mut := spanner.InsertOrUpdateMap("AuditChainAnchors", map[string]any{
			"TenantId":    anchor.TenantID,
			"DayBucket":   anchor.DayBucket,
			"HeadEventId": anchor.HeadEventID,
			"EventCount":  anchor.EventCount,
			"RootHash":    anchor.RootHash,
			"WormUri":     nullStr(anchor.WormURI),
			"SignerKeyId": nullStr(anchor.SignerKeyID),
			"AnchoredBy":  anchor.AnchoredBy,
			"AnchoredAt":  spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
			return fmt.Errorf("buffer AuditChainAnchors: %w", err)
		}

		rawBuf := outbox.NewSpannerTxnBuffer(txn)
		buf := tenantOutboxBuffer{tenantID: anchor.TenantID, inner: rawBuf}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateAuditAnchor, tenantID+":"+dayBucket, events.TopicLedger, events.AuditAnchorWrittenEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventAuditAnchorWritten,
				Version:   1,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			TenantID:    anchor.TenantID,
			DayBucket:   anchor.DayBucket,
			HeadEventID: anchor.HeadEventID,
			EventCount:  anchor.EventCount,
			RootHash:    anchor.RootHash,
			WormURI:     anchor.WormURI,
			SignerKeyID: anchor.SignerKeyID,
			AnchoredBy:  anchor.AnchoredBy,
			AnchoredAt:  now.Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return rawBuf.Flush(ctx)
	})
	if err != nil {
		return Anchor{}, err
	}
	return anchor, nil
}

// ListAnchors reads AuditChainAnchors using an explicit ReadOnlyTransaction.
func (r *SpannerRepository) ListAnchors(ctx context.Context, tenantID, dayBucket string, limit int) ([]Anchor, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []Anchor
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		params := map[string]any{"lim": int64(limit)}
		clauses := make([]string, 0, 2)
		if tenantID != "" {
			clauses = append(clauses, "TenantId = @tid")
			params["tid"] = tenantID
		}
		if dayBucket != "" {
			clauses = append(clauses, "DayBucket = @day")
			params["day"] = dayBucket
		}
		sql := `SELECT TenantId, DayBucket, HeadEventId, EventCount, RootHash, WormUri, SignerKeyId, AnchoredBy, AnchoredAt
		        FROM AuditChainAnchors`
		if len(clauses) > 0 {
			sql += " WHERE " + strings.Join(clauses, " AND ")
		}
		sql += " ORDER BY DayBucket DESC, AnchoredAt DESC LIMIT @lim"
		iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("query AuditChainAnchors: %w", nextErr)
			}
			a, scanErr := scanAnchorRow(row)
			if scanErr != nil {
				return scanErr
			}
			out = append(out, a)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CreateLegalHold inserts an AuditLegalHolds record and appends an immutable AuditEvents record in the same RW txn.
func (r *SpannerRepository) CreateLegalHold(ctx context.Context, in LegalHoldInput) (LegalHold, error) {
	if r == nil || r.client == nil {
		return LegalHold{}, ErrRepoUnavailable
	}
	now := r.now().UTC()
	hold := LegalHold{
		TenantID:   in.TenantID,
		HoldID:     "hold_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
		ObjectType: in.ObjectType,
		ObjectID:   in.ObjectID,
		Reason:     in.Reason,
		Status:     LegalHoldStatusActive,
		CreatedBy:  in.CreatedBy,
		ExpiresAt:  in.ExpiresAt,
		CreatedAt:  now,
	}

	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := map[string]any{
			"TenantId":   hold.TenantID,
			"HoldId":     hold.HoldID,
			"ObjectType": hold.ObjectType,
			"ObjectId":   hold.ObjectID,
			"Reason":     hold.Reason,
			"Status":     hold.Status,
			"CreatedBy":  hold.CreatedBy,
			"CreatedAt":  spanner.CommitTimestamp,
		}
		if hold.ExpiresAt != nil && !hold.ExpiresAt.IsZero() {
			m["ExpiresAt"] = hold.ExpiresAt.UTC()
		} else {
			m["ExpiresAt"] = spanner.NullTime{}
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("AuditLegalHolds", m)}); err != nil {
			return fmt.Errorf("insert AuditLegalHolds: %w", err)
		}
		afterRaw, _ := json.Marshal(map[string]any{
			"hold_id": hold.HoldID,
			"status":  hold.Status,
			"reason":  hold.Reason,
		})
		_, err := BufferEventInTxn(ctx, txn, EventInput{
			TenantID:        hold.TenantID,
			DayBucket:       now.Format("2006-01-02"),
			ActorID:         hold.CreatedBy,
			ActorType:       ActorTypeHuman,
			Action:          "audit.legal_hold.created",
			ObjectType:      hold.ObjectType,
			ObjectID:        hold.ObjectID,
			AfterJSON:       string(afterRaw),
			ReasonCode:      "LEGAL_HOLD",
			TraceID:         outbox.TraceIDFromContext(ctx),
			LegalHold:       true,
			LegalHoldReason: hold.Reason,
		})
		return err
	})
	if err != nil {
		return LegalHold{}, err
	}
	return hold, nil
}

// ListLegalHolds reads AuditLegalHolds inside a ReadOnlyTransaction.
func (r *SpannerRepository) ListLegalHolds(ctx context.Context, tenantID, objectType, objectID, status string, limit int) ([]LegalHold, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []LegalHold
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		params := map[string]any{"lim": int64(limit)}
		clauses := make([]string, 0, 4)
		if tenantID != "" {
			clauses = append(clauses, "TenantId = @tid")
			params["tid"] = tenantID
		}
		if objectType != "" {
			clauses = append(clauses, "LOWER(ObjectType) = LOWER(@otype)")
			params["otype"] = objectType
		}
		if objectID != "" {
			clauses = append(clauses, "ObjectId = @oid")
			params["oid"] = objectID
		}
		if status != "" {
			clauses = append(clauses, "UPPER(Status) = UPPER(@status)")
			params["status"] = status
		}
		sql := `SELECT TenantId, HoldId, ObjectType, ObjectId, Reason, Status, CreatedBy, ExpiresAt, CreatedAt
		        FROM AuditLegalHolds`
		if len(clauses) > 0 {
			sql += " WHERE " + strings.Join(clauses, " AND ")
		}
		sql += " ORDER BY CreatedAt DESC LIMIT @lim"
		iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
		defer iter.Stop()
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("query AuditLegalHolds: %w", nextErr)
			}
			var tid, holdID, ot, oid, reason, st, createdBy string
			var exp spanner.NullTime
			var createdAt time.Time
			if err := row.Columns(&tid, &holdID, &ot, &oid, &reason, &st, &createdBy, &exp, &createdAt); err != nil {
				return fmt.Errorf("scan AuditLegalHolds: %w", err)
			}
			h := LegalHold{
				TenantID:   tid,
				HoldID:     holdID,
				ObjectType: ot,
				ObjectID:   oid,
				Reason:     reason,
				Status:     st,
				CreatedBy:  createdBy,
				CreatedAt:  createdAt.UTC(),
			}
			if exp.Valid {
				t := exp.Time.UTC()
				h.ExpiresAt = &t
			}
			out = append(out, h)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListTenantsForDay returns distinct TenantIds with AuditEvents in dayBucket.
func (r *SpannerRepository) ListTenantsForDay(ctx context.Context, dayBucket string) ([]string, error) {
	if r == nil || r.client == nil {
		return nil, ErrRepoUnavailable
	}
	var tenants []string
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT DISTINCT TenantId FROM AuditEvents WHERE DayBucket = @day ORDER BY TenantId ASC`,
			Params: map[string]any{"day": dayBucket},
		})
		defer iter.Stop()
		for {
			row, nextErr := iter.Next()
			if nextErr == iterator.Done {
				break
			}
			if nextErr != nil {
				return fmt.Errorf("list audit tenants for day: %w", nextErr)
			}
			var tid string
			if err := row.Columns(&tid); err != nil {
				return err
			}
			if strings.TrimSpace(tid) != "" {
				tenants = append(tenants, tid)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tenants, nil
}

func scanAuditEventRow(row *spanner.Row) (AuditEvent, error) {
	var (
		ev                              AuditEvent
		actorType                       string
		role, objType, objID            spanner.NullString
		beforeJSON, afterJSON, diffURI  spanner.NullString
		reasonCode, approvalID, traceID spanner.NullString
		ip, deviceID, legalHoldReason   spanner.NullString
		retentionUntil                  spanner.NullTime
	)
	if err := row.Columns(
		&ev.TenantID, &ev.DayBucket, &ev.EventID, &ev.SeqNum,
		&ev.ActorID, &actorType, &role, &ev.Action,
		&objType, &objID, &beforeJSON, &afterJSON, &diffURI,
		&reasonCode, &approvalID, &traceID, &ip, &deviceID,
		&ev.PrevHash, &ev.EntryHash,
		&ev.LegalHold, &legalHoldReason, &retentionUntil, &ev.CreatedAt,
	); err != nil {
		return AuditEvent{}, fmt.Errorf("scan AuditEvent row: %w", err)
	}
	ev.ActorType = NormalizeActorType(actorType)
	ev.Role = role.StringVal
	ev.ObjectType = objType.StringVal
	ev.ObjectID = objID.StringVal
	ev.BeforeJSON = beforeJSON.StringVal
	ev.AfterJSON = afterJSON.StringVal
	ev.DiffGcsURI = diffURI.StringVal
	ev.ReasonCode = reasonCode.StringVal
	ev.ApprovalID = approvalID.StringVal
	ev.TraceID = traceID.StringVal
	ev.IP = ip.StringVal
	ev.DeviceID = deviceID.StringVal
	ev.LegalHoldReason = legalHoldReason.StringVal
	if retentionUntil.Valid {
		t := retentionUntil.Time.UTC()
		ev.RetentionUntil = &t
	}
	ev.CreatedAt = ev.CreatedAt.UTC()
	ev.Source = "audit_events"
	return ev, nil
}

func scanAnchorRow(row *spanner.Row) (Anchor, error) {
	var a Anchor
	var wormURI, signerKeyID spanner.NullString
	if err := row.Columns(
		&a.TenantID, &a.DayBucket, &a.HeadEventID, &a.EventCount,
		&a.RootHash, &wormURI, &signerKeyID, &a.AnchoredBy, &a.AnchoredAt,
	); err != nil {
		return Anchor{}, fmt.Errorf("scan AuditChainAnchors row: %w", err)
	}
	a.WormURI = wormURI.StringVal
	a.SignerKeyID = signerKeyID.StringVal
	a.AnchoredAt = a.AnchoredAt.UTC()
	return a, nil
}

func scanLegalHoldRow(tenantID string, row *spanner.Row) (LegalHold, error) {
	var h LegalHold
	var exp spanner.NullTime
	h.TenantID = tenantID
	if err := row.Columns(&h.HoldID, &h.ObjectType, &h.ObjectID, &h.Reason, &h.Status, &h.CreatedBy, &exp, &h.CreatedAt); err != nil {
		return LegalHold{}, fmt.Errorf("scan AuditLegalHolds row: %w", err)
	}
	if exp.Valid {
		t := exp.Time.UTC()
		h.ExpiresAt = &t
	}
	h.CreatedAt = h.CreatedAt.UTC()
	return h, nil
}

func nullStr(s string) spanner.NullString {
	if s == "" {
		return spanner.NullString{}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}

type tenantOutboxBuffer struct {
	tenantID string
	inner    outbox.TxnBuffer
}

func (b tenantOutboxBuffer) BufferOutbox(ctx context.Context, e outbox.Event) error {
	if strings.TrimSpace(e.SupplierID) == "" || e.SupplierID == outbox.PlatformSupplierID {
		e.SupplierID = b.tenantID
	}
	return b.inner.BufferOutbox(ctx, e)
}
