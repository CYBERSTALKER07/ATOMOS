package controltower

import (
	"context"
)

// Repository persists playbooks and runs.
type Repository interface {
	ListActivePlaybooks(ctx context.Context, supplierID string) ([]Playbook, error)
	GetPlaybook(ctx context.Context, playbookID string) (Playbook, error)
	CreatePlaybook(ctx context.Context, pb Playbook) error
	UpdatePlaybook(ctx context.Context, playbookID string, fields map[string]any) error
	DeactivatePlaybook(ctx context.Context, playbookID string) error

	ListOpenExceptions(ctx context.Context, supplierID string) ([]Exception, error)
	ListSupplierIDsWithOpenExceptions(ctx context.Context) ([]string, error)

	HasBlockingRun(ctx context.Context, exceptionID string) (bool, error)
	CreateRun(ctx context.Context, run PlaybookRun) error
	GetRun(ctx context.Context, runID string) (PlaybookRun, error)
	UpdateRun(ctx context.Context, run PlaybookRun) error
	ListRuns(ctx context.Context, supplierID, status string, limit int) ([]PlaybookRun, error)
	ListRunsForException(ctx context.Context, exceptionID string) ([]PlaybookRun, error)

	UpdateExceptionStatus(ctx context.Context, exceptionID, status string) error
	UpdateExceptionAssignee(ctx context.Context, exceptionID, role string) error
	RecordIntervention(ctx context.Context, manifestID, supplierID, operatorID, commandType, reasonCode, notes string) error
}
