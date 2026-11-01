package foundry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// MaxTransactionMutations is the maximum number of mutations permitted in a single Spanner transaction (UA-12).
const MaxTransactionMutations = 1000

var (
	// ErrTransactionMutationLimitExceeded is returned when a transaction attempts to buffer more than 1,000 mutations (UA-12).
	ErrTransactionMutationLimitExceeded = errors.New("transaction mutation limit exceeded (max 1000 mutations)")
	// ErrVersionConflict is returned when an optimistic concurrency check fails.
	ErrVersionConflict = errors.New("version conflict: optimistic locking failure")
)

// SpannerRepository persists Foundry Ontology, Kinetic Actions, Scenario Overlays,
// Depot Cashier sessions, Fleet DVIR Red-Tags, and Edge-Case Ledgers into Cloud Spanner
// with atomic OutboxEvents buffering.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Cloud Spanner backing store for the Foundry Engine.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	if client == nil {
		return nil
	}
	return &SpannerRepository{client: client}
}

// WithSpannerClient attaches a Cloud Spanner client to the Foundry Engine.
func (e *Engine) WithSpannerClient(client *spanner.Client) *Engine {
	if e == nil || client == nil {
		return e
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spannerRepo = NewSpannerRepository(client)
	return e
}

type spannerOutboxBuffer struct {
	events []outbox.Event
}

func (b *spannerOutboxBuffer) BufferOutbox(_ context.Context, ev outbox.Event) error {
	b.events = append(b.events, ev)
	return nil
}

func toOutboxMutations(eventsList []outbox.Event) []*spanner.Mutation {
	muts := make([]*spanner.Mutation, 0, len(eventsList))
	for _, ev := range eventsList {
		muts = append(muts, spanner.InsertOrUpdateMap("OutboxEvents", outbox.EventRowMap(ev)))
	}
	return muts
}

func toNullJSON(s string) spanner.NullJSON {
	if s == "" {
		return spanner.NullJSON{Valid: false}
	}
	return spanner.NullJSON{Value: json.RawMessage(s), Valid: true}
}

func nullJSONToString(nj spanner.NullJSON) string {
	if !nj.Valid || nj.Value == nil {
		return ""
	}
	switch v := nj.Value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case json.RawMessage:
		return string(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

type ontologyObjectKey struct {
	CellID     string
	ObjectType string
	ObjectID   string
}

// DeduplicateOntologyObjects deduplicates a slice of OntologyObject pointers by (CellId, ObjectType, ObjectId),
// keeping the last occurrence while preserving order.
func DeduplicateOntologyObjects(objs []*OntologyObject) []*OntologyObject {
	if len(objs) == 0 {
		return nil
	}
	seen := make(map[ontologyObjectKey]int)
	res := make([]*OntologyObject, 0, len(objs))
	for _, o := range objs {
		if o == nil {
			continue
		}
		k := ontologyObjectKey{CellID: o.CellID, ObjectType: o.ObjectType, ObjectID: o.ObjectID}
		if idx, exists := seen[k]; exists {
			res[idx] = o
		} else {
			seen[k] = len(res)
			res = append(res, o)
		}
	}
	return res
}

type scenarioPatchKey struct {
	ScenarioID string
	ObjectType string
	ObjectID   string
}

// DeduplicateScenarioPatches deduplicates a slice of ScenarioObjectPatch pointers by (ScenarioId, ObjectType, ObjectId),
// keeping the last occurrence while preserving order.
func DeduplicateScenarioPatches(patches []*ScenarioObjectPatch) []*ScenarioObjectPatch {
	if len(patches) == 0 {
		return nil
	}
	seen := make(map[scenarioPatchKey]int)
	res := make([]*ScenarioObjectPatch, 0, len(patches))
	for _, p := range patches {
		if p == nil {
			continue
		}
		k := scenarioPatchKey{ScenarioID: p.ScenarioID, ObjectType: p.ObjectType, ObjectID: p.ObjectID}
		if idx, exists := seen[k]; exists {
			res[idx] = p
		} else {
			seen[k] = len(res)
			res = append(res, p)
		}
	}
	return res
}

// PersistOntologyObject writes an OntologyObject to Cloud Spanner inside a ReadWriteTransaction.
// If expectedVersion is provided (and >= 0), it verifies that the existing object's version matches.
// It preserves the existing ValidFrom commit timestamp on updates.
func (r *SpannerRepository) PersistOntologyObject(ctx context.Context, obj *OntologyObject, expectedVersion ...int64) error {
	if r == nil || r.client == nil || obj == nil {
		return nil
	}
	if err := obj.PrepareForSpanner(); err != nil {
		return err
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var validFrom any = spanner.CommitTimestamp
		row, readErr := txn.ReadRow(ctx, "OntologyObjects", spanner.Key{obj.CellID, obj.ObjectType, obj.ObjectID}, []string{"Version", "ValidFrom"})
		if readErr != nil {
			if spanner.ErrCode(readErr) != codes.NotFound {
				return readErr
			}
			if len(expectedVersion) > 0 && expectedVersion[0] > 0 {
				return fmt.Errorf("%w: object does not exist, expected version %d", ErrVersionConflict, expectedVersion[0])
			}
		} else {
			var existingVer int64
			var existingValidFrom time.Time
			if err := row.Columns(&existingVer, &existingValidFrom); err != nil {
				return err
			}
			if len(expectedVersion) > 0 && expectedVersion[0] >= 0 && existingVer != expectedVersion[0] {
				return fmt.Errorf("%w: current version %d != expected %d", ErrVersionConflict, existingVer, expectedVersion[0])
			}
			validFrom = existingValidFrom
		}

		markings := obj.SecurityMarkings
		if markings == nil {
			markings = []string{}
		}

		mut := spanner.InsertOrUpdateMap("OntologyObjects", map[string]any{
			"CellId":           obj.CellID,
			"ObjectType":       obj.ObjectType,
			"ObjectId":         obj.ObjectID,
			"SupplierId":       obj.SupplierID,
			"Version":          obj.Version,
			"PropertiesJson":   toNullJSON(obj.PropertiesJson),
			"SecurityMarkings": markings,
			"ValidFrom":        validFrom,
			"UpdatedAt":        spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// PersistOntologyObjects batch persists multiple OntologyObjects, deduplicating by PK and enforcing the 1,000 mutation ceiling.
func (r *SpannerRepository) PersistOntologyObjects(ctx context.Context, objs []*OntologyObject) error {
	if len(objs) > MaxTransactionMutations {
		return ErrTransactionMutationLimitExceeded
	}
	deduped := DeduplicateOntologyObjects(objs)
	if len(deduped) > MaxTransactionMutations {
		return ErrTransactionMutationLimitExceeded
	}
	if r == nil || r.client == nil || len(objs) == 0 {
		return nil
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, len(deduped))
		for _, obj := range deduped {
			if err := obj.PrepareForSpanner(); err != nil {
				return err
			}
			markings := obj.SecurityMarkings
			if markings == nil {
				markings = []string{}
			}
			var validFrom any = spanner.CommitTimestamp
			if !obj.ValidFrom.IsZero() {
				validFrom = obj.ValidFrom
			}
			muts = append(muts, spanner.InsertOrUpdateMap("OntologyObjects", map[string]any{
				"CellId":           obj.CellID,
				"ObjectType":       obj.ObjectType,
				"ObjectId":         obj.ObjectID,
				"SupplierId":       obj.SupplierID,
				"Version":          obj.Version,
				"PropertiesJson":   toNullJSON(obj.PropertiesJson),
				"SecurityMarkings": markings,
				"ValidFrom":        validFrom,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}
		if len(muts) > MaxTransactionMutations {
			return ErrTransactionMutationLimitExceeded
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// GetOntologyObject reads an OntologyObject by key from Cloud Spanner with proper iterator disposal.
func (r *SpannerRepository) GetOntologyObject(ctx context.Context, cellID, objectType, objectID string) (*OntologyObject, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT CellId, ObjectType, ObjectId, SupplierId, Version, PropertiesJson, SecurityMarkings, ValidFrom, UpdatedAt
		      FROM OntologyObjects
		      WHERE CellId = @cellID AND ObjectType = @objectType AND ObjectId = @objectID`,
		Params: map[string]any{
			"cellID":     cellID,
			"objectType": objectType,
			"objectID":   objectID,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	var cell, oType, oID, suppID string
	var ver int64
	var propsNJ spanner.NullJSON
	var markings []string
	var validFrom, updatedAt time.Time
	if err := row.Columns(&cell, &oType, &oID, &suppID, &ver, &propsNJ, &markings, &validFrom, &updatedAt); err != nil {
		return nil, err
	}
	propsJSON := nullJSONToString(propsNJ)
	var props map[string]any
	if propsJSON != "" {
		_ = json.Unmarshal([]byte(propsJSON), &props)
	}
	return &OntologyObject{
		CellID:           cell,
		ObjectType:       oType,
		ObjectID:         oID,
		SupplierID:       suppID,
		Version:          ver,
		Properties:       props,
		PropertiesJson:   propsJSON,
		SecurityMarkings: markings,
		ValidFrom:        validFrom,
		UpdatedAt:        updatedAt,
	}, nil
}

// PersistOntologyGraphEdge writes an OntologyGraphEdge to Cloud Spanner.
func (r *SpannerRepository) PersistOntologyGraphEdge(ctx context.Context, edge *OntologyGraphEdge) error {
	if r == nil || r.client == nil || edge == nil {
		return nil
	}
	if err := edge.PrepareForSpanner(); err != nil {
		return err
	}
	var metaJSONVal any = nil
	if edge.MetadataJson != nil {
		metaJSONVal = toNullJSON(*edge.MetadataJson)
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("OntologyGraphEdges", map[string]any{
			"CellId":            edge.CellID,
			"SourceObjectType":  edge.SourceObjectType,
			"SourceObjectId":    edge.SourceObjectID,
			"LinkType":          edge.LinkType,
			"TargetObjectType":  edge.TargetObjectType,
			"TargetObjectId":    edge.TargetObjectID,
			"SupplierId":        edge.SupplierID,
			"CreatedByActionId": edge.CreatedByActionID,
			"MetadataJson":      metaJSONVal,
			"ValidFrom":         spanner.CommitTimestamp,
			"ValidTo":           edge.ValidTo,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// PersistKineticActionExecution writes an immutable KineticActionExecution and emits an OutboxEvent in the same transaction.
func (r *SpannerRepository) PersistKineticActionExecution(ctx context.Context, rec *ActionExecutionRecord) error {
	if r == nil || r.client == nil || rec == nil {
		return nil
	}
	if err := rec.PrepareForSpanner(); err != nil {
		return err
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("KineticActionExecutions", map[string]any{
			"ActionExecutionId":      rec.ActionExecutionID,
			"CellId":                 rec.CellID,
			"SupplierId":             rec.SupplierID,
			"ActionType":             rec.ActionType,
			"ActorId":                rec.ActorID,
			"ActorRole":              rec.ActorRole,
			"ScenarioId":             rec.ScenarioID,
			"Status":                 rec.Status,
			"PreconditionReportJson": toNullJSON(rec.PreconditionReportJson),
			"DiffPayloadJson":        toNullJSON(rec.DiffPayloadJson),
			"IdempotencyKey":         rec.IdempotencyKey,
			"ExecutedAt":             spanner.CommitTimestamp,
		})
		buf := &spannerOutboxBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateControlTower, rec.ActionExecutionID, events.TopicMain, map[string]any{
			"type":                rec.ActionType,
			"action_execution_id": rec.ActionExecutionID,
			"supplier_id":         rec.SupplierID,
			"status":              rec.Status,
		}); emitErr != nil {
			return emitErr
		}
		muts := append([]*spanner.Mutation{mut}, toOutboxMutations(buf.events)...)
		if len(muts) > MaxTransactionMutations {
			return ErrTransactionMutationLimitExceeded
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// PersistScenarioObjectPatches batch persists ScenarioObjectPatches with PK deduplication and 1,000 ceiling.
func (r *SpannerRepository) PersistScenarioObjectPatches(ctx context.Context, patches []*ScenarioObjectPatch) error {
	if len(patches) > MaxTransactionMutations {
		return ErrTransactionMutationLimitExceeded
	}
	deduped := DeduplicateScenarioPatches(patches)
	if len(deduped) > MaxTransactionMutations {
		return ErrTransactionMutationLimitExceeded
	}
	if r == nil || r.client == nil || len(patches) == 0 {
		return nil
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, len(deduped))
		for _, p := range deduped {
			if err := p.PrepareForSpanner(); err != nil {
				return err
			}
			muts = append(muts, spanner.InsertOrUpdateMap("ScenarioObjectPatches", map[string]any{
				"ScenarioId":  p.ScenarioID,
				"ObjectType":  p.ObjectType,
				"ObjectId":    p.ObjectID,
				"SupplierId":  p.SupplierID,
				"Operation":   p.Operation,
				"BaseVersion": p.BaseVersion,
				"PatchJson":   toNullJSON(p.PatchJson),
				"UpdatedAt":   spanner.CommitTimestamp,
			}))
		}
		if len(muts) > MaxTransactionMutations {
			return ErrTransactionMutationLimitExceeded
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// PersistCashierSession writes a single CashierBlindCountSession to Cloud Spanner.
func (r *SpannerRepository) PersistCashierSession(ctx context.Context, sess *CashierBlindCountSession) error {
	if r == nil || r.client == nil || sess == nil {
		return nil
	}
	if err := sess.PrepareForSpanner(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("CashierBlindCountSessions", map[string]any{
			"SessionId":                 sess.SessionID,
			"SupplierId":                sess.SupplierID,
			"DepotId":                   sess.DepotID,
			"DriverId":                  sess.DriverID,
			"RouteId":                   sess.RouteID,
			"BagSealBarcode":            sess.BagSealBarcode,
			"AttemptNumber":             sess.AttemptNumber,
			"DenominationBreakdownJson": toNullJSON(sess.DenominationBreakdownJson),
			"CountedCashMinor":          sess.CountedCashMinor,
			"ExpectedCashMinor":         sess.ExpectedCashMinor,
			"VarianceMinor":             sess.VarianceMinor,
			"Status":                    sess.Status,
			"ClearanceTokenHash":        sess.ClearanceTokenHash,
			"CreatedAt":                 spanner.CommitTimestamp,
			"ResolvedAt":                sess.ResolvedAt,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// PersistCashierSessionTransition atomically commits a Cashier session state transition,
// depot vault ledger entry, merchant daily AML cash exposure, and OutboxEvent row (SO-1, Invariant 8).
func (r *SpannerRepository) PersistCashierSessionTransition(
	ctx context.Context,
	sess *CashierBlindCountSession,
	vaultEntry *DepotVaultLedgerEntry,
	amlExposure *MerchantDailyCashExposure,
) error {
	if r == nil || r.client == nil || sess == nil {
		return nil
	}
	if err := sess.PrepareForSpanner(); err != nil {
		return err
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, 4)

		// 1. Cashier session mutation
		muts = append(muts, spanner.InsertOrUpdateMap("CashierBlindCountSessions", map[string]any{
			"SessionId":                 sess.SessionID,
			"SupplierId":                sess.SupplierID,
			"DepotId":                   sess.DepotID,
			"DriverId":                  sess.DriverID,
			"RouteId":                   sess.RouteID,
			"BagSealBarcode":            sess.BagSealBarcode,
			"AttemptNumber":             sess.AttemptNumber,
			"DenominationBreakdownJson": toNullJSON(sess.DenominationBreakdownJson),
			"CountedCashMinor":          sess.CountedCashMinor,
			"ExpectedCashMinor":         sess.ExpectedCashMinor,
			"VarianceMinor":             sess.VarianceMinor,
			"Status":                    sess.Status,
			"ClearanceTokenHash":        sess.ClearanceTokenHash,
			"CreatedAt":                 spanner.CommitTimestamp,
			"ResolvedAt":                sess.ResolvedAt,
		}))

		// 2. Depot vault ledger entry (if present)
		if vaultEntry != nil {
			muts = append(muts, spanner.InsertOrUpdateMap("DepotVaultLedgerEntries", map[string]any{
				"VaultEntryId":        vaultEntry.VaultEntryID,
				"DepotId":             vaultEntry.DepotID,
				"SupplierId":          vaultEntry.SupplierID,
				"SessionId":           vaultEntry.SessionID,
				"CITManifestId":       vaultEntry.CITManifestID,
				"EntryType":           vaultEntry.EntryType,
				"AmountMinor":         vaultEntry.AmountMinor,
				"RunningBalanceMinor": vaultEntry.RunningBalanceMinor,
				"CourierBadgeId":      vaultEntry.CourierBadgeID,
				"SealBarcode":         vaultEntry.SealBarcode,
				"CreatedAt":           spanner.CommitTimestamp,
			}))
		}

		// 3. Merchant daily cash exposure (if present)
		if amlExposure != nil {
			muts = append(muts, spanner.InsertOrUpdateMap("MerchantDailyCashExposure", map[string]any{
				"LegalEntityTIN":       amlExposure.LegalEntityTIN,
				"SupplierId":           amlExposure.SupplierID,
				"CalendarDate":         amlExposure.CalendarDate,
				"AccumulatedCashMinor": amlExposure.AccumulatedCashMinor,
				"UpdatedAt":            spanner.CommitTimestamp,
			}))
		}

		// 4. Outbox event (Invariant 8)
		buf := &spannerOutboxBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateControlTower, sess.SessionID, events.TopicMain, map[string]any{
			"type":               "CASHIER_SESSION_TRANSITION",
			"session_id":         sess.SessionID,
			"supplier_id":        sess.SupplierID,
			"depot_id":           sess.DepotID,
			"driver_id":          sess.DriverID,
			"status":             sess.Status,
			"counted_cash_minor": sess.CountedCashMinor,
		}); emitErr != nil {
			return emitErr
		}
		muts = append(muts, toOutboxMutations(buf.events)...)

		if len(muts) > MaxTransactionMutations {
			return ErrTransactionMutationLimitExceeded
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// GetCashierSession reads a CashierBlindCountSession by session ID from Cloud Spanner.
func (r *SpannerRepository) GetCashierSession(ctx context.Context, sessionID string) (*CashierBlindCountSession, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, SupplierId, DepotId, DriverId, RouteId, BagSealBarcode, AttemptNumber,
		             DenominationBreakdownJson, CountedCashMinor, ExpectedCashMinor, VarianceMinor, Status,
		             ClearanceTokenHash, CreatedAt, ResolvedAt
		      FROM CashierBlindCountSessions WHERE SessionId = @sessionID`,
		Params: map[string]any{"sessionID": sessionID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	var sess CashierBlindCountSession
	var denomNJ spanner.NullJSON
	if err := row.Columns(
		&sess.SessionID, &sess.SupplierID, &sess.DepotID, &sess.DriverID, &sess.RouteID, &sess.BagSealBarcode, &sess.AttemptNumber,
		&denomNJ, &sess.CountedCashMinor, &sess.ExpectedCashMinor, &sess.VarianceMinor, &sess.Status,
		&sess.ClearanceTokenHash, &sess.CreatedAt, &sess.ResolvedAt,
	); err != nil {
		return nil, err
	}
	sess.DenominationBreakdownJson = nullJSONToString(denomNJ)
	if sess.DenominationBreakdownJson != "" {
		_ = json.Unmarshal([]byte(sess.DenominationBreakdownJson), &sess.DenominationsUZS)
	}
	return &sess, nil
}

// PersistDVIRInspection writes a FleetDVIRInspection to Cloud Spanner.
func (r *SpannerRepository) PersistDVIRInspection(ctx context.Context, insp *FleetDVIRInspection) error {
	if r == nil || r.client == nil || insp == nil {
		return nil
	}
	if err := insp.PrepareForSpanner(); err != nil {
		return err
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("FleetDVIRInspections", map[string]any{
			"InspectionId":           insp.InspectionID,
			"SupplierId":             insp.SupplierID,
			"VehicleId":              insp.VehicleID,
			"DriverId":               insp.DriverID,
			"InspectionType":         insp.InspectionType,
			"ChecklistJson":          toNullJSON(insp.ChecklistJson),
			"CriticalDefectFound":    insp.CriticalDefectFound,
			"RedTagApplied":          insp.RedTagApplied,
			"OdometerKm":             insp.OdometerKm,
			"ReeferTempCelsius":      insp.ReeferTempCelsius,
			"Status":                 insp.Status,
			"MaintenanceWorkOrderId": insp.MaintenanceWorkOrderID,
			"CreatedAt":              spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// PersistFleetDVIRRedTag atomically commits a Fleet DVIR Red-Tag inspection, vehicle grounding
// state update in OntologyObjects, audit KineticAction execution, and OutboxEvent row (SO-1, Invariant 8).
func (r *SpannerRepository) PersistFleetDVIRRedTag(
	ctx context.Context,
	insp *FleetDVIRInspection,
	vehicleObj *OntologyObject,
	actionRec *ActionExecutionRecord,
) error {
	if r == nil || r.client == nil || insp == nil {
		return nil
	}
	if err := insp.PrepareForSpanner(); err != nil {
		return err
	}

	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		muts := make([]*spanner.Mutation, 0, 4)

		// 1. Fleet DVIR inspection
		muts = append(muts, spanner.InsertOrUpdateMap("FleetDVIRInspections", map[string]any{
			"InspectionId":           insp.InspectionID,
			"SupplierId":             insp.SupplierID,
			"VehicleId":              insp.VehicleID,
			"DriverId":               insp.DriverID,
			"InspectionType":         insp.InspectionType,
			"ChecklistJson":          toNullJSON(insp.ChecklistJson),
			"CriticalDefectFound":    insp.CriticalDefectFound,
			"RedTagApplied":          insp.RedTagApplied,
			"OdometerKm":             insp.OdometerKm,
			"ReeferTempCelsius":      insp.ReeferTempCelsius,
			"Status":                 insp.Status,
			"MaintenanceWorkOrderId": insp.MaintenanceWorkOrderID,
			"CreatedAt":              spanner.CommitTimestamp,
		}))

		// 2. Vehicle ontology object (if present)
		if vehicleObj != nil {
			if err := vehicleObj.PrepareForSpanner(); err != nil {
				return err
			}
			markings := vehicleObj.SecurityMarkings
			if markings == nil {
				markings = []string{}
			}
			var validFrom any = spanner.CommitTimestamp
			if !vehicleObj.ValidFrom.IsZero() {
				validFrom = vehicleObj.ValidFrom
			}
			muts = append(muts, spanner.InsertOrUpdateMap("OntologyObjects", map[string]any{
				"CellId":           vehicleObj.CellID,
				"ObjectType":       vehicleObj.ObjectType,
				"ObjectId":         vehicleObj.ObjectID,
				"SupplierId":       vehicleObj.SupplierID,
				"Version":          vehicleObj.Version,
				"PropertiesJson":   toNullJSON(vehicleObj.PropertiesJson),
				"SecurityMarkings": markings,
				"ValidFrom":        validFrom,
				"UpdatedAt":        spanner.CommitTimestamp,
			}))
		}

		// 3. Kinetic action execution record (if present)
		if actionRec != nil {
			if err := actionRec.PrepareForSpanner(); err != nil {
				return err
			}
			muts = append(muts, spanner.InsertOrUpdateMap("KineticActionExecutions", map[string]any{
				"ActionExecutionId":      actionRec.ActionExecutionID,
				"CellId":                 actionRec.CellID,
				"SupplierId":             actionRec.SupplierID,
				"ActionType":             actionRec.ActionType,
				"ActorId":                actionRec.ActorID,
				"ActorRole":              actionRec.ActorRole,
				"ScenarioId":             actionRec.ScenarioID,
				"Status":                 actionRec.Status,
				"PreconditionReportJson": toNullJSON(actionRec.PreconditionReportJson),
				"DiffPayloadJson":        toNullJSON(actionRec.DiffPayloadJson),
				"IdempotencyKey":         actionRec.IdempotencyKey,
				"ExecutedAt":             spanner.CommitTimestamp,
			}))
		}

		// 4. Outbox event (Invariant 8)
		buf := &spannerOutboxBuffer{}
		if emitErr := outbox.EmitJSON(ctx, buf, events.AggregateControlTower, insp.InspectionID, events.TopicMain, map[string]any{
			"type":                  "FLEET_DVIR_RED_TAG",
			"inspection_id":         insp.InspectionID,
			"supplier_id":           insp.SupplierID,
			"vehicle_id":            insp.VehicleID,
			"status":                insp.Status,
			"critical_defect_found": insp.CriticalDefectFound,
			"red_tag_applied":       insp.RedTagApplied,
		}); emitErr != nil {
			return emitErr
		}
		muts = append(muts, toOutboxMutations(buf.events)...)

		if len(muts) > MaxTransactionMutations {
			return ErrTransactionMutationLimitExceeded
		}
		return txn.BufferWrite(muts)
	})
	return err
}

// GetLatestDVIRForVehicle reads the most recent DVIR inspection for a vehicle.
func (r *SpannerRepository) GetLatestDVIRForVehicle(ctx context.Context, vehicleID string) (*FleetDVIRInspection, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT InspectionId, SupplierId, VehicleId, DriverId, InspectionType, ChecklistJson,
		             CriticalDefectFound, RedTagApplied, OdometerKm, ReeferTempCelsius, Status,
		             MaintenanceWorkOrderId, CreatedAt
		      FROM FleetDVIRInspections WHERE VehicleId = @vehicleID ORDER BY CreatedAt DESC LIMIT 1`,
		Params: map[string]any{"vehicleID": vehicleID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	var insp FleetDVIRInspection
	var checkNJ spanner.NullJSON
	if err := row.Columns(
		&insp.InspectionID, &insp.SupplierID, &insp.VehicleID, &insp.DriverID, &insp.InspectionType, &checkNJ,
		&insp.CriticalDefectFound, &insp.RedTagApplied, &insp.OdometerKm, &insp.ReeferTempCelsius, &insp.Status,
		&insp.MaintenanceWorkOrderID, &insp.CreatedAt,
	); err != nil {
		return nil, err
	}
	insp.ChecklistJson = nullJSONToString(checkNJ)
	if insp.ChecklistJson != "" {
		_ = json.Unmarshal([]byte(insp.ChecklistJson), &insp.Checklist)
	}
	return &insp, nil
}

// PersistAMLExposure writes a MerchantDailyCashExposure record to Cloud Spanner.
func (r *SpannerRepository) PersistAMLExposure(ctx context.Context, legalEntityTIN, supplierID, calendarDate string, accumulatedMinor int64) error {
	if r == nil || r.client == nil {
		return nil
	}
	parsedDate, err := civil.ParseDate(calendarDate)
	if err != nil {
		parsedDate = civil.DateOf(time.Now().UTC())
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("MerchantDailyCashExposure", map[string]any{
			"LegalEntityTIN":       legalEntityTIN,
			"SupplierId":           supplierID,
			"CalendarDate":         parsedDate,
			"AccumulatedCashMinor": accumulatedMinor,
			"UpdatedAt":            spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// GetAMLExposure reads the accumulated daily AML cash exposure for a legal entity TIN.
func (r *SpannerRepository) GetAMLExposure(ctx context.Context, legalEntityTIN, supplierID string, date civil.Date) (*MerchantDailyCashExposure, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT LegalEntityTIN, SupplierId, CalendarDate, AccumulatedCashMinor, UpdatedAt
		      FROM MerchantDailyCashExposure WHERE LegalEntityTIN = @tin AND SupplierId = @supplierID AND CalendarDate = @date`,
		Params: map[string]any{
			"tin":        legalEntityTIN,
			"supplierID": supplierID,
			"date":       date,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	var exp MerchantDailyCashExposure
	if err := row.Columns(&exp.LegalEntityTIN, &exp.SupplierID, &exp.CalendarDate, &exp.AccumulatedCashMinor, &exp.UpdatedAt); err != nil {
		return nil, err
	}
	return &exp, nil
}

// PersistGateCheckEvent writes a YardGateCheckEvent to Cloud Spanner.
func (r *SpannerRepository) PersistGateCheckEvent(ctx context.Context, evt *YardGateCheckEvent) error {
	if r == nil || r.client == nil || evt == nil {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("YardGateCheckEvents", map[string]any{
			"GateEventId":     evt.GateEventID,
			"DepotId":         evt.DepotID,
			"SupplierId":      evt.SupplierID,
			"VehicleId":       evt.VehicleID,
			"LicensePlate":    evt.LicensePlate,
			"Direction":       evt.Direction,
			"BoltSealBarcode": evt.BoltSealBarcode,
			"SealIntact":      evt.SealIntact,
			"AssignedBaySlot": evt.AssignedBaySlot,
			"CreatedAt":       spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// PersistReturnableContainerLedger updates the balance of returnable packaging for an account.
func (r *SpannerRepository) PersistReturnableContainerLedger(ctx context.Context, entry *ReturnableContainerLedger) error {
	if r == nil || r.client == nil || entry == nil {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("ReturnableContainerLedger", map[string]any{
			"AccountId":          entry.AccountID,
			"ContainerSku":       entry.ContainerSKU,
			"SupplierId":         entry.SupplierID,
			"HolderType":         entry.HolderType,
			"BalanceQty":         entry.BalanceQty,
			"DepositLockedMinor": entry.DepositLockedMinor,
			"UpdatedAt":          spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// GetReturnableContainerBalance reads the perpetual balance of a returnable container for an account.
func (r *SpannerRepository) GetReturnableContainerBalance(ctx context.Context, accountID, containerSKU string) (*ReturnableContainerLedger, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT AccountId, ContainerSku, SupplierId, HolderType, BalanceQty, DepositLockedMinor, UpdatedAt
		      FROM ReturnableContainerLedger WHERE AccountId = @accountID AND ContainerSku = @containerSKU`,
		Params: map[string]any{
			"accountID":    accountID,
			"containerSKU": containerSKU,
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	var entry ReturnableContainerLedger
	if err := row.Columns(
		&entry.AccountID, &entry.ContainerSKU, &entry.SupplierID, &entry.HolderType,
		&entry.BalanceQty, &entry.DepositLockedMinor, &entry.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &entry, nil
}

// PersistOfflineQuotaLease stores an active or expired driver offline quota lease in Cloud Spanner.
func (r *SpannerRepository) PersistOfflineQuotaLease(ctx context.Context, lease *OfflineQuotaLease) error {
	if r == nil || r.client == nil || lease == nil {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("OfflineQuotaLeases", map[string]any{
			"LeaseId":               lease.LeaseID,
			"OrderId":               lease.OrderID,
			"DriverId":              lease.DriverID,
			"SupplierId":            lease.SupplierID,
			"LegalEntityTIN":        lease.LegalEntityTIN,
			"CalendarDate":          lease.CalendarDate,
			"MaxCashAllowedMinor":   lease.MaxCashAllowedMinor,
			"MaxCreditAllowedMinor": lease.MaxCreditAllowedMinor,
			"ExpiresAt":             lease.ExpiresAt,
			"Status":                lease.Status,
			"CreatedAt":             spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	return err
}

// GetOfflineQuotaLease reads an offline quota lease by lease ID from Cloud Spanner.
func (r *SpannerRepository) GetOfflineQuotaLease(ctx context.Context, leaseID string) (*OfflineQuotaLease, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT LeaseId, OrderId, DriverId, SupplierId, LegalEntityTIN, CalendarDate,
		             MaxCashAllowedMinor, MaxCreditAllowedMinor, ExpiresAt, Status, CreatedAt
		      FROM OfflineQuotaLeases WHERE LeaseId = @leaseID`,
		Params: map[string]any{"leaseID": leaseID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	var lease OfflineQuotaLease
	if err := row.Columns(
		&lease.LeaseID, &lease.OrderID, &lease.DriverID, &lease.SupplierID, &lease.LegalEntityTIN, &lease.CalendarDate,
		&lease.MaxCashAllowedMinor, &lease.MaxCreditAllowedMinor, &lease.ExpiresAt, &lease.Status, &lease.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &lease, nil
}
