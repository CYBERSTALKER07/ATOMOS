package routing

import (
	"os"
	"strconv"
	"strings"

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
		solver:        &DispatchLocalSearchSolver{},
	}
}

// Solver defines the interface for solving routing problems.
type Solver interface {
	Solve(problem ReplanProblem) ([]string, error)
}

// DispatchLocalSearchSolver resequences remaining stops with nearest-neighbor + 2-opt
// (algorithm-aligned with dispatch.ResequenceStops; kept local to avoid import cycles).
type DispatchLocalSearchSolver struct {
	DepotLat float64
	DepotLng float64
}

func (s *DispatchLocalSearchSolver) Solve(problem ReplanProblem) ([]string, error) {
	stops := make([]geoStop, 0, len(problem.RemainingStops))
	for _, sc := range problem.RemainingStops {
		stops = append(stops, geoStop{
			OrderID: sc.OrderID,
			Lat:     sc.Lat,
			Lng:     sc.Lng,
		})
	}
	lat, lng := s.DepotLat, s.DepotLng
	if lat == 0 && lng == 0 && len(stops) > 0 {
		lat, lng = stops[0].Lat, stops[0].Lng
	}
	if problem.DepotLat != 0 || problem.DepotLng != 0 {
		lat, lng = problem.DepotLat, problem.DepotLng
	}
	ordered := resequenceStops(stops, lat, lng)
	seq := make([]string, 0, len(ordered))
	for _, o := range ordered {
		seq = append(seq, o.OrderID)
	}
	return seq, nil
}

// HeuristicSolver is retained for tests; delegates to DispatchLocalSearchSolver.
type HeuristicSolver struct{}

func (s *HeuristicSolver) Solve(problem ReplanProblem) ([]string, error) {
	return (&DispatchLocalSearchSolver{}).Solve(problem)
}

func replanCooldownSeconds() int64 {
	raw := strings.TrimSpace(os.Getenv("DISPATCH_REPLAN_COOLDOWN_SEC"))
	if raw == "" {
		return 60
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 60
	}
	return n
}

func maxReplansPerDay() int64 {
	raw := strings.TrimSpace(os.Getenv("DISPATCH_MAX_REPLANS_PER_DAY"))
	if raw == "" {
		return 48
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		return 48
	}
	return n
}
