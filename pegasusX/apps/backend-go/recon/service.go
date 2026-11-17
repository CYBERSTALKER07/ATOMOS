package recon

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
)

var periodIDPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// IsValidPeriodID verifies YYYY-MM formatting matching GL period semantics.
func IsValidPeriodID(periodID string) bool {
	return periodIDPattern.MatchString(strings.TrimSpace(periodID))
}

// Service orchestrates four-way reconciliation runs, break queries, and break resolution workflows.
type Service struct {
	repo Repository
}

// NewService constructs a Service backed by Cloud Spanner.
func NewService(client *spanner.Client) *Service {
	if client == nil {
		return &Service{}
	}
	return &Service{repo: NewSpannerRepository(client)}
}

// NewServiceWithRepository constructs a Service using an explicit Repository (for unit/harness tests).
func NewServiceWithRepository(repo Repository) *Service {
	return &Service{repo: repo}
}

// Repo returns the underlying Repository implementation.
func (s *Service) Repo() Repository {
	if s == nil {
		return nil
	}
	return s.repo
}

// RunReconciliation executes a consistent four-way reconciliation sweep and atomically persists results.
func (s *Service) RunReconciliation(ctx context.Context, req RunRequest) (RunOutput, error) {
	if s == nil || s.repo == nil {
		return RunOutput{}, fmt.Errorf("recon service: nil repository")
	}
	entityID := strings.TrimSpace(req.EntityID)
	if entityID == "" {
		return RunOutput{}, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}

	now := time.Now().UTC()
	periodID := strings.TrimSpace(req.PeriodID)
	if periodID == "" {
		periodID = gl.ComputePeriodID(ctx, entityID, req.CountryPack, now)
	} else if !IsValidPeriodID(periodID) {
		return RunOutput{}, fmt.Errorf("%w: period_id must be formatted YYYY-MM", ErrInvalidInput)
	}
	currency := gl.DefaultCurrencyForPack(req.CountryPack)

	mode := strings.ToUpper(strings.TrimSpace(req.Mode))
	switch mode {
	case "", ModeManual:
		mode = ModeManual
	case ModeNightly, ModeTriggered:
	default:
		return RunOutput{}, fmt.Errorf("%w: unsupported mode %q", ErrInvalidInput, req.Mode)
	}

	for _, d := range req.Dimensions {
		up := strings.ToUpper(strings.TrimSpace(d))
		switch up {
		case DimensionOrdersAR, DimensionStockGL, DimensionCashPSP, DimensionTaxFiscal:
		default:
			return RunOutput{}, fmt.Errorf("%w: unsupported dimension %q", ErrInvalidInput, d)
		}
	}

	snap, err := s.repo.LoadSnapshot(ctx, entityID, periodID, currency, now)
	if err != nil {
		return RunOutput{}, err
	}

	run, breaks := EvaluateSnapshot(snap, req.Dimensions, now)
	run.Mode = mode
	run.TriggeredBy = coalesce(strings.TrimSpace(req.TriggeredBy), "system")

	savedRun, savedBreaks, err := s.repo.SaveRunAndBreaks(ctx, run, breaks)
	if err != nil {
		return RunOutput{}, err
	}
	return RunOutput{Run: savedRun, Breaks: savedBreaks}, nil
}

// ListRuns returns reconciliation run history for an entity.
func (s *Service) ListRuns(ctx context.Context, filter ListRunsFilter) ([]ReconciliationRun, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("recon service: nil repository")
	}
	if strings.TrimSpace(filter.EntityID) == "" {
		return nil, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	if p := strings.TrimSpace(filter.PeriodID); p != "" && !IsValidPeriodID(p) {
		return nil, fmt.Errorf("%w: period_id must be formatted YYYY-MM", ErrInvalidInput)
	}
	return s.repo.ListRuns(ctx, filter)
}

// ListBreaks returns reconciliation breaks matching filter within an entity scope.
func (s *Service) ListBreaks(ctx context.Context, filter ListBreaksFilter) ([]ReconciliationBreak, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("recon service: nil repository")
	}
	if strings.TrimSpace(filter.EntityID) == "" {
		return nil, fmt.Errorf("%w: entity_id is required", ErrInvalidInput)
	}
	return s.repo.ListBreaks(ctx, filter)
}

// GetBreak retrieves a single break scoped by entityID.
func (s *Service) GetBreak(ctx context.Context, entityID, breakID string) (ReconciliationBreak, error) {
	if s == nil || s.repo == nil {
		return ReconciliationBreak{}, fmt.Errorf("recon service: nil repository")
	}
	if strings.TrimSpace(entityID) == "" || strings.TrimSpace(breakID) == "" {
		return ReconciliationBreak{}, fmt.Errorf("%w: entity_id and break_id required", ErrInvalidInput)
	}
	return s.repo.GetBreak(ctx, entityID, breakID)
}

// PatchBreak applies ACKNOWLEDGE, RESOLVE, or WAIVE to an existing break and emits recon.break.resolved when closed.
func (s *Service) PatchBreak(ctx context.Context, entityID, breakID string, req PatchBreakRequest) (ReconciliationBreak, error) {
	if s == nil || s.repo == nil {
		return ReconciliationBreak{}, fmt.Errorf("recon service: nil repository")
	}
	if strings.TrimSpace(entityID) == "" || strings.TrimSpace(breakID) == "" {
		return ReconciliationBreak{}, fmt.Errorf("%w: entity_id and break_id required", ErrInvalidInput)
	}
	if _, err := resolveBreakTransition(BreakStatusOpen, req); err != nil {
		// Validate action/note format before DB transaction, allowing transition check against real status inside UpdateBreak.
		if !isInvalidTransitionOnly(err) {
			return ReconciliationBreak{}, err
		}
	}
	return s.repo.UpdateBreak(ctx, entityID, breakID, req, time.Now().UTC())
}

func resolveBreakTransition(currentStatus string, req PatchBreakRequest) (string, error) {
	cur := strings.ToUpper(strings.TrimSpace(currentStatus))
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	note := strings.TrimSpace(coalesce(req.Note, req.ResolutionNote))

	switch action {
	case ActionAcknowledge:
		if cur != BreakStatusOpen && cur != BreakStatusAcknowledged {
			return "", fmt.Errorf("%w: cannot %s break in status %s", ErrInvalidTransition, action, cur)
		}
		return BreakStatusAcknowledged, nil
	case ActionResolve:
		if cur != BreakStatusOpen && cur != BreakStatusAcknowledged {
			return "", fmt.Errorf("%w: cannot %s break in status %s", ErrInvalidTransition, action, cur)
		}
		return BreakStatusResolved, nil
	case ActionWaive:
		if cur != BreakStatusOpen && cur != BreakStatusAcknowledged {
			return "", fmt.Errorf("%w: cannot %s break in status %s", ErrInvalidTransition, action, cur)
		}
		if note == "" {
			return "", fmt.Errorf("%w: resolution note is required to WAIVE a break", ErrInvalidInput)
		}
		return BreakStatusWaived, nil
	default:
		return "", fmt.Errorf("%w: unsupported action %q (expected ACKNOWLEDGE, RESOLVE, or WAIVE)", ErrInvalidInput, req.Action)
	}
}

func isInvalidTransitionOnly(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrInvalidTransition.Error())
}
