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

// ApproveRun executes a suggested run. Intermediate APPROVED is not persisted
// alone (avoids double UpdateRun + orphan APPROVED without action side-effects);
// ExecuteRun writes the single terminal status (EXECUTED|FAILED) + outbox.
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

// ExecuteRun runs playbook actions then finalizes the run in one UpdateRun.
// Local Spanner side-effects (exception ACK/ASSIGN) are deferred into the same
// RW txn as the terminal run status when the repository supports it.
// FREEZE_CREDIT is compensated if finalize fails after a successful freeze.
func (e *Engine) ExecuteRun(ctx context.Context, runID, actor string) error {
	if !e.Enabled() {
		return fmt.Errorf("playbooks_disabled")
	}
	if e.executor == nil {
		return fmt.Errorf("executor_unavailable")
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
	var deferred []localExceptionSideEffect
	frozeCredit := false
	var freezeRetailer, freezeSupplier string

	for i, action := range pb.Actions {
		if prior, ok := findPriorResult(run.ActionsResult, i); ok && prior.Status == "ok" {
			results = append(results, prior)
			continue
		}
		atype := strings.ToUpper(strings.TrimSpace(action.Type))
		// Defer pure Spanner exception mutations into finalize txn.
		if atype == "ACKNOWLEDGE_EXCEPTION" || atype == "ASSIGN_EXCEPTION" {
			if se, ok := planLocalExceptionSideEffect(i, action, ex); ok {
				deferred = append(deferred, se)
				results = append(results, ActionResult{Index: i, Type: action.Type, Status: "ok"})
				continue
			}
		}
		res := e.executor.ExecuteAction(ctx, runID, i, action, ex, actor)
		results = append(results, res)
		if res.Status == "failed" {
			failed = true
			break
		}
		if atype == "FREEZE_CREDIT" {
			frozeCredit = true
			freezeRetailer = ex.RetailerID
			freezeSupplier = ex.SupplierID
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

	// Prefer single RW: run row + deferred exception mutations + outbox.
	var finErr error
	if fin, ok := e.repo.(runFinalizer); ok && len(deferred) > 0 {
		finErr = fin.FinalizeRunWithExceptionEffects(ctx, run, deferred)
	} else {
		for _, se := range deferred {
			if se.Status != "" {
				if err := e.repo.UpdateExceptionStatus(ctx, se.ExceptionID, se.Status); err != nil {
					finErr = err
					break
				}
			}
			if se.AssigneeRole != "" {
				if err := e.repo.UpdateExceptionAssignee(ctx, se.ExceptionID, se.AssigneeRole); err != nil {
					finErr = err
					break
				}
			}
		}
		if finErr == nil {
			finErr = e.repo.UpdateRun(ctx, run)
		}
	}
	if finErr != nil {
		// Compensate credit freeze if we already mutated profile but could not finalize run.
		if frozeCredit && e.executor != nil {
			_ = e.executor.unfreezeCredit(ctx, freezeRetailer, freezeSupplier, actor)
		}
		return finErr
	}
	e.log.Info("playbook run executed", "run_id", runID, "status", run.Status, "actor", actor)
	return nil
}

// localExceptionSideEffect is applied in the same Spanner txn as run finalize.
type localExceptionSideEffect struct {
	Index        int
	ExceptionID  string
	Status       string // optional Status update
	AssigneeRole string // optional assignee
}

// runFinalizer is implemented by SpannerRepository for mega-txn finalize.
type runFinalizer interface {
	FinalizeRunWithExceptionEffects(ctx context.Context, run PlaybookRun, effects []localExceptionSideEffect) error
}

func planLocalExceptionSideEffect(idx int, action ActionSpec, ex Exception) (localExceptionSideEffect, bool) {
	atype := strings.ToUpper(strings.TrimSpace(action.Type))
	se := localExceptionSideEffect{Index: idx, ExceptionID: ex.ExceptionID}
	switch atype {
	case "ACKNOWLEDGE_EXCEPTION":
		se.Status = "ACKNOWLEDGED"
		return se, se.ExceptionID != ""
	case "ASSIGN_EXCEPTION":
		params := decodeStringMap(action.Params)
		role := strings.TrimSpace(params["role"])
		if role == "" {
			role = "SUPPLIER_OPS"
		}
		se.AssigneeRole = role
		return se, se.ExceptionID != ""
	default:
		return localExceptionSideEffect{}, false
	}
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
