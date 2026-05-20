package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/kafka"
	"backend-go/optimizationjobs"
	"backend-go/outbox"
	"backend-go/proximity"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	contract "optimizercontract"

	"github.com/google/uuid"
)

type autoDispatchQueuedResponse struct {
	Queued            bool   `json:"queued"`
	JobID             string `json:"job_id"`
	Status            string `json:"status"`
	SnapshotTimestamp string `json:"snapshot_timestamp"`
}

func queueAutoDispatchJob(ctx context.Context, client *spanner.Client, readRouter proximity.ReadRouter, supplierID string, filterOrderIDs []string, excludedTruckIDs []string, idempotencyKey string) (*autoDispatchQueuedResponse, *AutoDispatchResult, error) {
	snapshotAt := time.Now().UTC()

	orders, err := fetchDispatchableOrdersFresh(ctx, client, readRouter, supplierID, filterOrderIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch orders: %w", err)
	}
	if len(orders) == 0 {
		return nil, &AutoDispatchResult{
			SnapshotTimestamp: snapshotAt.Format(time.RFC3339),
			Manifests:         []TruckManifest{},
			Orphans:           []OrphanOrder{},
		}, nil
	}

	drivers, err := fetchAvailableDriversFresh(ctx, client, supplierID)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch drivers: %w", err)
	}
	if len(excludedTruckIDs) > 0 {
		excludeSet := make(map[string]bool, len(excludedTruckIDs))
		for _, id := range excludedTruckIDs {
			excludeSet[id] = true
		}
		filtered := drivers[:0]
		for _, driver := range drivers {
			if !excludeSet[driver.DriverID] {
				filtered = append(filtered, driver)
			}
		}
		drivers = filtered
	}

	jobID := uuid.NewString()
	envelope, err := buildAutoDispatchEnvelope(ctx, client, supplierID, jobID, idempotencyKey, snapshotAt, orders, drivers)
	if err != nil {
		return nil, nil, fmt.Errorf("build optimization envelope: %w", err)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal optimization envelope: %w", err)
	}

	job, err := optimizationjobs.New(optimizationjobs.CreateParams{
		JobID:           jobID,
		SupplierID:      supplierID,
		JobType:         optimizationjobs.JobTypeAutoDispatch,
		SolverType:      optimizationjobs.SolverTypeVRP,
		TraceID:         telemetry.TraceIDFromContext(ctx),
		IdempotencyKey:  idempotencyKey,
		SourceEventType: kafka.EventOptimizationQueued,
		Payload:         payload,
		RequestedAt:     snapshotAt,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("build optimization job: %w", err)
	}

	resolvedJobID, resolvedStatus, err := enqueueOptimizationJob(ctx, client, job, envelope)
	if err != nil {
		return nil, nil, err
	}
	if isActiveOptimizationJobStatus(resolvedStatus) {
		if err := cache.MarkSupplierOptimizationJobActive(ctx, supplierID, resolvedJobID); err != nil {
			slog.WarnContext(ctx, "optimization job active index add failed", "supplier_id", supplierID, "job_id", resolvedJobID, "err", err)
		}
	}

	return &autoDispatchQueuedResponse{
		Queued:            true,
		JobID:             resolvedJobID,
		Status:            string(resolvedStatus),
		SnapshotTimestamp: snapshotAt.Format(time.RFC3339Nano),
	}, nil, nil
}

func buildAutoDispatchEnvelope(ctx context.Context, client *spanner.Client, supplierID string, jobID string, idempotencyKey string, snapshotAt time.Time, orders []dispatchableOrder, drivers []availableDriver) (contract.OptimizationJobEnvelope, error) {
	targetCells, err := resolveDispatchTargetCells(ctx, client, supplierID, auth.EffectiveWarehouseID(ctx))
	if err != nil {
		return contract.OptimizationJobEnvelope{}, fmt.Errorf("resolve target h3 cells: %w", err)
	}

	targetCells = append([]string(nil), targetCells...)
	sort.Strings(targetCells)

	depotNodeUUID, originLat, originLng := resolveDispatchQueueOrigin(ctx, client, supplierID, orders)
	vrp := buildVRPJobPayload(depotNodeUUID, originLat, originLng, orders, drivers)

	return contract.OptimizationJobEnvelope{
		V:                 contract.V,
		JobID:             jobID,
		SupplierID:        supplierID,
		JobType:           contract.OptimizationJobTypeAutoDispatch,
		SolverType:        contract.OptimizationSolverTypeVRP,
		TraceID:           telemetry.TraceIDFromContext(ctx),
		IdempotencyKey:    idempotencyKey,
		SourceEventType:   kafka.EventOptimizationQueued,
		TargetH3Cells:     targetCells,
		MatrixSize:        int32(len(vrp.DistanceMatrixKM)),
		DispatchTimestamp: snapshotAt.Format(time.RFC3339Nano),
		Status:            contract.OptimizationJobStatusQueued,
		VRP:               &vrp,
	}, nil
}

func enqueueOptimizationJob(ctx context.Context, client *spanner.Client, job optimizationjobs.Job, envelope contract.OptimizationJobEnvelope) (string, contract.OptimizationJobStatus, error) {
	resolvedJobID := job.JobID
	resolvedStatus := contract.OptimizationJobStatus(job.Status)
	traceID := telemetry.TraceIDFromContext(ctx)

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if job.IdempotencyKey != "" {
			existingJobID, existingStatus, found, err := findOptimizationJobByIdempotency(ctx, txn, job.SupplierID, job.IdempotencyKey)
			if err != nil {
				return fmt.Errorf("lookup optimization job by idempotency key: %w", err)
			}
			if found {
				resolvedJobID = existingJobID
				resolvedStatus = existingStatus
				return nil
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{optimizationjobs.InsertMutation(job)}); err != nil {
			return fmt.Errorf("insert optimization job: %w", err)
		}

		if err := outbox.EmitJSON(txn, "OptimizationJob", job.JobID, kafka.EventOptimizationQueued, kafka.TopicOptimizerJobs, envelope, traceID); err != nil {
			return fmt.Errorf("emit optimization job outbox event: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("queue optimization job %s: %w", job.JobID, err)
	}

	return resolvedJobID, resolvedStatus, nil
}

func findOptimizationJobByIdempotency(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string, idempotencyKey string) (string, contract.OptimizationJobStatus, bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT JobId, Status
			FROM OptimizationJobs@{FORCE_INDEX=Idx_OptimizationJobs_BySupplierIdempotency}
			WHERE SupplierId = @supplierId
			  AND IdempotencyKey = @idempotencyKey
			ORDER BY RequestedAt DESC
			LIMIT 1`,
		Params: map[string]interface{}{
			"supplierId":     supplierID,
			"idempotencyKey": idempotencyKey,
		},
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}

	var jobID string
	var status string
	if err := row.Columns(&jobID, &status); err != nil {
		return "", "", false, err
	}

	return jobID, contract.OptimizationJobStatus(status), true, nil
}

func resolveDispatchQueueOrigin(ctx context.Context, client *spanner.Client, supplierID string, orders []dispatchableOrder) (string, float64, float64) {
	warehouseID := auth.EffectiveWarehouseID(ctx)
	if lat, lng, ok := fetchWarehouseOrigin(ctx, client, warehouseID); ok {
		return "warehouse:" + warehouseID, lat, lng
	}

	if len(orders) == 0 {
		return "supplier:" + supplierID + ":dispatch-depot", 0, 0
	}

	var sumLat float64
	var sumLng float64
	for _, order := range orders {
		sumLat += order.Lat
		sumLng += order.Lng
	}

	count := float64(len(orders))
	return "supplier:" + supplierID + ":dispatch-depot", sumLat / count, sumLng / count
}

func buildVRPJobPayload(depotNodeUUID string, originLat float64, originLng float64, orders []dispatchableOrder, drivers []availableDriver) contract.VRPJobPayload {
	coordinates := make([]dispatchCoordinate, 0, len(orders)+1)
	coordinates = append(coordinates, dispatchCoordinate{Lat: originLat, Lng: originLng})

	dropOffNodeUUIDs := make([]string, 0, len(orders))
	nodes := make([]contract.VRPNodeProjectionPayload, 0, len(orders))
	nodeDemands := make([]contract.VRPNodeDemandPayload, 0, len(orders))
	nodeTimeWindows := make([]contract.VRPNodeTimeWindowPayload, 0, len(orders))
	for _, order := range orders {
		coordinates = append(coordinates, dispatchCoordinate{Lat: order.Lat, Lng: order.Lng})
		dropOffNodeUUIDs = append(dropOffNodeUUIDs, order.OrderID)
		nodes = append(nodes, contract.VRPNodeProjectionPayload{
			NodeUUID:             order.OrderID,
			OrderID:              order.OrderID,
			RetailerID:           order.RetailerID,
			RetailerName:         order.RetailerName,
			Lat:                  order.Lat,
			Lng:                  order.Lng,
			Amount:               order.Amount,
			DemandVU:             order.VolumeVU,
			ReceivingWindowOpen:  order.ReceivingWindowOpen,
			ReceivingWindowClose: order.ReceivingWindowClose,
		})
		nodeDemands = append(nodeDemands, contract.VRPNodeDemandPayload{
			NodeUUID: order.OrderID,
			DemandVU: order.VolumeVU,
		})
		startWindowHours, endWindowHours := resolveWindowHours(order.ReceivingWindowOpen, order.ReceivingWindowClose)
		nodeTimeWindows = append(nodeTimeWindows, contract.VRPNodeTimeWindowPayload{
			NodeUUID:         order.OrderID,
			StartWindowHours: startWindowHours,
			EndWindowHours:   endWindowHours,
		})
	}

	vehicles := make([]contract.VRPVehiclePayload, 0, len(drivers))
	for _, driver := range drivers {
		vehicles = append(vehicles, contract.VRPVehiclePayload{
			VehicleUUID:      driver.VehicleID,
			DriverUUID:       driver.DriverID,
			DriverName:       driver.Name,
			VehicleType:      driver.VehicleType,
			VehicleClass:     driver.VehicleClass,
			CapacityVU:       driver.MaxVolumeVU,
			StartWindowHours: 0,
			EndWindowHours:   24,
		})
	}

	return contract.VRPJobPayload{
		DepotNodeUUID: depotNodeUUID,
		Depot: &contract.VRPDepotPayload{
			NodeUUID: depotNodeUUID,
			Label:    resolveDepotLabel(depotNodeUUID),
			Lat:      originLat,
			Lng:      originLng,
		},
		DropOffNodeUUIDs: dropOffNodeUUIDs,
		DistanceMatrixKM: buildDistanceMatrixKM(coordinates),
		Vehicles:         vehicles,
		Nodes:            nodes,
		NodeDemands:      nodeDemands,
		NodeTimeWindows:  nodeTimeWindows,
	}
}

func isActiveOptimizationJobStatus(status contract.OptimizationJobStatus) bool {
	switch status {
	case contract.OptimizationJobStatusQueued, contract.OptimizationJobStatusPublished, contract.OptimizationJobStatusRunning, contract.OptimizationJobStatusApplying:
		return true
	default:
		return false
	}
}

func resolveDepotLabel(nodeUUID string) string {
	switch {
	case strings.HasPrefix(nodeUUID, "warehouse:"):
		return "Warehouse origin"
	case strings.HasPrefix(nodeUUID, "supplier:"):
		return "Dispatch centroid"
	default:
		return nodeUUID
	}
}

type dispatchCoordinate struct {
	Lat float64
	Lng float64
}

func buildDistanceMatrixKM(coordinates []dispatchCoordinate) [][]float64 {
	matrix := make([][]float64, len(coordinates))
	for rowIdx := range coordinates {
		matrix[rowIdx] = make([]float64, len(coordinates))
		for colIdx := range coordinates {
			if rowIdx == colIdx {
				continue
			}
			matrix[rowIdx][colIdx] = haversineKM(coordinates[rowIdx], coordinates[colIdx])
		}
	}
	return matrix
}

func haversineKM(origin dispatchCoordinate, destination dispatchCoordinate) float64 {
	const earthRadiusKM = 6371.0

	toRadians := func(value float64) float64 {
		return value * math.Pi / 180.0
	}

	dLat := toRadians(destination.Lat - origin.Lat)
	dLng := toRadians(destination.Lng - origin.Lng)
	lat1 := toRadians(origin.Lat)
	lat2 := toRadians(destination.Lat)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKM * c
}

func resolveWindowHours(open string, close string) (float64, float64) {
	startMinutes := parseTimeHHMM(open)
	endMinutes := parseTimeHHMM(close)

	if startMinutes < 0 {
		startMinutes = 0
	}
	if endMinutes <= startMinutes {
		endMinutes = 24 * 60
	}

	return float64(startMinutes) / 60.0, float64(endMinutes) / 60.0
}
