package optimizercontract

// OptimizationJobType identifies the backend domain workflow that queued the
// solver run. Values stay aligned with backend-go/optimizationjobs job types.
type OptimizationJobType string

const (
	OptimizationJobTypeAutoDispatch  OptimizationJobType = "AUTO_DISPATCH"
	OptimizationJobTypeDispatchQueue OptimizationJobType = "DISPATCH_QUEUE"
)

// OptimizationJobStatus tracks the durable state of a queued optimization job.
// Values are additive with the backend OptimizationJobs ledger.
type OptimizationJobStatus string

const (
	OptimizationJobStatusQueued    OptimizationJobStatus = "QUEUED"
	OptimizationJobStatusPublished OptimizationJobStatus = "PUBLISHED"
	OptimizationJobStatusRunning   OptimizationJobStatus = "RUNNING"
	OptimizationJobStatusSolved    OptimizationJobStatus = "SOLVED"
	OptimizationJobStatusApplying  OptimizationJobStatus = "APPLYING"
	OptimizationJobStatusApplied   OptimizationJobStatus = "APPLIED"
	OptimizationJobStatusFailed    OptimizationJobStatus = "FAILED"
	OptimizationJobStatusCancelled OptimizationJobStatus = "CANCELLED"
)

// OptimizationSolverType identifies which solver family should process the
// envelope.
type OptimizationSolverType string

const (
	OptimizationSolverTypeVRP   OptimizationSolverType = "VRP"
	OptimizationSolverTypeCPSAT OptimizationSolverType = "CP_SAT"
)

// OptimizationSolverStatus describes the solver's result quality for a
// completed optimization run.
type OptimizationSolverStatus string

const (
	OptimizationSolverStatusOptimal      OptimizationSolverStatus = "OPTIMAL"
	OptimizationSolverStatusFeasible     OptimizationSolverStatus = "FEASIBLE"
	OptimizationSolverStatusInfeasible   OptimizationSolverStatus = "INFEASIBLE"
	OptimizationSolverStatusModelInvalid OptimizationSolverStatus = "MODEL_INVALID"
)

// OptimizationJobEnvelope is the canonical Kafka payload for
// pegasus-optimizer-jobs. It carries the durable job identity, routing scope,
// matrix sizing metadata, and a backend-owned dispatch snapshot so the worker
// never has to re-read backend state to execute the solve.
type OptimizationJobEnvelope struct {
	V                 string                 `json:"v"`
	JobID             string                 `json:"job_id"`
	SupplierID        string                 `json:"supplier_id"`
	JobType           OptimizationJobType    `json:"job_type"`
	SolverType        OptimizationSolverType `json:"solver_type"`
	TraceID           string                 `json:"trace_id,omitempty"`
	IdempotencyKey    string                 `json:"idempotency_key,omitempty"`
	SourceEventType   string                 `json:"source_event_type,omitempty"`
	TargetH3Cells     []string               `json:"target_h3_cells,omitempty"`
	MatrixSize        int32                  `json:"matrix_size"`
	DispatchTimestamp string                 `json:"dispatch_timestamp"`
	Status            OptimizationJobStatus  `json:"status"`
	VRP               *VRPJobPayload         `json:"vrp,omitempty"`
	CPSAT             *CPSATJobPayload       `json:"cpsat,omitempty"`
}

// VRPJobPayload is the backend-owned route-planning snapshot handed to the
// worker. It is intentionally solver-execution-ready so the worker only needs
// to translate into its internal request envelope.
type VRPJobPayload struct {
	DepotNodeUUID     string                     `json:"depot_node_uuid"`
	Depot             *VRPDepotPayload           `json:"depot,omitempty"`
	DropOffNodeUUIDs  []string                   `json:"drop_off_node_uuids"`
	DistanceMatrixKM  [][]float64                `json:"distance_matrix_km"`
	Vehicles          []VRPVehiclePayload        `json:"vehicles"`
	Nodes             []VRPNodeProjectionPayload `json:"nodes,omitempty"`
	NodeDemands       []VRPNodeDemandPayload     `json:"node_demands"`
	NodeTimeWindows   []VRPNodeTimeWindowPayload `json:"node_time_windows"`
	SolverTimeLimitMs int64                      `json:"solver_time_limit_ms,omitempty"`
}

// VRPDepotPayload captures the dispatch origin metadata so projection reads do
// not need a second lookup to render the starting point.
type VRPDepotPayload struct {
	NodeUUID string  `json:"node_uuid"`
	Label    string  `json:"label,omitempty"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

// VRPVehiclePayload is one available vehicle in the route-planning snapshot.
type VRPVehiclePayload struct {
	VehicleUUID      string  `json:"vehicle_uuid"`
	DriverUUID       string  `json:"driver_uuid"`
	DriverName       string  `json:"driver_name,omitempty"`
	VehicleType      string  `json:"vehicle_type,omitempty"`
	VehicleClass     string  `json:"vehicle_class,omitempty"`
	CapacityVU       float64 `json:"capacity_vu"`
	StartWindowHours float64 `json:"start_window_hours"`
	EndWindowHours   float64 `json:"end_window_hours"`
}

// VRPNodeProjectionPayload captures render-ready stop metadata for projection
// reads while keeping the solver-specific demand/time-window payloads additive.
type VRPNodeProjectionPayload struct {
	NodeUUID             string  `json:"node_uuid"`
	OrderID              string  `json:"order_id,omitempty"`
	RetailerID           string  `json:"retailer_id,omitempty"`
	RetailerName         string  `json:"retailer_name,omitempty"`
	Lat                  float64 `json:"lat"`
	Lng                  float64 `json:"lng"`
	Amount               int64   `json:"amount,omitempty"`
	DemandVU             float64 `json:"demand_vu"`
	ReceivingWindowOpen  string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose string  `json:"receiving_window_close,omitempty"`
}

// VRPNodeDemandPayload is the volumetric demand for one stop node.
type VRPNodeDemandPayload struct {
	NodeUUID string  `json:"node_uuid"`
	DemandVU float64 `json:"demand_vu"`
}

// VRPNodeTimeWindowPayload expresses the delivery window for one stop node in
// fractional hours from the start of day.
type VRPNodeTimeWindowPayload struct {
	NodeUUID         string  `json:"node_uuid"`
	StartWindowHours float64 `json:"start_window_hours"`
	EndWindowHours   float64 `json:"end_window_hours"`
}

// CPSATJobPayload is reserved for future queued factory scheduling jobs. K2
// only wires VRP auto-dispatch, but the envelope stays additive.
type CPSATJobPayload struct {
	FactorySlots         []CPSATFactorySlotPayload         `json:"factory_slots"`
	ManifestRequirements []CPSATManifestRequirementPayload `json:"manifest_requirements"`
	SolverTimeLimitMs    int64                             `json:"solver_time_limit_ms,omitempty"`
	NumSearchWorkers     int32                             `json:"num_search_workers,omitempty"`
}

type CPSATFactorySlotPayload struct {
	FactoryNodeUUID string  `json:"factory_node_uuid"`
	SlotCapacity    float64 `json:"slot_capacity"`
}

type CPSATManifestRequirementPayload struct {
	ManifestID               string   `json:"manifest_id"`
	RequiredCapacity         float64  `json:"required_capacity"`
	PriorityScore            float64  `json:"priority_score"`
	EligibleFactoryNodeUUIDs []string `json:"eligible_factory_node_uuids"`
}

// SolverMetadata is copied onto solved-result envelopes so downstream readers
// can decode scaled metrics without importing worker internals.
type SolverMetadata struct {
	JobID            string                 `json:"job_id"`
	TraceID          string                 `json:"trace_id"`
	SupplierID       string                 `json:"supplier_id"`
	SolverType       OptimizationSolverType `json:"solver_type"`
	ScaleFactor      int64                  `json:"scale_factor"`
	IdempotencyKey   string                 `json:"idempotency_key,omitempty"`
	SourceEventType  string                 `json:"source_event_type,omitempty"`
	RequestedAtUnixM int64                  `json:"requested_at_unix_ms"`
}

// VRPResultEnvelope is the canonical solved-route payload written to
// OptimizationJobs.ResultPayload and OutboxEvents.
type VRPResultEnvelope struct {
	Meta                SolverMetadata           `json:"meta"`
	Status              OptimizationSolverStatus `json:"status"`
	TimedOut            bool                     `json:"timed_out"`
	MatrixSize          int32                    `json:"matrix_size"`
	ObjectiveCostScaled int64                    `json:"objective_cost_scaled"`
	Routes              []VehicleRouteEnvelope   `json:"routes"`
	UnassignedNodeUUIDs []string                 `json:"unassigned_node_uuids"`
	Warnings            []string                 `json:"warnings,omitempty"`
}

// VehicleRouteEnvelope is one solved vehicle route in stop order.
type VehicleRouteEnvelope struct {
	VehicleUUID      string   `json:"vehicle_uuid"`
	DriverUUID       string   `json:"driver_uuid"`
	OrderedNodeUUIDs []string `json:"ordered_node_uuids"`
	LoadScaled       int64    `json:"load_scaled"`
	RouteCostScaled  int64    `json:"route_cost_scaled"`
}

// CPSATResultEnvelope is the canonical solved factory-slot payload written to
// OptimizationJobs.ResultPayload and OutboxEvents.
type CPSATResultEnvelope struct {
	Meta                  SolverMetadata           `json:"meta"`
	Status                OptimizationSolverStatus `json:"status"`
	TimedOut              bool                     `json:"timed_out"`
	MatrixSize            int32                    `json:"matrix_size"`
	ObjectiveScoreScaled  int64                    `json:"objective_score_scaled"`
	Assignments           []AssignmentEnvelope     `json:"assignments"`
	UnassignedManifestIDs []string                 `json:"unassigned_manifest_ids"`
	Warnings              []string                 `json:"warnings,omitempty"`
}

// AssignmentEnvelope is one CP-SAT assignment decision.
type AssignmentEnvelope struct {
	ManifestID      string `json:"manifest_id"`
	FactoryNodeUUID string `json:"factory_node_uuid"`
	Assigned        bool   `json:"assigned"`
}

// OptimizationSolvedEvent is the canonical terminal outbox payload for a
// completed optimization job.
type OptimizationSolvedEvent struct {
	JobID      string                   `json:"job_id"`
	TraceID    string                   `json:"trace_id"`
	SupplierID string                   `json:"supplier_id"`
	SolverType OptimizationSolverType   `json:"solver_type"`
	Status     OptimizationSolverStatus `json:"status"`
	TimedOut   bool                     `json:"timed_out"`
	MatrixSize int32                    `json:"matrix_size"`
	ProducedAt string                   `json:"produced_at"`
	Warnings   []string                 `json:"warnings,omitempty"`
	VRP        *VRPResultEnvelope       `json:"vrp,omitempty"`
	CPSAT      *CPSATResultEnvelope     `json:"cpsat,omitempty"`
}
