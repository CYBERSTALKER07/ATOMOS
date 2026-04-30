package optimizercontract

// V is the wire-contract version. Bumped only via coordinated migration of
// both the backend client and the ai-worker server.
const V = "v1"

// SolvePath is the canonical HTTP route the ai-worker mounts the solver on.
const SolvePath = "/v1/optimizer/solve"

// AuthHeader is the shared-secret header carrying INTERNAL_API_KEY. The
// optimiser is an internal service — it never sees user JWTs.
const AuthHeader = "X-Internal-Api-Key"

// SolverSource discriminates which algorithm produced a manifest. Persisted on
// OrderManifests.OptimizerSource so analytics can A/B compare outcomes.
type SolverSource string

const (
	// SourceVRP is the Clarke-Wright Savings + 2-opt local-search optimiser
	// (Phase 2). Higher-quality routes; bounded by the 2.5 s timeout.
	SourceVRP SolverSource = "VRP_CLARKE_WRIGHT"

	// SourceFallback is the Phase 1 K-Means clustering + binpack pipeline.
	// Used when the optimiser times out, returns 5xx, or is unreachable.
	SourceFallback SolverSource = "KMEANS_BINPACK"
)

// SolveRequest is the input to POST /v1/optimizer/solve. The backend
// hydrates every field from Spanner inside a single stale read; the optimiser
// performs ZERO database I/O — it is a pure function of this payload.
type SolveRequest struct {
	// V must equal the package constant V. Mismatches return ErrCodeVersion.
	V string `json:"v"`

	// TraceID propagates the originating HTTP request's trace through the
	// optimiser logs and into every emitted Kafka event downstream.
	TraceID string `json:"trace_id"`

	// SupplierID scopes the request. The optimiser never crosses suppliers.
	SupplierID string `json:"supplier_id"`

	// HomeNodeID is the warehouse or factory the routes depart from. All
	// vehicles in Vehicles[] are home-based here.
	HomeNodeID string `json:"home_node_id"`

	// DepartureTime is the wall-clock departure (RFC3339, retailer TZ). Used
	// to evaluate Stops[].WindowOpen / WindowClose feasibility.
	DepartureTime string `json:"departure_time"`

	// Stops is the candidate delivery set. A stop with no feasible vehicle
	// becomes an Orphan in the response, never silently dropped.
	Stops []Stop `json:"stops"`

	// Vehicles is the available fleet. Empty fleet → ErrCodeEmptyFleet.
	Vehicles []Vehicle `json:"vehicles"`

	// Tunables override per-call solver parameters. Nil = use defaults.
	Tunables *Tunables `json:"tunables,omitempty"`
}

// Stop is one candidate delivery. Volume is in volumetric units (VU);
// receiving windows are HH:MM 24-hour strings or empty for "any time".
type Stop struct {
	OrderID    string  `json:"order_id"`
	RetailerID string  `json:"retailer_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	H3Cell     string  `json:"h3_cell"`
	VolumeVU   float64 `json:"volume_vu"`

	// WindowOpen / WindowClose are HH:MM strings. Empty = no constraint.
	// The optimiser treats both-set as a hard SLA constraint; missing one
	// degrades to "no constraint" (matches the Spanner-NULL semantics).
	WindowOpen  string `json:"window_open,omitempty"`
	WindowClose string `json:"window_close,omitempty"`

	// ServiceMinutes is the dwell time at the stop (unloading + handover).
	// Default 5 min if zero.
	ServiceMinutes int `json:"service_minutes,omitempty"`

	// Priority is a savings-rank boost for the Clarke-Wright solver.
	// Recovery orders (overflow-bounced back to pool) carry +10 000 so they
	// get first dibs on vehicle volume in the next dispatch cycle.
	Priority int `json:"priority,omitempty"`
}

// Vehicle is one available truck. MaxVolumeVU is the raw spec; the optimiser
// applies the global TetrisBuffer (0.95) before fit checks.
type Vehicle struct {
	VehicleID    string  `json:"vehicle_id"`
	DriverID     string  `json:"driver_id"`
	MaxVolumeVU  float64 `json:"max_volume_vu"`
	StartLat     float64 `json:"start_lat"`
	StartLng     float64 `json:"start_lng"`
	AvgSpeedKmph float64 `json:"avg_speed_kmph"`
}

// Tunables are per-call overrides for the Phase 2 solver. All fields are
// optional; zero values mean "use compiled-in defaults".
type Tunables struct {
	// TetrisBuffer is the volumetric safety margin (default 0.95).
	TetrisBuffer float64 `json:"tetris_buffer,omitempty"`

	// TwoOptIterations caps local-search rounds per route (default 200).
	TwoOptIterations int `json:"two_opt_iterations,omitempty"`

	// MaxStopsPerRoute caps stops per manifest (default 25, matches Google
	// Maps waypoint ceiling).
	MaxStopsPerRoute int `json:"max_stops_per_route,omitempty"`
}

// SolveResponse is the optimiser's reply on success (HTTP 200).
type SolveResponse struct {
	V       string       `json:"v"`
	TraceID string       `json:"trace_id"`
	Source  SolverSource `json:"source"`

	// Routes are the produced manifests. Each route's stops are listed in
	// execution order; the backend persists them as OrderManifestStops with
	// SequenceIndex matching slice index.
	Routes []Route `json:"routes"`

	// Orphans are stops that could not be placed (capacity, window, or no
	// reachable vehicle). The backend re-queues these for the next tick.
	Orphans []Orphan `json:"orphans"`

	// Stats is solver telemetry. Persisted on the manifest row for analytics.
	Stats Stats `json:"stats"`
}

// Route is one optimised manifest in the response.
type Route struct {
	VehicleID   string  `json:"vehicle_id"`
	DriverID    string  `json:"driver_id"`
	Stops       []Stop  `json:"stops"`
	TotalVU     float64 `json:"total_vu"`
	UtilPct     float64 `json:"util_pct"`
	DistanceKm  float64 `json:"distance_km"`
	DurationMin int     `json:"duration_min"`
}

// Orphan is one stop the optimiser could not place.
type Orphan struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// Stats is solver-level telemetry returned per request.
type Stats struct {
	ElapsedMs            int     `json:"elapsed_ms"`
	StopsConsidered      int     `json:"stops_considered"`
	StopsPlaced          int     `json:"stops_placed"`
	StopsOrphaned        int     `json:"stops_orphaned"`
	VehiclesUsed         int     `json:"vehicles_used"`
	AvgUtilisationPct    float64 `json:"avg_utilisation_pct"`
	TwoOptImprovementPct float64 `json:"two_opt_improvement_pct"`
}
