package credit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ScoreHistoryRepository persists CreditScoreHistory rows, manages network consent & dispute metrics,
// and writes governed limit changes + CreditLimitChanged/CreditHold events atomically.
type ScoreHistoryRepository interface {
	SaveScoreHistory(ctx context.Context, rec CreditScoreRecord, updateLatestView bool) error
	ListScoreHistory(ctx context.Context, retailerID, supplierID, scope string, limit int) ([]CreditScoreRecord, error)
	LoadNetworkInputs(ctx context.Context, retailerID string) (NetworkScoreInputs, error)
	LoadRelationshipMetrics(ctx context.Context, retailerID, supplierID string) (ScoreMetrics, bool, error)
	ApplyGovernedLimitChange(ctx context.Context, in ApplyLimitRepoInput) (Profile, error)
}

// ConsentingSupplierProfile represents one supplier-retailer relationship with verified consent.
type ConsentingSupplierProfile struct {
	Profile         Profile
	Terms           RetailerPaymentTerms
	Metrics         ScoreMetrics
	ConsentVerified bool
}

// NetworkScoreInputs holds cross-supplier consenting data & isolated disputes for network scoring.
type NetworkScoreInputs struct {
	RetailerID            string
	ConsentingProfiles    []ConsentingSupplierProfile
	TotalConsentingCount  int64
	DisputesIsolatedCount int64
	DisputedAmountMinor   int64
}

// ApplyLimitRepoInput bundles fields for atomic governed limit change + outbox + audit.
type ApplyLimitRepoInput struct {
	RetailerID      string
	SupplierID      string
	NewLimitMinor   int64
	OldLimitMinor   int64
	ConsentVerified *bool
	ApprovalID      string
	ActorID         string
	ActorRole       string
	Reason          string
	ReasonCodesJSON string
	ScoreRecord     *CreditScoreRecord
	Now             time.Time
	AuditID         string
}

// MemoryScoreHistoryRepository implements ScoreHistoryRepository in memory for unit tests.
type MemoryScoreHistoryRepository struct {
	mu              sync.RWMutex
	history         map[string][]CreditScoreRecord // retailerID -> records (newest last)
	latest          map[string]RetailerCreditScore // retailerID -> latest view
	termsByKey      map[string]ConsentingSupplierProfile
	disputesByKey   map[string]int64 // supplierID|retailerID -> disputed invoice count
	disputeAmtByKey map[string]int64
	relMetricsByKey map[string]ScoreMetrics
	profilesRepo    Repository
	policyRepo      PolicyRepository
}

// NewMemoryScoreHistoryRepository builds an in-memory ScoreHistoryRepository.
func NewMemoryScoreHistoryRepository(profilesRepo Repository, policyRepo PolicyRepository) *MemoryScoreHistoryRepository {
	return &MemoryScoreHistoryRepository{
		history:         make(map[string][]CreditScoreRecord),
		latest:          make(map[string]RetailerCreditScore),
		termsByKey:      make(map[string]ConsentingSupplierProfile),
		disputesByKey:   make(map[string]int64),
		disputeAmtByKey: make(map[string]int64),
		relMetricsByKey: make(map[string]ScoreMetrics),
		profilesRepo:    profilesRepo,
		policyRepo:      policyRepo,
	}
}

// SeedConsentingRelationship seeds a supplier-retailer relationship for network scoring tests.
func (m *MemoryScoreHistoryRepository) SeedConsentingRelationship(p Profile, t RetailerPaymentTerms, metrics ScoreMetrics, consent bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.termsByKey[p.RetailerID+"|"+p.SupplierID] = ConsentingSupplierProfile{
		Profile:         p,
		Terms:           t,
		Metrics:         metrics,
		ConsentVerified: consent,
	}
	m.relMetricsByKey[p.RetailerID+"|"+p.SupplierID] = metrics
}

// SeedDisputes seeds supplier-specific disputed invoices for dispute-isolation tests.
func (m *MemoryScoreHistoryRepository) SeedDisputes(retailerID, supplierID string, count, amountMinor int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := retailerID + "|" + supplierID
	m.disputesByKey[k] = count
	m.disputeAmtByKey[k] = amountMinor
}

func (m *MemoryScoreHistoryRepository) SaveScoreHistory(_ context.Context, rec CreditScoreRecord, updateLatestView bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history[rec.RetailerID] = append(m.history[rec.RetailerID], rec)
	if updateLatestView {
		m.latest[rec.RetailerID] = RetailerCreditScore{
			RetailerID:          rec.RetailerID,
			Score:               rec.Score,
			PdBps:               rec.PdBps,
			RiskTier:            rec.RiskTier,
			SuggestedLimitMinor: rec.SuggestedLimitMinor,
			FactorsJSON:         rec.FactorsJSON,
			ReasonCodes:         rec.ReasonCodes,
			ReasonCodesJSON:     rec.ReasonCodesJSON,
			WindowStart:         rec.WindowStart,
			WindowEnd:           rec.WindowEnd,
			ComputedAt:          rec.ComputedAt,
		}
	}
	return nil
}

func (m *MemoryScoreHistoryRepository) ListScoreHistory(_ context.Context, retailerID, supplierID, scope string, limit int) ([]CreditScoreRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rows := m.history[retailerID]
	out := make([]CreditScoreRecord, 0, len(rows))
	for _, r := range rows {
		if supplierID != "" && r.SupplierID != supplierID {
			continue
		}
		if scope != "" && !strings.EqualFold(r.Scope, scope) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].ComputedAt.Equal(out[j].ComputedAt) {
			return out[i].ComputedAt.After(out[j].ComputedAt)
		}
		return out[i].ScoreID > out[j].ScoreID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *MemoryScoreHistoryRepository) LoadNetworkInputs(_ context.Context, retailerID string) (NetworkScoreInputs, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := NetworkScoreInputs{RetailerID: retailerID}
	for k, csp := range m.termsByKey {
		if !strings.HasPrefix(k, retailerID+"|") {
			continue
		}
		if !csp.ConsentVerified {
			continue
		}
		out.ConsentingProfiles = append(out.ConsentingProfiles, csp)
		out.TotalConsentingCount++
		if dc := m.disputesByKey[k]; dc > 0 {
			out.DisputesIsolatedCount += dc
			out.DisputedAmountMinor += m.disputeAmtByKey[k]
		}
	}
	return out, nil
}

func (m *MemoryScoreHistoryRepository) LoadRelationshipMetrics(_ context.Context, retailerID, supplierID string) (ScoreMetrics, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k := retailerID + "|" + supplierID
	sm, ok := m.relMetricsByKey[k]
	consent := false
	if csp, ok2 := m.termsByKey[k]; ok2 {
		consent = csp.ConsentVerified
		if !ok {
			sm = csp.Metrics
			ok = true
		}
	}
	if dc := m.disputesByKey[k]; dc > 0 {
		sm.DisputedInvoicesCount = dc
		sm.DisputedAmountMinor = m.disputeAmtByKey[k]
	}
	if !ok && m.policyRepo != nil {
		if prog, found, err := m.policyRepo.GetProgram(context.Background(), supplierID); err == nil && found && prog.GlobalDefaultLimitMinor > 0 {
			sm.StarterLimitMinor = prog.GlobalDefaultLimitMinor
		}
	}
	return sm, consent, nil
}

func (m *MemoryScoreHistoryRepository) ApplyGovernedLimitChange(ctx context.Context, in ApplyLimitRepoInput) (Profile, error) {
	if m.profilesRepo == nil {
		return Profile{}, ErrProfileNotFound
	}
	existing, found, err := m.profilesRepo.GetProfile(ctx, in.RetailerID, in.SupplierID)
	if err != nil {
		return Profile{}, err
	}
	if !found {
		existing = Profile{
			RetailerID:       in.RetailerID,
			SupplierID:       in.SupplierID,
			CreditLimitMinor: in.OldLimitMinor,
			Status:           StatusActive,
			CreatedAt:        in.Now,
		}
	}
	oldLimit := existing.CreditLimitMinor
	existing.CreditLimitMinor = in.NewLimitMinor
	existing.AvailableCreditMinor = existing.Available()
	existing.UpdatedAt = in.Now
	if err := m.profilesRepo.UpsertProfile(ctx, existing, func(txn outbox.TxnBuffer) error {
		if txn == nil {
			return nil
		}
		tsStr := in.Now.UTC().Format(time.RFC3339Nano)
		if err := outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, in.RetailerID, events.TopicMain, events.CreditLimitChangedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventCreditLimitChanged,
				Version:   1,
				Timestamp: tsStr,
			},
			RetailerID:         in.RetailerID,
			SupplierID:         in.SupplierID,
			PreviousLimitMinor: oldLimit,
			NewLimitMinor:      in.NewLimitMinor,
			ApprovalID:         in.ApprovalID,
			ActorID:            in.ActorID,
			Reason:             in.Reason,
			ReasonCodesJSON:    in.ReasonCodesJSON,
			OccurredAt:         tsStr,
		}); err != nil {
			return err
		}
		return outbox.EmitJSON(ctx, txn, events.AggregateCreditProfile, in.RetailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: tsStr},
			ProfileID:        profileID(in.RetailerID, in.SupplierID),
			RetailerID:       in.RetailerID,
			SupplierID:       in.SupplierID,
			CreditLimitMinor: existing.CreditLimitMinor,
			CurrentBalance:   existing.CurrentBalanceMinor,
			Delinquent:       existing.DelinquencyCount > 0,
			Reason:           "limit_changed:" + in.Reason,
		})
	}); err != nil {
		return Profile{}, err
	}
	if in.ConsentVerified != nil {
		m.mu.Lock()
		k := in.RetailerID + "|" + in.SupplierID
		csp := m.termsByKey[k]
		csp.Profile = existing
		csp.ConsentVerified = *in.ConsentVerified
		m.termsByKey[k] = csp
		m.mu.Unlock()
	}
	if in.ScoreRecord != nil {
		rec := *in.ScoreRecord
		rec.CurrentLimitMinor = in.NewLimitMinor
		_ = m.SaveScoreHistory(ctx, rec, false)
	}
	return existing, nil
}

// SpannerScoreHistoryRepo persists CreditScoreHistory, RetailerCreditScores, ConsentVerified,
// and governed limit changes atomically in Cloud Spanner using strong reads and RW transactions.
type SpannerScoreHistoryRepo struct {
	client *spanner.Client
}

// NewSpannerScoreHistoryRepo constructs a Spanner-backed ScoreHistoryRepository.
func NewSpannerScoreHistoryRepo(client *spanner.Client) *SpannerScoreHistoryRepo {
	return &SpannerScoreHistoryRepo{client: client}
}

func (r *SpannerScoreHistoryRepo) SaveScoreHistory(ctx context.Context, rec CreditScoreRecord, updateLatestView bool) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner client unavailable")
	}
	rcJSON := rec.ReasonCodesJSON
	if rcJSON == "" {
		b, _ := json.Marshal(rec.ReasonCodes)
		rcJSON = string(b)
	}
	if rcJSON == "" || rcJSON == "null" {
		rcJSON = "[]"
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"RetailerId":            rec.RetailerID,
			"ScoreId":               rec.ScoreID,
			"Scope":                 rec.Scope,
			"ConsentVerified":       rec.ConsentVerified,
			"ModelVersion":          rec.ModelVersion,
			"ScorerSource":          rec.ScorerSource,
			"Score":                 rec.Score,
			"PdBps":                 rec.PdBps,
			"RiskTier":              string(rec.RiskTier),
			"SuggestedLimitMinor":   rec.SuggestedLimitMinor,
			"CurrentLimitMinor":     rec.CurrentLimitMinor,
			"ReasonCodesJson":       rcJSON,
			"FeaturesHash":          rec.FeaturesHash,
			"DisputesIsolatedCount": rec.DisputesIsolatedCount,
			"WindowStart":           rec.WindowStart.UTC(),
			"WindowEnd":             rec.WindowEnd.UTC(),
			"ComputedAt":            spanner.CommitTimestamp,
			"CreatedAt":             spanner.CommitTimestamp,
		}
		if strings.TrimSpace(rec.SupplierID) != "" {
			row["SupplierId"] = rec.SupplierID
		}
		if strings.TrimSpace(rec.ConsentScope) != "" {
			row["ConsentScope"] = rec.ConsentScope
		}
		if strings.TrimSpace(rec.FactorsJSON) != "" {
			row["FactorsJson"] = rec.FactorsJSON
		}
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("CreditScoreHistory", row),
		}
		if updateLatestView {
			var factorsAny any
			if strings.TrimSpace(rec.FactorsJSON) != "" && json.Valid([]byte(rec.FactorsJSON)) {
				factorsAny = spanner.NullJSON{Valid: true, Value: json.RawMessage(rec.FactorsJSON)}
			}
			muts = append(muts, spanner.InsertOrUpdateMap("RetailerCreditScores", map[string]any{
				"RetailerId":          rec.RetailerID,
				"Score":               rec.Score,
				"RiskTier":            string(rec.RiskTier),
				"SuggestedLimitMinor": rec.SuggestedLimitMinor,
				"FactorsJson":         factorsAny,
				"WindowStart":         rec.WindowStart.UTC(),
				"WindowEnd":           rec.WindowEnd.UTC(),
				"ComputedAt":          rec.ComputedAt.UTC(),
			}))
		}
		if rec.Scope == ScoreScopeRelationship && strings.TrimSpace(rec.SupplierID) != "" {
			// Best-effort update profile RiskScore / LastEvaluatedAt if profile row exists.
			if _, pErr := txn.ReadRow(txCtx, "RetailerCreditProfiles", spanner.Key{rec.RetailerID, rec.SupplierID}, []string{"Version"}); pErr == nil {
				evalAt := rec.ComputedAt.UTC()
				if evalAt.IsZero() {
					evalAt = time.Now().UTC()
				}
				muts = append(muts, spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
					"RetailerId":      rec.RetailerID,
					"SupplierId":      rec.SupplierID,
					"RiskScore":       rec.Score,
					"LastEvaluatedAt": evalAt,
				}))
			}
		}

		buf := outbox.NewSpannerTxnBuffer(txn)
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateCreditScore, rec.ScoreID, events.TopicIntel, events.CreditScoreComputedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventCreditScoreComputed,
				Version:   1,
				Timestamp: rec.ComputedAt.UTC().Format(time.RFC3339Nano),
			},
			ScoreID:             rec.ScoreID,
			RetailerID:          rec.RetailerID,
			SupplierID:          rec.SupplierID,
			Scope:               rec.Scope,
			ConsentVerified:     rec.ConsentVerified,
			ModelVersion:        rec.ModelVersion,
			ScorerSource:        rec.ScorerSource,
			Score:               rec.Score,
			PdBps:               rec.PdBps,
			RiskTier:            string(rec.RiskTier),
			SuggestedLimitMinor: rec.SuggestedLimitMinor,
			ReasonCodesJSON:     rcJSON,
			FeaturesHash:        rec.FeaturesHash,
			ComputedAt:          rec.ComputedAt.UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		return buf.Flush(txCtx)
	})
	return err
}

func (r *SpannerScoreHistoryRepo) ListScoreHistory(ctx context.Context, retailerID, supplierID, scope string, limit int) ([]CreditScoreRecord, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner client unavailable")
	}
	if limit <= 0 || limit > 200 {
		limit = 25
	}
	sql := `SELECT RetailerId, ScoreId, Scope, COALESCE(SupplierId, ''), ConsentVerified,
	               COALESCE(ConsentScope, ''), ModelVersion, ScorerSource, Score, PdBps,
	               RiskTier, SuggestedLimitMinor, CurrentLimitMinor, ReasonCodesJson,
	               COALESCE(FactorsJson, ''), FeaturesHash, DisputesIsolatedCount,
	               WindowStart, WindowEnd, ComputedAt, CreatedAt
	        FROM CreditScoreHistory
	        WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID, "lim": int64(limit)}
	if strings.TrimSpace(supplierID) != "" {
		sql += " AND (SupplierId = @sid OR Scope = 'NETWORK')"
		params["sid"] = strings.TrimSpace(supplierID)
	}
	if strings.TrimSpace(scope) != "" {
		sql += " AND Scope = @scope"
		params["scope"] = strings.ToUpper(strings.TrimSpace(scope))
	}
	sql += " ORDER BY ComputedAt DESC LIMIT @lim"

	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []CreditScoreRecord
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var rec CreditScoreRecord
		var tier string
		if err := row.Columns(
			&rec.RetailerID, &rec.ScoreID, &rec.Scope, &rec.SupplierID, &rec.ConsentVerified,
			&rec.ConsentScope, &rec.ModelVersion, &rec.ScorerSource, &rec.Score, &rec.PdBps,
			&tier, &rec.SuggestedLimitMinor, &rec.CurrentLimitMinor, &rec.ReasonCodesJSON,
			&rec.FactorsJSON, &rec.FeaturesHash, &rec.DisputesIsolatedCount,
			&rec.WindowStart, &rec.WindowEnd, &rec.ComputedAt, &rec.CreatedAt,
		); err != nil {
			return nil, err
		}
		rec.RiskTier = RiskTier(tier)
		if strings.TrimSpace(rec.ReasonCodesJSON) != "" {
			_ = json.Unmarshal([]byte(rec.ReasonCodesJSON), &rec.ReasonCodes)
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *SpannerScoreHistoryRepo) LoadRelationshipMetrics(ctx context.Context, retailerID, supplierID string) (ScoreMetrics, bool, error) {
	var m ScoreMetrics
	if r == nil || r.client == nil {
		return m, false, nil
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	now := time.Now().UTC()
	since := now.Add(-90 * 24 * time.Hour)
	var consent bool

	// 1. Terms + ConsentVerified + Program starter limit.
	if row, err := ro.ReadRow(ctx, "RetailerPaymentTerms", spanner.Key{retailerID, supplierID},
		[]string{"CreditLimitMinor", "ConsentVerified"}); err == nil {
		var termsLimit int64
		var cv spanner.NullBool
		if colErr := row.Columns(&termsLimit, &cv); colErr == nil {
			if termsLimit > 0 {
				m.StarterLimitMinor = termsLimit
			}
			if cv.Valid {
				consent = cv.Bool
			}
		}
	}
	if m.StarterLimitMinor <= 0 {
		if row, err := ro.ReadRow(ctx, "SupplierCreditPrograms", spanner.Key{supplierID}, []string{"GlobalDefaultLimitMinor"}); err == nil {
			var defLimit int64
			if colErr := row.Columns(&defLimit); colErr == nil && defLimit > 0 {
				m.StarterLimitMinor = defLimit
			}
		}
	}

	// 2. Open/overdue undisputed AR vs disputed AR.
	invIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT Status, BalanceMinor, PrincipalMinor, DueAt
		      FROM ArInvoices
		      WHERE RetailerId = @rid AND SupplierId = @sid
		      LIMIT 200`,
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	})
	for {
		row, err := invIter.Next()
		if errors.Is(err, iterator.Done) || err != nil {
			break
		}
		var st string
		var bal, prin int64
		var due time.Time
		if err := row.Columns(&st, &bal, &prin, &due); err != nil {
			continue
		}
		up := strings.ToUpper(strings.TrimSpace(st))
		if up == "DISPUTED" {
			m.DisputedInvoicesCount++
			m.DisputedAmountMinor += bal
			continue
		}
		if up == "SETTLED" || up == "PAID" {
			m.PaidVolume90dMinor += prin
			m.PaymentsLast90d++
			m.ExpectedPayments90++
			continue
		}
		if up == "OPEN" || up == "PARTIAL" || up == "OVERDUE" {
			m.ExpectedPayments90++
			if !due.IsZero() && now.After(due) {
				dpd := int64(now.Sub(due).Hours() / 24)
				if dpd > m.MaxDaysPastDue {
					m.MaxDaysPastDue = dpd
				}
			}
		}
	}
	invIter.Stop()

	// 3. Cleared OrderCreditReservations + 90d Orders baseline.
	resIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(1), COALESCE(SUM(AmountMinor), 0)
		      FROM OrderCreditReservations
		      WHERE RetailerId = @rid AND SupplierId = @sid AND Status = 'CLEARED' AND UpdatedAt >= @since`,
		Params: map[string]any{"rid": retailerID, "sid": supplierID, "since": since},
	})
	if row, err := resIter.Next(); err == nil {
		var cnt, amt int64
		if row.Columns(&cnt, &amt) == nil {
			if cnt > m.PaymentsLast90d {
				m.PaymentsLast90d = cnt
			}
			if amt > m.PaidVolume90dMinor {
				m.PaidVolume90dMinor = amt
			}
		}
	}
	resIter.Stop()

	ordIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(1), COALESCE(SUM(TotalMinor), 0)
		      FROM Orders
		      WHERE RetailerId = @rid AND SupplierId = @sid
		        AND Status NOT IN ('CANCELLED', 'REJECTED', 'VOIDED')
		        AND CreatedAt >= @since`,
		Params: map[string]any{"rid": retailerID, "sid": supplierID, "since": since},
	})
	if row, err := ordIter.Next(); err == nil {
		var cnt, totalSum int64
		if row.Columns(&cnt, &totalSum) == nil {
			m.OrderCount90d = cnt
			if cnt > 0 {
				m.AvgOrderMinor = totalSum / cnt
			}
		}
	}
	ordIter.Stop()
	if m.ExpectedPayments90 < m.PaymentsLast90d {
		m.ExpectedPayments90 = m.PaymentsLast90d
	}

	return m, consent, nil
}

func (r *SpannerScoreHistoryRepo) LoadNetworkInputs(ctx context.Context, retailerID string) (NetworkScoreInputs, error) {
	out := NetworkScoreInputs{RetailerID: retailerID}
	if r == nil || r.client == nil {
		return out, nil
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	// Load only supplier relationships where ConsentVerified = true.
	tIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT SupplierId, CreditEnabled, TermsDays, GracePeriodDays, CreditLimitMinor, UseGlobalDefaults, ConsentVerified
		      FROM RetailerPaymentTerms
		      WHERE RetailerId = @rid AND ConsentVerified = true`,
		Params: map[string]any{"rid": retailerID},
	})
	var consentingSuppliers []RetailerPaymentTerms
	for {
		row, err := tIter.Next()
		if errors.Is(err, iterator.Done) || err != nil {
			break
		}
		var t RetailerPaymentTerms
		t.RetailerID = retailerID
		if err := row.Columns(&t.SupplierID, &t.CreditEnabled, &t.TermsDays, &t.GracePeriodDays, &t.CreditLimitMinor, &t.UseGlobalDefaults, &t.ConsentVerified); err == nil {
			consentingSuppliers = append(consentingSuppliers, t)
		}
	}
	tIter.Stop()

	for _, t := range consentingSuppliers {
		var p Profile
		pRow, err := ro.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, t.SupplierID}, []string{
			"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor", "ReservedMinor",
			"AvailableCreditMinor", "RiskScore", "DelinquencyCount", "PrepayOnlyLocked", "Status",
			"LastEvaluatedAt", "Version", "CreatedAt", "UpdatedAt",
		})
		if err == nil {
			if scanned, sErr := scanProfileRow(pRow); sErr == nil {
				p = scanned
			}
		} else {
			p = Profile{
				RetailerID:       retailerID,
				SupplierID:       t.SupplierID,
				CreditLimitMinor: t.CreditLimitMinor,
				Status:           StatusActive,
			}
		}
		metrics, _, _ := r.LoadRelationshipMetrics(ctx, retailerID, t.SupplierID)
		out.DisputesIsolatedCount += metrics.DisputedInvoicesCount
		out.DisputedAmountMinor += metrics.DisputedAmountMinor
		out.ConsentingProfiles = append(out.ConsentingProfiles, ConsentingSupplierProfile{
			Profile:         p,
			Terms:           t,
			Metrics:         metrics,
			ConsentVerified: true,
		})
		out.TotalConsentingCount++
	}
	return out, nil
}

func (r *SpannerScoreHistoryRepo) ApplyGovernedLimitChange(ctx context.Context, in ApplyLimitRepoInput) (Profile, error) {
	if r == nil || r.client == nil {
		return Profile{}, fmt.Errorf("spanner client unavailable")
	}
	var updated Profile
	err := r.runLimitTxn(ctx, in, &updated)
	return updated, err
}

func (r *SpannerScoreHistoryRepo) runLimitTxn(ctx context.Context, in ApplyLimitRepoInput, out *Profile) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		cols := []string{
			"RetailerId", "SupplierId", "CreditLimitMinor", "CurrentBalanceMinor",
			"ReservedMinor", "AvailableCreditMinor", "RiskScore", "DelinquencyCount",
			"PrepayOnlyLocked", "Status", "LastEvaluatedAt", "Version", "CreatedAt", "UpdatedAt",
		}
		p := Profile{
			RetailerID: in.RetailerID,
			SupplierID: in.SupplierID,
			Status:     StatusActive,
			Version:    1,
			CreatedAt:  in.Now,
			UpdatedAt:  in.Now,
		}
		row, err := txn.ReadRow(txCtx, "RetailerCreditProfiles", spanner.Key{in.RetailerID, in.SupplierID}, cols)
		if err == nil {
			scanned, sErr := scanProfileRow(row)
			if sErr != nil {
				return sErr
			}
			p = scanned
			p.Version++
			p.UpdatedAt = in.Now
		} else if spanner.ErrCode(err) != codes.NotFound {
			return err
		}
		oldLimit := p.CreditLimitMinor
		p.CreditLimitMinor = in.NewLimitMinor
		p.AvailableCreditMinor = p.Available()
		if p.LastEvaluatedAt.IsZero() {
			p.LastEvaluatedAt = in.Now
		}
		if in.ScoreRecord != nil {
			p.RiskScore = in.ScoreRecord.Score
		}

		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           p.RetailerID,
				"SupplierId":           p.SupplierID,
				"CreditLimitMinor":     p.CreditLimitMinor,
				"CurrentBalanceMinor":  p.CurrentBalanceMinor,
				"ReservedMinor":        p.ReservedMinor,
				"AvailableCreditMinor": p.AvailableCreditMinor,
				"RiskScore":            p.RiskScore,
				"DelinquencyCount":     p.DelinquencyCount,
				"PrepayOnlyLocked":     p.PrepayOnlyLocked,
				"Status":               string(p.Status),
				"LastEvaluatedAt":      p.LastEvaluatedAt.UTC(),
				"Version":              p.Version,
				"CreatedAt":            p.CreatedAt,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		}

		if tRow, tErr := txn.ReadRow(txCtx, "RetailerPaymentTerms", spanner.Key{in.RetailerID, in.SupplierID}, []string{"Version"}); tErr == nil {
			var tv int64
			_ = tRow.Columns(&tv)
			updTerms := map[string]any{
				"RetailerId":       in.RetailerID,
				"SupplierId":       in.SupplierID,
				"CreditLimitMinor": in.NewLimitMinor,
				"Version":          tv + 1,
				"UpdatedAt":        spanner.CommitTimestamp,
			}
			if in.ConsentVerified != nil {
				updTerms["ConsentVerified"] = *in.ConsentVerified
			}
			muts = append(muts, spanner.UpdateMap("RetailerPaymentTerms", updTerms))
		} else if in.ConsentVerified != nil && *in.ConsentVerified {
			muts = append(muts, spanner.InsertOrUpdateMap("RetailerPaymentTerms", map[string]any{
				"RetailerId":        in.RetailerID,
				"SupplierId":        in.SupplierID,
				"CreditEnabled":     true,
				"TermsDays":         int64(30),
				"GracePeriodDays":   int64(0),
				"CreditLimitMinor":  in.NewLimitMinor,
				"UseGlobalDefaults": false,
				"ConsentVerified":   true,
				"Version":           int64(1),
				"UpdatedAt":         spanner.CommitTimestamp,
			}))
		}

		if in.ScoreRecord != nil {
			rec := *in.ScoreRecord
			rcJSON := rec.ReasonCodesJSON
			if rcJSON == "" {
				b, _ := json.Marshal(rec.ReasonCodes)
				rcJSON = string(b)
			}
			if rcJSON == "" || rcJSON == "null" {
				rcJSON = "[]"
			}
			scoreMap := map[string]any{
				"RetailerId":            rec.RetailerID,
				"ScoreId":               rec.ScoreID,
				"Scope":                 rec.Scope,
				"SupplierId":            in.SupplierID,
				"ConsentVerified":       rec.ConsentVerified,
				"ModelVersion":          rec.ModelVersion,
				"ScorerSource":          rec.ScorerSource,
				"Score":                 rec.Score,
				"PdBps":                 rec.PdBps,
				"RiskTier":              string(rec.RiskTier),
				"SuggestedLimitMinor":   rec.SuggestedLimitMinor,
				"CurrentLimitMinor":     in.NewLimitMinor,
				"ReasonCodesJson":       rcJSON,
				"FeaturesHash":          rec.FeaturesHash,
				"DisputesIsolatedCount": rec.DisputesIsolatedCount,
				"WindowStart":           rec.WindowStart.UTC(),
				"WindowEnd":             rec.WindowEnd.UTC(),
				"ComputedAt":            spanner.CommitTimestamp,
				"CreatedAt":             spanner.CommitTimestamp,
			}
			muts = append(muts, spanner.InsertOrUpdateMap("CreditScoreHistory", scoreMap))
		}

		// Append-only CreditPolicyAudit (InsertMap only per immutable-ledger gate!).
		beforeJSON, _ := json.Marshal(map[string]any{"credit_limit_minor": oldLimit})
		afterJSON, _ := json.Marshal(map[string]any{
			"credit_limit_minor": in.NewLimitMinor,
			"approval_id":        in.ApprovalID,
			"reason":             in.Reason,
		})
		muts = append(muts, spanner.InsertMap("CreditPolicyAudit", map[string]any{
			"AuditId":     clampStr(in.AuditID, 36),
			"SupplierId":  clampStr(in.SupplierID, 36),
			"RetailerId":  nullableStr(clampStr(in.RetailerID, 36)),
			"Action":      "LIMIT_CHANGE",
			"ActorUserId": nullableStr(clampStr(in.ActorID, 36)),
			"ActorRole":   nullableStr(clampStr(in.ActorRole, 32)),
			"BeforeJson":  spanner.NullJSON{Valid: true, Value: json.RawMessage(beforeJSON)},
			"AfterJson":   spanner.NullJSON{Valid: true, Value: json.RawMessage(afterJSON)},
			"CreatedAt":   spanner.CommitTimestamp,
		}))

		buf := outbox.NewSpannerTxnBuffer(txn)
		tsStr := in.Now.UTC().Format(time.RFC3339Nano)
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateCreditProfile, in.RetailerID, events.TopicMain, events.CreditLimitChangedEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventCreditLimitChanged,
				Version:   1,
				Timestamp: tsStr,
			},
			RetailerID:         in.RetailerID,
			SupplierID:         in.SupplierID,
			PreviousLimitMinor: oldLimit,
			NewLimitMinor:      in.NewLimitMinor,
			ApprovalID:         in.ApprovalID,
			ActorID:            in.ActorID,
			Reason:             in.Reason,
			ReasonCodesJSON:    in.ReasonCodesJSON,
			OccurredAt:         tsStr,
		}); err != nil {
			return err
		}
		if err := outbox.EmitJSON(txCtx, buf, events.AggregateCreditProfile, in.RetailerID, events.TopicMain, events.CreditProfileEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventRetailerCreditProfileChanged, Timestamp: tsStr},
			ProfileID:        profileID(in.RetailerID, in.SupplierID),
			RetailerID:       in.RetailerID,
			SupplierID:       in.SupplierID,
			CreditLimitMinor: p.CreditLimitMinor,
			CurrentBalance:   p.CurrentBalanceMinor,
			Delinquent:       p.DelinquencyCount > 0,
			Reason:           "limit_changed:" + in.Reason,
		}); err != nil {
			return err
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		if err := buf.Flush(txCtx); err != nil {
			return err
		}
		*out = p
		return nil
	})
	return err
}

func nullableStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func clampStr(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if maxLen > 0 && len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
