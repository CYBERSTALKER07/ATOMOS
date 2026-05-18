package model

import "encoding/json"

type SolverType string

const (
	SolverTypeVRP   SolverType = "VRP"
	SolverTypeCPSAT SolverType = "CP_SAT"
)

type OptimizationJob struct {
	JobID            string          `json:"job_id"`
	TraceID          string          `json:"trace_id"`
	SupplierID       string          `json:"supplier_id"`
	SolverType       SolverType      `json:"solver_type"`
	IdempotencyKey   string          `json:"idempotency_key"`
	SourceEventType  string          `json:"source_event_type"`
	RequestedAtUnixM int64           `json:"requested_at_unix_ms"`
	Payload          json.RawMessage `json:"payload"`
}

type SolverMetadata struct {
	JobID            string
	TraceID          string
	SupplierID       string
	SolverType       SolverType
	ScaleFactor      int64
	IdempotencyKey   string
	SourceEventType  string
	RequestedAtUnixM int64
}

// --- VRP payloads ---

type VRPPayload struct {
	DepotNodeUUID     string                  `json:"depot_node_uuid"`
	DropOffNodeUUIDs  []string                `json:"drop_off_node_uuids"`
	DistanceMatrixKM  [][]float64             `json:"distance_matrix_km"`
	Vehicles          []VehiclePayload        `json:"vehicles"`
	NodeDemands       []NodeDemandPayload     `json:"node_demands"`
	NodeTimeWindows   []NodeTimeWindowPayload `json:"node_time_windows"`
	SolverTimeLimitMs int64                   `json:"solver_time_limit_ms,omitempty"`
}

type VehiclePayload struct {
	VehicleUUID      string  `json:"vehicle_uuid"`
	DriverUUID       string  `json:"driver_uuid"`
	CapacityVU       float64 `json:"capacity_vu"`
	StartWindowHours float64 `json:"start_window_hours"`
	EndWindowHours   float64 `json:"end_window_hours"`
}

type NodeDemandPayload struct {
	NodeUUID string  `json:"node_uuid"`
	DemandVU float64 `json:"demand_vu"`
}

type NodeTimeWindowPayload struct {
	NodeUUID         string  `json:"node_uuid"`
	StartWindowHours float64 `json:"start_window_hours"`
	EndWindowHours   float64 `json:"end_window_hours"`
}

type VRPRequestEnvelope struct {
	Meta                 SolverMetadata
	DepotNodeUUID        string
	IndexToUUID          []string
	UUIDToIndex          map[string]int
	DistanceMatrixScaled [][]int64
	Vehicles             []VehicleEnvelope
	NodeDemands          []NodeDemandEnvelope
	NodeTimeWindows      []NodeTimeWindowEnvelope
	SolverTimeLimitMs    int64
	ReturnBestEffort     bool
}

type VehicleEnvelope struct {
	VehicleUUID           string
	DriverUUID            string
	CapacityScaled        int64
	StartTimeWindowScaled int64
	EndTimeWindowScaled   int64
}

type NodeDemandEnvelope struct {
	NodeUUID     string
	DemandScaled int64
}

type NodeTimeWindowEnvelope struct {
	NodeUUID              string
	StartTimeWindowScaled int64
	EndTimeWindowScaled   int64
}

type VRPResultEnvelope struct {
	Meta                SolverMetadata         `json:"meta"`
	Feasible            bool                   `json:"feasible"`
	TimedOut            bool                   `json:"timed_out"`
	ObjectiveCostScaled int64                  `json:"objective_cost_scaled"`
	Routes              []VehicleRouteEnvelope `json:"routes"`
	UnassignedNodeUUIDs []string               `json:"unassigned_node_uuids"`
	Warnings            []string               `json:"warnings"`
}

type VehicleRouteEnvelope struct {
	VehicleUUID      string   `json:"vehicle_uuid"`
	DriverUUID       string   `json:"driver_uuid"`
	OrderedNodeUUIDs []string `json:"ordered_node_uuids"`
	LoadScaled       int64    `json:"load_scaled"`
	RouteCostScaled  int64    `json:"route_cost_scaled"`
}

// --- CP-SAT payloads ---

type CPSATPayload struct {
	FactorySlots         []FactorySlotPayload         `json:"factory_slots"`
	ManifestRequirements []ManifestRequirementPayload `json:"manifest_requirements"`
	SolverTimeLimitMs    int64                        `json:"solver_time_limit_ms,omitempty"`
	NumSearchWorkers     int32                        `json:"num_search_workers,omitempty"`
}

type FactorySlotPayload struct {
	FactoryNodeUUID string  `json:"factory_node_uuid"`
	SlotCapacity    float64 `json:"slot_capacity"`
}

type ManifestRequirementPayload struct {
	ManifestID               string   `json:"manifest_id"`
	RequiredCapacity         float64  `json:"required_capacity"`
	PriorityScore            float64  `json:"priority_score"`
	EligibleFactoryNodeUUIDs []string `json:"eligible_factory_node_uuids"`
}

type CPSATRequestEnvelope struct {
	Meta                 SolverMetadata
	FactorySlots         []FactorySlotEnvelope
	ManifestRequirements []ManifestRequirementEnvelope
	SolverTimeLimitMs    int64
	ReturnBestEffort     bool
	NumSearchWorkers     int32
}

type FactorySlotEnvelope struct {
	FactoryNodeUUID    string
	SlotCapacityScaled int64
}

type ManifestRequirementEnvelope struct {
	ManifestID               string
	RequiredCapacityScaled   int64
	PriorityScoreScaled      int64
	EligibleFactoryNodeUUIDs []string
}

type CPSATResultEnvelope struct {
	Meta                  SolverMetadata       `json:"meta"`
	Feasible              bool                 `json:"feasible"`
	TimedOut              bool                 `json:"timed_out"`
	ObjectiveScoreScaled  int64                `json:"objective_score_scaled"`
	Assignments           []AssignmentEnvelope `json:"assignments"`
	UnassignedManifestIDs []string             `json:"unassigned_manifest_ids"`
	Warnings              []string             `json:"warnings"`
}

type AssignmentEnvelope struct {
	ManifestID      string `json:"manifest_id"`
	FactoryNodeUUID string `json:"factory_node_uuid"`
	Assigned        bool   `json:"assigned"`
}

// --- Outbox event projection ---

type OptimizationSolvedEvent struct {
	JobID      string               `json:"job_id"`
	TraceID    string               `json:"trace_id"`
	SupplierID string               `json:"supplier_id"`
	SolverType SolverType           `json:"solver_type"`
	Feasible   bool                 `json:"feasible"`
	TimedOut   bool                 `json:"timed_out"`
	ProducedAt string               `json:"produced_at"`
	Warnings   []string             `json:"warnings,omitempty"`
	VRP        *VRPResultEnvelope   `json:"vrp,omitempty"`
	CPSAT      *CPSATResultEnvelope `json:"cpsat,omitempty"`
}
