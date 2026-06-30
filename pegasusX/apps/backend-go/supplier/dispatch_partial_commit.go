package supplier

import "fmt"

// dispatchPartialCommitError is returned when a multi-chunk dispatch execute commits
// some routes before a later chunk fails. Idempotency keys are released on failure so
// operators may retry warehouse-scoped execute after reconciling partial manifests.
type dispatchPartialCommitError struct {
	CommittedRoutes []DispatchExecuteRoute
	FailedChunk     int
	TotalChunks     int
	TotalRoutes     int
	Cause           error
}

func (e *dispatchPartialCommitError) Error() string {
	return fmt.Sprintf(
		"dispatch_partial_commit: committed_routes=%d failed_chunk=%d/%d total_routes=%d: %v",
		len(e.CommittedRoutes), e.FailedChunk, e.TotalChunks, e.TotalRoutes, e.Cause,
	)
}

func (e *dispatchPartialCommitError) Unwrap() error { return e.Cause }
