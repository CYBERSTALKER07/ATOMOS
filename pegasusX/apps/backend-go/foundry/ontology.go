// Package foundry implements the Palantir Foundry-grade Semantic-Kinetic-Dynamic
// architecture, Role 7 (Depot Cashier & AML Vault), Role 9 (Fleet, Yard & DVIR Red-Tag),
// and the 12 mandatory cross-role pre-execution edge-case guards for PegasusX.
package foundry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	MarkingAMLInvestigation         = "MARKING:AML_INVESTIGATION"
	MarkingDiscrepantQuarantine     = "MARKING:DISCREPANT_QUARANTINE"
	MarkingThermalQuarantineSpoiled = "MARKING:THERMAL_QUARANTINE_SPOILED"
	MarkingMicroSkimmingSuspect     = "MARKING:MICRO_SKIMMING_SUSPECT"
	MarkingLotRecallQuarantine      = "MARKING:LOT_RECALL_QUARANTINE"
)

var propagatingMarkings = map[string]bool{
	MarkingAMLInvestigation:         true,
	MarkingDiscrepantQuarantine:     true,
	MarkingThermalQuarantineSpoiled: true,
	MarkingMicroSkimmingSuspect:     true,
	MarkingLotRecallQuarantine:      true,
}

var (
	ErrObjectNotFound           = errors.New("foundry: ontology object not found")
	ErrInvalidObjectIdentity    = errors.New("foundry: cell_id, object_type, and object_id are required")
	ErrPreconditionFailed       = errors.New("foundry: pre-execution kinetic guard failed")
	ErrScenarioWatermarkDrift   = errors.New("foundry: scenario base version watermark conflict")
	ErrDuplicateBagSeal         = errors.New("foundry: tamper-evident bag seal barcode already processed")
	ErrBlindRecountRequired     = errors.New("foundry: variance exceeds tolerance; blind recount #2 required")
	ErrAMLCeilingExceeded       = errors.New("foundry: daily cash AML exposure ceiling exceeded across legal entity TIN")
	ErrOfflineLeaseExceeded     = errors.New("foundry: offline cash/credit collection exceeds pre-allocated lease quota")
	ErrInvalidRRNProof          = errors.New("foundry: RRN tender reclassification proof invalid or outside 15-minute ePOD window")
	ErrVehicleRedTagged         = errors.New("foundry: vehicle is RED_TAG_QUARANTINE (OUT_OF_SERVICE) and blocked from dispatch")
	ErrGateOutboundSealInvalid  = errors.New("foundry: outbound vehicle blocked at gate due to missing or broken bolt seal")
	ErrFiscalPreFlightBlocked   = errors.New("foundry: wave release blocked by missing IKPU tax code or expiring PKCS#12 certificate")
	ErrSlicePayoutUnfiscalized  = errors.New("foundry: settlement slice blocked from payout batch until Soliq EHF is REGISTERED_SOLIQ")
	ErrSliceImmutableAfterBatch = errors.New("foundry: BATCHED settlement slice is immutable; use post-settlement clawback slice")
	ErrCrushabilityViolation    = errors.New("foundry: 3D load plan rejected due to heavy item stacked above fragile carton")
	ErrFXReconfirmationRequired = errors.New("foundry: FX movement exceeded contractual corridor cap; retailer reconfirmation required")
	ErrToteHeartbeatActive      = errors.New("foundry: pick tote heartbeat is still active (< 20m since last scan)")
)

// OntologyObject represents a strongly-typed business entity in the Semantic Layer.
type OntologyObject struct {
	CellID           string         `json:"cell_id" spanner:"CellId"`
	ObjectType       string         `json:"object_type" spanner:"ObjectType"`
	ObjectID         string         `json:"object_id" spanner:"ObjectId"`
	SupplierID       string         `json:"supplier_id" spanner:"SupplierId"`
	Version          int64          `json:"version" spanner:"Version"`
	Properties       map[string]any `json:"properties,omitempty" spanner:"-"`
	PropertiesJson   string         `json:"properties_json,omitempty" spanner:"PropertiesJson"`
	SecurityMarkings []string       `json:"security_markings,omitempty" spanner:"SecurityMarkings"`
	ValidFrom        time.Time      `json:"valid_from" spanner:"ValidFrom"`
	UpdatedAt        time.Time      `json:"updated_at" spanner:"UpdatedAt"`
}

type OntologyObjects = OntologyObject

func (o *OntologyObject) PrepareForSpanner() error {
	if o.PropertiesJson == "" && o.Properties != nil {
		b, err := json.Marshal(o.Properties)
		if err != nil {
			return err
		}
		o.PropertiesJson = string(b)
	} else if o.PropertiesJson != "" && o.Properties == nil {
		var props map[string]any
		if err := json.Unmarshal([]byte(o.PropertiesJson), &props); err == nil {
			o.Properties = props
		}
	}
	return nil
}

// OntologyGraphEdge represents a directed relationship between two OntologyObjects.
type OntologyGraphEdge struct {
	CellID            string            `json:"cell_id" spanner:"CellId"`
	SourceObjectType  string            `json:"source_object_type" spanner:"SourceObjectType"`
	SourceObjectID    string            `json:"source_object_id" spanner:"SourceObjectId"`
	LinkType          string            `json:"link_type" spanner:"LinkType"`
	TargetObjectType  string            `json:"target_object_type" spanner:"TargetObjectType"`
	TargetObjectID    string            `json:"target_object_id" spanner:"TargetObjectId"`
	SupplierID        string            `json:"supplier_id" spanner:"SupplierId"`
	CreatedByActionID string            `json:"created_by_action_id" spanner:"CreatedByActionId"`
	Metadata          map[string]string `json:"metadata,omitempty" spanner:"-"`
	MetadataJson      *string           `json:"metadata_json,omitempty" spanner:"MetadataJson"`
	ValidFrom         time.Time         `json:"valid_from" spanner:"ValidFrom"`
	ValidTo           *time.Time        `json:"valid_to,omitempty" spanner:"ValidTo"`
}

type OntologyGraphEdges = OntologyGraphEdge

func (e *OntologyGraphEdge) PrepareForSpanner() error {
	if e.MetadataJson == nil && e.Metadata != nil {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		s := string(b)
		e.MetadataJson = &s
	} else if e.MetadataJson != nil && e.Metadata == nil {
		var m map[string]string
		if err := json.Unmarshal([]byte(*e.MetadataJson), &m); err == nil {
			e.Metadata = m
		}
	}
	return nil
}

// RecallBlastRadiusReport summarizes all downstream entities impacted by a contaminated factory lot.
type RecallBlastRadiusReport struct {
	CellID                     string   `json:"cell_id"`
	SupplierID                 string   `json:"supplier_id"`
	ContaminatedRawMaterialLot string   `json:"contaminated_raw_material_lot"`
	AffectedWorkOrders         []string `json:"affected_work_orders"`
	AffectedFinishedLots       []string `json:"affected_finished_lots"`
	QuarantinedSSCCPallets     []string `json:"quarantined_sscc_pallets"`
	QuarantinedWarehouseTotes  []string `json:"quarantined_warehouse_totes"`
	InTransitDriverDropBlocks  []string `json:"in_transit_driver_drop_blocks"`
	RetailerReversePickupOrders []string `json:"retailer_reverse_pickup_orders"`
	ImpactedSoliqFacturas      []string `json:"impacted_soliq_facturas"`
}

// Engine coordinates the Foundry Semantic, Kinetic, Dynamic, Cashier, Fleet, and Edge-Guard layers.
type Engine struct {
	mu sync.RWMutex

	now         func() time.Time
	spannerRepo *SpannerRepository

	objects map[string]*OntologyObject
	edges   map[string][]OntologyGraphEdge

	preconditions map[string][]PreconditionEvaluator
	actionsByKey  map[string]*ActionExecutionRecord
	actionHistory []*ActionExecutionRecord

	scenarioPatches map[string]map[string]*ScenarioPatch

	cashierSessions      map[string]*CashierBlindCountSession
	bagSeals             map[string]string
	driverShiftVariances map[string][]int64
	vaultLedger          map[string][]*DepotVaultLedgerEntry
	vaultBalances        map[string]int64
	amlDailyExposure     map[string]int64
	offlineLeases        map[string]*OfflineQuotaLease

	dvirByVehicle        map[string]*FleetDVIRInspection
	vehiclePredictiveHold map[string]bool
	vehicleDriverPairing map[string]string
	gateEvents           []*YardGateCheckEvent

	containerLedger      map[string]*ReturnableContainerBalance
	supplierReserveMinor map[string]int64
	toteStates           map[string]*PickToteState
}

// NewEngine initializes a thread-safe Foundry Engine with all 10-step pre-execution guards wired.
func NewEngine() *Engine {
	eng := &Engine{
		now:                   func() time.Time { return time.Now().UTC() },
		objects:               make(map[string]*OntologyObject),
		edges:                 make(map[string][]OntologyGraphEdge),
		preconditions:         make(map[string][]PreconditionEvaluator),
		actionsByKey:          make(map[string]*ActionExecutionRecord),
		scenarioPatches:       make(map[string]map[string]*ScenarioPatch),
		cashierSessions:       make(map[string]*CashierBlindCountSession),
		bagSeals:              make(map[string]string),
		driverShiftVariances:  make(map[string][]int64),
		vaultLedger:           make(map[string][]*DepotVaultLedgerEntry),
		vaultBalances:         make(map[string]int64),
		amlDailyExposure:      make(map[string]int64),
		offlineLeases:         make(map[string]*OfflineQuotaLease),
		dvirByVehicle:         make(map[string]*FleetDVIRInspection),
		vehiclePredictiveHold: make(map[string]bool),
		vehicleDriverPairing:  make(map[string]string),
		containerLedger:       make(map[string]*ReturnableContainerBalance),
		supplierReserveMinor:  make(map[string]int64),
		toteStates:            make(map[string]*PickToteState),
	}
	eng.registerDefaultPreExecutionGuards()
	return eng
}

// ActionHistoryCount returns the current count of items in the actionHistory ring buffer.
func (e *Engine) ActionHistoryCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.actionHistory)
}

// GateEventsCount returns the current count of items in the gateEvents ring buffer.
func (e *Engine) GateEventsCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.gateEvents)
}

// VaultLedgerCount returns the current count of items in the vaultLedger ring buffer for a depot.
func (e *Engine) VaultLedgerCount(depotID string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.vaultLedger[strings.TrimSpace(depotID)])
}

// DriverShiftVarianceCount returns the current count of variance history items for a driver.
func (e *Engine) DriverShiftVarianceCount(driverID string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.driverShiftVariances[strings.TrimSpace(driverID)])
}

func objectKey(cellID, objectType, objectID string) string {
	return fmt.Sprintf("%s::%s::%s", strings.TrimSpace(cellID), strings.TrimSpace(objectType), strings.TrimSpace(objectID))
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return make(map[string]any)
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneObject(o *OntologyObject) *OntologyObject {
	if o == nil {
		return nil
	}
	return &OntologyObject{
		CellID:           o.CellID,
		ObjectType:       o.ObjectType,
		ObjectID:         o.ObjectID,
		SupplierID:       o.SupplierID,
		Version:          o.Version,
		Properties:       cloneMap(o.Properties),
		PropertiesJson:   o.PropertiesJson,
		SecurityMarkings: slices.Clone(o.SecurityMarkings),
		ValidFrom:        o.ValidFrom,
		UpdatedAt:        o.UpdatedAt,
	}
}

func mergeMarkings(existing, incoming []string) []string {
	seen := make(map[string]bool, len(existing)+len(incoming))
	var out []string
	for _, m := range existing {
		m = strings.TrimSpace(m)
		if m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	for _, m := range incoming {
		m = strings.TrimSpace(m)
		if m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	slices.Sort(out)
	return out
}

// UpsertObject creates or updates an OntologyObject and increments its monotonic Version.
func (e *Engine) UpsertObject(ctx context.Context, obj OntologyObject) (*OntologyObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cellID := strings.TrimSpace(obj.CellID)
	objType := strings.TrimSpace(obj.ObjectType)
	objID := strings.TrimSpace(obj.ObjectID)
	if cellID == "" || objType == "" || objID == "" {
		return nil, ErrInvalidObjectIdentity
	}

	key := objectKey(cellID, objType, objID)
	now := e.now()

	e.mu.RLock()
	repo := e.spannerRepo
	existing, hasExisting := e.objects[key]
	var baseVer int64
	_ = baseVer
	var proposed *OntologyObject
	if hasExisting {
		baseVer = existing.Version
		proposed = cloneObject(existing)
		proposed.Version++
		if obj.SupplierID != "" {
			proposed.SupplierID = obj.SupplierID
		}
		if proposed.Properties == nil {
			proposed.Properties = make(map[string]any)
		}
		for k, v := range obj.Properties {
			proposed.Properties[k] = v
		}
		proposed.SecurityMarkings = mergeMarkings(proposed.SecurityMarkings, obj.SecurityMarkings)
		proposed.UpdatedAt = now
	} else {
		proposed = &OntologyObject{
			CellID:           cellID,
			ObjectType:       objType,
			ObjectID:         objID,
			SupplierID:       strings.TrimSpace(obj.SupplierID),
			Version:          1,
			Properties:       cloneMap(obj.Properties),
			SecurityMarkings: mergeMarkings(nil, obj.SecurityMarkings),
			ValidFrom:        now,
			UpdatedAt:        now,
		}
	}
	e.mu.RUnlock()

	_ = proposed.PrepareForSpanner()

	if repo != nil {
		// Pure Spanner persistence first: zero in-memory mutations before Spanner commit.
		if err := repo.PersistOntologyObject(ctx, proposed); err != nil {
			return nil, err
		}
	}

	// Update in-memory read-through cache strictly post-commit.
	e.mu.Lock()
	e.objects[key] = proposed
	e.mu.Unlock()

	return cloneObject(proposed), nil
}

func (e *Engine) upsertObjectLocked(cellID, objType, objID, supplierID string, props map[string]any, markings []string) *OntologyObject {
	key := objectKey(cellID, objType, objID)
	now := e.now()
	if existing, ok := e.objects[key]; ok {
		existing.Version++
		if supplierID != "" {
			existing.SupplierID = supplierID
		}
		if existing.Properties == nil {
			existing.Properties = make(map[string]any)
		}
		for k, v := range props {
			existing.Properties[k] = v
		}
		existing.SecurityMarkings = mergeMarkings(existing.SecurityMarkings, markings)
		existing.UpdatedAt = now
		_ = existing.PrepareForSpanner()
		return cloneObject(existing)
	}

	created := &OntologyObject{
		CellID:           cellID,
		ObjectType:       objType,
		ObjectID:         objID,
		SupplierID:       strings.TrimSpace(supplierID),
		Version:          1,
		Properties:       cloneMap(props),
		SecurityMarkings: mergeMarkings(nil, markings),
		ValidFrom:        now,
		UpdatedAt:        now,
	}
	_ = created.PrepareForSpanner()
	e.objects[key] = created
	return cloneObject(created)
}

// GetObject retrieves a live OntologyObject by its canonical coordinates.
// In-memory data structures serve strictly as a read-through cache backed by Spanner.
func (e *Engine) GetObject(ctx context.Context, cellID, objectType, objectID string) (*OntologyObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := objectKey(cellID, objectType, objectID)
	e.mu.RLock()
	obj, ok := e.objects[key]
	repo := e.spannerRepo
	e.mu.RUnlock()

	if ok {
		return cloneObject(obj), nil
	}

	if repo != nil {
		spannerObj, err := repo.GetOntologyObject(ctx, cellID, objectType, objectID)
		if err == nil && spannerObj != nil {
			e.mu.Lock()
			e.objects[key] = cloneObject(spannerObj)
			e.mu.Unlock()
			return cloneObject(spannerObj), nil
		}
	}

	return nil, ErrObjectNotFound
}

// LinkObjects creates a directed edge between two OntologyObjects and propagates mandatory SecurityMarkings.
func (e *Engine) LinkObjects(ctx context.Context, edge OntologyGraphEdge) (*OntologyGraphEdge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cellID := strings.TrimSpace(edge.CellID)
	srcType := strings.TrimSpace(edge.SourceObjectType)
	srcID := strings.TrimSpace(edge.SourceObjectID)
	linkType := strings.TrimSpace(edge.LinkType)
	dstType := strings.TrimSpace(edge.TargetObjectType)
	dstID := strings.TrimSpace(edge.TargetObjectID)
	if cellID == "" || srcType == "" || srcID == "" || linkType == "" || dstType == "" || dstID == "" {
		return nil, ErrInvalidObjectIdentity
	}

	e.mu.Lock()

	srcKey := objectKey(cellID, srcType, srcID)
	dstKey := objectKey(cellID, dstType, dstID)
	srcObj, ok := e.objects[srcKey]
	if !ok {
		e.mu.Unlock()
		return nil, fmt.Errorf("%w: source %s/%s", ErrObjectNotFound, srcType, srcID)
	}
	dstObj, ok := e.objects[dstKey]
	if !ok {
		e.mu.Unlock()
		return nil, fmt.Errorf("%w: target %s/%s", ErrObjectNotFound, dstType, dstID)
	}

	now := e.now()
	created := OntologyGraphEdge{
		CellID:            cellID,
		SourceObjectType:  srcType,
		SourceObjectID:    srcID,
		LinkType:          linkType,
		TargetObjectType:  dstType,
		TargetObjectID:    dstID,
		SupplierID:        edge.SupplierID,
		CreatedByActionID: edge.CreatedByActionID,
		Metadata:          edge.Metadata,
		ValidFrom:         now,
	}

	existingEdges := e.edges[srcKey]
	replaced := false
	for i := range existingEdges {
		if existingEdges[i].LinkType == linkType &&
			existingEdges[i].TargetObjectType == dstType &&
			existingEdges[i].TargetObjectID == dstID {
			existingEdges[i] = created
			replaced = true
			break
		}
	}
	if !replaced {
		e.edges[srcKey] = append(existingEdges, created)
	}

	// Propagate mandatory security markings downstream across the lineage graph.
	var inherited []string
	for _, m := range srcObj.SecurityMarkings {
		if propagatingMarkings[m] {
			inherited = append(inherited, m)
		}
	}
	if len(inherited) > 0 {
		e.propagateMarkingsLocked(dstObj, inherited)
	}
	repo := e.spannerRepo
	e.mu.Unlock()

	if repo != nil {
		if err := repo.PersistOntologyGraphEdge(ctx, &created); err != nil {
			return nil, err
		}
	}
	return &created, nil
}

func (e *Engine) propagateMarkingsLocked(start *OntologyObject, markings []string) {
	queue := []*OntologyObject{start}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		k := objectKey(curr.CellID, curr.ObjectType, curr.ObjectID)
		if visited[k] {
			continue
		}
		visited[k] = true

		beforeLen := len(curr.SecurityMarkings)
		curr.SecurityMarkings = mergeMarkings(curr.SecurityMarkings, markings)
		if len(curr.SecurityMarkings) > beforeLen {
			curr.Version++
			curr.UpdatedAt = e.now()
		}

		for _, edge := range e.edges[k] {
			if edge.ValidTo != nil {
				continue
			}
			targetKey := objectKey(edge.CellID, edge.TargetObjectType, edge.TargetObjectID)
			if nextObj, ok := e.objects[targetKey]; ok && !visited[targetKey] {
				queue = append(queue, nextObj)
			}
		}
	}
}

// SearchAround executes a multi-hop breadth-first traversal across the Ontology Graph.
func (e *Engine) SearchAround(
	ctx context.Context,
	cellID string,
	startObjectType string,
	startObjectIDs []string,
	allowedLinkTypes []string,
	maxHops int,
) ([]*OntologyObject, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxHops <= 0 {
		maxHops = 1
	}
	if maxHops > 12 {
		maxHops = 12
	}

	linkFilter := make(map[string]bool, len(allowedLinkTypes))
	for _, lt := range allowedLinkTypes {
		lt = strings.TrimSpace(lt)
		if lt != "" {
			linkFilter[lt] = true
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	type hopNode struct {
		key   string
		depth int
	}

	visited := make(map[string]bool)
	var queue []hopNode
	for _, id := range startObjectIDs {
		k := objectKey(cellID, startObjectType, id)
		if _, ok := e.objects[k]; ok && !visited[k] {
			visited[k] = true
			queue = append(queue, hopNode{key: k, depth: 0})
		}
	}

	var results []*OntologyObject
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.depth > 0 {
			if obj, ok := e.objects[curr.key]; ok {
				results = append(results, cloneObject(obj))
			}
		}
		if curr.depth >= maxHops {
			continue
		}

		for _, edge := range e.edges[curr.key] {
			if edge.ValidTo != nil {
				continue
			}
			if len(linkFilter) > 0 && !linkFilter[edge.LinkType] {
				continue
			}
			nextKey := objectKey(edge.CellID, edge.TargetObjectType, edge.TargetObjectID)
			if !visited[nextKey] {
				visited[nextKey] = true
				queue = append(queue, hopNode{key: nextKey, depth: curr.depth + 1})
			}
		}
	}

	return results, nil
}

// TriggerContaminatedLotRecall executes a 1-click forward genealogy traversal from a RawMaterialLot
// across WorkOrders, FinishedLots, SSCCPallets, Totes, Orders, and SoliqFacturas, stamping
// MARKING:LOT_RECALL_QUARANTINE on every downstream object.
func (e *Engine) TriggerContaminatedLotRecall(
	ctx context.Context,
	cellID string,
	supplierID string,
	rawMaterialLotID string,
) (*RecallBlastRadiusReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	e.mu.Lock()
	rootKey := objectKey(cellID, "RawMaterialLot", rawMaterialLotID)
	rootObj, ok := e.objects[rootKey]
	if !ok {
		e.mu.Unlock()
		return nil, fmt.Errorf("%w: RawMaterialLot/%s", ErrObjectNotFound, rawMaterialLotID)
	}
	e.propagateMarkingsLocked(rootObj, []string{MarkingLotRecallQuarantine})
	e.mu.Unlock()

	impacted, err := e.SearchAround(ctx, cellID, "RawMaterialLot", []string{rawMaterialLotID}, nil, 8)
	if err != nil {
		return nil, err
	}

	report := &RecallBlastRadiusReport{
		CellID:                     cellID,
		SupplierID:                 supplierID,
		ContaminatedRawMaterialLot: rawMaterialLotID,
	}

	for _, obj := range impacted {
		switch obj.ObjectType {
		case "WorkOrder":
			report.AffectedWorkOrders = append(report.AffectedWorkOrders, obj.ObjectID)
		case "FinishedLot":
			report.AffectedFinishedLots = append(report.AffectedFinishedLots, obj.ObjectID)
		case "SSCCPallet":
			report.QuarantinedSSCCPallets = append(report.QuarantinedSSCCPallets, obj.ObjectID)
		case "Tote":
			report.QuarantinedWarehouseTotes = append(report.QuarantinedWarehouseTotes, obj.ObjectID)
		case "Order":
			status, _ := obj.Properties["status"].(string)
			if status == "IN_TRANSIT" || status == "LOADED" || status == "ARRIVED" {
				report.InTransitDriverDropBlocks = append(report.InTransitDriverDropBlocks, obj.ObjectID)
			} else {
				report.RetailerReversePickupOrders = append(report.RetailerReversePickupOrders, obj.ObjectID)
			}
		case "SoliqFactura":
			report.ImpactedSoliqFacturas = append(report.ImpactedSoliqFacturas, obj.ObjectID)
		}
	}

	return report, nil
}
