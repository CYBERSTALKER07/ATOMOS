package dispatch

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"

	internalKafka "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"
)

// PersistInput is the input shape for PersistDraftManifests. Source is the
// attribution string from plan.OptimizeAndValidate ("optimizer",
// "fallback_phase1", "fallback_validation_rejected") so analytics and the
// admin telemetry layer can attribute the routing decision.
type PersistInput struct {
	SupplierID  string
	WarehouseID string
	Source      string
	Result      *AssignmentResult
	// FleetByDriver maps DriverID to the truck/vehicle id (Vehicles.VehicleId)
	// for that driver. Required because DispatchRoute carries DriverID only.
	FleetByDriver map[string]string
}

// PersistDraftManifests writes one SupplierTruckManifests row per route plus
// one ManifestOrders row per stop, atomically with a MANIFEST_DRAFT_CREATED
// outbox event per manifest. Returns the list of created ManifestIds in the
// same order as Result.Routes.
//
// All work happens inside a single ReadWriteTransaction — Spanner rolls back
// the whole batch (manifest rows + manifest-order rows + outbox rows) on any
// commit abort, so partial state is impossible by construction.
//
// The 20k-cell mutation cap is the only bound: roughly N(routes) +
// N(stops_total) + N(routes outbox rows). For typical dispatch waves
// (≤30 routes, ≤200 stops) we are nowhere near the limit.
func PersistDraftManifests(ctx context.Context, sc *spanner.Client, in PersistInput) ([]string, error) {
	if sc == nil {
		return nil, fmt.Errorf("dispatch.Persist: nil spanner client")
	}
	if in.Result == nil {
		return nil, fmt.Errorf("dispatch.Persist: nil result")
	}
	if in.SupplierID == "" {
		return nil, fmt.Errorf("dispatch.Persist: SupplierID is required")
	}

	manifestIDs := make([]string, len(in.Result.Routes))
	for i := range in.Result.Routes {
		manifestIDs[i] = uuid.NewString()
	}

	_, err := sc.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(in.Result.Routes)*4)
		for ri, route := range in.Result.Routes {
			manifestID := manifestIDs[ri]
			truckID := in.FleetByDriver[route.DriverID]

			mutations = append(mutations, spanner.InsertOrUpdate(
				"SupplierTruckManifests",
				[]string{
					"ManifestId", "SupplierId", "WarehouseId", "TruckId",
					"DriverId", "State", "TotalVolumeVU", "MaxVolumeVU",
					"StopCount", "CreatedAt", "UpdatedAt",
				},
				[]interface{}{
					manifestID, in.SupplierID, nullableString(in.WarehouseID), truckID,
					route.DriverID, "DRAFT", route.LoadedVolume, route.MaxVolume,
					int64(len(route.Orders)), spanner.CommitTimestamp, spanner.CommitTimestamp,
				},
			))

			for stopIdx, o := range route.Orders {
				mutations = append(mutations, spanner.InsertOrUpdate(
					"ManifestOrders",
					[]string{
						"ManifestId", "OrderId", "SequenceIndex",
						"LoadingOrder", "VolumeVU", "State",
					},
					[]interface{}{
						manifestID, o.OrderID, int64(stopIdx + 1),
						int64(len(route.Orders) - stopIdx), o.Volume, "ASSIGNED",
					},
				))
			}
		}

		if err := txn.BufferWrite(mutations); err != nil {
			return fmt.Errorf("buffer write: %w", err)
		}

		// One outbox event per manifest. Aggregate type "Manifest" keeps the
		// notification dispatcher's switch consistent with the existing
		// EventManifestRebalanced / EventManifestCancelled producers.
		now := time.Now().UTC()
		for ri, route := range in.Result.Routes {
			manifestID := manifestIDs[ri]
			payload := internalKafka.ManifestLifecycleEvent{
				ManifestID:  manifestID,
				SupplierId:  in.SupplierID,
				DriverID:    route.DriverID,
				TruckID:     in.FleetByDriver[route.DriverID],
				State:       "DRAFT",
				StopCount:   len(route.Orders),
				VolumeVU:    route.LoadedVolume,
				MaxVolumeVU: route.MaxVolume,
				Timestamp:   now,
			}
			if err := outbox.EmitJSON(
				txn, "Manifest", manifestID,
				internalKafka.EventManifestDraftCreated, internalKafka.TopicMain,
				payload,
				telemetry.TraceIDFromContext(ctx),
			); err != nil {
				return fmt.Errorf("emit MANIFEST_DRAFT_CREATED %s: %w", manifestID, err)
			}
			if err := outbox.EmitJSON(
				txn, "Manifest", manifestID,
				internalKafka.EventPayloadSync, internalKafka.TopicMain,
				internalKafka.PayloadSyncEvent{
					SupplierID:  in.SupplierID,
					WarehouseID: in.WarehouseID,
					ManifestID:  manifestID,
					Reason:      internalKafka.EventManifestDraftCreated,
					Timestamp:   now,
				},
				telemetry.TraceIDFromContext(ctx),
			); err != nil {
				return fmt.Errorf("emit PAYLOAD_SYNC %s: %w", manifestID, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return manifestIDs, nil
}

// nullableString returns spanner.NullString — needed because WarehouseId is
// nullable in the SupplierTruckManifests DDL (supplier-wide manifests).
func nullableString(s string) spanner.NullString {
	if s == "" {
		return spanner.NullString{Valid: false}
	}
	return spanner.NullString{StringVal: s, Valid: true}
}
