package routing

import (
	"cloud.google.com/go/spanner"
)

// Service provides advanced routing operations like ReplanRoute.
type Service struct {
	spannerClient *spanner.Client
	solver        Solver
}

// NewService creates a new Routing Service.
func NewService(spannerClient *spanner.Client) *Service {
	return &Service{
		spannerClient: spannerClient,
		solver:        &HeuristicSolver{},
	}
}

// Solver defines the interface for solving routing problems.
type Solver interface {
	Solve(problem ReplanProblem) ([]string, error)
}

// HeuristicSolver provides a fast low-latency implementation for replanning.
type HeuristicSolver struct{}

func (s *HeuristicSolver) Solve(problem ReplanProblem) ([]string, error) {
	// Fast heuristic: preserve current sequence for Phase 1.
	// In reality, this would apply nearest-neighbor + 2-opt.
	var seq []string
	for _, s := range problem.RemainingStops {
		seq = append(seq, s.OrderID)
	}
	return seq, nil
}
