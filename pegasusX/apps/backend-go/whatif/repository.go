package whatif

import (
	"context"
	"time"
)

// Repository abstracts live dataset reads and WhatIfSimulations persistence.
type Repository interface {
	LoadSimulationDataset(ctx context.Context, tenantID, scenarioID, entityID string) (SimulationDataset, error)
	SaveSimulation(
		ctx context.Context,
		sim WhatIfSimulation,
		auditRec AuditRecorder,
		auditIn AuditRecordInput,
	) (WhatIfSimulation, bool, error)
	ListSimulations(ctx context.Context, f ListFilter) ([]WhatIfSimulation, error)
	GetSimulation(ctx context.Context, tenantID, simID string) (WhatIfSimulation, bool, error)
	CommitSimulationProposals(
		ctx context.Context,
		tenantID, simID string,
		mutate func(cur WhatIfSimulation) (WhatIfSimulation, error),
		auditRec AuditRecorder,
		auditIn func(updated WhatIfSimulation) AuditRecordInput,
		committedAt time.Time,
	) (WhatIfSimulation, error)
}
