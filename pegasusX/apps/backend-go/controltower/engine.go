package controltower

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/segment"
)

// Engine evaluates playbooks and executes runs.
type Engine struct {
	repo        Repository
	executor    *ActionExecutor
	segmentSvc  *segment.Service
	cfg         Config
	log         *slog.Logger
	now         func() time.Time
}

func NewEngine(repo Repository, executor *ActionExecutor, segmentSvc *segment.Service, cfg Config, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	return &Engine{
		repo:       repo,
		executor:   executor,
		segmentSvc: segmentSvc,
		cfg:        cfg,
		log:        log,
		now:        time.Now,
	}
}

func (e *Engine) Enabled() bool {
	return e != nil && e.cfg.Enabled
}

// Evaluate matches open exceptions to playbooks and creates runs.
func (e *Engine) Evaluate(ctx context.Context, supplierID string, exceptionIDs []string) error {
	if !e.Enabled() || e.repo == nil {
		return nil
	}
	supplierID = strings.TrimSpace(supplierID)
	exceptions, err := e.repo.ListOpenExceptions(ctx, supplierID)
	if err != nil {
		return err
	}
	if len(exceptionIDs) > 0 {
		allow := make(map[string]bool, len(exceptionIDs))
		for _, id := range exceptionIDs {
			allow[strings.TrimSpace(id)] = true
		}
		filtered := make([]Exception, 0, len(exceptionIDs))
		for _, ex := range exceptions {
			if allow[ex.ExceptionID] {
				filtered = append(filtered, ex)
			}
		}
		exceptions = filtered
	}
	playbooks, err := e.repo.ListActivePlaybooks(ctx, supplierID)
	if err != nil {
		return err
	}
	now := e.now()
	for _, ex := range exceptions {
		if err := e.enrichException(ctx, &ex); err != nil {
			e.log.Warn("playbook enrich exception failed", "exception_id", ex.ExceptionID, "err", err)
		}
		blocking, err := e.repo.HasBlockingRun(ctx, ex.ExceptionID)
		if err != nil {
			e.log.Warn("playbook blocking run check failed", "exception_id", ex.ExceptionID, "err", err)
			continue
		}
		if blocking {
			continue
		}
		for _, pb := range playbooks {
			if !pb.MatchRules.MatchesException(ex, now) {
				continue
			}
			mode := RunModeSuggested
			if pb.AutoExecute && e.cfg.AutoExecute && allActionsAutoSafe(pb.Actions) {
				mode = RunModeAuto
			}
			run := PlaybookRun{
				RunID:       uuid.NewString(),
				PlaybookID:  pb.PlaybookID,
				ExceptionID: ex.ExceptionID,
				SupplierID:  supplierID,
				Mode:        mode,
				Status:      RunStatusSuggested,
				CreatedAt:   now,
			}
			if err := e.repo.CreateRun(ctx, run); err != nil {
				e.log.Warn("playbook create run failed", "playbook_id", pb.PlaybookID, "err", err)
				break
			}
			if mode == RunModeAuto {
				if err := e.ExecuteRun(ctx, run.RunID, "system:auto"); err != nil {
					e.log.Warn("playbook auto execute failed", "run_id", run.RunID, "err", err)
				}
			}
			break // first match wins
		}
	}
	return nil
}

func (e *Engine) enrichException(ctx context.Context, ex *Exception) error {
	if e.segmentSvc != nil && ex.RetailerID != "" {
		seg, err := e.segmentSvc.GetRetailerSegment(ctx, ex.RetailerID)
		if err == nil {
			ex.RetailerSegment = seg
		}
	}
	return nil
}

func allActionsAutoSafe(actions []ActionSpec) bool {
	for _, a := range actions {
		if !IsAutoSafeAction(a.Type) {
			return false
		}
	}
	return true
}

// ApproveRun marks a suggested run approved and executes it.
func (e *Engine) ApproveRun(ctx context.Context, runID, actor string) error {
	if !e.Enabled() {
		return fmt.Errorf("playbooks_disabled")
	}
	run, err := e.repo.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run.Status != RunStatusSuggested {
		return fmt.Errorf("run_not_suggested")
	}
	run.Status = RunStatusApproved
	if err := e.repo.UpdateRun(ctx, run); err != nil {
		return err
	}
	return e.ExecuteRun(ctx, runID, actor)
}

// SkipRun marks a run skipped.
func (e *Engine) SkipRun(ctx context.Context, runID, actor string) error {
	if !e.Enabled() {
		return fmt.Errorf("playbooks_disabled")
	}
	run, err := e.repo.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	run.Status = RunStatusSkipped
	run.ExecutedBy = actor
	now := e.now()
	run.ExecutedAt = &now
	return e.repo.UpdateRun(ctx, run)
}

// ExecuteRun runs playbook actions for a run.
func (e *Engine) ExecuteRun(ctx context.Context, runID, actor string) error {
	if !e.Enabled() {
		return fmt.Errorf("playbooks_disabled")
	}
	run, err := e.repo.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	pb, err := e.repo.GetPlaybook(ctx, run.PlaybookID)
	if err != nil {
		return err
	}
	exceptions, err := e.repo.ListOpenExceptions(ctx, run.SupplierID)
	if err != nil {
		return err
	}
	var ex Exception
	for _, candidate := range exceptions {
		if candidate.ExceptionID == run.ExceptionID {
			ex = candidate
			break
		}
	}
	if ex.ExceptionID == "" {
		ex = Exception{ExceptionID: run.ExceptionID, SupplierID: run.SupplierID}
	}
	_ = e.enrichException(ctx, &ex)

	results := make([]ActionResult, 0, len(pb.Actions))
	failed := false
	for i, action := range pb.Actions {
		if prior, ok := findPriorResult(run.ActionsResult, i); ok && prior.Status == "ok" {
			results = append(results, prior)
			continue
		}
		res := e.executor.ExecuteAction(ctx, runID, i, action, ex, actor)
		results = append(results, res)
		if res.Status == "failed" {
			failed = true
			break
		}
	}
	now := e.now()
	run.ExecutedAt = &now
	run.ExecutedBy = actor
	run.ActionsResult = results
	if failed {
		run.Status = RunStatusFailed
	} else {
		run.Status = RunStatusExecuted
	}
	if err := e.repo.UpdateRun(ctx, run); err != nil {
		return err
	}
	e.log.Info("playbook run executed", "run_id", runID, "status", run.Status, "actor", actor)
	return nil
}

func findPriorResult(results []ActionResult, idx int) (ActionResult, bool) {
	for _, r := range results {
		if r.Index == idx && r.Status == "ok" {
			return r, true
		}
	}
	return ActionResult{}, false
}

// ListSuggestedRuns returns suggested runs for supplier UI.
func (e *Engine) ListSuggestedRuns(ctx context.Context, supplierID string) ([]PlaybookRun, error) {
	if e.repo == nil {
		return nil, nil
	}
	return e.repo.ListRuns(ctx, supplierID, RunStatusSuggested, 100)
}
