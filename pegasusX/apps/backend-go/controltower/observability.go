package controltower

import (
	"net/http"
)

// CellHealth matches the Swift Codable model in Control Tower macOS.
type CellHealth struct {
	CellID        string  `json:"cellId"`
	Region        string  `json:"region"`
	Role          string  `json:"role"`
	ActiveNodes   int     `json:"activeNodes"`
	QPS           int     `json:"qps"`
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	Status        string  `json:"status"`
	P99LatencyMs  float64 `json:"p99LatencyMs"`
}

// SpannerClusterMetrics matches the Swift Codable model in Control Tower macOS.
type SpannerClusterMetrics struct {
	P99ReadMs               float64            `json:"p99ReadMs"`
	P99WriteMs              float64            `json:"p99WriteMs"`
	P99CommitMs             float64            `json:"p99CommitMs"`
	ActiveSessions          int                `json:"activeSessions"`
	MaxSessions             int                `json:"maxSessions"`
	StaleReadOffloadPercent float64            `json:"staleReadOffloadPercent"`
	HourlyHistory           []LatencyDataPoint `json:"hourlyHistory"`
}

// LatencyDataPoint is a point-in-time latency metric.
type LatencyDataPoint struct {
	Hour     string  `json:"hour"`
	ReadMs   float64 `json:"readMs"`
	WriteMs  float64 `json:"writeMs"`
	CommitMs float64 `json:"commitMs"`
}

// KafkaPartitionLag matches the Swift Codable model in Control Tower macOS.
type KafkaPartitionLag struct {
	Topic         string `json:"topic"`
	Partitions    int    `json:"partitions"`
	MaxLagMs      int    `json:"maxLagMs"`
	DLQCount      int    `json:"dlqCount"`
	ConsumerGroup string `json:"consumerGroup"`
	Status        string `json:"status"`
}

// RedisPermitPool matches the Swift Codable model in Control Tower macOS.
type RedisPermitPool struct {
	PoolName      string  `json:"poolName"`
	ActivePermits int     `json:"activePermits"`
	MaxPermits    int     `json:"maxPermits"`
	QueueWaitMs   float64 `json:"queueWaitMs"`
}

// TenderWaterfallSlice matches the Swift Codable model in Control Tower macOS.
type TenderWaterfallSlice struct {
	Label       string  `json:"label"`
	AmountTiyin int64   `json:"amountTiyin"`
	Percentage  float64 `json:"percentage"`
	IsDeduction bool    `json:"isDeduction"`
	Detail      string  `json:"detail"`
}

// AttributedCostItem matches the Swift Codable model in Control Tower macOS.
type AttributedCostItem struct {
	Role             string  `json:"role"`
	EntityID         string  `json:"entityId"`
	ResourceName     string  `json:"resourceName"`
	MonthlyCostCents int64   `json:"monthlyCostCents"`
	UsageMetric      string  `json:"usageMetric"`
	TrendPercent     float64 `json:"trendPercent"`
}

// RolePerformanceScorecard matches the Swift Codable model in Control Tower macOS.
type RolePerformanceScorecard struct {
	Role                  string  `json:"role"`
	PrimaryMetricName     string  `json:"primaryMetricName"`
	PrimaryMetricValue    string  `json:"primaryMetricValue"`
	PrimaryMetricSubtext  string  `json:"primaryMetricSubtext"`
	SecondaryMetricName   string  `json:"secondaryMetricName"`
	SecondaryMetricValue  string  `json:"secondaryMetricValue"`
	ComplianceRatePercent float64 `json:"complianceRatePercent"`
	ActiveAlertCount      int     `json:"activeAlertCount"`
}

// HandleCells handles GET /v1/control-tower/cells
func (h *Handlers) HandleCells(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	cells := []CellHealth{
		{
			CellID:        "cell-uz-tashkent",
			Region:        "Toshkent Sh.",
			Role:          "Primary Core",
			ActiveNodes:   16,
			QPS:           12450,
			CPUPercent:    42.5,
			MemoryPercent: 64.2,
			Status:        "HEALTHY",
			P99LatencyMs:  4.2,
		},
		{
			CellID:        "cell-uz-samarkand",
			Region:        "Samarqand",
			Role:          "Secondary Core",
			ActiveNodes:   8,
			QPS:           4120,
			CPUPercent:    28.0,
			MemoryPercent: 51.6,
			Status:        "HEALTHY",
			P99LatencyMs:  5.8,
		},
		{
			CellID:        "cell-uz-fergana",
			Region:        "Fargʻona",
			Role:          "Edge Ingest",
			ActiveNodes:   6,
			QPS:           2890,
			CPUPercent:    31.4,
			MemoryPercent: 48.0,
			Status:        "HEALTHY",
			P99LatencyMs:  6.4,
		},
	}
	writeJSON(w, http.StatusOK, cells)
}

// HandleSpannerMetrics handles GET /v1/control-tower/metrics/spanner
func (h *Handlers) HandleSpannerMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	metrics := SpannerClusterMetrics{
		P99ReadMs:               4.2,
		P99WriteMs:              18.4,
		P99CommitMs:             24.1,
		ActiveSessions:          1840,
		MaxSessions:             2000,
		StaleReadOffloadPercent: 78.4,
		HourlyHistory: []LatencyDataPoint{
			{Hour: "10:00", ReadMs: 3.8, WriteMs: 17.2, CommitMs: 22.1},
			{Hour: "11:00", ReadMs: 4.1, WriteMs: 18.0, CommitMs: 23.4},
			{Hour: "12:00", ReadMs: 4.5, WriteMs: 19.5, CommitMs: 25.0},
			{Hour: "13:00", ReadMs: 4.8, WriteMs: 20.1, CommitMs: 26.2},
			{Hour: "14:00", ReadMs: 4.3, WriteMs: 18.9, CommitMs: 24.8},
			{Hour: "15:00", ReadMs: 4.2, WriteMs: 18.4, CommitMs: 24.1},
		},
	}
	writeJSON(w, http.StatusOK, metrics)
}

// HandleKafkaMetrics handles GET /v1/control-tower/metrics/kafka
func (h *Handlers) HandleKafkaMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	lags := []KafkaPartitionLag{
		{Topic: "orders.created", Partitions: 12, MaxLagMs: 142, DLQCount: 0, ConsumerGroup: "order-lifecycle-svc", Status: "HEALTHY"},
		{Topic: "inventory.crdt.sync", Partitions: 12, MaxLagMs: 280, DLQCount: 0, ConsumerGroup: "depot-balance-svc", Status: "HEALTHY"},
		{Topic: "compliance.fiscal.soliq", Partitions: 8, MaxLagMs: 85, DLQCount: 3, ConsumerGroup: "fiscal-gateway-worker", Status: "DEGRADED"},
		{Topic: "telemetry.coldchain.dvir", Partitions: 16, MaxLagMs: 45, DLQCount: 0, ConsumerGroup: "fleet-telematics-hub", Status: "HEALTHY"},
	}
	writeJSON(w, http.StatusOK, lags)
}

// HandleRedisMetrics handles GET /v1/control-tower/metrics/redis
func (h *Handlers) HandleRedisMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	pools := []RedisPermitPool{
		{PoolName: "supplier-catalog-sync", ActivePermits: 32, MaxPermits: 100, QueueWaitMs: 1.2},
		{PoolName: "warehouse-wave-execution", ActivePermits: 18, MaxPermits: 50, QueueWaitMs: 0.8},
		{PoolName: "driver-location-ingest", ActivePermits: 145, MaxPermits: 200, QueueWaitMs: 3.4},
		{PoolName: "soliq-fiscal-dispatch", ActivePermits: 12, MaxPermits: 25, QueueWaitMs: 0.4},
	}
	writeJSON(w, http.StatusOK, pools)
}

// HandleWaterfall handles GET /v1/control-tower/waterfall
func (h *Handlers) HandleWaterfall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	role := r.URL.Query().Get("role")
	var slices []TenderWaterfallSlice
	switch role {
	case "Warehouse Perspective":
		slices = []TenderWaterfallSlice{
			{Label: "Allocated Depot GMV", AmountTiyin: 45_000_000_00, Percentage: 100.0, IsDeduction: false, Detail: "Goods received and cross-docked"},
			{Label: "Wave Handling Surcharge", AmountTiyin: 1_620_000_00, Percentage: 3.50, IsDeduction: false, Detail: "Earned depot operations revenue"},
			{Label: "Cold-Chain Energy Levy", AmountTiyin: 450_000_00, Percentage: 1.00, IsDeduction: true, Detail: "Sub-zero reefer chamber utility cost"},
			{Label: "Net Depot Credit", AmountTiyin: 1_170_000_00, Percentage: 2.60, IsDeduction: false, Detail: "Net treasury credit for depot operations"},
		}
	case "Retailer Perspective":
		slices = []TenderWaterfallSlice{
			{Label: "Retail Invoiced Value", AmountTiyin: 32_500_000_00, Percentage: 100.0, IsDeduction: false, Detail: "Doorstep merchandise billing total"},
			{Label: "Promotional Allowance", AmountTiyin: 1_625_000_00, Percentage: 5.00, IsDeduction: true, Detail: "Supplier co-funded promo discount"},
			{Label: "Cash-on-Delivery Fee", AmountTiyin: 325_000_00, Percentage: 1.00, IsDeduction: true, Detail: "Cash vault security & blind turn-in fee"},
			{Label: "Net Payable at Doorstep", AmountTiyin: 30_550_000_00, Percentage: 94.00, IsDeduction: false, Detail: "Total collection due upon physical delivery"},
		}
	case "Fleet & Carrier":
		slices = []TenderWaterfallSlice{
			{Label: "Freight Order Billable", AmountTiyin: 18_400_000_00, Percentage: 100.0, IsDeduction: false, Detail: "Base line-haul and route compensation"},
			{Label: "DVIR Fuel Incentive", AmountTiyin: 552_000_00, Percentage: 3.00, IsDeduction: false, Detail: "Clean pre-trip inspection bonus"},
			{Label: "Rescue Penalty Escrow", AmountTiyin: 0, Percentage: 0.00, IsDeduction: true, Detail: "Zero mid-route breakdown deductions"},
			{Label: "Net Carrier Disbursable", AmountTiyin: 18_952_000_00, Percentage: 103.00, IsDeduction: false, Detail: "Total dispatches payable to fleet partner"},
		}
	default: // "Overall Ecosystem", "Supplier Perspective", etc.
		slices = []TenderWaterfallSlice{
			{Label: "Gross GMV", AmountTiyin: 100_000_000_00, Percentage: 100.0, IsDeduction: false, Detail: "Total order value before splits"},
			{Label: "Platform Fee", AmountTiyin: 2_500_000_00, Percentage: 2.50, IsDeduction: true, Detail: "Configured tenant platform take-rate"},
			{Label: "Logistics Fulfillment", AmountTiyin: 3_500_000_00, Percentage: 3.50, IsDeduction: true, Detail: "Depot handling and last-mile carrier tariff"},
			{Label: "Soliq Fiscal VAT", AmountTiyin: 12_000_000_00, Percentage: 12.00, IsDeduction: true, Detail: "Statutory VAT computed per MXIK code"},
			{Label: "Gateway Settlement", AmountTiyin: 500_000_00, Percentage: 0.50, IsDeduction: true, Detail: "Payment processing gateway clearing"},
			{Label: "Net Supplier Payout", AmountTiyin: 81_500_000_00, Percentage: 81.50, IsDeduction: false, Detail: "Disbursable net payout to supplier bank account"},
		}
	}
	writeJSON(w, http.StatusOK, slices)
}

// HandleAttributedCosts handles GET /v1/control-tower/costs/attributed
func (h *Handlers) HandleAttributedCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	costs := []AttributedCostItem{
		{Role: "Supplier", EntityID: "SPL-SAM-901", ResourceName: "Cloud Spanner Mutations", MonthlyCostCents: 1420_50, UsageMetric: "18.4M mutations", TrendPercent: 4.2},
		{Role: "Supplier", EntityID: "SPL-SAM-901", ResourceName: "GTIN Catalog Matching", MonthlyCostCents: 380_20, UsageMetric: "24.5k inferences", TrendPercent: -1.5},
		{Role: "Warehouse", EntityID: "DEPOT-TAS-CENTRAL", ResourceName: "Wave Planning Compute", MonthlyCostCents: 910_80, UsageMetric: "480 batch runs", TrendPercent: 2.1},
		{Role: "Warehouse", EntityID: "DEPOT-TAS-CENTRAL", ResourceName: "Cold-Chain Telemetry Ingest", MonthlyCostCents: 290_40, UsageMetric: "12.8M MQTT msgs", TrendPercent: 0.0},
		{Role: "Fleet", EntityID: "FLT-EAST-CARRIER", ResourceName: "Routing Graph Solver (VRP)", MonthlyCostCents: 1840_00, UsageMetric: "14.2k routes solved", TrendPercent: 5.6},
		{Role: "Retailer", EntityID: "TIN-309182746", ResourceName: "Catalog Dynamic Search", MonthlyCostCents: 420_10, UsageMetric: "1.2M queries", TrendPercent: 8.4},
	}
	writeJSON(w, http.StatusOK, costs)
}

// HandleScorecards handles GET /v1/control-tower/scorecards
func (h *Handlers) HandleScorecards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	scorecards := []RolePerformanceScorecard{
		{
			Role:                  "Supplier Network",
			PrimaryMetricName:     "OTIF Compliance",
			PrimaryMetricValue:    "98.4%",
			PrimaryMetricSubtext:  "Target: ≥ 98.0%",
			SecondaryMetricName:   "Soliq Factura Rate",
			SecondaryMetricValue:  "99.8%",
			ComplianceRatePercent: 98.4,
			ActiveAlertCount:      0,
		},
		{
			Role:                  "Warehouse Operations",
			PrimaryMetricName:     "Avg Wave Cycle",
			PrimaryMetricValue:    "42.5 min",
			PrimaryMetricSubtext:  "Target: ≤ 45 min",
			SecondaryMetricName:   "Vault Variance",
			SecondaryMetricValue:  "0.04%",
			ComplianceRatePercent: 99.2,
			ActiveAlertCount:      0,
		},
		{
			Role:                  "Fleet & Logistics",
			PrimaryMetricName:     "DVIR Walkaround Pass",
			PrimaryMetricValue:    "99.1%",
			PrimaryMetricSubtext:  "Target: ≥ 99.0%",
			SecondaryMetricName:   "ALPR Gate Turnaround",
			SecondaryMetricValue:  "8.2 min",
			ComplianceRatePercent: 99.1,
			ActiveAlertCount:      0,
		},
		{
			Role:                  "Retailer Network",
			PrimaryMetricName:     "Doorstep Acceptance",
			PrimaryMetricValue:    "98.9%",
			PrimaryMetricSubtext:  "Target: ≥ 97.5%",
			SecondaryMetricName:   "Card Payment Ratio",
			SecondaryMetricValue:  "68.4%",
			ComplianceRatePercent: 98.9,
			ActiveAlertCount:      0,
		},
	}
	writeJSON(w, http.StatusOK, scorecards)
}
