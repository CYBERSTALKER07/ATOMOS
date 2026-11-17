package planning

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// SeasonalTemplate describes a hard-coded seasonal surge curve.
type SeasonalTemplate struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	StartMonth      int     `json:"start_month"`
	StartDay        int     `json:"start_day"`
	EndMonth        int     `json:"end_month"`
	EndDay          int     `json:"end_day"`
	Multiplier      float64 `json:"multiplier"`
	ConfidenceFloor float64 `json:"confidence_floor"`
}

// SeasonalOverrideInput is a custom date-range override.
type SeasonalOverrideInput struct {
	TemplateID string   `json:"template_id"`
	StartDate  string   `json:"start_date"`
	EndDate    string   `json:"end_date"`
	Name       string   `json:"name,omitempty"`
	Multiplier *float64 `json:"multiplier,omitempty"`
}

// SeasonalOverrideRow is a persisted custom season.
type SeasonalOverrideRow struct {
	OverrideID string  `json:"override_id"`
	SupplierID string  `json:"supplier_id"`
	TemplateID string  `json:"template_id"`
	Name       string  `json:"name,omitempty"`
	StartDate  string  `json:"start_date"`
	EndDate    string  `json:"end_date"`
	IsActive   bool    `json:"is_active"`
	Multiplier float64 `json:"multiplier"`
}

func templateFromCore(tpl seasonalcore.Template) SeasonalTemplate {
	return SeasonalTemplate{
		ID:              tpl.ID,
		Name:            tpl.Name,
		StartMonth:      tpl.StartMonth,
		StartDay:        tpl.StartDay,
		EndMonth:        tpl.EndMonth,
		EndDay:          tpl.EndDay,
		Multiplier:      tpl.Multiplier,
		ConfidenceFloor: tpl.ConfidenceFloor,
	}
}

// ComposeSeasonalOverrides deterministically orders and composes all active
// overlapping seasonal override rows without dropping any window.
func ComposeSeasonalOverrides(rows []SeasonalOverrideRow) *SeasonalTemplate {
	if len(rows) == 0 {
		return nil
	}
	ordered := make([]SeasonalOverrideRow, len(rows))
	copy(ordered, rows)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].StartDate != ordered[j].StartDate {
			return ordered[i].StartDate < ordered[j].StartDate
		}
		if ordered[i].EndDate != ordered[j].EndDate {
			return ordered[i].EndDate < ordered[j].EndDate
		}
		return ordered[i].OverrideID < ordered[j].OverrideID
	})
	mults := make([]float64, 0, len(ordered))
	ids := make([]string, 0, len(ordered))
	names := make([]string, 0, len(ordered))
	for _, r := range ordered {
		m := r.Multiplier
		if m <= 0 {
			m = seasonalcore.ResolveOverrideMultiplier(nil, r.TemplateID)
		} else {
			m = seasonalcore.ClampMultiplier(m)
		}
		mults = append(mults, m)
		ids = append(ids, r.OverrideID)
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = strings.TrimSpace(r.TemplateID)
		}
		if name != "" {
			names = append(names, name)
		}
	}
	composedMult := seasonalcore.ClampMultiplier(seasonalcore.ComposeMultipliers(mults))
	if len(ordered) == 1 {
		return &SeasonalTemplate{
			ID:              "custom:" + ordered[0].OverrideID,
			Name:            strings.Join(names, " + "),
			Multiplier:      composedMult,
			ConfidenceFloor: 0.75,
		}
	}
	return &SeasonalTemplate{
		ID:              "custom:" + strings.Join(ids, "+"),
		Name:            strings.Join(names, " + "),
		Multiplier:      composedMult,
		ConfidenceFloor: 0.75,
	}
}

// ActiveSeasonalTemplate returns the composed active template for a date (custom overrides win).
// Callers that need the multiplier for quantity math must apply tpl.Multiplier.
func (s *Service) ActiveSeasonalTemplate(ctx context.Context, supplierID string, on time.Time) (*SeasonalTemplate, string, error) {
	overrides, err := s.activeCustomOverrides(ctx, supplierID, on)
	if err != nil {
		return nil, "", err
	}
	if composed := ComposeSeasonalOverrides(overrides); composed != nil {
		return composed, "seasonal_template", nil
	}
	var activeBuiltins []seasonalcore.Template
	for _, core := range seasonalcore.Builtins {
		if seasonalcore.ActiveOn(core, on) {
			activeBuiltins = append(activeBuiltins, core)
		}
	}
	if len(activeBuiltins) == 1 {
		tpl := templateFromCore(activeBuiltins[0])
		return &tpl, "seasonal_template", nil
	}
	if len(activeBuiltins) > 1 {
		mults := make([]float64, 0, len(activeBuiltins))
		ids := make([]string, 0, len(activeBuiltins))
		names := make([]string, 0, len(activeBuiltins))
		for _, core := range activeBuiltins {
			mults = append(mults, core.Multiplier)
			ids = append(ids, core.ID)
			names = append(names, core.Name)
		}
		tpl := SeasonalTemplate{
			ID:              strings.Join(ids, "+"),
			Name:            strings.Join(names, " + "),
			Multiplier:      seasonalcore.ClampMultiplier(seasonalcore.ComposeMultipliers(mults)),
			ConfidenceFloor: 0.75,
		}
		return &tpl, "seasonal_template", nil
	}
	return nil, "", nil
}

func (s *Service) activeCustomOverrides(ctx context.Context, supplierID string, on time.Time) ([]SeasonalOverrideRow, error) {
	if s == nil || s.Spanner == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT OverrideId, SupplierId, TemplateId, COALESCE(Name,''), StartDate, EndDate, IsActive,
		             COALESCE(Multiplier, 0)
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		        AND StartDate <= @on AND EndDate >= @on
		      ORDER BY StartDate ASC, OverrideId ASC`,
		Params: map[string]any{"sid": supplierID, "on": civil.DateOf(on.UTC())},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.Spanner.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	var rows []SeasonalOverrideRow
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var r SeasonalOverrideRow
		var start, end civil.Date
		if err := row.Columns(&r.OverrideID, &r.SupplierID, &r.TemplateID, &r.Name, &start, &end, &r.IsActive, &r.Multiplier); err != nil {
			return nil, err
		}
		r.StartDate = start.String()
		r.EndDate = end.String()
		if r.Multiplier <= 0 {
			r.Multiplier = seasonalcore.ResolveOverrideMultiplier(nil, r.TemplateID)
		} else {
			r.Multiplier = seasonalcore.ClampMultiplier(r.Multiplier)
		}
		rows = append(rows, r)
	}
	return rows, nil
}

// SeedUzbekistanHolidays idempotently seeds the 2025-2035 Uzbekistan holiday schedule for supplierID.
func (s *Service) SeedUzbekistanHolidays(ctx context.Context, supplierID string) (int, error) {
	if s == nil || s.Spanner == nil {
		return 0, errors.New("planning unavailable")
	}
	n, err := seasonalcore.SeedUzbekistanHolidays(ctx, s.Spanner, supplierID)
	if err != nil {
		return 0, err
	}
	for _, tid := range []string{"uz_ramadan", "uz_ramazon_hayit", "uz_qurbon_hayit", "uz_navruz", "uz_independence_day"} {
		s.invalidateSeasonalCache(ctx, supplierID, tid)
	}
	return n, nil
}

// CreateSeasonalOverride persists a custom seasonal window with a resolved Multiplier.
func (s *Service) CreateSeasonalOverride(ctx context.Context, supplierID string, in SeasonalOverrideInput) (SeasonalOverrideRow, error) {
	if s == nil || s.Spanner == nil {
		return SeasonalOverrideRow{}, errors.New("planning unavailable")
	}
	start, err := time.Parse("2006-01-02", strings.TrimSpace(in.StartDate))
	if err != nil {
		return SeasonalOverrideRow{}, errors.New("invalid_start_date")
	}
	end, err := time.Parse("2006-01-02", strings.TrimSpace(in.EndDate))
	if err != nil {
		return SeasonalOverrideRow{}, errors.New("invalid_end_date")
	}
	if end.Before(start) {
		return SeasonalOverrideRow{}, errors.New("end_before_start")
	}
	overrideID := uuid.NewString()
	templateID := strings.TrimSpace(in.TemplateID)
	if templateID == "" {
		templateID = "custom"
	}
	mult := seasonalcore.ResolveOverrideMultiplier(in.Multiplier, templateID)
	row := SeasonalOverrideRow{
		OverrideID: overrideID,
		SupplierID: supplierID,
		TemplateID: templateID,
		Name:       strings.TrimSpace(in.Name),
		StartDate:  start.Format("2006-01-02"),
		EndDate:    end.Format("2006-01-02"),
		IsActive:   true,
		Multiplier: mult,
	}
	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("SeasonalTemplateOverrides", map[string]any{
			"OverrideId": overrideID,
			"SupplierId": supplierID,
			"TemplateId": templateID,
			"Name":       row.Name,
			"StartDate":  civil.DateOf(start),
			"EndDate":    civil.DateOf(end),
			"IsActive":   true,
			"Multiplier": mult,
			"CreatedAt":  spanner.CommitTimestamp,
		})})
	})
	if err == nil {
		s.invalidateSeasonalCache(ctx, supplierID, templateID)
	}
	return row, err
}

// ListSeasonalOverrides returns active custom overrides for a supplier.
func (s *Service) ListSeasonalOverrides(ctx context.Context, supplierID string) ([]SeasonalOverrideRow, error) {
	if s == nil || s.Spanner == nil {
		return nil, errors.New("planning unavailable")
	}
	stmt := spanner.Statement{
		SQL: `SELECT OverrideId, SupplierId, TemplateId, COALESCE(Name,''), StartDate, EndDate, IsActive,
		             COALESCE(Multiplier, 0)
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		      ORDER BY StartDate ASC, OverrideId ASC LIMIT 200`,
		Params: map[string]any{"sid": supplierID},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.Spanner.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	var rows []SeasonalOverrideRow
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var r SeasonalOverrideRow
		var start, end civil.Date
		if err := row.Columns(&r.OverrideID, &r.SupplierID, &r.TemplateID, &r.Name, &start, &end, &r.IsActive, &r.Multiplier); err != nil {
			continue
		}
		r.StartDate = start.String()
		r.EndDate = end.String()
		if r.Multiplier <= 0 {
			r.Multiplier = seasonalcore.ResolveOverrideMultiplier(nil, r.TemplateID)
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (s *Service) invalidateSeasonalCache(ctx context.Context, supplierID, templateID string) {
	if s == nil || s.Cache == nil {
		return
	}
	s.Cache.Invalidate(ctx, SeasonalCacheKey(supplierID, templateID))
}

// CacheSeasonalTemplate stores active template metadata in Redis.
func (s *Service) CacheSeasonalTemplate(ctx context.Context, supplierID string, tpl SeasonalTemplate) error {
	if s == nil || s.Cache == nil {
		return nil
	}
	raw, err := json.Marshal(tpl)
	if err != nil {
		return err
	}
	return s.Cache.Set(ctx, SeasonalCacheKey(supplierID, tpl.ID), raw, time.Duration(seasonalCacheTTL)*time.Second)
}

// BuiltinSeasonalTemplates returns hard-coded templates for API listing.
func BuiltinSeasonalTemplates() []SeasonalTemplate {
	out := make([]SeasonalTemplate, len(seasonalcore.Builtins))
	for i, core := range seasonalcore.Builtins {
		out[i] = templateFromCore(core)
	}
	return out
}
