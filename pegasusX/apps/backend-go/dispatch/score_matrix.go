package dispatch

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
)

// AttachRoadMatrix builds a RoadKm callback from an OSRM distance matrix over
// the given points (depot + order coords). When OSRM is nil/fails, leaves ctx unchanged.
// Pure scoring remains free of network I/O after this attach (G6.D).
func AttachRoadMatrix(ctx context.Context, score ScoreContext, osrm *routing.OSRMClient, points []routing.LatLng) ScoreContext {
	if osrm == nil || len(points) < 2 || !DispatchScoreUseOSRM() {
		if score.MatrixSource == "" {
			score.MatrixSource = MatrixSourceHaversine
		}
		return score
	}
	matrix, err := osrm.DistanceMatrix(ctx, points)
	if err != nil || len(matrix) == 0 {
		score.MatrixSource = MatrixSourceHaversine
		return score
	}
	// Index points for nearest-match lookup (exact lat/lng key).
	type key struct{ lat, lng float64 }
	idx := make(map[key]int, len(points))
	for i, p := range points {
		idx[key{p.Lat, p.Lng}] = i
	}
	score.RoadKm = func(fromLat, fromLng, toLat, toLng float64) (float64, bool) {
		i, okI := idx[key{fromLat, fromLng}]
		j, okJ := idx[key{toLat, toLng}]
		if !okI || !okJ || i >= len(matrix) || j >= len(matrix[i]) {
			return 0, false
		}
		m := matrix[i][j]
		if m < 0 {
			return 0, false
		}
		return float64(m) / 1000.0, true // meters → km
	}
	score.MatrixSource = MatrixSourceOSRM
	return score
}
