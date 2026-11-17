package partner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const (
	OnecStatusPending  = "PENDING"
	OnecStatusAcked    = "ACKED"
	OnecStatusRejected = "REJECTED"
	OnecStatusImported = "IMPORTED"

	OnecDocJournalEntry = "JOURNAL_ENTRY"
	OnecDocImportBatch  = "IMPORT_BATCH"

	maxOnecDocKeyLen    = 128
	maxOnecRefLen       = 128
	maxOnecReasonLen    = 512
	defaultOnecListLim  = 100
	maxOnecListLim      = 500
)

var (
	ErrOnecPostingNotFound = errors.New("onec_posting_not_found")
	ErrInvalidOnecInput    = errors.New("invalid_onec_input")
)

// OnecPosting models a durable PartnerOnecPostings row.
type OnecPosting struct {
	PostingID     string     `json:"posting_id"`
	SupplierID    string     `json:"supplier_id"`
	DocumentKey   string     `json:"document_key"`
	DocumentType  string     `json:"document_type"`
	ExportJobID   string     `json:"export_job_id,omitempty"`
	PayloadJSON   string     `json:"payload_json"`
	ExposureMinor int64      `json:"exposure_minor"`
	Status        string     `json:"status"`
	OnecRef       string     `json:"onec_ref,omitempty"`
	RejectReason  string     `json:"reject_reason,omitempty"`
	Attempts      int64      `json:"attempts"`
	AckedAt       *time.Time `json:"acked_at,omitempty"`
	RejectedAt    *time.Time `json:"rejected_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// OnecPostingRepository persists PartnerOnecPostings for durable 1C pull/ack/reject and import idempotency.
type OnecPostingRepository interface {
	CreateIfAbsent(ctx context.Context, p OnecPosting) (OnecPosting, bool, error)
	Get(ctx context.Context, supplierID, postingID string) (OnecPosting, bool, error)
	GetByDocumentKey(ctx context.Context, supplierID, documentKey string) (OnecPosting, bool, error)
	ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]OnecPosting, error)
	Ack(ctx context.Context, supplierID, postingIDOrDocKey, onecRef string, now time.Time) (OnecPosting, error)
	Reject(ctx context.Context, supplierID, postingIDOrDocKey, reason string, exposureMinor *int64, now time.Time) (OnecPosting, error)
	Requeue(ctx context.Context, supplierID, postingIDOrDocKey string, now time.Time) (OnecPosting, error)
	Delete(ctx context.Context, supplierID, postingID string) error
}

func normalizeOnecDocKey(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) <= maxOnecDocKeyLen {
		return s
	}
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func truncateOnecField(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func prepareOnecPosting(p *OnecPosting, now time.Time) error {
	if p == nil {
		return ErrInvalidOnecInput
	}
	p.SupplierID = strings.TrimSpace(p.SupplierID)
	p.DocumentKey = normalizeOnecDocKey(p.DocumentKey)
	p.DocumentType = truncateOnecField(strings.ToUpper(p.DocumentType), 32)
	p.ExportJobID = strings.TrimSpace(p.ExportJobID)
	p.OnecRef = truncateOnecField(p.OnecRef, maxOnecRefLen)
	p.RejectReason = truncateOnecField(p.RejectReason, maxOnecReasonLen)
	p.Status = strings.ToUpper(strings.TrimSpace(p.Status))
	if p.SupplierID == "" || p.DocumentKey == "" {
		return ErrInvalidOnecInput
	}
	if p.DocumentType == "" {
		p.DocumentType = OnecDocJournalEntry
	}
	if p.PayloadJSON == "" {
		p.PayloadJSON = "{}"
	}
	if p.ExposureMinor < 0 {
		p.ExposureMinor = 0
	}
	if p.Status == "" {
		p.Status = OnecStatusPending
	}
	if strings.TrimSpace(p.PostingID) == "" {
		p.PostingID = uuid.NewString()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now.UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now.UTC()
	}
	return nil
}

// MemoryOnecPostingRepository provides an in-memory PartnerOnecPostings store for unit tests and nil-Spanner fallback.
type MemoryOnecPostingRepository struct {
	mu      sync.Mutex
	byID    map[string]OnecPosting
	byDoc   map[string]string // supplierID|docKey -> postingID
}

// NewMemoryOnecPostingRepository initializes an in-memory OnecPostingRepository.
func NewMemoryOnecPostingRepository() *MemoryOnecPostingRepository {
	return &MemoryOnecPostingRepository{
		byID:  map[string]OnecPosting{},
		byDoc: map[string]string{},
	}
}

func onecMemDocKey(supplierID, docKey string) string {
	return strings.TrimSpace(supplierID) + "|" + normalizeOnecDocKey(docKey)
}

func cloneOnecPosting(p OnecPosting) OnecPosting {
	if p.AckedAt != nil {
		t := *p.AckedAt
		p.AckedAt = &t
	}
	if p.RejectedAt != nil {
		t := *p.RejectedAt
		p.RejectedAt = &t
	}
	return p
}

func (r *MemoryOnecPostingRepository) CreateIfAbsent(_ context.Context, p OnecPosting) (OnecPosting, bool, error) {
	if err := prepareOnecPosting(&p, time.Now().UTC()); err != nil {
		return OnecPosting{}, false, err
	}
	dk := onecMemDocKey(p.SupplierID, p.DocumentKey)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existingID, ok := r.byDoc[dk]; ok {
		if existing, exists := r.byID[existingID]; exists {
			return cloneOnecPosting(existing), false, nil
		}
	}
	r.byID[p.PostingID] = cloneOnecPosting(p)
	r.byDoc[dk] = p.PostingID
	return cloneOnecPosting(p), true, nil
}

func (r *MemoryOnecPostingRepository) Get(_ context.Context, supplierID, postingID string) (OnecPosting, bool, error) {
	sid := strings.TrimSpace(supplierID)
	pid := strings.TrimSpace(postingID)
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byID[pid]
	if !ok || p.SupplierID != sid {
		return OnecPosting{}, false, nil
	}
	return cloneOnecPosting(p), true, nil
}

func (r *MemoryOnecPostingRepository) GetByDocumentKey(_ context.Context, supplierID, documentKey string) (OnecPosting, bool, error) {
	dk := onecMemDocKey(supplierID, documentKey)
	r.mu.Lock()
	defer r.mu.Unlock()
	pid, ok := r.byDoc[dk]
	if !ok {
		return OnecPosting{}, false, nil
	}
	p, ok := r.byID[pid]
	if !ok || p.SupplierID != strings.TrimSpace(supplierID) {
		return OnecPosting{}, false, nil
	}
	return cloneOnecPosting(p), true, nil
}

func (r *MemoryOnecPostingRepository) ListBySupplier(_ context.Context, supplierID, status string, limit int) ([]OnecPosting, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return nil, ErrInvalidOnecInput
	}
	st := strings.ToUpper(strings.TrimSpace(status))
	if limit <= 0 {
		limit = defaultOnecListLim
	}
	if limit > maxOnecListLim {
		limit = maxOnecListLim
	}
	r.mu.Lock()
	out := make([]OnecPosting, 0)
	for _, p := range r.byID {
		if p.SupplierID != sid {
			continue
		}
		if st != "" && st != "ALL" && p.Status != st {
			continue
		}
		out = append(out, cloneOnecPosting(p))
	}
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].PostingID < out[j].PostingID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryOnecPostingRepository) findLocked(supplierID, postingIDOrDocKey string) (OnecPosting, bool) {
	sid := strings.TrimSpace(supplierID)
	key := strings.TrimSpace(postingIDOrDocKey)
	if p, ok := r.byID[key]; ok && p.SupplierID == sid {
		return p, true
	}
	if pid, ok := r.byDoc[onecMemDocKey(sid, key)]; ok {
		if p, exists := r.byID[pid]; exists && p.SupplierID == sid {
			return p, true
		}
	}
	return OnecPosting{}, false
}

func (r *MemoryOnecPostingRepository) Ack(_ context.Context, supplierID, postingIDOrDocKey, onecRef string, now time.Time) (OnecPosting, error) {
	if strings.TrimSpace(supplierID) == "" || strings.TrimSpace(postingIDOrDocKey) == "" {
		return OnecPosting{}, ErrInvalidOnecInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ref := truncateOnecField(onecRef, maxOnecRefLen)
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.findLocked(supplierID, postingIDOrDocKey)
	if !ok {
		return OnecPosting{}, ErrOnecPostingNotFound
	}
	if p.Status == OnecStatusAcked {
		if ref != "" && p.OnecRef == "" {
			p.OnecRef = ref
			p.UpdatedAt = now.UTC()
			r.byID[p.PostingID] = cloneOnecPosting(p)
		}
		return cloneOnecPosting(p), nil
	}
	ts := now.UTC()
	p.Status = OnecStatusAcked
	if ref != "" {
		p.OnecRef = ref
	}
	p.RejectReason = ""
	p.AckedAt = &ts
	p.UpdatedAt = ts
	r.byID[p.PostingID] = cloneOnecPosting(p)
	return cloneOnecPosting(p), nil
}

func (r *MemoryOnecPostingRepository) Reject(_ context.Context, supplierID, postingIDOrDocKey, reason string, exposureMinor *int64, now time.Time) (OnecPosting, error) {
	if strings.TrimSpace(supplierID) == "" || strings.TrimSpace(postingIDOrDocKey) == "" {
		return OnecPosting{}, ErrInvalidOnecInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cleanReason := truncateOnecField(reason, maxOnecReasonLen)
	if cleanReason == "" {
		cleanReason = "rejected_by_1c"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.findLocked(supplierID, postingIDOrDocKey)
	if !ok {
		return OnecPosting{}, ErrOnecPostingNotFound
	}
	ts := now.UTC()
	wasRejectedSame := p.Status == OnecStatusRejected && p.RejectReason == cleanReason
	if !wasRejectedSame {
		p.Attempts++
	}
	p.Status = OnecStatusRejected
	p.RejectReason = cleanReason
	if exposureMinor != nil && *exposureMinor >= 0 {
		p.ExposureMinor = *exposureMinor
	}
	p.RejectedAt = &ts
	p.UpdatedAt = ts
	r.byID[p.PostingID] = cloneOnecPosting(p)
	return cloneOnecPosting(p), nil
}

func (r *MemoryOnecPostingRepository) Requeue(_ context.Context, supplierID, postingIDOrDocKey string, now time.Time) (OnecPosting, error) {
	if strings.TrimSpace(supplierID) == "" || strings.TrimSpace(postingIDOrDocKey) == "" {
		return OnecPosting{}, ErrInvalidOnecInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.findLocked(supplierID, postingIDOrDocKey)
	if !ok {
		return OnecPosting{}, ErrOnecPostingNotFound
	}
	p.Status = OnecStatusPending
	p.RejectReason = ""
	p.UpdatedAt = now.UTC()
	r.byID[p.PostingID] = cloneOnecPosting(p)
	return cloneOnecPosting(p), nil
}

func (r *MemoryOnecPostingRepository) Delete(_ context.Context, supplierID, postingID string) error {
	sid := strings.TrimSpace(supplierID)
	pid := strings.TrimSpace(postingID)
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byID[pid]
	if !ok || p.SupplierID != sid {
		return nil
	}
	delete(r.byID, pid)
	delete(r.byDoc, onecMemDocKey(sid, p.DocumentKey))
	return nil
}

// SpannerOnecPostingRepository persists PartnerOnecPostings in Spanner.
type SpannerOnecPostingRepository struct {
	client *spanner.Client
}

// NewSpannerOnecPostingRepository creates a Spanner-backed OnecPostingRepository.
func NewSpannerOnecPostingRepository(client *spanner.Client) *SpannerOnecPostingRepository {
	return &SpannerOnecPostingRepository{client: client}
}

const onecPostingSelectCols = `PostingId, SupplierId, DocumentKey, DocumentType, ExportJobId, PayloadJson, ExposureMinor, Status, OnecRef, RejectReason, Attempts, AckedAt, RejectedAt, CreatedAt, UpdatedAt`

func scanOnecPosting(row *spanner.Row) (OnecPosting, error) {
	var p OnecPosting
	var exportJobID, onecRef, rejectReason spanner.NullString
	var ackedAt, rejectedAt spanner.NullTime
	if err := row.Columns(
		&p.PostingID, &p.SupplierID, &p.DocumentKey, &p.DocumentType,
		&exportJobID, &p.PayloadJSON, &p.ExposureMinor, &p.Status,
		&onecRef, &rejectReason, &p.Attempts, &ackedAt, &rejectedAt,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return OnecPosting{}, err
	}
	p.ExportJobID = exportJobID.StringVal
	p.OnecRef = onecRef.StringVal
	p.RejectReason = rejectReason.StringVal
	if ackedAt.Valid {
		t := ackedAt.Time.UTC()
		p.AckedAt = &t
	}
	if rejectedAt.Valid {
		t := rejectedAt.Time.UTC()
		p.RejectedAt = &t
	}
	p.CreatedAt = p.CreatedAt.UTC()
	p.UpdatedAt = p.UpdatedAt.UTC()
	return p, nil
}

func (r *SpannerOnecPostingRepository) CreateIfAbsent(ctx context.Context, p OnecPosting) (OnecPosting, bool, error) {
	if r == nil || r.client == nil {
		return OnecPosting{}, false, fmt.Errorf("spanner_unavailable")
	}
	if err := prepareOnecPosting(&p, time.Now().UTC()); err != nil {
		return OnecPosting{}, false, err
	}
	var out OnecPosting
	var created bool
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		iter := txn.Query(ctx, spanner.Statement{
			SQL: `SELECT ` + onecPostingSelectCols + `
				FROM PartnerOnecPostings
				WHERE SupplierId = @sid AND DocumentKey = @dk
				LIMIT 1`,
			Params: map[string]any{"sid": p.SupplierID, "dk": p.DocumentKey},
		})
		row, err := iter.Next()
		iter.Stop()
		if err == nil {
			existing, scanErr := scanOnecPosting(row)
			if scanErr != nil {
				return scanErr
			}
			out = existing
			created = false
			return nil
		}
		if err != iterator.Done {
			return err
		}

		m := map[string]any{
			"PostingId":     p.PostingID,
			"SupplierId":    p.SupplierID,
			"DocumentKey":   p.DocumentKey,
			"DocumentType":  p.DocumentType,
			"ExportJobId":   nullableStr(p.ExportJobID),
			"PayloadJson":   p.PayloadJSON,
			"ExposureMinor": p.ExposureMinor,
			"Status":        p.Status,
			"OnecRef":       nullableStr(p.OnecRef),
			"RejectReason":  nullableStr(p.RejectReason),
			"Attempts":      p.Attempts,
			"CreatedAt":     spanner.CommitTimestamp,
			"UpdatedAt":     spanner.CommitTimestamp,
		}
		if p.AckedAt != nil {
			m["AckedAt"] = p.AckedAt.UTC()
		}
		if p.RejectedAt != nil {
			m["RejectedAt"] = p.RejectedAt.UTC()
		}
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("PartnerOnecPostings", m),
		}); err != nil {
			return err
		}
		out = p
		created = true
		return nil
	})
	if err != nil {
		return OnecPosting{}, false, err
	}
	return out, created, nil
}

func (r *SpannerOnecPostingRepository) Get(ctx context.Context, supplierID, postingID string) (OnecPosting, bool, error) {
	if r == nil || r.client == nil {
		return OnecPosting{}, false, fmt.Errorf("spanner_unavailable")
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ` + onecPostingSelectCols + `
			FROM PartnerOnecPostings
			WHERE SupplierId = @sid AND PostingId = @pid
			LIMIT 1`,
		Params: map[string]any{"sid": strings.TrimSpace(supplierID), "pid": strings.TrimSpace(postingID)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return OnecPosting{}, false, nil
	}
	if err != nil {
		return OnecPosting{}, false, err
	}
	p, err := scanOnecPosting(row)
	if err != nil {
		return OnecPosting{}, false, err
	}
	return p, true, nil
}

func (r *SpannerOnecPostingRepository) GetByDocumentKey(ctx context.Context, supplierID, documentKey string) (OnecPosting, bool, error) {
	if r == nil || r.client == nil {
		return OnecPosting{}, false, fmt.Errorf("spanner_unavailable")
	}
	iter := r.client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ` + onecPostingSelectCols + `
			FROM PartnerOnecPostings
			WHERE SupplierId = @sid AND DocumentKey = @dk
			LIMIT 1`,
		Params: map[string]any{"sid": strings.TrimSpace(supplierID), "dk": normalizeOnecDocKey(documentKey)},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return OnecPosting{}, false, nil
	}
	if err != nil {
		return OnecPosting{}, false, err
	}
	p, err := scanOnecPosting(row)
	if err != nil {
		return OnecPosting{}, false, err
	}
	return p, true, nil
}

func (r *SpannerOnecPostingRepository) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]OnecPosting, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner_unavailable")
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return nil, ErrInvalidOnecInput
	}
	if limit <= 0 {
		limit = defaultOnecListLim
	}
	if limit > maxOnecListLim {
		limit = maxOnecListLim
	}
	st := strings.ToUpper(strings.TrimSpace(status))
	var stmt spanner.Statement
	if st == "" || st == "ALL" {
		stmt = spanner.Statement{
			SQL: `SELECT ` + onecPostingSelectCols + `
				FROM PartnerOnecPostings
				WHERE SupplierId = @sid
				ORDER BY CreatedAt ASC, PostingId ASC
				LIMIT @lim`,
			Params: map[string]any{"sid": sid, "lim": int64(limit)},
		}
	} else {
		stmt = spanner.Statement{
			SQL: `SELECT ` + onecPostingSelectCols + `
				FROM PartnerOnecPostings
				WHERE SupplierId = @sid AND Status = @st
				ORDER BY CreatedAt ASC, PostingId ASC
				LIMIT @lim`,
			Params: map[string]any{"sid": sid, "st": st, "lim": int64(limit)},
		}
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]OnecPosting, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := scanOnecPosting(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func readOnecPostingInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, postingIDOrDocKey string) (OnecPosting, error) {
	sid := strings.TrimSpace(supplierID)
	id := strings.TrimSpace(postingIDOrDocKey)
	dk := normalizeOnecDocKey(id)
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT ` + onecPostingSelectCols + `
			FROM PartnerOnecPostings
			WHERE SupplierId = @sid AND (PostingId = @id OR DocumentKey = @dk)
			LIMIT 1`,
		Params: map[string]any{"sid": sid, "id": id, "dk": dk},
	})
	row, err := iter.Next()
	iter.Stop()
	if err == iterator.Done {
		return OnecPosting{}, ErrOnecPostingNotFound
	}
	if err != nil {
		return OnecPosting{}, err
	}
	return scanOnecPosting(row)
}

func (r *SpannerOnecPostingRepository) Ack(ctx context.Context, supplierID, postingIDOrDocKey, onecRef string, now time.Time) (OnecPosting, error) {
	if r == nil || r.client == nil {
		return OnecPosting{}, fmt.Errorf("spanner_unavailable")
	}
	if strings.TrimSpace(supplierID) == "" || strings.TrimSpace(postingIDOrDocKey) == "" {
		return OnecPosting{}, ErrInvalidOnecInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ref := truncateOnecField(onecRef, maxOnecRefLen)
	var out OnecPosting
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cur, err := readOnecPostingInTxn(ctx, txn, supplierID, postingIDOrDocKey)
		if err != nil {
			return err
		}
		if cur.Status == OnecStatusAcked {
			if ref != "" && cur.OnecRef == "" {
				cur.OnecRef = ref
				cur.UpdatedAt = now.UTC()
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.UpdateMap("PartnerOnecPostings", map[string]any{
						"PostingId": cur.PostingID,
						"OnecRef":   ref,
						"UpdatedAt": spanner.CommitTimestamp,
					}),
				}); err != nil {
					return err
				}
			}
			out = cur
			return nil
		}
		ts := now.UTC()
		cur.Status = OnecStatusAcked
		if ref != "" {
			cur.OnecRef = ref
		}
		cur.RejectReason = ""
		cur.AckedAt = &ts
		cur.UpdatedAt = ts
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("PartnerOnecPostings", map[string]any{
				"PostingId":    cur.PostingID,
				"Status":       OnecStatusAcked,
				"OnecRef":      nullableStr(cur.OnecRef),
				"RejectReason": nil,
				"AckedAt":      ts,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		out = cur
		return nil
	})
	if err != nil {
		return OnecPosting{}, err
	}
	return out, nil
}

func (r *SpannerOnecPostingRepository) Reject(ctx context.Context, supplierID, postingIDOrDocKey, reason string, exposureMinor *int64, now time.Time) (OnecPosting, error) {
	if r == nil || r.client == nil {
		return OnecPosting{}, fmt.Errorf("spanner_unavailable")
	}
	if strings.TrimSpace(supplierID) == "" || strings.TrimSpace(postingIDOrDocKey) == "" {
		return OnecPosting{}, ErrInvalidOnecInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cleanReason := truncateOnecField(reason, maxOnecReasonLen)
	if cleanReason == "" {
		cleanReason = "rejected_by_1c"
	}
	var out OnecPosting
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cur, err := readOnecPostingInTxn(ctx, txn, supplierID, postingIDOrDocKey)
		if err != nil {
			return err
		}
		ts := now.UTC()
		if !(cur.Status == OnecStatusRejected && cur.RejectReason == cleanReason) {
			cur.Attempts++
		}
		cur.Status = OnecStatusRejected
		cur.RejectReason = cleanReason
		if exposureMinor != nil && *exposureMinor >= 0 {
			cur.ExposureMinor = *exposureMinor
		}
		cur.RejectedAt = &ts
		cur.UpdatedAt = ts
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("PartnerOnecPostings", map[string]any{
				"PostingId":     cur.PostingID,
				"Status":        OnecStatusRejected,
				"RejectReason":  cleanReason,
				"ExposureMinor": cur.ExposureMinor,
				"Attempts":      cur.Attempts,
				"RejectedAt":    ts,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		out = cur
		return nil
	})
	if err != nil {
		return OnecPosting{}, err
	}
	return out, nil
}

func (r *SpannerOnecPostingRepository) Requeue(ctx context.Context, supplierID, postingIDOrDocKey string, now time.Time) (OnecPosting, error) {
	if r == nil || r.client == nil {
		return OnecPosting{}, fmt.Errorf("spanner_unavailable")
	}
	if strings.TrimSpace(supplierID) == "" || strings.TrimSpace(postingIDOrDocKey) == "" {
		return OnecPosting{}, ErrInvalidOnecInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var out OnecPosting
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cur, err := readOnecPostingInTxn(ctx, txn, supplierID, postingIDOrDocKey)
		if err != nil {
			return err
		}
		cur.Status = OnecStatusPending
		cur.RejectReason = ""
		cur.UpdatedAt = now.UTC()
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("PartnerOnecPostings", map[string]any{
				"PostingId":    cur.PostingID,
				"Status":       OnecStatusPending,
				"RejectReason": nil,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		out = cur
		return nil
	})
	if err != nil {
		return OnecPosting{}, err
	}
	return out, nil
}

func (r *SpannerOnecPostingRepository) Delete(ctx context.Context, supplierID, postingID string) error {
	if r == nil || r.client == nil {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		cur, err := readOnecPostingInTxn(ctx, txn, supplierID, postingID)
		if errors.Is(err, ErrOnecPostingNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("PartnerOnecPostings", spanner.Key{cur.PostingID}),
		})
	})
	return err
}
