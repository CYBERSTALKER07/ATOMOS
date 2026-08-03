package credit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// Policy feature flag (SSMR).
func PolicyV2Enabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CREDIT_POLICY_V2_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// SupplierCreditProgram is org-level credit program state.
type SupplierCreditProgram struct {
	SupplierID              string     `json:"supplier_id"`
	ProgramEnabled          bool       `json:"program_enabled"`
	EnabledAt               *time.Time `json:"enabled_at,omitempty"`
	EnabledByUserID         string     `json:"enabled_by_user_id,omitempty"`
	DisabledAt              *time.Time `json:"disabled_at,omitempty"`
	DisabledByActor         string     `json:"disabled_by_actor,omitempty"`
	DisableReason           string     `json:"disable_reason,omitempty"`
	GlobalTermsDays         int64      `json:"global_terms_days"`
	GlobalGraceDays         int64      `json:"global_grace_days"`
	GlobalDefaultLimitMinor int64      `json:"global_default_limit_minor"`
	Timezone                string     `json:"timezone"`
	Version                 int64      `json:"version"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// RetailerPaymentTerms is per-retailer relationship + Net terms.
type RetailerPaymentTerms struct {
	RetailerID        string     `json:"retailer_id"`
	SupplierID        string     `json:"supplier_id"`
	CreditEnabled     bool       `json:"credit_enabled"`
	EnabledAt         *time.Time `json:"enabled_at,omitempty"`
	EnabledByUserID   string     `json:"enabled_by_user_id,omitempty"`
	DisabledAt        *time.Time `json:"disabled_at,omitempty"`
	DisabledByActor   string     `json:"disabled_by_actor,omitempty"`
	TermsDays         int64      `json:"terms_days"`
	GracePeriodDays   int64      `json:"grace_period_days"`
	CreditLimitMinor  int64      `json:"credit_limit_minor"`
	UseGlobalDefaults bool       `json:"use_global_defaults"`
	Version           int64      `json:"version"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ResolvedTerms is effective Net terms after global/override merge.
type ResolvedTerms struct {
	TermsDays        int64  `json:"terms_days"`
	GracePeriodDays  int64  `json:"grace_period_days"`
	CreditLimitMinor int64  `json:"credit_limit_minor"`
	Timezone         string `json:"timezone"`
}

// CreditPolicyAuditRow is an immutable policy change record.
type CreditPolicyAuditRow struct {
	AuditID      string
	SupplierID   string
	RetailerID   string
	Action       string
	ActorUserID  string
	ActorRole    string
	BeforeJSON   string
	AfterJSON    string
	WarningAckAt *time.Time
	TicketID     string
	CreatedAt    time.Time
}

// PolicyRepository persists programs, terms, and audit.
type PolicyRepository interface {
	GetProgram(ctx context.Context, supplierID string) (SupplierCreditProgram, bool, error)
	UpsertProgram(ctx context.Context, p SupplierCreditProgram) error
	GetTerms(ctx context.Context, retailerID, supplierID string) (RetailerPaymentTerms, bool, error)
	ListTermsBySupplier(ctx context.Context, supplierID string, limit int) ([]RetailerPaymentTerms, error)
	ListTermsByRetailer(ctx context.Context, retailerID string, limit int) ([]RetailerPaymentTerms, error)
	UpsertTerms(ctx context.Context, t RetailerPaymentTerms) error
	AppendAudit(ctx context.Context, a CreditPolicyAuditRow) error
}

// MemoryPolicyRepository is used in tests and when Spanner tables are absent.
type MemoryPolicyRepository struct {
	mu       sync.RWMutex
	programs map[string]SupplierCreditProgram
	terms    map[string]RetailerPaymentTerms
	audits   []CreditPolicyAuditRow
}

func NewMemoryPolicyRepository() *MemoryPolicyRepository {
	return &MemoryPolicyRepository{
		programs: map[string]SupplierCreditProgram{},
		terms:    map[string]RetailerPaymentTerms{},
	}
}

func termsKey(rid, sid string) string { return rid + ":" + sid }

func (r *MemoryPolicyRepository) GetProgram(_ context.Context, supplierID string) (SupplierCreditProgram, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.programs[supplierID]
	return p, ok, nil
}

func (r *MemoryPolicyRepository) UpsertProgram(_ context.Context, p SupplierCreditProgram) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.programs[p.SupplierID] = p
	return nil
}

func (r *MemoryPolicyRepository) GetTerms(_ context.Context, retailerID, supplierID string) (RetailerPaymentTerms, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.terms[termsKey(retailerID, supplierID)]
	return t, ok, nil
}

func (r *MemoryPolicyRepository) ListTermsBySupplier(_ context.Context, supplierID string, limit int) ([]RetailerPaymentTerms, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]RetailerPaymentTerms, 0)
	for _, t := range r.terms {
		if t.SupplierID == supplierID {
			out = append(out, t)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryPolicyRepository) ListTermsByRetailer(_ context.Context, retailerID string, limit int) ([]RetailerPaymentTerms, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]RetailerPaymentTerms, 0)
	for _, t := range r.terms {
		if t.RetailerID == retailerID {
			out = append(out, t)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *MemoryPolicyRepository) UpsertTerms(_ context.Context, t RetailerPaymentTerms) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.terms[termsKey(t.RetailerID, t.SupplierID)] = t
	return nil
}

func (r *MemoryPolicyRepository) AppendAudit(_ context.Context, a CreditPolicyAuditRow) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.audits = append(r.audits, a)
	return nil
}

// SpannerPolicyRepository persists policy tables.
type SpannerPolicyRepository struct {
	client *spanner.Client
}

func NewSpannerPolicyRepository(client *spanner.Client) *SpannerPolicyRepository {
	return &SpannerPolicyRepository{client: client}
}

func (r *SpannerPolicyRepository) GetProgram(ctx context.Context, supplierID string) (SupplierCreditProgram, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "SupplierCreditPrograms", spanner.Key{supplierID},
		[]string{"SupplierId", "ProgramEnabled", "EnabledAt", "EnabledByUserId", "DisabledAt", "DisabledByActor",
			"DisableReason", "GlobalTermsDays", "GlobalGraceDays", "GlobalDefaultLimitMinor", "Timezone", "Version", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return SupplierCreditProgram{}, false, nil
		}
		return SupplierCreditProgram{}, false, err
	}
	p, err := scanProgram(row)
	return p, err == nil, err
}

func scanProgram(row *spanner.Row) (SupplierCreditProgram, error) {
	var p SupplierCreditProgram
	var enabledAt, disabledAt spanner.NullTime
	var enabledBy, disabledBy, reason, tz spanner.NullString
	if err := row.Columns(&p.SupplierID, &p.ProgramEnabled, &enabledAt, &enabledBy, &disabledAt, &disabledBy,
		&reason, &p.GlobalTermsDays, &p.GlobalGraceDays, &p.GlobalDefaultLimitMinor, &tz, &p.Version, &p.UpdatedAt); err != nil {
		return p, err
	}
	if enabledAt.Valid {
		t := enabledAt.Time
		p.EnabledAt = &t
	}
	if disabledAt.Valid {
		t := disabledAt.Time
		p.DisabledAt = &t
	}
	p.EnabledByUserID = enabledBy.StringVal
	p.DisabledByActor = disabledBy.StringVal
	p.DisableReason = reason.StringVal
	p.Timezone = tz.StringVal
	if p.Timezone == "" {
		p.Timezone = "Asia/Tashkent"
	}
	return p, nil
}

func (r *SpannerPolicyRepository) UpsertProgram(ctx context.Context, p SupplierCreditProgram) error {
	m := map[string]any{
		"SupplierId":              p.SupplierID,
		"ProgramEnabled":          p.ProgramEnabled,
		"EnabledByUserId":         nullable(p.EnabledByUserID),
		"DisabledByActor":         nullable(p.DisabledByActor),
		"DisableReason":           nullable(p.DisableReason),
		"GlobalTermsDays":         p.GlobalTermsDays,
		"GlobalGraceDays":         p.GlobalGraceDays,
		"GlobalDefaultLimitMinor": p.GlobalDefaultLimitMinor,
		"Timezone":                p.Timezone,
		"Version":                 p.Version,
		"UpdatedAt":               spanner.CommitTimestamp,
	}
	if p.EnabledAt != nil {
		m["EnabledAt"] = *p.EnabledAt
	}
	if p.DisabledAt != nil {
		m["DisabledAt"] = *p.DisabledAt
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("SupplierCreditPrograms", m)})
	return err
}

func (r *SpannerPolicyRepository) GetTerms(ctx context.Context, retailerID, supplierID string) (RetailerPaymentTerms, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "RetailerPaymentTerms", spanner.Key{retailerID, supplierID},
		[]string{"RetailerId", "SupplierId", "CreditEnabled", "EnabledAt", "EnabledByUserId", "DisabledAt", "DisabledByActor",
			"TermsDays", "GracePeriodDays", "CreditLimitMinor", "UseGlobalDefaults", "Version", "UpdatedAt"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return RetailerPaymentTerms{}, false, nil
		}
		return RetailerPaymentTerms{}, false, err
	}
	t, err := scanTerms(row)
	return t, err == nil, err
}

func scanTerms(row *spanner.Row) (RetailerPaymentTerms, error) {
	var t RetailerPaymentTerms
	var enabledAt, disabledAt spanner.NullTime
	var enabledBy, disabledBy spanner.NullString
	if err := row.Columns(&t.RetailerID, &t.SupplierID, &t.CreditEnabled, &enabledAt, &enabledBy, &disabledAt, &disabledBy,
		&t.TermsDays, &t.GracePeriodDays, &t.CreditLimitMinor, &t.UseGlobalDefaults, &t.Version, &t.UpdatedAt); err != nil {
		return t, err
	}
	if enabledAt.Valid {
		x := enabledAt.Time
		t.EnabledAt = &x
	}
	if disabledAt.Valid {
		x := disabledAt.Time
		t.DisabledAt = &x
	}
	t.EnabledByUserID = enabledBy.StringVal
	t.DisabledByActor = disabledBy.StringVal
	return t, nil
}

func (r *SpannerPolicyRepository) ListTermsBySupplier(ctx context.Context, supplierID string, limit int) ([]RetailerPaymentTerms, error) {
	if limit <= 0 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf(`SELECT RetailerId, SupplierId, CreditEnabled, EnabledAt, EnabledByUserId, DisabledAt, DisabledByActor, TermsDays, GracePeriodDays, CreditLimitMinor, UseGlobalDefaults, Version, UpdatedAt FROM RetailerPaymentTerms WHERE SupplierId = @sid ORDER BY UpdatedAt DESC LIMIT %d`, limit),
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]RetailerPaymentTerms, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := scanTerms(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *SpannerPolicyRepository) ListTermsByRetailer(ctx context.Context, retailerID string, limit int) ([]RetailerPaymentTerms, error) {
	if limit <= 0 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf(`SELECT RetailerId, SupplierId, CreditEnabled, EnabledAt, EnabledByUserId, DisabledAt, DisabledByActor, TermsDays, GracePeriodDays, CreditLimitMinor, UseGlobalDefaults, Version, UpdatedAt FROM RetailerPaymentTerms WHERE RetailerId = @rid ORDER BY UpdatedAt DESC LIMIT %d`, limit),
		Params: map[string]any{"rid": retailerID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]RetailerPaymentTerms, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		t, err := scanTerms(row)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *SpannerPolicyRepository) UpsertTerms(ctx context.Context, t RetailerPaymentTerms) error {
	m := map[string]any{
		"RetailerId":        t.RetailerID,
		"SupplierId":        t.SupplierID,
		"CreditEnabled":     t.CreditEnabled,
		"EnabledByUserId":   nullable(t.EnabledByUserID),
		"DisabledByActor":   nullable(t.DisabledByActor),
		"TermsDays":         t.TermsDays,
		"GracePeriodDays":   t.GracePeriodDays,
		"CreditLimitMinor":  t.CreditLimitMinor,
		"UseGlobalDefaults": t.UseGlobalDefaults,
		"Version":           t.Version,
		"UpdatedAt":         spanner.CommitTimestamp,
	}
	if t.EnabledAt != nil {
		m["EnabledAt"] = *t.EnabledAt
	}
	if t.DisabledAt != nil {
		m["DisabledAt"] = *t.DisabledAt
	}
	_, err := r.client.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerPaymentTerms", m)})
	return err
}

func (r *SpannerPolicyRepository) AppendAudit(ctx context.Context, a CreditPolicyAuditRow) error {
	m := map[string]any{
		"AuditId":     a.AuditID,
		"SupplierId":  a.SupplierID,
		"Action":      a.Action,
		"ActorUserId": nullable(a.ActorUserID),
		"ActorRole":   nullable(a.ActorRole),
		"TicketId":    nullable(a.TicketID),
		"CreatedAt":   spanner.CommitTimestamp,
	}
	if a.RetailerID != "" {
		m["RetailerId"] = a.RetailerID
	}
	if a.BeforeJSON != "" {
		m["BeforeJson"] = spanner.NullJSON{Valid: true, Value: json.RawMessage(a.BeforeJSON)}
	}
	if a.AfterJSON != "" {
		m["AfterJson"] = spanner.NullJSON{Valid: true, Value: json.RawMessage(a.AfterJSON)}
	}
	if a.WarningAckAt != nil {
		m["WarningAckAt"] = *a.WarningAckAt
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("CreditPolicyAudit", m)})
	})
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// PolicyService owns irreversible enable / admin disable / terms resolution.
type PolicyService struct {
	repo    PolicyRepository
	credit  *Service
	now     func() time.Time
	newID   func() string
}

func NewPolicyService(repo PolicyRepository, credit *Service) *PolicyService {
	return &PolicyService{
		repo:   repo,
		credit: credit,
		now:    func() time.Time { return time.Now().UTC() },
		newID:  func() string { return fmt.Sprintf("cpa_%d", time.Now().UnixNano()) },
	}
}

func (s *PolicyService) SetNow(fn func() time.Time) { s.now = fn }

// CreditPathAllowed implements PolicyGate.
func (s *PolicyService) CreditPathAllowed(ctx context.Context, retailerID, supplierID string) (bool, string, int64, error) {
	if s == nil || s.repo == nil {
		return true, "", 0, nil // no policy → legacy behavior
	}
	if !PolicyV2Enabled() {
		return true, "", 0, nil
	}
	prog, ok, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return false, "", 0, err
	}
	if !ok || !prog.ProgramEnabled {
		return false, "credit_program_not_enabled", 0, nil
	}
	terms, ok, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return false, "", 0, err
	}
	if !ok || !terms.CreditEnabled {
		return false, "credit_relationship_not_enabled", 0, nil
	}
	resolved := s.resolveTerms(prog, terms)
	return true, "", resolved.TermsDays, nil
}

// ResolveDueAt implements PolicyGate — dueAt = creditLeaveAt + TermsDays (supplier TZ calendar days).
func (s *PolicyService) ResolveDueAt(ctx context.Context, retailerID, supplierID string, creditLeaveAt time.Time) (time.Time, int64, error) {
	prog, _, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return time.Time{}, 0, err
	}
	terms, _, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return time.Time{}, 0, err
	}
	resolved := s.resolveTerms(prog, terms)
	loc := time.UTC
	if resolved.Timezone != "" {
		if l, lerr := time.LoadLocation(resolved.Timezone); lerr == nil {
			loc = l
		}
	}
	local := creditLeaveAt.In(loc)
	dueLocal := time.Date(local.Year(), local.Month(), local.Day(), 23, 59, 59, 0, loc).
		AddDate(0, 0, int(resolved.TermsDays))
	return dueLocal.UTC(), resolved.TermsDays, nil
}

func (s *PolicyService) resolveTerms(prog SupplierCreditProgram, terms RetailerPaymentTerms) ResolvedTerms {
	tz := prog.Timezone
	if tz == "" {
		tz = "Asia/Tashkent"
	}
	if terms.UseGlobalDefaults || !terms.CreditEnabled {
		days := prog.GlobalTermsDays
		if days <= 0 {
			days = 30
		}
		return ResolvedTerms{
			TermsDays:        days,
			GracePeriodDays:  prog.GlobalGraceDays,
			CreditLimitMinor: prog.GlobalDefaultLimitMinor,
			Timezone:         tz,
		}
	}
	days := terms.TermsDays
	if days <= 0 {
		days = prog.GlobalTermsDays
		if days <= 0 {
			days = 30
		}
	}
	limit := terms.CreditLimitMinor
	if limit <= 0 {
		limit = prog.GlobalDefaultLimitMinor
	}
	return ResolvedTerms{
		TermsDays:        days,
		GracePeriodDays:  terms.GracePeriodDays,
		CreditLimitMinor: limit,
		Timezone:         tz,
	}
}

// EnableProgram turns on supplier credit program (irreversible self-serve).
func (s *PolicyService) EnableProgram(ctx context.Context, supplierID, actorUserID, actorRole string, warningAck bool, ackAt time.Time, defaults *SupplierCreditProgram) (SupplierCreditProgram, error) {
	if !warningAck {
		return SupplierCreditProgram{}, ErrWarningAckRequired
	}
	existing, found, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return SupplierCreditProgram{}, err
	}
	if found && existing.ProgramEnabled {
		return existing, nil // idempotent
	}
	now := s.now()
	p := SupplierCreditProgram{
		SupplierID:              supplierID,
		ProgramEnabled:          true,
		EnabledAt:               &now,
		EnabledByUserID:         actorUserID,
		GlobalTermsDays:         30,
		GlobalGraceDays:         0,
		GlobalDefaultLimitMinor: 0,
		Timezone:                "Asia/Tashkent",
		Version:                 1,
		UpdatedAt:               now,
	}
	if defaults != nil {
		if defaults.GlobalTermsDays > 0 {
			p.GlobalTermsDays = defaults.GlobalTermsDays
		}
		if defaults.GlobalGraceDays >= 0 {
			p.GlobalGraceDays = defaults.GlobalGraceDays
		}
		if defaults.GlobalDefaultLimitMinor >= 0 {
			p.GlobalDefaultLimitMinor = defaults.GlobalDefaultLimitMinor
		}
		if defaults.Timezone != "" {
			p.Timezone = defaults.Timezone
		}
	}
	if found {
		p.Version = existing.Version + 1
	}
	before, _ := json.Marshal(existing)
	after, _ := json.Marshal(p)
	if err := s.repo.UpsertProgram(ctx, p); err != nil {
		return SupplierCreditProgram{}, err
	}
	_ = s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, Action: "PROGRAM_ENABLE",
		ActorUserID: actorUserID, ActorRole: actorRole,
		BeforeJSON: string(before), AfterJSON: string(after), WarningAckAt: &ackAt, CreatedAt: now,
	})
	return p, nil
}

// PatchProgramDefaults updates global terms (allowed after enable).
func (s *PolicyService) PatchProgramDefaults(ctx context.Context, supplierID, actorUserID, actorRole string, termsDays, graceDays, limitMinor *int64, tz *string) (SupplierCreditProgram, error) {
	p, ok, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return SupplierCreditProgram{}, err
	}
	if !ok || !p.ProgramEnabled {
		return SupplierCreditProgram{}, ErrProgramDisabled
	}
	before, _ := json.Marshal(p)
	if termsDays != nil && *termsDays > 0 {
		p.GlobalTermsDays = *termsDays
	}
	if graceDays != nil && *graceDays >= 0 {
		p.GlobalGraceDays = *graceDays
	}
	if limitMinor != nil && *limitMinor >= 0 {
		p.GlobalDefaultLimitMinor = *limitMinor
	}
	if tz != nil && *tz != "" {
		p.Timezone = *tz
	}
	p.Version++
	p.UpdatedAt = s.now()
	after, _ := json.Marshal(p)
	if err := s.repo.UpsertProgram(ctx, p); err != nil {
		return SupplierCreditProgram{}, err
	}
	_ = s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, Action: "TERMS_PATCH",
		ActorUserID: actorUserID, ActorRole: actorRole,
		BeforeJSON: string(before), AfterJSON: string(after), CreatedAt: s.now(),
	})
	return p, nil
}

// EnableRelationship irreversibly enables credit for one retailer.
func (s *PolicyService) EnableRelationship(ctx context.Context, supplierID, retailerID, actorUserID, actorRole string, warningAck bool, ackAt time.Time, termsDays, graceDays, limitMinor int64, useGlobal bool) (RetailerPaymentTerms, error) {
	if !warningAck {
		return RetailerPaymentTerms{}, ErrWarningAckRequired
	}
	prog, ok, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return RetailerPaymentTerms{}, err
	}
	if !ok || !prog.ProgramEnabled {
		return RetailerPaymentTerms{}, ErrProgramDisabled
	}
	existing, found, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return RetailerPaymentTerms{}, err
	}
	if found && existing.CreditEnabled {
		return existing, nil // idempotent
	}
	now := s.now()
	t := RetailerPaymentTerms{
		RetailerID:        retailerID,
		SupplierID:        supplierID,
		CreditEnabled:     true,
		EnabledAt:         &now,
		EnabledByUserID:   actorUserID,
		TermsDays:         termsDays,
		GracePeriodDays:   graceDays,
		CreditLimitMinor:  limitMinor,
		UseGlobalDefaults: useGlobal,
		Version:           1,
		UpdatedAt:         now,
	}
	if useGlobal {
		t.TermsDays = prog.GlobalTermsDays
		t.GracePeriodDays = prog.GlobalGraceDays
		if limitMinor <= 0 {
			t.CreditLimitMinor = prog.GlobalDefaultLimitMinor
		}
	}
	if t.TermsDays <= 0 {
		t.TermsDays = 30
	}
	if found {
		t.Version = existing.Version + 1
	}
	before, _ := json.Marshal(existing)
	after, _ := json.Marshal(t)
	if err := s.repo.UpsertTerms(ctx, t); err != nil {
		return RetailerPaymentTerms{}, err
	}
	// Mirror onto credit profile as ACTIVE with limit.
	if s.credit != nil {
		limit := t.CreditLimitMinor
		if t.UseGlobalDefaults {
			limit = prog.GlobalDefaultLimitMinor
			if t.CreditLimitMinor > 0 {
				limit = t.CreditLimitMinor
			}
		}
		_ = s.credit.UpsertProfile(ctx, Profile{
			RetailerID:       retailerID,
			SupplierID:       supplierID,
			CreditLimitMinor: limit,
			Status:           StatusActive,
			RiskTier:         RiskTierMedium,
		}, actorUserID, "relationship_enable")
	}
	_ = s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, RetailerID: retailerID, Action: "REL_ENABLE",
		ActorUserID: actorUserID, ActorRole: actorRole,
		BeforeJSON: string(before), AfterJSON: string(after), WarningAckAt: &ackAt, CreatedAt: now,
	})
	return t, nil
}

// PatchRelationshipTerms updates days/limit/grace (allowed; does not disable).
func (s *PolicyService) PatchRelationshipTerms(ctx context.Context, supplierID, retailerID, actorUserID, actorRole string, termsDays, graceDays, limitMinor *int64, useGlobal *bool) (RetailerPaymentTerms, error) {
	t, ok, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return RetailerPaymentTerms{}, err
	}
	if !ok || !t.CreditEnabled {
		return RetailerPaymentTerms{}, ErrCreditNotEnabled
	}
	before, _ := json.Marshal(t)
	if termsDays != nil && *termsDays > 0 {
		t.TermsDays = *termsDays
	}
	if graceDays != nil && *graceDays >= 0 {
		t.GracePeriodDays = *graceDays
	}
	if limitMinor != nil && *limitMinor >= 0 {
		t.CreditLimitMinor = *limitMinor
	}
	if useGlobal != nil {
		t.UseGlobalDefaults = *useGlobal
	}
	t.Version++
	t.UpdatedAt = s.now()
	after, _ := json.Marshal(t)
	if err := s.repo.UpsertTerms(ctx, t); err != nil {
		return RetailerPaymentTerms{}, err
	}
	if s.credit != nil && limitMinor != nil {
		prof, found, _ := s.credit.GetProfile(ctx, retailerID, supplierID)
		if found {
			prof.CreditLimitMinor = *limitMinor
			_ = s.credit.UpsertProfile(ctx, prof, actorUserID, "terms_limit_patch")
		}
	}
	_ = s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, RetailerID: retailerID, Action: "TERMS_PATCH",
		ActorUserID: actorUserID, ActorRole: actorRole,
		BeforeJSON: string(before), AfterJSON: string(after), CreatedAt: s.now(),
	})
	return t, nil
}

// RejectSelfServeDisable always returns 403 semantics.
func (s *PolicyService) RejectSelfServeDisable() error {
	return ErrDisableRequiresSupport
}

// AdminDisableRelationship permanently disables (support only). Open AR remains collectible.
func (s *PolicyService) AdminDisableRelationship(ctx context.Context, supplierID, retailerID, actorUserID, actorRole, ticketID, reason string) error {
	if strings.TrimSpace(ticketID) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("ticket_id_and_reason_required")
	}
	t, ok, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCreditNotEnabled
	}
	before, _ := json.Marshal(t)
	now := s.now()
	t.CreditEnabled = false
	t.DisabledAt = &now
	t.DisabledByActor = fmt.Sprintf("%s:%s", actorRole, actorUserID)
	t.Version++
	t.UpdatedAt = now
	after, _ := json.Marshal(t)
	if err := s.repo.UpsertTerms(ctx, t); err != nil {
		return err
	}
	if s.credit != nil {
		prof, found, _ := s.credit.GetProfile(ctx, retailerID, supplierID)
		if found {
			prof.Status = StatusInactive
			_ = s.credit.UpsertProfile(ctx, prof, actorUserID, "admin_disable:"+ticketID)
		}
	}
	return s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, RetailerID: retailerID, Action: "ADMIN_DISABLE",
		ActorUserID: actorUserID, ActorRole: actorRole, TicketID: ticketID,
		BeforeJSON: string(before), AfterJSON: string(after), CreatedAt: now,
	})
}

// AdminDisableProgram kills the org-level program.
func (s *PolicyService) AdminDisableProgram(ctx context.Context, supplierID, actorUserID, actorRole, ticketID, reason string) error {
	if strings.TrimSpace(ticketID) == "" || strings.TrimSpace(reason) == "" {
		return fmt.Errorf("ticket_id_and_reason_required")
	}
	p, ok, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProgramDisabled
	}
	before, _ := json.Marshal(p)
	now := s.now()
	p.ProgramEnabled = false
	p.DisabledAt = &now
	p.DisabledByActor = fmt.Sprintf("%s:%s", actorRole, actorUserID)
	p.DisableReason = reason
	p.Version++
	p.UpdatedAt = now
	after, _ := json.Marshal(p)
	if err := s.repo.UpsertProgram(ctx, p); err != nil {
		return err
	}
	return s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, Action: "ADMIN_DISABLE",
		ActorUserID: actorUserID, ActorRole: actorRole, TicketID: ticketID,
		BeforeJSON: string(before), AfterJSON: string(after), CreatedAt: now,
	})
}

// HoldRelationship sets profile FROZEN without clearing CreditEnabled.
func (s *PolicyService) HoldRelationship(ctx context.Context, supplierID, retailerID, actorUserID, actorRole string) error {
	if s.credit == nil {
		return fmt.Errorf("credit service unavailable")
	}
	prof, found, err := s.credit.GetProfile(ctx, retailerID, supplierID)
	if err != nil {
		return err
	}
	if !found {
		return ErrProfileNotFound
	}
	before, _ := json.Marshal(prof)
	prof.Status = StatusFrozen
	if err := s.credit.UpsertProfile(ctx, prof, actorUserID, "hold"); err != nil {
		return err
	}
	after, _ := json.Marshal(prof)
	return s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, RetailerID: retailerID, Action: "HOLD",
		ActorUserID: actorUserID, ActorRole: actorRole,
		BeforeJSON: string(before), AfterJSON: string(after), CreatedAt: s.now(),
	})
}

// UnholdRelationship clears freeze back to ACTIVE.
func (s *PolicyService) UnholdRelationship(ctx context.Context, supplierID, retailerID, actorUserID, actorRole string) error {
	if s.credit == nil {
		return fmt.Errorf("credit service unavailable")
	}
	terms, ok, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return err
	}
	if !ok || !terms.CreditEnabled {
		return ErrCreditNotEnabled
	}
	prof, found, err := s.credit.GetProfile(ctx, retailerID, supplierID)
	if err != nil {
		return err
	}
	if !found {
		return ErrProfileNotFound
	}
	before, _ := json.Marshal(prof)
	prof.Status = StatusActive
	if err := s.credit.UpsertProfile(ctx, prof, actorUserID, "unhold"); err != nil {
		return err
	}
	after, _ := json.Marshal(prof)
	return s.repo.AppendAudit(ctx, CreditPolicyAuditRow{
		AuditID: s.newID(), SupplierID: supplierID, RetailerID: retailerID, Action: "UNHOLD",
		ActorUserID: actorUserID, ActorRole: actorRole,
		BeforeJSON: string(before), AfterJSON: string(after), CreatedAt: s.now(),
	})
}

func (s *PolicyService) GetProgram(ctx context.Context, supplierID string) (SupplierCreditProgram, bool, error) {
	return s.repo.GetProgram(ctx, supplierID)
}

func (s *PolicyService) ListRelationships(ctx context.Context, supplierID string, limit int) ([]RetailerPaymentTerms, error) {
	return s.repo.ListTermsBySupplier(ctx, supplierID, limit)
}

func (s *PolicyService) ListRetailerRelationships(ctx context.Context, retailerID string, limit int) ([]RetailerPaymentTerms, error) {
	return s.repo.ListTermsByRetailer(ctx, retailerID, limit)
}

func (s *PolicyService) ResolveTermsFor(ctx context.Context, retailerID, supplierID string) (ResolvedTerms, error) {
	prog, _, err := s.repo.GetProgram(ctx, supplierID)
	if err != nil {
		return ResolvedTerms{}, err
	}
	terms, _, err := s.repo.GetTerms(ctx, retailerID, supplierID)
	if err != nil {
		return ResolvedTerms{}, err
	}
	return s.resolveTerms(prog, terms), nil
}
