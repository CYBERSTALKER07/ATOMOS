package foundry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	BaySlotInboundStaging    = "INBOUND_STAGING"
	BaySlotCrossDock         = "CROSS_DOCK"
	BaySlotOutboundLoading   = "OUTBOUND_LOADING"
	BaySlotMechanicalService = "MECHANICAL_SERVICE"

	DVIRStatusDispatchAuthorized = "DISPATCH_AUTHORIZED"
	DVIRStatusRedTagQuarantine   = "RED_TAG_QUARANTINE"
)

// DVIRChecklist8Point captures the mandatory 8-point Pre-Trip Digital Walkaround.
type DVIRChecklist8Point struct {
	SteeringAndHornOK             bool `json:"steering_and_horn_ok"`
	ServiceAndParkingBrakesOK     bool `json:"service_and_parking_brakes_ok"`
	TiresTreadAndPressureOK       bool `json:"tires_tread_and_pressure_ok"`
	LightsAndHazardsOK            bool `json:"lights_and_hazards_ok"`
	WipersAndFluidsOK             bool `json:"wipers_and_fluids_ok"`
	EmergencyKitAndExtinguisherOK bool `json:"emergency_kit_and_extinguisher_ok"`
	CargoBoxSanitationOK          bool `json:"cargo_box_sanitation_ok"`
	ReeferTempCalibratedOK        bool `json:"reefer_temp_calibrated_ok"`
}

// HasCriticalDefect returns true and the list of failed safety points if any of the 8 checks fail.
func (c DVIRChecklist8Point) HasCriticalDefect(requiresColdChain bool) (bool, []string) {
	var failed []string
	if !c.SteeringAndHornOK {
		failed = append(failed, "steering_and_horn")
	}
	if !c.ServiceAndParkingBrakesOK {
		failed = append(failed, "service_and_parking_brakes")
	}
	if !c.TiresTreadAndPressureOK {
		failed = append(failed, "tires_tread_and_pressure")
	}
	if !c.LightsAndHazardsOK {
		failed = append(failed, "lights_and_hazards")
	}
	if !c.WipersAndFluidsOK {
		failed = append(failed, "wipers_and_fluids")
	}
	if !c.EmergencyKitAndExtinguisherOK {
		failed = append(failed, "emergency_kit_and_extinguisher")
	}
	if !c.CargoBoxSanitationOK {
		failed = append(failed, "cargo_box_sanitation")
	}
	if requiresColdChain && !c.ReeferTempCalibratedOK {
		failed = append(failed, "reefer_temp_calibrated")
	}
	return len(failed) > 0, failed
}

// VehicleGroundedRedTagError is returned when a vehicle is grounded due to critical defects (UA-8).
type VehicleGroundedRedTagError struct {
	InspectionID    string   `json:"inspection_id"`
	VehicleID       string   `json:"vehicle_id"`
	CriticalDefects []string `json:"critical_defects"`
	Reason          string   `json:"reason"`
}

func (e *VehicleGroundedRedTagError) Error() string {
	return fmt.Sprintf("vehicle %s is grounded under RED_TAG_QUARANTINE (inspection: %s): critical defects: %v. %s",
		e.VehicleID, e.InspectionID, e.CriticalDefects, e.Reason)
}

func (e *VehicleGroundedRedTagError) Unwrap() error {
	return ErrVehicleRedTagged
}

// FleetDVIRInspection represents a Pre-Trip or Post-Trip vehicle inspection record.
type FleetDVIRInspection struct {
	InspectionID           string              `json:"inspection_id" spanner:"InspectionId"`
	SupplierID             string              `json:"supplier_id" spanner:"SupplierId"`
	VehicleID              string              `json:"vehicle_id" spanner:"VehicleId"`
	DriverID               string              `json:"driver_id" spanner:"DriverId"`
	InspectionType         string              `json:"inspection_type" spanner:"InspectionType"` // PRE_TRIP | POST_TRIP
	Checklist              DVIRChecklist8Point `json:"checklist,omitempty" spanner:"-"`
	ChecklistJson          string              `json:"checklist_json,omitempty" spanner:"ChecklistJson"`
	CriticalDefectFound    bool                `json:"critical_defect_found" spanner:"CriticalDefectFound"`
	RedTagApplied          bool                `json:"red_tag_applied" spanner:"RedTagApplied"`
	OdometerKm             int64               `json:"odometer_km" spanner:"OdometerKm"`
	ReeferTempCelsius      *float64            `json:"reefer_temp_celsius,omitempty" spanner:"ReeferTempCelsius"`
	Status                 string              `json:"status" spanner:"Status"`
	MaintenanceWorkOrderID *string             `json:"maintenance_work_order_id,omitempty" spanner:"MaintenanceWorkOrderId"`
	CreatedAt              time.Time           `json:"created_at" spanner:"CreatedAt"`
	CellID                 string              `json:"cell_id,omitempty" spanner:"-"`
	RequiresColdChain      bool                `json:"requires_cold_chain,omitempty" spanner:"-"`
	FailedPoints           []string            `json:"failed_points,omitempty" spanner:"-"`
	PreventativeServiceDue bool                `json:"preventative_service_due,omitempty" spanner:"-"`
}

type FleetDVIRInspections = FleetDVIRInspection

func (f *FleetDVIRInspection) PrepareForSpanner() error {
	if f.ChecklistJson == "" {
		b, err := json.Marshal(f.Checklist)
		if err != nil {
			return err
		}
		f.ChecklistJson = string(b)
	} else if f.Checklist == (DVIRChecklist8Point{}) {
		var c DVIRChecklist8Point
		if err := json.Unmarshal([]byte(f.ChecklistJson), &c); err == nil {
			f.Checklist = c
		}
	}
	return nil
}

// YardGateCheckEvent records ALPR gate check-in/out and dynamic yard bay slotting.
type YardGateCheckEvent struct {
	GateEventID     string    `json:"gate_event_id" spanner:"GateEventId"`
	DepotID         string    `json:"depot_id" spanner:"DepotId"`
	SupplierID      string    `json:"supplier_id" spanner:"SupplierId"`
	VehicleID       string    `json:"vehicle_id" spanner:"VehicleId"`
	LicensePlate    string    `json:"license_plate" spanner:"LicensePlate"`
	Direction       string    `json:"direction" spanner:"Direction"` // GATE_IN | GATE_OUT
	BoltSealBarcode *string   `json:"bolt_seal_barcode,omitempty" spanner:"BoltSealBarcode"`
	SealIntact      bool      `json:"seal_intact" spanner:"SealIntact"`
	AssignedBaySlot string    `json:"assigned_bay_slot" spanner:"AssignedBaySlot"`
	CreatedAt       time.Time `json:"created_at" spanner:"CreatedAt"`
	VehiclePurpose  string    `json:"vehicle_purpose,omitempty" spanner:"-"`
}

type YardGateCheckEvents = YardGateCheckEvent

// PairDriverWithVehicle pairs a driver with a vehicle prior to DVIR inspection.
func (e *Engine) PairDriverWithVehicle(vehicleID, driverID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vehicleDriverPairing[strings.TrimSpace(vehicleID)] = strings.TrimSpace(driverID)
}

// SubmitDVIRInspection evaluates the 8-point walkaround and enforces the Red-Tag Grounding Gate (Guard #12).
func (e *Engine) SubmitDVIRInspection(ctx context.Context, req FleetDVIRInspection) (*FleetDVIRInspection, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	vehicleID := strings.TrimSpace(req.VehicleID)
	driverID := strings.TrimSpace(req.DriverID)
	if vehicleID == "" || driverID == "" {
		return nil, fmt.Errorf("%w: vehicle_id and driver_id are required", ErrPreconditionFailed)
	}

	hasDefect, failedPoints := req.Checklist.HasCriticalDefect(req.RequiresColdChain)
	status := DVIRStatusDispatchAuthorized
	var workOrderID string
	var woPtr *string
	if hasDefect {
		status = DVIRStatusRedTagQuarantine
		workOrderID = "wo_" + uuid.NewString()
		woPtr = &workOrderID
	}

	// Preventative maintenance check: oil change every 10,000 km or brake relining every 30,000 km.
	prevDue := req.OdometerKm > 0 && (req.OdometerKm%10000 < 150 || req.OdometerKm%30000 < 250)

	cellID := strings.TrimSpace(req.CellID)
	if cellID == "" {
		cellID = "cell-uz"
	}

	record := &FleetDVIRInspection{
		InspectionID:           uuid.NewString(),
		CellID:                 cellID,
		SupplierID:             strings.TrimSpace(req.SupplierID),
		VehicleID:              vehicleID,
		DriverID:               driverID,
		InspectionType:         strings.ToUpper(strings.TrimSpace(req.InspectionType)),
		RequiresColdChain:      req.RequiresColdChain,
		Checklist:              req.Checklist,
		FailedPoints:           failedPoints,
		CriticalDefectFound:    hasDefect,
		RedTagApplied:          hasDefect,
		OdometerKm:             req.OdometerKm,
		ReeferTempCelsius:      req.ReeferTempCelsius,
		Status:                 status,
		MaintenanceWorkOrderID: woPtr,
		PreventativeServiceDue: prevDue,
		CreatedAt:              e.now(),
	}
	if record.InspectionType == "" {
		record.InspectionType = "PRE_TRIP"
	}
	_ = record.PrepareForSpanner()

	e.mu.Lock()
	defer e.mu.Unlock()

	e.dvirByVehicle[vehicleID] = record
	if hasDefect {
		// Uncouple driver-vehicle pairing immediately on Red-Tag!
		delete(e.vehicleDriverPairing, vehicleID)
		e.upsertObjectLocked(cellID, "Vehicle", vehicleID, record.SupplierID, map[string]any{
			"status":                    "OUT_OF_SERVICE",
			"red_tag_applied":           true,
			"maintenance_work_order_id": workOrderID,
			"failed_points":             failedPoints,
		}, []string{MarkingDiscrepantQuarantine})
	} else {
		e.vehicleDriverPairing[vehicleID] = driverID
		e.upsertObjectLocked(cellID, "Vehicle", vehicleID, record.SupplierID, map[string]any{
			"status":          "DISPATCH_AUTHORIZED",
			"red_tag_applied": false,
			"paired_driver":   driverID,
		}, nil)
	}
	if e.spannerRepo != nil {
		_ = e.spannerRepo.PersistDVIRInspection(ctx, record)
	}

	return record, nil
}

// EvaluatePredictiveReeferHealth monitors compressor cooling curves and places degrading vans on hold before loading.
func (e *Engine) EvaluatePredictiveReeferHealth(
	ctx context.Context,
	vehicleID string,
	coolingRateDegPerMin float64,
	minRequiredRateDegPerMin float64,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	vehicleID = strings.TrimSpace(vehicleID)
	if vehicleID == "" {
		return false, ErrInvalidObjectIdentity
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	hold := coolingRateDegPerMin < minRequiredRateDegPerMin
	e.vehiclePredictiveHold[vehicleID] = hold
	if hold {
		delete(e.vehicleDriverPairing, vehicleID)
	}
	return hold, nil
}

// AssertVehicleDispatchAuthorized is the pre-execution gate called before Payloader loading or Driver route start.
func (e *Engine) AssertVehicleDispatchAuthorized(ctx context.Context, vehicleID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	vehicleID = strings.TrimSpace(vehicleID)
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.vehiclePredictiveHold[vehicleID] {
		return &VehicleGroundedRedTagError{
			VehicleID:       vehicleID,
			CriticalDefects: []string{"reefer_compressor_degradation"},
			Reason:          "vehicle is on PRE_FAILURE_MAINTENANCE_HOLD due to reefer compressor degradation",
		}
	}
	insp, ok := e.dvirByVehicle[vehicleID]
	if !ok {
		return &VehicleGroundedRedTagError{
			VehicleID:       vehicleID,
			CriticalDefects: []string{"missing_pre_trip_dvir"},
			Reason:          "vehicle has no completed Pre-Trip DVIR inspection",
		}
	}
	if insp.RedTagApplied || insp.Status != DVIRStatusDispatchAuthorized {
		return &VehicleGroundedRedTagError{
			InspectionID:    insp.InspectionID,
			VehicleID:       vehicleID,
			CriticalDefects: insp.FailedPoints,
			Reason:          "vehicle failed Pre-Trip walkaround safety checks",
		}
	}
	return nil
}

// RecordGateCheckEvent logs ALPR gate check-in/out, enforces outbound bolt seal integrity, and assigns a yard slot.
func (e *Engine) RecordGateCheckEvent(ctx context.Context, req YardGateCheckEvent) (*YardGateCheckEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	vehicleID := strings.TrimSpace(req.VehicleID)
	plate := strings.TrimSpace(req.LicensePlate)
	dir := strings.ToUpper(strings.TrimSpace(req.Direction))
	if vehicleID == "" || plate == "" || (dir != "GATE_IN" && dir != "GATE_OUT") {
		return nil, fmt.Errorf("%w: vehicle_id, license_plate, and direction (GATE_IN|GATE_OUT) required", ErrPreconditionFailed)
	}

	var sealBarcodeStr string
	var sealPtr *string
	if req.BoltSealBarcode != nil {
		sealBarcodeStr = strings.TrimSpace(*req.BoltSealBarcode)
		if sealBarcodeStr != "" {
			sealPtr = &sealBarcodeStr
		}
	}

	if dir == "GATE_OUT" {
		if !req.SealIntact || sealBarcodeStr == "" {
			return nil, ErrGateOutboundSealInvalid
		}
		if err := e.AssertVehicleDispatchAuthorized(ctx, vehicleID); err != nil {
			return nil, err
		}
	}

	slot := BaySlotOutboundLoading
	switch strings.ToUpper(strings.TrimSpace(req.VehiclePurpose)) {
	case "INBOUND_FTL":
		slot = BaySlotInboundStaging
	case "CROSS_DOCK":
		slot = BaySlotCrossDock
	case "WORKSHOP":
		slot = BaySlotMechanicalService
	}
	if !req.SealIntact && dir == "GATE_IN" {
		slot = BaySlotMechanicalService
	}

	event := &YardGateCheckEvent{
		GateEventID:     uuid.NewString(),
		DepotID:         strings.TrimSpace(req.DepotID),
		SupplierID:      strings.TrimSpace(req.SupplierID),
		VehicleID:       vehicleID,
		LicensePlate:    plate,
		Direction:       dir,
		VehiclePurpose:  req.VehiclePurpose,
		BoltSealBarcode: sealPtr,
		SealIntact:      req.SealIntact,
		AssignedBaySlot: slot,
		CreatedAt:       e.now(),
	}

	e.mu.Lock()
	e.gateEvents = append(e.gateEvents, event)
	if len(e.gateEvents) > 1000 {
		e.gateEvents = e.gateEvents[len(e.gateEvents)-1000:]
	}
	e.mu.Unlock()
	return event, nil
}
