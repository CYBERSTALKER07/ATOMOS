package entityresolution

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

const (
	DedupStatusClean              = "CLEAN"
	DedupStatusDuplicateCandidate = "DUPLICATE_CANDIDATE"
	DedupStatusMerged             = "MERGED"

	defaultRetailerLinksLimit = 100
	maxRetailerLinksLimit     = 500
	maxLinkFieldLen           = 32
	maxLinkLegalNameLen       = 255
)

var (
	ErrInvalidRetailerLinkInput  = errors.New("invalid_retailer_link_input")
	ErrRetailerLinkNotFound      = errors.New("retailer_link_not_found")
	ErrRetailerMergeTargetNotFound = errors.New("retailer_merge_target_not_found")
)

// SupplierRetailerLink mirrors one row of SupplierRetailerLinks scoped strictly to SupplierId.
type SupplierRetailerLink struct {
	SupplierID      string    `json:"supplier_id"`
	RetailerID      string    `json:"retailer_id"`
	TaxIDTIN        string    `json:"tax_id_tin,omitempty"`
	NormalizedPhone string    `json:"normalized_phone,omitempty"`
	LegalName       string    `json:"legal_name,omitempty"`
	DedupClusterID  string    `json:"dedup_cluster_id,omitempty"`
	DedupStatus     string    `json:"dedup_status"`
	MergedIntoID    string    `json:"merged_into_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// RetailerLinkRepository manages tenant-scoped SupplierRetailerLinks records and dedup clusters.
type RetailerLinkRepository interface {
	Upsert(ctx context.Context, link SupplierRetailerLink, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error)
	Get(ctx context.Context, supplierID, retailerID string) (SupplierRetailerLink, bool, error)
	List(ctx context.Context, supplierID, dedupStatus string, limit int) ([]SupplierRetailerLink, error)
	Resolve(ctx context.Context, supplierID, retailerID, action, mergedIntoID string, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error)
}

func digitsOnly(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeTIN deterministically normalizes tax IDs (INN/STIR/PINFL) for exact tenant-scoped matching.
func NormalizeTIN(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if d := digitsOnly(trimmed); d != "" {
		if len(d) > maxLinkFieldLen {
			return d[:maxLinkFieldLen]
		}
		return d
	}
	var b strings.Builder
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	out := b.String()
	if len(out) > maxLinkFieldLen {
		return out[:maxLinkFieldLen]
	}
	return out
}

// NormalizeUZPhone deterministically normalizes Uzbekistan (+998...) and international dial numbers.
func NormalizeUZPhone(raw string) string {
	d := digitsOnly(raw)
	if d == "" {
		return ""
	}
	var normalized string
	switch {
	case len(d) == 14 && strings.HasPrefix(d, "00998"):
		normalized = "+" + d[2:]
	case len(d) == 13 && strings.HasPrefix(d, "0998"):
		normalized = "+" + d[1:]
	case len(d) == 12 && strings.HasPrefix(d, "998"):
		normalized = "+" + d
	case len(d) == 10 && strings.HasPrefix(d, "0"):
		normalized = "+998" + d[1:]
	case len(d) == 9:
		normalized = "+998" + d
	default:
		normalized = "+" + d
	}
	if len(normalized) > maxLinkFieldLen {
		return normalized[:maxLinkFieldLen]
	}
	return normalized
}

func truncateLegalName(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) <= maxLinkLegalNameLen {
		return s
	}
	return s[:maxLinkLegalNameLen]
}

// ApplyRetailerLinkUpsert computes the deterministic normalized row and any peer dedup-cluster updates.
// activePeers must already be scoped to the same SupplierId, exclude incoming.RetailerID, and exclude MERGED rows.
func ApplyRetailerLinkUpsert(
	now time.Time,
	incoming SupplierRetailerLink,
	existing *SupplierRetailerLink,
	activePeers []SupplierRetailerLink,
	oldClusterPeers []SupplierRetailerLink,
) (SupplierRetailerLink, []SupplierRetailerLink) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	out := SupplierRetailerLink{
		SupplierID:      strings.TrimSpace(incoming.SupplierID),
		RetailerID:      strings.TrimSpace(incoming.RetailerID),
		TaxIDTIN:        NormalizeTIN(incoming.TaxIDTIN),
		NormalizedPhone: NormalizeUZPhone(incoming.NormalizedPhone),
		LegalName:       truncateLegalName(incoming.LegalName),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	oldClusterID := ""
	if existing != nil {
		if !existing.CreatedAt.IsZero() {
			out.CreatedAt = existing.CreatedAt.UTC()
		}
		if out.TaxIDTIN == "" {
			out.TaxIDTIN = NormalizeTIN(existing.TaxIDTIN)
		}
		if out.NormalizedPhone == "" {
			out.NormalizedPhone = NormalizeUZPhone(existing.NormalizedPhone)
		}
		if out.LegalName == "" {
			out.LegalName = truncateLegalName(existing.LegalName)
		}
		oldClusterID = strings.TrimSpace(existing.DedupClusterID)
	}

	matched := make([]SupplierRetailerLink, 0)
	for _, peer := range activePeers {
		if peer.SupplierID != out.SupplierID || peer.RetailerID == out.RetailerID {
			continue
		}
		if strings.ToUpper(strings.TrimSpace(peer.DedupStatus)) == DedupStatusMerged {
			continue
		}
		tinMatch := out.TaxIDTIN != "" && NormalizeTIN(peer.TaxIDTIN) == out.TaxIDTIN
		phoneMatch := out.NormalizedPhone != "" && NormalizeUZPhone(peer.NormalizedPhone) == out.NormalizedPhone
		if tinMatch || phoneMatch {
			matched = append(matched, peer)
		}
	}

	updatedPeers := make([]SupplierRetailerLink, 0)
	if len(matched) > 0 {
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].RetailerID < matched[j].RetailerID
		})
		clusterID := ""
		for _, peer := range matched {
			if cid := strings.TrimSpace(peer.DedupClusterID); cid != "" {
				clusterID = cid
				break
			}
		}
		if clusterID == "" {
			clusterID = oldClusterID
		}
		if clusterID == "" {
			clusterID = uuid.NewString()
		}
		out.DedupClusterID = clusterID
		out.DedupStatus = DedupStatusDuplicateCandidate
		out.MergedIntoID = ""

		for _, peer := range matched {
			if strings.TrimSpace(peer.DedupClusterID) == clusterID && strings.ToUpper(strings.TrimSpace(peer.DedupStatus)) == DedupStatusDuplicateCandidate {
				continue
			}
			peer.DedupClusterID = clusterID
			peer.DedupStatus = DedupStatusDuplicateCandidate
			peer.UpdatedAt = now
			updatedPeers = append(updatedPeers, peer)
		}
		return out, updatedPeers
	}

	out.DedupClusterID = ""
	out.DedupStatus = DedupStatusClean
	out.MergedIntoID = ""

	if oldClusterID != "" {
		remaining := filterRemainingClusterCandidates(oldClusterPeers, out.SupplierID, out.RetailerID, oldClusterID)
		if len(remaining) == 1 {
			sole := remaining[0]
			sole.DedupClusterID = ""
			sole.DedupStatus = DedupStatusClean
			sole.UpdatedAt = now
			updatedPeers = append(updatedPeers, sole)
		}
	}
	return out, updatedPeers
}

// ApplyRetailerLinkResolution marks a candidate merged or clean and dissolves single-member orphan clusters.
func ApplyRetailerLinkResolution(
	now time.Time,
	action string,
	mergedIntoID string,
	target SupplierRetailerLink,
	targetMergeExists bool,
	clusterPeers []SupplierRetailerLink,
) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	act := strings.ToLower(strings.TrimSpace(action))
	oldClusterID := strings.TrimSpace(target.DedupClusterID)
	updated := target
	updated.UpdatedAt = now

	switch act {
	case "merge_into", "merge":
		mid := strings.TrimSpace(mergedIntoID)
		if mid == "" || mid == target.RetailerID || !targetMergeExists {
			return SupplierRetailerLink{}, nil, ErrRetailerMergeTargetNotFound
		}
		updated.DedupStatus = DedupStatusMerged
		updated.MergedIntoID = mid
	case "mark_clean", "clean":
		updated.DedupStatus = DedupStatusClean
		updated.DedupClusterID = ""
		updated.MergedIntoID = ""
	default:
		return SupplierRetailerLink{}, nil, fmt.Errorf("%w: unsupported action %q", ErrInvalidRetailerLinkInput, action)
	}

	var updatedPeers []SupplierRetailerLink
	if oldClusterID != "" {
		remaining := filterRemainingClusterCandidates(clusterPeers, target.SupplierID, target.RetailerID, oldClusterID)
		if len(remaining) == 1 {
			sole := remaining[0]
			sole.DedupClusterID = ""
			sole.DedupStatus = DedupStatusClean
			sole.UpdatedAt = now
			updatedPeers = append(updatedPeers, sole)
		}
	}
	return updated, updatedPeers, nil
}

func filterRemainingClusterCandidates(peers []SupplierRetailerLink, supplierID, excludeRetailerID, clusterID string) []SupplierRetailerLink {
	out := make([]SupplierRetailerLink, 0, len(peers))
	for _, peer := range peers {
		if peer.SupplierID != supplierID || peer.RetailerID == excludeRetailerID {
			continue
		}
		if strings.TrimSpace(peer.DedupClusterID) != clusterID {
			continue
		}
		if strings.ToUpper(strings.TrimSpace(peer.DedupStatus)) != DedupStatusDuplicateCandidate {
			continue
		}
		out = append(out, peer)
	}
	return out
}

// MemoryRetailerLinkRepository stores SupplierRetailerLinks strictly partitioned by SupplierId.
type MemoryRetailerLinkRepository struct {
	mu   sync.RWMutex
	rows map[string]map[string]SupplierRetailerLink
}

// NewMemoryRetailerLinkRepository returns an empty tenant-safe in-memory repository.
func NewMemoryRetailerLinkRepository() *MemoryRetailerLinkRepository {
	return &MemoryRetailerLinkRepository{
		rows: make(map[string]map[string]SupplierRetailerLink),
	}
}

func (r *MemoryRetailerLinkRepository) Upsert(_ context.Context, link SupplierRetailerLink, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	sid := strings.TrimSpace(link.SupplierID)
	rid := strings.TrimSpace(link.RetailerID)
	if sid == "" || rid == "" {
		return SupplierRetailerLink{}, nil, ErrInvalidRetailerLinkInput
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, ok := r.rows[sid]
	if !ok {
		bucket = make(map[string]SupplierRetailerLink)
		r.rows[sid] = bucket
	}

	var existingPtr *SupplierRetailerLink
	if ex, found := bucket[rid]; found {
		cp := ex
		existingPtr = &cp
	}

	peers := make([]SupplierRetailerLink, 0, len(bucket))
	for k, v := range bucket {
		if k != rid {
			peers = append(peers, v)
		}
	}

	saved, updatedPeers := ApplyRetailerLinkUpsert(now, link, existingPtr, peers, peers)
	bucket[rid] = saved
	for _, peer := range updatedPeers {
		bucket[peer.RetailerID] = peer
	}
	return saved, updatedPeers, nil
}

func (r *MemoryRetailerLinkRepository) Get(_ context.Context, supplierID, retailerID string) (SupplierRetailerLink, bool, error) {
	sid := strings.TrimSpace(supplierID)
	rid := strings.TrimSpace(retailerID)
	if sid == "" || rid == "" {
		return SupplierRetailerLink{}, false, ErrInvalidRetailerLinkInput
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	bucket := r.rows[sid]
	if bucket == nil {
		return SupplierRetailerLink{}, false, nil
	}
	row, ok := bucket[rid]
	return row, ok, nil
}

func (r *MemoryRetailerLinkRepository) List(_ context.Context, supplierID, dedupStatus string, limit int) ([]SupplierRetailerLink, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return nil, ErrInvalidRetailerLinkInput
	}
	if limit <= 0 {
		limit = defaultRetailerLinksLimit
	}
	if limit > maxRetailerLinksLimit {
		limit = maxRetailerLinksLimit
	}
	filterStatus := strings.ToUpper(strings.TrimSpace(dedupStatus))

	r.mu.RLock()
	bucket := r.rows[sid]
	out := make([]SupplierRetailerLink, 0, len(bucket))
	for _, row := range bucket {
		if filterStatus != "" && strings.ToUpper(strings.TrimSpace(row.DedupStatus)) != filterStatus {
			continue
		}
		out = append(out, row)
	}
	r.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].RetailerID < out[j].RetailerID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *MemoryRetailerLinkRepository) Resolve(_ context.Context, supplierID, retailerID, action, mergedIntoID string, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	sid := strings.TrimSpace(supplierID)
	rid := strings.TrimSpace(retailerID)
	if sid == "" || rid == "" {
		return SupplierRetailerLink{}, nil, ErrInvalidRetailerLinkInput
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	bucket := r.rows[sid]
	if bucket == nil {
		return SupplierRetailerLink{}, nil, ErrRetailerLinkNotFound
	}
	target, ok := bucket[rid]
	if !ok {
		return SupplierRetailerLink{}, nil, ErrRetailerLinkNotFound
	}
	mid := strings.TrimSpace(mergedIntoID)
	_, mergeExists := bucket[mid]

	peers := make([]SupplierRetailerLink, 0, len(bucket))
	for k, v := range bucket {
		if k != rid {
			peers = append(peers, v)
		}
	}

	updated, updatedPeers, err := ApplyRetailerLinkResolution(now, action, mid, target, mergeExists, peers)
	if err != nil {
		return SupplierRetailerLink{}, nil, err
	}
	bucket[rid] = updated
	for _, peer := range updatedPeers {
		bucket[peer.RetailerID] = peer
	}
	return updated, updatedPeers, nil
}

// SpannerRetailerLinkRepository persists SupplierRetailerLinks strictly scoped to SupplierId.
type SpannerRetailerLinkRepository struct {
	client *spanner.Client
}

// NewSpannerRetailerLinkRepository constructs a Spanner-backed RetailerLinkRepository.
func NewSpannerRetailerLinkRepository(client *spanner.Client) *SpannerRetailerLinkRepository {
	return &SpannerRetailerLinkRepository{client: client}
}

const retailerLinkSelectCols = `SupplierId, RetailerId, TaxIdTIN, NormalizedPhone, LegalName, DedupClusterId, DedupStatus, MergedIntoId, CreatedAt, UpdatedAt`

func scanRetailerLink(row *spanner.Row) (SupplierRetailerLink, error) {
	var link SupplierRetailerLink
	var tin, phone, legalName, clusterID, mergedInto spanner.NullString
	if err := row.Columns(
		&link.SupplierID,
		&link.RetailerID,
		&tin,
		&phone,
		&legalName,
		&clusterID,
		&link.DedupStatus,
		&mergedInto,
		&link.CreatedAt,
		&link.UpdatedAt,
	); err != nil {
		return SupplierRetailerLink{}, err
	}
	link.TaxIDTIN = tin.StringVal
	link.NormalizedPhone = phone.StringVal
	link.LegalName = legalName.StringVal
	link.DedupClusterID = clusterID.StringVal
	link.MergedIntoID = mergedInto.StringVal
	link.CreatedAt = link.CreatedAt.UTC()
	link.UpdatedAt = link.UpdatedAt.UTC()
	return link, nil
}

func nullLinkString(s string) spanner.NullString {
	trimmed := strings.TrimSpace(s)
	return spanner.NullString{StringVal: trimmed, Valid: trimmed != ""}
}

func retailerLinkUpdateMutation(link SupplierRetailerLink) *spanner.Mutation {
	return spanner.UpdateMap("SupplierRetailerLinks", map[string]any{
		"SupplierId":      link.SupplierID,
		"RetailerId":      link.RetailerID,
		"TaxIdTIN":        nullLinkString(link.TaxIDTIN),
		"NormalizedPhone": nullLinkString(link.NormalizedPhone),
		"LegalName":       nullLinkString(link.LegalName),
		"DedupClusterId":  nullLinkString(link.DedupClusterID),
		"DedupStatus":     link.DedupStatus,
		"MergedIntoId":    nullLinkString(link.MergedIntoID),
		"UpdatedAt":       spanner.CommitTimestamp,
	})
}

func retailerLinkInsertMutation(link SupplierRetailerLink) *spanner.Mutation {
	return spanner.InsertMap("SupplierRetailerLinks", map[string]any{
		"SupplierId":      link.SupplierID,
		"RetailerId":      link.RetailerID,
		"TaxIdTIN":        nullLinkString(link.TaxIDTIN),
		"NormalizedPhone": nullLinkString(link.NormalizedPhone),
		"LegalName":       nullLinkString(link.LegalName),
		"DedupClusterId":  nullLinkString(link.DedupClusterID),
		"DedupStatus":     link.DedupStatus,
		"MergedIntoId":    nullLinkString(link.MergedIntoID),
		"CreatedAt":       spanner.CommitTimestamp,
		"UpdatedAt":       spanner.CommitTimestamp,
	})
}

func querySpannerRetailerLinks(ctx context.Context, queryer interface {
	Query(context.Context, spanner.Statement) *spanner.RowIterator
}, stmt spanner.Statement) ([]SupplierRetailerLink, error) {
	iter := queryer.Query(ctx, stmt)
	defer iter.Stop()
	out := make([]SupplierRetailerLink, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		item, err := scanRetailerLink(row)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
}

func (r *SpannerRetailerLinkRepository) Upsert(ctx context.Context, link SupplierRetailerLink, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	if r == nil || r.client == nil {
		return SupplierRetailerLink{}, nil, fmt.Errorf("spanner_unavailable")
	}
	sid := strings.TrimSpace(link.SupplierID)
	rid := strings.TrimSpace(link.RetailerID)
	if sid == "" || rid == "" {
		return SupplierRetailerLink{}, nil, ErrInvalidRetailerLinkInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	var saved SupplierRetailerLink
	var updatedPeers []SupplierRetailerLink

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		existingRows, err := querySpannerRetailerLinks(ctx, txn, spanner.Statement{
			SQL: `SELECT ` + retailerLinkSelectCols + `
				FROM SupplierRetailerLinks
				WHERE SupplierId = @sid AND RetailerId = @rid
				LIMIT 1`,
			Params: map[string]any{"sid": sid, "rid": rid},
		})
		if err != nil {
			return err
		}

		var existingPtr *SupplierRetailerLink
		if len(existingRows) > 0 {
			cp := existingRows[0]
			existingPtr = &cp
		}

		effTIN := NormalizeTIN(link.TaxIDTIN)
		effPhone := NormalizeUZPhone(link.NormalizedPhone)
		if existingPtr != nil {
			if effTIN == "" {
				effTIN = NormalizeTIN(existingPtr.TaxIDTIN)
			}
			if effPhone == "" {
				effPhone = NormalizeUZPhone(existingPtr.NormalizedPhone)
			}
		}

		var activePeers []SupplierRetailerLink
		if effTIN != "" || effPhone != "" {
			activePeers, err = querySpannerRetailerLinks(ctx, txn, spanner.Statement{
				SQL: `SELECT ` + retailerLinkSelectCols + `
					FROM SupplierRetailerLinks
					WHERE SupplierId = @sid
					  AND RetailerId != @rid
					  AND DedupStatus != 'MERGED'
					  AND (
					    (@tin != '' AND TaxIdTIN = @tin)
					    OR (@phone != '' AND NormalizedPhone = @phone)
					  )`,
				Params: map[string]any{
					"sid":   sid,
					"rid":   rid,
					"tin":   effTIN,
					"phone": effPhone,
				},
			})
			if err != nil {
				return err
			}
		}

		var oldClusterPeers []SupplierRetailerLink
		if existingPtr != nil && strings.TrimSpace(existingPtr.DedupClusterID) != "" {
			oldClusterPeers, err = querySpannerRetailerLinks(ctx, txn, spanner.Statement{
				SQL: `SELECT ` + retailerLinkSelectCols + `
					FROM SupplierRetailerLinks
					WHERE SupplierId = @sid
					  AND DedupClusterId = @cid
					  AND RetailerId != @rid
					  AND DedupStatus = 'DUPLICATE_CANDIDATE'`,
				Params: map[string]any{
					"sid": sid,
					"rid": rid,
					"cid": strings.TrimSpace(existingPtr.DedupClusterID),
				},
			})
			if err != nil {
				return err
			}
		}

		saved, updatedPeers = ApplyRetailerLinkUpsert(now, link, existingPtr, activePeers, oldClusterPeers)
		muts := make([]*spanner.Mutation, 0, 1+len(updatedPeers))
		if existingPtr == nil {
			muts = append(muts, retailerLinkInsertMutation(saved))
		} else {
			muts = append(muts, retailerLinkUpdateMutation(saved))
		}
		for _, peer := range updatedPeers {
			muts = append(muts, retailerLinkUpdateMutation(peer))
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return SupplierRetailerLink{}, nil, err
	}
	return saved, updatedPeers, nil
}

func (r *SpannerRetailerLinkRepository) Get(ctx context.Context, supplierID, retailerID string) (SupplierRetailerLink, bool, error) {
	if r == nil || r.client == nil {
		return SupplierRetailerLink{}, false, fmt.Errorf("spanner_unavailable")
	}
	sid := strings.TrimSpace(supplierID)
	rid := strings.TrimSpace(retailerID)
	if sid == "" || rid == "" {
		return SupplierRetailerLink{}, false, ErrInvalidRetailerLinkInput
	}
	rows, err := querySpannerRetailerLinks(ctx, r.client.Single(), spanner.Statement{
		SQL: `SELECT ` + retailerLinkSelectCols + `
			FROM SupplierRetailerLinks
			WHERE SupplierId = @sid AND RetailerId = @rid
			LIMIT 1`,
		Params: map[string]any{"sid": sid, "rid": rid},
	})
	if err != nil {
		return SupplierRetailerLink{}, false, err
	}
	if len(rows) == 0 {
		return SupplierRetailerLink{}, false, nil
	}
	return rows[0], true, nil
}

func (r *SpannerRetailerLinkRepository) List(ctx context.Context, supplierID, dedupStatus string, limit int) ([]SupplierRetailerLink, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner_unavailable")
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return nil, ErrInvalidRetailerLinkInput
	}
	if limit <= 0 {
		limit = defaultRetailerLinksLimit
	}
	if limit > maxRetailerLinksLimit {
		limit = maxRetailerLinksLimit
	}
	st := strings.ToUpper(strings.TrimSpace(dedupStatus))
	if st == "" || st == "ALL" {
		return querySpannerRetailerLinks(ctx, r.client.Single(), spanner.Statement{
			SQL: `SELECT ` + retailerLinkSelectCols + `
				FROM SupplierRetailerLinks
				WHERE SupplierId = @sid
				ORDER BY UpdatedAt DESC, RetailerId ASC
				LIMIT @lim`,
			Params: map[string]any{"sid": sid, "lim": int64(limit)},
		})
	}
	return querySpannerRetailerLinks(ctx, r.client.Single(), spanner.Statement{
		SQL: `SELECT ` + retailerLinkSelectCols + `
			FROM SupplierRetailerLinks
			WHERE SupplierId = @sid AND DedupStatus = @st
			ORDER BY UpdatedAt DESC, RetailerId ASC
			LIMIT @lim`,
		Params: map[string]any{"sid": sid, "st": st, "lim": int64(limit)},
	})
}

func (r *SpannerRetailerLinkRepository) Resolve(ctx context.Context, supplierID, retailerID, action, mergedIntoID string, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	if r == nil || r.client == nil {
		return SupplierRetailerLink{}, nil, fmt.Errorf("spanner_unavailable")
	}
	sid := strings.TrimSpace(supplierID)
	rid := strings.TrimSpace(retailerID)
	if sid == "" || rid == "" {
		return SupplierRetailerLink{}, nil, ErrInvalidRetailerLinkInput
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	var updated SupplierRetailerLink
	var updatedPeers []SupplierRetailerLink
	mid := strings.TrimSpace(mergedIntoID)

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		targetRows, err := querySpannerRetailerLinks(ctx, txn, spanner.Statement{
			SQL: `SELECT ` + retailerLinkSelectCols + `
				FROM SupplierRetailerLinks
				WHERE SupplierId = @sid AND RetailerId = @rid
				LIMIT 1`,
			Params: map[string]any{"sid": sid, "rid": rid},
		})
		if err != nil {
			return err
		}
		if len(targetRows) == 0 {
			return ErrRetailerLinkNotFound
		}
		target := targetRows[0]

		mergeExists := false
		if strings.EqualFold(strings.TrimSpace(action), "merge_into") || strings.EqualFold(strings.TrimSpace(action), "merge") {
			if mid != "" && mid != rid {
				mergeRows, err := querySpannerRetailerLinks(ctx, txn, spanner.Statement{
					SQL: `SELECT ` + retailerLinkSelectCols + `
						FROM SupplierRetailerLinks
						WHERE SupplierId = @sid AND RetailerId = @mid
						LIMIT 1`,
					Params: map[string]any{"sid": sid, "mid": mid},
				})
				if err != nil {
					return err
				}
				mergeExists = len(mergeRows) > 0
			}
		}

		var clusterPeers []SupplierRetailerLink
		if strings.TrimSpace(target.DedupClusterID) != "" {
			clusterPeers, err = querySpannerRetailerLinks(ctx, txn, spanner.Statement{
				SQL: `SELECT ` + retailerLinkSelectCols + `
					FROM SupplierRetailerLinks
					WHERE SupplierId = @sid
					  AND DedupClusterId = @cid
					  AND RetailerId != @rid
					  AND DedupStatus = 'DUPLICATE_CANDIDATE'`,
				Params: map[string]any{
					"sid": sid,
					"rid": rid,
					"cid": strings.TrimSpace(target.DedupClusterID),
				},
			})
			if err != nil {
				return err
			}
		}

		var resErr error
		updated, updatedPeers, resErr = ApplyRetailerLinkResolution(now, action, mid, target, mergeExists, clusterPeers)
		if resErr != nil {
			return resErr
		}
		muts := make([]*spanner.Mutation, 0, 1+len(updatedPeers))
		muts = append(muts, retailerLinkUpdateMutation(updated))
		for _, peer := range updatedPeers {
			muts = append(muts, retailerLinkUpdateMutation(peer))
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return SupplierRetailerLink{}, nil, err
	}
	return updated, updatedPeers, nil
}

// UpsertSupplierRetailerLink normalizes TIN/phone and persists a tenant-scoped SupplierRetailerLinks row.
func UpsertSupplierRetailerLink(ctx context.Context, client *spanner.Client, link SupplierRetailerLink, now time.Time) (SupplierRetailerLink, []SupplierRetailerLink, error) {
	return NewSpannerRetailerLinkRepository(client).Upsert(ctx, link, now)
}
