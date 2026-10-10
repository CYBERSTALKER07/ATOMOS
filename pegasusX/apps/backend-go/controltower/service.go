package controltower

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service is the application facade for playbook HTTP and worker entrypoints.
type Service struct {
	engine *Engine
	repo   Repository
	cfg    Config
}

func NewService(repo Repository, engine *Engine, cfg Config) *Service {
	return &Service{repo: repo, engine: engine, cfg: cfg}
}

func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

func (s *Service) Evaluate(ctx context.Context, supplierID string, exceptionIDs []string) error {
	if s.engine == nil {
		return nil
	}
	return s.engine.Evaluate(ctx, supplierID, exceptionIDs)
}

func (s *Service) ApproveRun(ctx context.Context, runID, actor string) error {
	return s.engine.ApproveRun(ctx, runID, actor)
}

func (s *Service) SkipRun(ctx context.Context, runID, actor string) error {
	return s.engine.SkipRun(ctx, runID, actor)
}

func (s *Service) ListPlaybooks(ctx context.Context, supplierID string) ([]Playbook, error) {
	if s.repo == nil {
		return nil, nil
	}
	return s.repo.ListActivePlaybooks(ctx, supplierID)
}

func (s *Service) CreatePlaybook(ctx context.Context, supplierID, actor string, in CreatePlaybookInput) (Playbook, error) {
	if s.repo == nil {
		return Playbook{}, fmt.Errorf("repository_unavailable")
	}
	matchRaw, _ := json.Marshal(in.MatchRules)
	actionsRaw, _ := json.Marshal(in.Actions)
	now := time.Now()
	pb := Playbook{
		PlaybookID:    uuid.NewString(),
		SupplierID:    supplierID,
		Name:          strings.TrimSpace(in.Name),
		Description:   in.Description,
		IsActive:      true,
		Priority:      in.Priority,
		MatchRules:    in.MatchRules,
		MatchRulesRaw: matchRaw,
		Actions:       in.Actions,
		ActionsRaw:    actionsRaw,
		AutoExecute:   in.AutoExecute,
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedBy:     actor,
	}
	if pb.Priority == 0 {
		pb.Priority = 50
	}
	if err := s.repo.CreatePlaybook(ctx, pb); err != nil {
		return Playbook{}, err
	}
	return pb, nil
}

func (s *Service) UpdatePlaybook(ctx context.Context, playbookID string, in UpdatePlaybookInput) error {
	fields := map[string]any{}
	if in.Name != nil {
		fields["Name"] = *in.Name
	}
	if in.Description != nil {
		fields["Description"] = *in.Description
	}
	if in.Priority != nil {
		fields["Priority"] = *in.Priority
	}
	if in.MatchRules != nil {
		raw, _ := json.Marshal(*in.MatchRules)
		fields["MatchRulesJson"] = raw
	}
	if in.Actions != nil {
		raw, _ := json.Marshal(*in.Actions)
		fields["ActionsJson"] = raw
	}
	if in.AutoExecute != nil {
		fields["AutoExecute"] = *in.AutoExecute
	}
	if in.IsActive != nil {
		fields["IsActive"] = *in.IsActive
	}
	if len(fields) == 0 {
		return fmt.Errorf("no_fields_to_update")
	}
	return s.repo.UpdatePlaybook(ctx, playbookID, fields)
}

func (s *Service) DeactivatePlaybook(ctx context.Context, playbookID string) error {
	return s.repo.DeactivatePlaybook(ctx, playbookID)
}

func (s *Service) ListRuns(ctx context.Context, supplierID, status string) ([]PlaybookRun, error) {
	return s.repo.ListRuns(ctx, supplierID, status, 100)
}

func (s *Service) ListRunsForException(ctx context.Context, exceptionID string) ([]PlaybookRun, error) {
	return s.repo.ListRunsForException(ctx, exceptionID)
}

func (s *Service) SeedIfEmpty(ctx context.Context) error {
	if sp, ok := s.repo.(*SpannerRepository); ok {
		return sp.SeedPlatformPlaybooks(ctx)
	}
	return nil
}

func (s *Service) ListSupplierIDsWithOpenExceptions(ctx context.Context) ([]string, error) {
	return s.repo.ListSupplierIDsWithOpenExceptions(ctx)
}

// ListOpenScored ranks open exceptions and attaches matching playbook recommendations.
func (s *Service) ListOpenScored(ctx context.Context, supplierID string, limit int) ([]ScoredException, error) {
	if s == nil || s.repo == nil || !s.Enabled() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	exceptions, err := s.repo.ListOpenExceptions(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	playbooks, err := s.repo.ListActivePlaybooks(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	scored := make([]ScoredException, 0, len(exceptions))
	for i := range exceptions {
		ex := exceptions[i]
		if s.engine != nil {
			_ = s.engine.enrichException(ctx, &ex)
		}
		scored = append(scored, buildScoredException(ex, playbooks, now))
	}
	sortScoredExceptions(scored)
	if len(scored) > limit {
		scored = scored[:limit]
	}
	return scored, nil
}

func sortScoredExceptions(rows []ScoredException) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return rows[i].ExceptionID < rows[j].ExceptionID
	})
}

// ScoredByOrderID returns scored exceptions keyed by order_id for enrichment.
func (s *Service) ScoredByOrderID(ctx context.Context, supplierID string) (map[string]ScoredException, error) {
	rows, err := s.ListOpenScored(ctx, supplierID, 200)
	if err != nil {
		return nil, err
	}
	out := make(map[string]ScoredException, len(rows))
	for _, row := range rows {
		oid := strings.TrimSpace(row.OrderID)
		if oid == "" {
			continue
		}
		if _, exists := out[oid]; !exists {
			out[oid] = row
		}
	}
	return out, nil
}

func (s *Service) AbortManifest(ctx context.Context, manifestID, supplierID, operatorID, reasonCode, notes string) error {
	return s.repo.RecordIntervention(ctx, manifestID, supplierID, operatorID, "ABORT_MANIFEST", reasonCode, notes)
}
