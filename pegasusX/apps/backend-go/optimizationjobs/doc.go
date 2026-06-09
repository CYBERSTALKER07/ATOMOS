// Package optimizationjobs provides the Spanner-backed durable queue for
// async optimizer dispatch. Jobs are inserted as part of the auto-dispatch
// or planner workflows, then picked up by the optimization workers.
package optimizationjobs
