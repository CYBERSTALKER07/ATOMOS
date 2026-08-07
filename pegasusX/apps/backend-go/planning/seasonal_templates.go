package planning

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/seasonalcore"
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

// ActiveSeasonalTemplate returns the active template for a date (custom overrides win).
// Callers that need the multiplier for quantity math must apply tpl.Multiplier.
func (s *Service) ActiveSeasonalTemplate(ctx context.Context, supplierID string, on time.Time) (*SeasonalTemplate, string, error) {
	if custom, err := s.activeCustomOverride(ctx, supplierID, on); err != nil {
		return nil, "", err
	} else if custom != nil {
		mult := custom.Multiplier
		if mult <= 0 {
			mult = seasonalcore.ResolveOverrideMultiplier(nil, custom.TemplateID)
		}
		tpl := SeasonalTemplate{
			ID:              "custom:" + custom.OverrideID,
			Name:            custom.Name,
			Multiplier:      mult,
			ConfidenceFloor: 0.75,
		}
		return &tpl, "seasonal_template", nil
	}
	for _, core := range seasonalcore.Builtins {
		if seasonalcore.ActiveOn(core, on) {
			tpl := templateFromCore(core)
			return &tpl, "seasonal_template", nil
		}
	}
	return nil, "", nil
}

func (s *Service) activeCustomOverride(ctx context.Context, supplierID string, on time.Time) (*SeasonalOverrideRow, error) {
	if s == nil || s.Spanner == nil {
		return nil, nil
	}
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OverrideId, SupplierId, TemplateId, COALESCE(Name,''), StartDate, EndDate, IsActive,
		             COALESCE(Multiplier, 0)
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		        AND StartDate <= @on AND EndDate >= @on
		      ORDER BY StartDate DESC LIMIT 1`,
		Params: map[string]any{"sid": supplierID, "on": on},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r SeasonalOverrideRow
	var start, end time.Time
	if err := row.Columns(&r.OverrideID, &r.SupplierID, &r.TemplateID, &r.Name, &start, &end, &r.IsActive, &r.Multiplier); err != nil {
		return nil, err
	}
	r.StartDate = start.Format("2006-01-02")
	r.EndDate = end.Format("2006-01-02")
	if r.Multiplier <= 0 {
		r.Multiplier = seasonalcore.ResolveOverrideMultiplier(nil, r.TemplateID)
	}
	return &r, nil
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
			"StartDate":  start,
			"EndDate":    end,
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
	iter := s.Spanner.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OverrideId, SupplierId, TemplateId, COALESCE(Name,''), StartDate, EndDate, IsActive,
		             COALESCE(Multiplier, 0)
		      FROM SeasonalTemplateOverrides
		      WHERE SupplierId = @sid AND IsActive = true
		      ORDER BY StartDate DESC LIMIT 50`,
		Params: map[string]any{"sid": supplierID},
	})
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
		var start, end time.Time
		if err := row.Columns(&r.OverrideID, &r.SupplierID, &r.TemplateID, &r.Name, &start, &end, &r.IsActive, &r.Multiplier); err != nil {
			continue
		}
		r.StartDate = start.Format("2006-01-02")
		r.EndDate = end.Format("2006-01-02")
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
