package roi

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Service orchestrates M1.8 ROI baseline capture, value event attribution,
// platform-fee ROI multiple calculation (in bps), and QBR exports.
type Service struct {
	repo       Repository
	feeReader  PlatformFeeReader
	histReader HistoricalBaselineReader
	audit      AuditRecorder
	now        func() time.Time
}

// NewService constructs a Spanner-backed ROI service.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{
			now: func() time.Time { return time.Now().UTC() },
		}
	}
	sr := NewSpannerRepository(client)
	return &Service{
		repo:       sr,
		feeReader:  sr,
		histReader: sr,
		now:        func() time.Time { return time.Now().UTC() },
	}
}

// NewServiceWithRepository constructs an ROI service backed by a custom Repository (e.g., MemoryRepository).
func NewServiceWithRepository(repo Repository) *Service {
	svc := &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
	if fr, ok := repo.(PlatformFeeReader); ok {
		svc.feeReader = fr
	}
	if hr, ok := repo.(HistoricalBaselineReader); ok {
		svc.histReader = hr
	}
	return svc
}

// Repo exposes the underlying Repository.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// SetPlatformFeeReader sets a custom platform fee reader.
func (s *Service) SetPlatformFeeReader(r PlatformFeeReader) {
	if s != nil {
		s.feeReader = r
	}
}

// SetHistoricalBaselineReader sets a custom historical baseline reader.
func (s *Service) SetHistoricalBaselineReader(r HistoricalBaselineReader) {
	if s != nil {
		s.histReader = r
	}
}

// SetAuditRecorder attaches an immutable audit recorder.
func (s *Service) SetAuditRecorder(rec AuditRecorder) {
	if s != nil {
		s.audit = rec
	}
}

func (s *Service) currentNow() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

// CaptureBaseline snapshots 90-day historical AR/bad-debt/expired-stock metrics
// augmented by onboarding survey estimates and persists the baseline row.
func (s *Service) CaptureBaseline(ctx context.Context, in CaptureBaselineInput) (Baseline, error) {
	if s == nil || s.repo == nil {
		return Baseline{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	if tid == "" {
		return Baseline{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if in.Survey != nil {
		if err := validateNoFloats(in.Survey); err != nil {
			return Baseline{}, err
		}
	}
	if in.BaselineStockoutMinor < 0 || in.BaselineLogisticsMinor < 0 || in.BaselineLabourHours < 0 || in.SurveyHourlyWageMinor < 0 {
		return Baseline{}, fmt.Errorf("%w: baseline metrics must be non-negative", ErrInvalidInput)
	}
	if in.BaselineArMinor != nil && *in.BaselineArMinor < 0 {
		return Baseline{}, fmt.Errorf("%w: baseline_ar_minor must be non-negative", ErrInvalidInput)
	}
	if in.BaselineBadDebtMinor != nil && *in.BaselineBadDebtMinor < 0 {
		return Baseline{}, fmt.Errorf("%w: baseline_bad_debt_minor must be non-negative", ErrInvalidInput)
	}
	if in.BaselineExpiredMinor != nil && *in.BaselineExpiredMinor < 0 {
		return Baseline{}, fmt.Errorf("%w: baseline_expired_minor must be non-negative", ErrInvalidInput)
	}

	windowDays := in.WindowDays
	if windowDays <= 0 {
		windowDays = 90
	}
	if windowDays > 365 {
		windowDays = 365
	}
	now := s.currentNow()
	periodEnd := now
	periodStart := now.Add(-time.Duration(windowDays) * 24 * time.Hour)

	var hist HistoricalMetrics
	if s.histReader != nil {
		var err error
		hist, err = s.histReader.ReadHistoricalMetrics(ctx, tid, periodStart, periodEnd)
		if err != nil {
			return Baseline{}, err
		}
	}

	arMinor := hist.ArMinor
	if in.BaselineArMinor != nil {
		arMinor = *in.BaselineArMinor
	}
	badDebtMinor := hist.BadDebtMinor
	if in.BaselineBadDebtMinor != nil {
		badDebtMinor = *in.BaselineBadDebtMinor
	}
	expiredMinor := hist.ExpiredMinor
	if in.BaselineExpiredMinor != nil {
		expiredMinor = *in.BaselineExpiredMinor
	}

	cur := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cur == "" {
		cur = strings.ToUpper(strings.TrimSpace(hist.Currency))
	}
	if cur == "" {
		cur = "UZS"
	}

	surveyMap := in.Survey
	if surveyMap == nil {
		surveyMap = map[string]any{}
	}
	surveyRaw, err := json.Marshal(surveyMap)
	if err != nil {
		return Baseline{}, fmt.Errorf("%w: marshal survey: %v", ErrInvalidInput, err)
	}
	capturedBy := strings.TrimSpace(in.CapturedBy)
	if capturedBy == "" {
		capturedBy = auth.ActorFromContext(ctx)
	}

	b := Baseline{
		TenantID:               tid,
		BaselineID:             "roib_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
		WindowDays:             windowDays,
		BaselineArMinor:        arMinor,
		BaselineBadDebtMinor:   badDebtMinor,
		BaselineExpiredMinor:   expiredMinor,
		BaselineStockoutMinor:  in.BaselineStockoutMinor,
		BaselineLogisticsMinor: in.BaselineLogisticsMinor,
		BaselineLabourHours:    in.BaselineLabourHours,
		SurveyHourlyWageMinor:  in.SurveyHourlyWageMinor,
		SurveyNotes:            strings.TrimSpace(in.SurveyNotes),
		SurveyJSON:             string(surveyRaw),
		Survey:                 surveyMap,
		Currency:               cur,
		CapturedBy:             capturedBy,
		PeriodStart:            periodStart,
		PeriodEnd:              periodEnd,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := s.repo.UpsertBaseline(ctx, b); err != nil {
		return Baseline{}, err
	}
	if s.audit != nil {
		afterRaw, _ := json.Marshal(b)
		_ = s.audit.RecordEvent(ctx, AuditRecordInput{
			TenantID:   tid,
			ActorID:    capturedBy,
			ActorType:  auth.ActorTypeHuman,
			Action:     "roi.baseline.captured",
			ObjectType: "RoiBaseline",
			ObjectID:   b.BaselineID,
			AfterJSON:  string(afterRaw),
		})
	}
	return b, nil
}

// GetLatestBaseline fetches the most recently captured baseline for tenantID.
func (s *Service) GetLatestBaseline(ctx context.Context, tenantID string) (Baseline, bool, error) {
	if s == nil || s.repo == nil {
		return Baseline{}, false, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return Baseline{}, false, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	return s.repo.GetLatestBaseline(ctx, tid)
}

// RecordEvent validates and records a single ROI value attribution event and emits roi.event.recorded.
func (s *Service) RecordEvent(ctx context.Context, in RecordEventInput) (RoiEvent, error) {
	if s == nil || s.repo == nil {
		return RoiEvent{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(in.TenantID)
	if tid == "" {
		return RoiEvent{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	cat, valid := NormalizeCategory(in.Category)
	if !valid {
		return RoiEvent{}, fmt.Errorf("%w: unsupported roi category %q", ErrInvalidInput, in.Category)
	}
	if in.SavingsMinor < 0 {
		return RoiEvent{}, fmt.Errorf("%w: savings_minor must be >= 0", ErrInvalidInput)
	}

	if idem := strings.TrimSpace(in.IdempotencyKey); idem != "" {
		if existing, found, err := s.repo.FindEventByIdempotency(ctx, tid, idem); err != nil {
			return RoiEvent{}, err
		} else if found {
			return existing, nil
		}
	}

	methodology := strings.TrimSpace(in.Methodology)
	if methodology == "" {
		methodology = DefaultMethodology(cat)
	}
	confBps := in.ConfidenceBps
	if confBps <= 0 {
		confBps = DefaultConfidenceBps(cat)
	}
	if confBps > 10000 {
		return RoiEvent{}, fmt.Errorf("%w: confidence_bps must be between 0 and 10000", ErrInvalidInput)
	}
	isEstimate := false
	if in.IsEstimate != nil {
		isEstimate = *in.IsEstimate
	}
	if cat == CategoryLabourSaved {
		// Labour savings are derived from survey hourly rates and are always flagged as estimates.
		isEstimate = true
	}

	cur := strings.ToUpper(strings.TrimSpace(in.Currency))
	if cur == "" {
		cur = "UZS"
	}
	recordedBy := strings.TrimSpace(in.RecordedBy)
	if recordedBy == "" {
		recordedBy = auth.ActorFromContext(ctx)
	}
	now := s.currentNow()
	occurredAt := in.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = now
	}

	ev := RoiEvent{
		TenantID:        tid,
		EventID:         "roie_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24],
		Category:        cat,
		SavingsMinor:    in.SavingsMinor,
		Currency:        cur,
		Methodology:     methodology,
		ConfidenceBps:   confBps,
		IsEstimate:      isEstimate,
		HoldoutControl:  in.HoldoutControl,
		ProposalID:      strings.TrimSpace(in.ProposalID),
		EvidenceRefType: strings.TrimSpace(in.EvidenceRefType),
		EvidenceRefID:   strings.TrimSpace(in.EvidenceRefID),
		Notes:           strings.TrimSpace(in.Notes),
		RecordedBy:      recordedBy,
		IdempotencyKey:  strings.TrimSpace(in.IdempotencyKey),
		OccurredAt:      occurredAt,
		CreatedAt:       now,
	}
	if err := s.repo.CreateEvent(ctx, ev); err != nil {
		return RoiEvent{}, err
	}
	return ev, nil
}

// ListEvents lists ROI events matching filter f.
func (s *Service) ListEvents(ctx context.Context, f EventsFilter) ([]RoiEvent, error) {
	if s == nil || s.repo == nil {
		return nil, ErrRepoUnavailable
	}
	f.TenantID = strings.TrimSpace(f.TenantID)
	if f.TenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if cat := strings.TrimSpace(f.Category); cat != "" {
		norm, ok := NormalizeCategory(cat)
		if !ok {
			return nil, fmt.Errorf("%w: unsupported roi category %q", ErrInvalidInput, cat)
		}
		f.Category = norm
	}
	return s.repo.ListEvents(ctx, f)
}

// GetSummary computes aggregated savings by category, treatment vs holdout comparison,
// platform fees from BillingMeterEvents, and net ROI multiple in basis points.
func (s *Service) GetSummary(ctx context.Context, tenantID string, windowDays int) (Summary, error) {
	if s == nil || s.repo == nil {
		return Summary{}, ErrRepoUnavailable
	}
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return Summary{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if windowDays <= 0 {
		windowDays = 90
	}
	if windowDays > 3650 {
		windowDays = 3650
	}
	now := s.currentNow()
	since := now.Add(-time.Duration(windowDays) * 24 * time.Hour)
	until := now.Add(time.Minute)

	eventsList, err := s.repo.ListEvents(ctx, EventsFilter{
		TenantID: tid,
		Since:    &since,
		Until:    &until,
		Limit:    5000,
	})
	if err != nil {
		return Summary{}, err
	}

	var baselinePtr *Baseline
	if b, ok, err := s.repo.GetLatestBaseline(ctx, tid); err != nil {
		return Summary{}, err
	} else if ok {
		baselinePtr = &b
	}

	var platformFeesMinor int64
	feeCurrency := ""
	if s.feeReader != nil {
		fees, cur, fErr := s.feeReader.ReadPlatformFeesMinor(ctx, tid, since, until)
		if fErr != nil {
			return Summary{}, fErr
		}
		platformFeesMinor = fees
		feeCurrency = cur
	}

	currency := feeCurrency
	if currency == "" && baselinePtr != nil {
		currency = baselinePtr.Currency
	}

	type catAgg struct {
		count          int64
		totalMinor     int64
		realizedMinor  int64
		estimatedMinor int64
		holdoutMinor   int64
		confSum        int64
		methodology    string
	}
	aggs := make(map[string]*catAgg, len(AllCategories()))
	for _, c := range AllCategories() {
		aggs[c] = &catAgg{methodology: DefaultMethodology(c)}
	}

	var (
		totalSavingsMinor     int64
		realizedSavingsMinor  int64
		estimatedSavingsMinor int64
		holdoutSavingsMinor   int64
		treatmentCount        int64
		holdoutCount          int64
	)

	for _, ev := range eventsList {
		if currency == "" && ev.Currency != "" {
			currency = ev.Currency
		}
		a, ok := aggs[ev.Category]
		if !ok {
			a = &catAgg{methodology: DefaultMethodology(ev.Category)}
			aggs[ev.Category] = a
		}
		a.count++
		a.confSum += ev.ConfidenceBps
		if strings.TrimSpace(ev.Methodology) != "" {
			a.methodology = ev.Methodology
		}

		if ev.HoldoutControl {
			holdoutCount++
			holdoutSavingsMinor += ev.SavingsMinor
			a.holdoutMinor += ev.SavingsMinor
			continue
		}

		treatmentCount++
		totalSavingsMinor += ev.SavingsMinor
		a.totalMinor += ev.SavingsMinor
		if ev.IsEstimate {
			estimatedSavingsMinor += ev.SavingsMinor
			a.estimatedMinor += ev.SavingsMinor
		} else {
			realizedSavingsMinor += ev.SavingsMinor
			a.realizedMinor += ev.SavingsMinor
		}
	}

	if currency == "" {
		currency = "UZS"
	}

	categories := make([]CategorySummary, 0, len(aggs))
	for _, cat := range AllCategories() {
		a := aggs[cat]
		avgConf := DefaultConfidenceBps(cat)
		if a.count > 0 {
			avgConf = a.confSum / a.count
		}
		categories = append(categories, CategorySummary{
			Category:            cat,
			EventCount:          a.count,
			TotalSavingsMinor:   a.totalMinor,
			RealizedMinor:       a.realizedMinor,
			EstimatedMinor:      a.estimatedMinor,
			HoldoutSavingsMinor: a.holdoutMinor,
			AvgConfidenceBps:    avgConf,
			Methodology:         a.methodology,
		})
	}
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].TotalSavingsMinor != categories[j].TotalSavingsMinor {
			return categories[i].TotalSavingsMinor > categories[j].TotalSavingsMinor
		}
		return categories[i].Category < categories[j].Category
	})

	netSavingsMinor := totalSavingsMinor - platformFeesMinor
	var roiMultipleBps int64
	if platformFeesMinor > 0 {
		roiMultipleBps = (totalSavingsMinor * 10000) / platformFeesMinor
	}

	incrementalLiftMinor := totalSavingsMinor - holdoutSavingsMinor
	var liftBps int64
	if holdoutSavingsMinor > 0 {
		liftBps = (incrementalLiftMinor * 10000) / holdoutSavingsMinor
	} else if totalSavingsMinor > 0 && treatmentCount > 0 {
		liftBps = 10000
	}

	return Summary{
		TenantID:              tid,
		Currency:              currency,
		WindowDays:            int64(windowDays),
		PeriodStart:           since,
		PeriodEnd:             now,
		TotalSavingsMinor:     totalSavingsMinor,
		RealizedSavingsMinor:  realizedSavingsMinor,
		EstimatedSavingsMinor: estimatedSavingsMinor,
		HoldoutSavingsMinor:   holdoutSavingsMinor,
		PlatformFeesMinor:     platformFeesMinor,
		NetSavingsMinor:       netSavingsMinor,
		RoiMultipleBps:        roiMultipleBps,
		Categories:            categories,
		HoldoutStats: HoldoutStats{
			TreatmentEventCount:   treatmentCount,
			TreatmentSavingsMinor: totalSavingsMinor,
			HoldoutEventCount:     holdoutCount,
			HoldoutSavingsMinor:   holdoutSavingsMinor,
			IncrementalLiftMinor:  incrementalLiftMinor,
			LiftBps:               liftBps,
		},
		Baseline: baselinePtr,
	}, nil
}

// ExportQBR builds a structured QBR JSON report combining Summary and underlying events.
func (s *Service) ExportQBR(ctx context.Context, tenantID string, windowDays int) (QBRReport, error) {
	sum, err := s.GetSummary(ctx, tenantID, windowDays)
	if err != nil {
		return QBRReport{}, err
	}
	since := sum.PeriodStart
	until := sum.PeriodEnd.Add(time.Minute)
	evs, err := s.repo.ListEvents(ctx, EventsFilter{
		TenantID: sum.TenantID,
		Since:    &since,
		Until:    &until,
		Limit:    5000,
	})
	if err != nil {
		return QBRReport{}, err
	}
	if evs == nil {
		evs = []RoiEvent{}
	}
	return QBRReport{
		GeneratedAt: s.currentNow(),
		Summary:     sum,
		Events:      evs,
	}, nil
}

// ExportQBRCSV renders the QBR report as RFC 4180 CSV with summary KPIs,
// per-category methodology & confidence, holdout comparison, and event rows.
func (s *Service) ExportQBRCSV(ctx context.Context, tenantID string, windowDays int) ([]byte, error) {
	rep, err := s.ExportQBR(ctx, tenantID, windowDays)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	_ = w.Write([]string{
		"section", "tenant_id", "category", "event_count", "savings_minor",
		"realized_minor", "estimated_minor", "holdout_savings_minor",
		"platform_fees_minor", "net_savings_minor", "roi_multiple_bps",
		"confidence_bps", "is_estimate", "holdout_control", "currency",
		"proposal_id", "evidence_ref_id", "methodology", "occurred_at",
	})
	sum := rep.Summary
	_ = w.Write([]string{
		"SUMMARY", sum.TenantID, "ALL",
		strconv.FormatInt(sum.HoldoutStats.TreatmentEventCount, 10),
		strconv.FormatInt(sum.TotalSavingsMinor, 10),
		strconv.FormatInt(sum.RealizedSavingsMinor, 10),
		strconv.FormatInt(sum.EstimatedSavingsMinor, 10),
		strconv.FormatInt(sum.HoldoutSavingsMinor, 10),
		strconv.FormatInt(sum.PlatformFeesMinor, 10),
		strconv.FormatInt(sum.NetSavingsMinor, 10),
		strconv.FormatInt(sum.RoiMultipleBps, 10),
		"", "", "", sum.Currency, "", "",
		"Net ROI multiple in bps (10000 = 1.00x) comparing treatment savings against BillingMeterEvents platform fees",
		rep.GeneratedAt.Format(time.RFC3339),
	})
	for _, cat := range sum.Categories {
		_ = w.Write([]string{
			"CATEGORY", sum.TenantID, cat.Category,
			strconv.FormatInt(cat.EventCount, 10),
			strconv.FormatInt(cat.TotalSavingsMinor, 10),
			strconv.FormatInt(cat.RealizedMinor, 10),
			strconv.FormatInt(cat.EstimatedMinor, 10),
			strconv.FormatInt(cat.HoldoutSavingsMinor, 10),
			"", "", "",
			strconv.FormatInt(cat.AvgConfidenceBps, 10),
			"", "", sum.Currency, "", "", cat.Methodology, "",
		})
	}
	for _, ev := range rep.Events {
		_ = w.Write([]string{
			"EVENT", ev.TenantID, ev.Category, "1",
			strconv.FormatInt(ev.SavingsMinor, 10),
			"", "", "", "", "", "",
			strconv.FormatInt(ev.ConfidenceBps, 10),
			strconv.FormatBool(ev.IsEstimate),
			strconv.FormatBool(ev.HoldoutControl),
			ev.Currency, ev.ProposalID, ev.EvidenceRefID,
			ev.Methodology, ev.OccurredAt.Format(time.RFC3339),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
