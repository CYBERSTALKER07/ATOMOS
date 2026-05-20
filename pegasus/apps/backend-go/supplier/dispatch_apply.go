package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"backend-go/cache"
	"backend-go/kafka"
	"backend-go/kafka/workerpool"
	"backend-go/optimizationjobs"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	goKafka "github.com/segmentio/kafka-go"

	contract "optimizercontract"
)

const (
	dispatchApplyMaxAttempts   = 3
	dispatchApplyRetryBase     = 200 * time.Millisecond
	dispatchApplyFailureCode   = "APPLY_FAILED"
	dispatchJobRouteNamespace  = "optimization-route"
	dispatchJobManifestNS      = "optimization-manifest"
	dispatchJobWarehousePrefix = "warehouse:"
)

type dispatchJobVRPData struct {
	envelope    contract.OptimizationJobEnvelope
	solved      contract.OptimizationSolvedEvent
	scaleFactor float64
}

type dispatchJobRoutePlan struct {
	ManifestID      string
	RouteID         string
	DriverID        string
	DriverName      string
	VehicleUUID     string
	VehicleType     string
	VehicleClass    string
	CapacityVU      float64
	LoadVU          float64
	RouteCostKM     float64
	GeoZone         string
	Stops           []dispatchProjectionStop
	Orders          []DispatchOrder
	LoadingManifest []LoadingManifestEntry
}

func decodeDispatchJobVRP(job optimizationjobs.Job) (*dispatchJobVRPData, error) {
	if job.SolverType != optimizationjobs.SolverTypeVRP || len(job.Payload) == 0 || len(job.ResultPayload) == 0 {
		return nil, nil
	}

	var envelope contract.OptimizationJobEnvelope
	if err := json.Unmarshal(job.Payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode optimization job envelope: %w", err)
	}
	var solved contract.OptimizationSolvedEvent
	if err := json.Unmarshal(job.ResultPayload, &solved); err != nil {
		return nil, fmt.Errorf("decode optimization job result payload: %w", err)
	}
	if envelope.VRP == nil || solved.VRP == nil {
		return nil, nil
	}

	scaleFactor := float64(solved.VRP.Meta.ScaleFactor)
	if scaleFactor <= 0 {
		scaleFactor = 1
	}

	return &dispatchJobVRPData{
		envelope:    envelope,
		solved:      solved,
		scaleFactor: scaleFactor,
	}, nil
}

func buildDispatchJobRoutePlans(job optimizationjobs.Job, data *dispatchJobVRPData) []dispatchJobRoutePlan {
	if data == nil || data.envelope.VRP == nil || data.solved.VRP == nil {
		return []dispatchJobRoutePlan{}
	}

	nodesByID := make(map[string]contract.VRPNodeProjectionPayload, len(data.envelope.VRP.Nodes))
	for _, node := range data.envelope.VRP.Nodes {
		nodesByID[node.NodeUUID] = node
	}
	vehiclesByID := make(map[string]contract.VRPVehiclePayload, len(data.envelope.VRP.Vehicles))
	for _, vehicle := range data.envelope.VRP.Vehicles {
		vehiclesByID[vehicle.VehicleUUID] = vehicle
	}

	plans := make([]dispatchJobRoutePlan, 0, len(data.solved.VRP.Routes))
	for routeIndex, route := range data.solved.VRP.Routes {
		vehicle := vehiclesByID[route.VehicleUUID]
		driverID := route.DriverUUID
		if driverID == "" {
			driverID = vehicle.DriverUUID
		}

		geoOrders := make([]GeoOrder, 0, len(route.OrderedNodeUUIDs))
		orders := make([]DispatchOrder, 0, len(route.OrderedNodeUUIDs))
		stops := make([]dispatchProjectionStop, 0, len(route.OrderedNodeUUIDs))
		for stopIndex, nodeUUID := range route.OrderedNodeUUIDs {
			node := nodesByID[nodeUUID]
			stop := buildProjectionStop(node, nodeUUID, stopIndex+1)
			stops = append(stops, stop)
			geoOrders = append(geoOrders, GeoOrder{
				OrderID:              stop.OrderID,
				RetailerID:           stop.RetailerID,
				RetailerName:         stop.RetailerName,
				Amount:               stop.Amount,
				Lat:                  stop.Lat,
				Lng:                  stop.Lng,
				Volume:               stop.DemandVU,
				ReceivingWindowOpen:  stop.ReceivingWindowOpen,
				ReceivingWindowClose: stop.ReceivingWindowClose,
			})
			orders = append(orders, DispatchOrder{
				OrderID:              stop.OrderID,
				RetailerID:           stop.RetailerID,
				RetailerName:         stop.RetailerName,
				Amount:               stop.Amount,
				VolumeVU:             stop.DemandVU,
				Lat:                  stop.Lat,
				Lng:                  stop.Lng,
				ReceivingWindowOpen:  stop.ReceivingWindowOpen,
				ReceivingWindowClose: stop.ReceivingWindowClose,
			})
		}

		routeID := dispatchJobRouteID(job.JobID, driverID, route.VehicleUUID, routeIndex)
		plans = append(plans, dispatchJobRoutePlan{
			ManifestID:      dispatchJobManifestID(routeID),
			RouteID:         routeID,
			DriverID:        driverID,
			DriverName:      vehicle.DriverName,
			VehicleUUID:     route.VehicleUUID,
			VehicleType:     vehicle.VehicleType,
			VehicleClass:    vehicle.VehicleClass,
			CapacityVU:      vehicle.CapacityVU,
			LoadVU:          float64(route.LoadScaled) / data.scaleFactor,
			RouteCostKM:     float64(route.RouteCostScaled) / data.scaleFactor,
			GeoZone:         routeGeoZone(geoOrders),
			Stops:           stops,
			Orders:          orders,
			LoadingManifest: buildDispatchLoadingManifest(geoOrders),
		})
	}

	return plans
}

func buildDispatchLoadingManifest(geoOrders []GeoOrder) []LoadingManifestEntry {
	n := len(geoOrders)
	loadingManifest := make([]LoadingManifestEntry, n)
	for idx, order := range geoOrders {
		sequence := n - idx
		instruction := fmt.Sprintf("Load position %d of %d", sequence, n)
		if sequence == 1 {
			instruction = "Load first — Back of Truck"
		} else if sequence == n {
			instruction = "Load last — By the Doors"
		}
		loadingManifest[sequence-1] = LoadingManifestEntry{
			LoadSequence: sequence,
			OrderID:      order.OrderID,
			RetailerName: order.RetailerName,
			VolumeVU:     order.Volume,
			Lat:          order.Lat,
			Lng:          order.Lng,
			Instruction:  instruction,
		}
	}
	return loadingManifest
}

func dispatchJobWarehouseID(data *dispatchJobVRPData) string {
	if data == nil || data.envelope.VRP == nil {
		return ""
	}

	depotNodeUUID := data.envelope.VRP.DepotNodeUUID
	if depotNodeUUID == "" && data.envelope.VRP.Depot != nil {
		depotNodeUUID = data.envelope.VRP.Depot.NodeUUID
	}
	if !strings.HasPrefix(depotNodeUUID, dispatchJobWarehousePrefix) {
		return ""
	}
	return strings.TrimPrefix(depotNodeUUID, dispatchJobWarehousePrefix)
}

func dispatchJobRouteID(jobID string, driverID string, vehicleID string, routeIndex int) string {
	seed := strings.Join([]string{dispatchJobRouteNamespace, jobID, driverID, vehicleID, strconv.Itoa(routeIndex)}, ":")
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(seed)).String()
}

func dispatchJobManifestID(routeID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(dispatchJobManifestNS+":"+routeID)).String()
}

func StartOptimizationApplyConsumer(ctx context.Context, spannerClient *spanner.Client, manifestSvc *ManifestService, brokerAddress string) {
	if spannerClient == nil || manifestSvc == nil {
		slog.ErrorContext(ctx, "optimization apply consumer missing dependencies")
		return
	}

	reader := goKafka.NewReader(goKafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    kafka.TopicMain,
		GroupID:  "pegasus-optimization-apply-group",
		MinBytes: 1,
		MaxBytes: 10 << 20,
	})

	pool, err := workerpool.New(workerpool.Config{
		Source: reader,
		Name:   "optimization-apply",
		Logger: slog.Default(),
		Handler: func(ctx context.Context, m goKafka.Message) error {
			if kafka.EventType(m.Headers, m.Key) != kafka.EventOptimizationSolved {
				return nil
			}

			var lastErr error
			for attempt := 1; attempt <= dispatchApplyMaxAttempts; attempt++ {
				if err := handleOptimizationSolvedApply(ctx, spannerClient, manifestSvc, m.Value); err == nil {
					return nil
				} else {
					lastErr = err
				}
				if attempt == dispatchApplyMaxAttempts {
					break
				}
				if err := sleepDispatchApplyBackoff(ctx, attempt); err != nil {
					return fmt.Errorf("optimization apply backoff interrupted: %w", err)
				}
			}
			return fmt.Errorf("optimization apply retries exhausted: %w", lastErr)
		},
		OnFailure: func(ctx context.Context, m goKafka.Message, handlerErr error) {
			event, ok := parseOptimizationSolvedEvent(m.Value)
			if !ok {
				slog.ErrorContext(ctx, "optimization apply consumer failed to decode terminal event", "err", handlerErr, "partition", m.Partition, "offset", m.Offset)
				return
			}
			if err := markDispatchJobApplyFailed(ctx, spannerClient, event.SupplierID, event.JobID, handlerErr); err != nil {
				slog.ErrorContext(ctx, "optimization apply failure persistence failed", "job_id", event.JobID, "supplier_id", event.SupplierID, "err", err)
			}
		},
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to init optimization apply pool", "err", err)
		return
	}

	go func() {
		defer reader.Close()
		if err := pool.Run(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "optimization apply consumer exited", "err", err)
		}
	}()
}

func handleOptimizationSolvedApply(ctx context.Context, spannerClient *spanner.Client, manifestSvc *ManifestService, value []byte) error {
	event, ok := parseOptimizationSolvedEvent(value)
	if !ok {
		return fmt.Errorf("decode optimization solved event")
	}
	if event.TraceID != "" {
		ctx = telemetry.WithTraceID(ctx, event.TraceID)
	}
	if event.JobID == "" || event.SupplierID == "" {
		return nil
	}
	return applySolvedDispatchJob(ctx, spannerClient, manifestSvc, event.SupplierID, event.JobID)
}

func parseOptimizationSolvedEvent(value []byte) (contract.OptimizationSolvedEvent, bool) {
	var event contract.OptimizationSolvedEvent
	if err := json.Unmarshal(value, &event); err != nil {
		return contract.OptimizationSolvedEvent{}, false
	}
	return event, true
}

func applySolvedDispatchJob(ctx context.Context, spannerClient *spanner.Client, manifestSvc *ManifestService, supplierID string, jobID string) error {
	shouldContinue, err := ensureDispatchJobApplying(ctx, spannerClient, supplierID, jobID)
	if err != nil {
		return err
	}
	if !shouldContinue {
		return nil
	}
	if err := cache.MarkSupplierOptimizationJobActive(ctx, supplierID, jobID); err != nil {
		slog.WarnContext(ctx, "optimization apply active-set add failed", "supplier_id", supplierID, "job_id", jobID, "err", err)
	}

	job, found, err := optimizationjobs.GetByID(ctx, spannerClient, supplierID, jobID)
	if err != nil {
		return fmt.Errorf("load optimization job %s: %w", jobID, err)
	}
	if !found {
		return fmt.Errorf("optimization job %s not found for supplier %s", jobID, supplierID)
	}

	data, err := decodeDispatchJobVRP(job)
	if err != nil {
		return err
	}
	if data == nil {
		if err := markDispatchJobApplied(ctx, spannerClient, supplierID, jobID); err != nil {
			return err
		}
		if err := cache.RemoveSupplierOptimizationJobActive(ctx, supplierID, jobID); err != nil {
			slog.WarnContext(ctx, "optimization apply active-set remove failed", "supplier_id", supplierID, "job_id", jobID, "err", err)
		}
		return nil
	}

	warehouseID := dispatchJobWarehouseID(data)
	for _, plan := range buildDispatchJobRoutePlans(job, data) {
		if _, err := manifestSvc.CreateDraftManifestWithIdentity(
			ctx,
			plan.ManifestID,
			supplierID,
			warehouseID,
			plan.RouteID,
			plan.DriverID,
			plan.DriverID,
			plan.CapacityVU,
			plan.GeoZone,
			plan.Orders,
			plan.LoadingManifest,
		); err != nil {
			return fmt.Errorf("apply optimization route %s: %w", plan.RouteID, err)
		}
	}

	if err := markDispatchJobApplied(ctx, spannerClient, supplierID, jobID); err != nil {
		return err
	}
	if err := cache.RemoveSupplierOptimizationJobActive(ctx, supplierID, jobID); err != nil {
		slog.WarnContext(ctx, "optimization apply active-set remove failed", "supplier_id", supplierID, "job_id", jobID, "err", err)
	}
	return nil
}

func ensureDispatchJobApplying(ctx context.Context, spannerClient *spanner.Client, supplierID string, jobID string) (bool, error) {
	shouldContinue := false
	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "OptimizationJobs", spanner.Key{jobID}, []string{"SupplierId", "Status"})
		if err != nil {
			return fmt.Errorf("read optimization job status: %w", err)
		}
		var rowSupplierID string
		var rowStatus string
		if err := row.Columns(&rowSupplierID, &rowStatus); err != nil {
			return fmt.Errorf("decode optimization job status: %w", err)
		}
		if rowSupplierID != supplierID {
			return fmt.Errorf("optimization job %s supplier mismatch", jobID)
		}

		switch optimizationjobs.Status(rowStatus) {
		case optimizationjobs.StatusApplied, optimizationjobs.StatusCancelled, optimizationjobs.StatusFailed:
			shouldContinue = false
			return nil
		case optimizationjobs.StatusApplying:
			shouldContinue = true
			return nil
		case optimizationjobs.StatusSolved:
			shouldContinue = true
			return txn.BufferWrite([]*spanner.Mutation{spanner.Update("OptimizationJobs",
				[]string{"JobId", "Status", "FailureCode", "FailureMessage", "UpdatedAt"},
				[]interface{}{jobID, string(optimizationjobs.StatusApplying), "", "", spanner.CommitTimestamp},
			)})
		default:
			return fmt.Errorf("optimization job %s is %s, expected SOLVED/APPLYING", jobID, rowStatus)
		}
	})
	if err != nil {
		return false, fmt.Errorf("transition optimization job %s to APPLYING: %w", jobID, err)
	}
	return shouldContinue, nil
}

func markDispatchJobApplied(ctx context.Context, spannerClient *spanner.Client, supplierID string, jobID string) error {
	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "OptimizationJobs", spanner.Key{jobID}, []string{"SupplierId", "Status"})
		if err != nil {
			return fmt.Errorf("read optimization job before applied: %w", err)
		}
		var rowSupplierID string
		var rowStatus string
		if err := row.Columns(&rowSupplierID, &rowStatus); err != nil {
			return fmt.Errorf("decode optimization job before applied: %w", err)
		}
		if rowSupplierID != supplierID {
			return fmt.Errorf("optimization job %s supplier mismatch", jobID)
		}
		if optimizationjobs.Status(rowStatus) == optimizationjobs.StatusApplied {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update("OptimizationJobs",
			[]string{"JobId", "Status", "AppliedAt", "FailureCode", "FailureMessage", "UpdatedAt"},
			[]interface{}{jobID, string(optimizationjobs.StatusApplied), time.Now().UTC(), "", "", spanner.CommitTimestamp},
		)})
	})
	if err != nil {
		return fmt.Errorf("mark optimization job %s applied: %w", jobID, err)
	}
	return nil
}

func markDispatchJobApplyFailed(ctx context.Context, spannerClient *spanner.Client, supplierID string, jobID string, cause error) error {
	if jobID == "" || supplierID == "" {
		return nil
	}
	message := cause.Error()
	_, err := spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "OptimizationJobs", spanner.Key{jobID}, []string{"SupplierId", "Status"})
		if err != nil {
			return fmt.Errorf("read optimization job before failure: %w", err)
		}
		var rowSupplierID string
		var rowStatus string
		if err := row.Columns(&rowSupplierID, &rowStatus); err != nil {
			return fmt.Errorf("decode optimization job before failure: %w", err)
		}
		if rowSupplierID != supplierID {
			return fmt.Errorf("optimization job %s supplier mismatch", jobID)
		}
		if optimizationjobs.Status(rowStatus) == optimizationjobs.StatusApplied || optimizationjobs.Status(rowStatus) == optimizationjobs.StatusCancelled {
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{spanner.Update("OptimizationJobs",
			[]string{"JobId", "Status", "FailureCode", "FailureMessage", "UpdatedAt"},
			[]interface{}{jobID, string(optimizationjobs.StatusFailed), dispatchApplyFailureCode, message, spanner.CommitTimestamp},
		)})
	})
	if err != nil {
		return fmt.Errorf("mark optimization job %s apply failed: %w", jobID, err)
	}
	if err := cache.RemoveSupplierOptimizationJobActive(ctx, supplierID, jobID); err != nil {
		slog.WarnContext(ctx, "optimization apply active-set remove failed after failure", "supplier_id", supplierID, "job_id", jobID, "err", err)
	}
	return nil
}

func sleepDispatchApplyBackoff(ctx context.Context, attempt int) error {
	if attempt < 1 {
		attempt = 1
	}
	backoff := dispatchApplyRetryBase * time.Duration(1<<(attempt-1))
	jitter := time.Duration(rand.Int63n(int64(dispatchApplyRetryBase)))
	wait := backoff + jitter
	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
