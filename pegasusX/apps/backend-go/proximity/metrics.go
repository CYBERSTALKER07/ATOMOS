package proximity

import (
	"fmt"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SpatialCompactionRatio measures the compaction ratio (original_cells / compacted_cells)
	SpatialCompactionRatio = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "void_spatial_compaction_ratio",
			Help:    "Ratio of uncompacted H3 cells to compacted H3 cells",
			Buckets: []float64{1, 2, 5, 10, 50, 100, 500},
		},
		[]string{"zone_type", "resolution"},
	)

	// SpatialEgressOverflowTotal counts the number of times coverage was truncated
	// to fit within MaxCompactedEgressCells
	SpatialEgressOverflowTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "void_spatial_egress_overflow_total",
			Help: "Total count of spatial payloads truncated due to egress circuit breaker",
		},
		[]string{"zone_type", "resolution"},
	)
)

// RecordCompactionRatio records the compaction ratio.
func RecordCompactionRatio(zoneType string, resolution int, originalCount, compactedCount int) {
	if compactedCount > 0 {
		ratio := float64(originalCount) / float64(compactedCount)
		SpatialCompactionRatio.WithLabelValues(zoneType, fmt.Sprint(resolution)).Observe(ratio)
	}
}

// RecordEgressOverflow records an egress overflow event.
func RecordEgressOverflow(zoneType string, resolution int) {
	SpatialEgressOverflowTotal.WithLabelValues(zoneType, fmt.Sprint(resolution)).Inc()
}
