package plan

import "sync/atomic"

// SourceCounters tracks per-source attribution for OptimizeAndValidate calls.
// All fields are lock-free atomics — safe to read from analytics handlers
// while the dispatch shadow / primary path increments concurrently.
type SourceCounters struct {
	Optimizer          atomic.Int64 // solver returned a valid plan
	FallbackPhase1     atomic.Int64 // solver unreachable / errored
	FallbackValidation atomic.Int64 // solver returned an over-capacity plan
	Errors             atomic.Int64 // fallback itself failed (P0 alert signal)
}

// Record increments the counter that matches the given source string.
// Errors counter is incremented separately by the caller when err != nil.
// Safe to call with a nil receiver — analytics-disabled call sites become
// zero-cost.
func (c *SourceCounters) Record(source string) {
	if c == nil {
		return
	}
	switch source {
	case SourceOptimizer:
		c.Optimizer.Add(1)
	case SourceFallbackPhase1:
		c.FallbackPhase1.Add(1)
	case SourceFallbackValidation:
		c.FallbackValidation.Add(1)
	}
}

// RecordError increments the Errors counter. nil-safe.
func (c *SourceCounters) RecordError() {
	if c == nil {
		return
	}
	c.Errors.Add(1)
}
