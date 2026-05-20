package proximity

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// SpatialCompactionRatio tracks compacted/raw H3 payload ratios.
// Exported as void_spatial_compaction_ratio{zone_type,resolution}.
var SpatialCompactionRatio = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "void",
		Subsystem: "spatial",
		Name:      "compaction_ratio",
		Help:      "Ratio of compacted H3 cell count to raw H3 cell count for egress payloads.",
		Buckets:   []float64{0.05, 0.1, 0.2, 0.35, 0.5, 0.65, 0.8, 1.0},
	},
	[]string{"zone_type", "resolution"},
)

// SpatialEgressOverflowTotal counts compacted payload overflows that trigger
// egress truncation before response emission.
// Exported as void_spatial_egress_overflow_total{zone_type,resolution}.
var SpatialEgressOverflowTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "void",
		Subsystem: "spatial",
		Name:      "egress_overflow_total",
		Help:      "Total compacted spatial payload overflows requiring egress truncation.",
	},
	[]string{"zone_type", "resolution"},
)
