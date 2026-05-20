package supplier

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"backend-go/cache"
	"backend-go/optimizationjobs"

	"cloud.google.com/go/spanner"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	contract "optimizercontract"
)

const defaultSupplierDispatchJobsTestDatabase = "projects/pegasus-logistics/instances/pegasus-dev/databases/pegasus-db"

func TestListActiveDispatchJobs_RedisPrimaryAndSpannerFallback_Integration(t *testing.T) {
	if os.Getenv("RUN_SPANNER_INTEGRATION") != "1" {
		t.Skip("set RUN_SPANNER_INTEGRATION=1 to run Spanner-backed supplier dispatch job tests")
	}
	ensureSupplierDispatchJobsSpannerReachable(t)

	ctx := context.Background()
	client, err := spanner.NewClient(ctx, supplierDispatchJobsTestDatabase(), option.WithEndpoint(spannerDispatchJobsEmulatorHost()), option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("spanner.NewClient() error = %v", err)
	}
	defer client.Close()

	supplierID := "SUP-DISPATCH-JOBS-" + uuid.NewString()
	queuedJobID := "queued-" + uuid.NewString()
	runningJobID := "running-" + uuid.NewString()
	solvedJobID := "solved-" + uuid.NewString()

	seededJobIDs := []string{queuedJobID, runningJobID, solvedJobID}
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		seedDispatchJobMutation(t, supplierID, queuedJobID, optimizationjobs.StatusQueued, time.Now().UTC().Add(-3*time.Minute), nil, nil),
		seedDispatchJobMutation(t, supplierID, runningJobID, optimizationjobs.StatusRunning, time.Now().UTC().Add(-2*time.Minute), ptrTime(time.Now().UTC().Add(-90*time.Second)), nil),
		seedDispatchJobMutation(t, supplierID, solvedJobID, optimizationjobs.StatusSolved, time.Now().UTC().Add(-1*time.Minute), ptrTime(time.Now().UTC().Add(-50*time.Second)), ptrTime(time.Now().UTC().Add(-20*time.Second))),
	}); err != nil {
		t.Fatalf("seed OptimizationJobs: %v", err)
	}
	t.Cleanup(func() {
		mutations := make([]*spanner.Mutation, 0, len(seededJobIDs))
		for _, jobID := range seededJobIDs {
			mutations = append(mutations, spanner.Delete("OptimizationJobs", spanner.Key{jobID}))
		}
		_, _ = client.Apply(context.Background(), mutations)
	})

	originalCacheClient := cache.Client
	t.Cleanup(func() {
		cache.Client = originalCacheClient
	})

	mr := startDispatchJobsMiniRedis(t)
	defer mr.Close()
	cache.Client = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	if err := cache.MarkSupplierOptimizationJobActive(ctx, supplierID, queuedJobID); err != nil {
		t.Fatalf("MarkSupplierOptimizationJobActive queued: %v", err)
	}
	if err := cache.MarkSupplierOptimizationJobActive(ctx, supplierID, runningJobID); err != nil {
		t.Fatalf("MarkSupplierOptimizationJobActive running: %v", err)
	}
	if err := cache.MarkSupplierOptimizationJobActive(ctx, supplierID, solvedJobID); err != nil {
		t.Fatalf("MarkSupplierOptimizationJobActive solved: %v", err)
	}

	primary, err := listActiveDispatchJobs(ctx, client, supplierID)
	if err != nil {
		t.Fatalf("listActiveDispatchJobs(redis primary): %v", err)
	}
	if primary.Source != "redis_index" {
		t.Fatalf("primary.Source = %q, want %q", primary.Source, "redis_index")
	}
	if primary.Degraded {
		t.Fatal("primary.Degraded = true, want false")
	}
	if len(primary.Jobs) != 2 {
		t.Fatalf("len(primary.Jobs) = %d, want 2", len(primary.Jobs))
	}
	for _, job := range primary.Jobs {
		if job.JobID == solvedJobID {
			t.Fatalf("solved job %s should have been filtered from redis primary results", solvedJobID)
		}
	}
	remainingIDs, err := mr.SMembers(cache.SupplierJobsActiveKey(supplierID))
	if err != nil {
		t.Fatalf("miniredis SMEMBERS: %v", err)
	}
	for _, remainingID := range remainingIDs {
		if remainingID == solvedJobID {
			t.Fatalf("stale solved job %s should have been removed from redis active set", solvedJobID)
		}
	}

	cache.Client = redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
		MaxRetries:   0,
	})

	fallback, err := listActiveDispatchJobs(ctx, client, supplierID)
	if err != nil {
		t.Fatalf("listActiveDispatchJobs(spanner fallback): %v", err)
	}
	if fallback.Source != "spanner_fallback" {
		t.Fatalf("fallback.Source = %q, want %q", fallback.Source, "spanner_fallback")
	}
	if !fallback.Degraded {
		t.Fatal("fallback.Degraded = false, want true")
	}
	if len(fallback.Jobs) != 2 {
		t.Fatalf("len(fallback.Jobs) = %d, want 2", len(fallback.Jobs))
	}
	for _, job := range fallback.Jobs {
		if job.Status != string(optimizationjobs.StatusQueued) && job.Status != string(optimizationjobs.StatusRunning) {
			t.Fatalf("fallback returned terminal status %q for job %s", job.Status, job.JobID)
		}
	}
}

func TestBuildDispatchJobProjection_SolvedVRPIncludesRoutesAndUnassigned(t *testing.T) {
	now := time.Now().UTC()
	payload, err := json.Marshal(contract.OptimizationJobEnvelope{
		V:          "v1",
		JobID:      "job-1",
		SupplierID: "supplier-1",
		JobType:    contract.OptimizationJobTypeAutoDispatch,
		SolverType: contract.OptimizationSolverTypeVRP,
		Status:     contract.OptimizationJobStatusSolved,
		VRP: &contract.VRPJobPayload{
			DepotNodeUUID: "depot-1",
			Depot:         &contract.VRPDepotPayload{NodeUUID: "depot-1", Label: "Main Depot", Lat: 41.2995, Lng: 69.2401},
			Vehicles: []contract.VRPVehiclePayload{{
				VehicleUUID:  "veh-1",
				DriverUUID:   "drv-1",
				DriverName:   "Driver One",
				VehicleType:  "TRUCK",
				VehicleClass: "CLASS_A",
				CapacityVU:   40,
			}},
			Nodes: []contract.VRPNodeProjectionPayload{
				{NodeUUID: "node-a", OrderID: "ORD-A", RetailerID: "RET-A", RetailerName: "Retailer A", Lat: 41.31, Lng: 69.25, Amount: 120000, DemandVU: 10, ReceivingWindowOpen: "09:00", ReceivingWindowClose: "12:00"},
				{NodeUUID: "node-b", OrderID: "ORD-B", RetailerID: "RET-B", RetailerName: "Retailer B", Lat: 41.28, Lng: 69.22, Amount: 220000, DemandVU: 12, ReceivingWindowOpen: "10:00", ReceivingWindowClose: "14:00"},
				{NodeUUID: "node-z", OrderID: "ORD-Z", RetailerID: "RET-Z", RetailerName: "Retailer Z", Lat: 41.33, Lng: 69.18, Amount: 90000, DemandVU: 8, ReceivingWindowOpen: "11:00", ReceivingWindowClose: "15:00"},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resultPayload, err := json.Marshal(contract.OptimizationSolvedEvent{
		JobID:      "job-1",
		TraceID:    "trace-1",
		SupplierID: "supplier-1",
		SolverType: contract.OptimizationSolverTypeVRP,
		Status:     contract.OptimizationSolverStatusOptimal,
		TimedOut:   false,
		MatrixSize: 3,
		ProducedAt: now.Format(time.RFC3339Nano),
		Warnings:   []string{"projection-check"},
		VRP: &contract.VRPResultEnvelope{
			Meta: contract.SolverMetadata{
				JobID:            "job-1",
				TraceID:          "trace-1",
				SupplierID:       "supplier-1",
				SolverType:       contract.OptimizationSolverTypeVRP,
				ScaleFactor:      10000,
				RequestedAtUnixM: now.Add(-time.Minute).UnixMilli(),
			},
			Status:              contract.OptimizationSolverStatusOptimal,
			TimedOut:            false,
			MatrixSize:          3,
			ObjectiveCostScaled: 65400,
			Routes: []contract.VehicleRouteEnvelope{{
				VehicleUUID:      "veh-1",
				DriverUUID:       "drv-1",
				OrderedNodeUUIDs: []string{"node-a", "node-b"},
				LoadScaled:       220000,
				RouteCostScaled:  65400,
			}},
			UnassignedNodeUUIDs: []string{"node-z"},
			Warnings:            []string{"projection-check"},
		},
	})
	if err != nil {
		t.Fatalf("marshal result payload: %v", err)
	}

	completedAt := now
	projection, err := buildDispatchJobProjection(optimizationjobs.Job{
		JobID:         "job-1",
		SupplierID:    "supplier-1",
		SolverType:    optimizationjobs.SolverTypeVRP,
		Status:        optimizationjobs.StatusSolved,
		Payload:       payload,
		ResultPayload: resultPayload,
		RequestedAt:   now.Add(-2 * time.Minute),
		UpdatedAt:     now,
		CompletedAt:   &completedAt,
	})
	if err != nil {
		t.Fatalf("buildDispatchJobProjection() error = %v", err)
	}

	if !projection.Ready {
		t.Fatal("projection.Ready = false, want true")
	}
	if projection.Depot == nil || projection.Depot.NodeUUID != "depot-1" {
		t.Fatalf("projection.Depot = %#v, want depot-1", projection.Depot)
	}
	if len(projection.Routes) != 1 {
		t.Fatalf("len(projection.Routes) = %d, want 1", len(projection.Routes))
	}
	route := projection.Routes[0]
	wantRouteID := dispatchJobRouteID("job-1", "drv-1", "veh-1", 0)
	if route.DriverName != "Driver One" {
		t.Fatalf("route.DriverName = %q, want %q", route.DriverName, "Driver One")
	}
	if route.RouteID != wantRouteID {
		t.Fatalf("route.RouteID = %q, want %q", route.RouteID, wantRouteID)
	}
	if route.ManifestID != dispatchJobManifestID(wantRouteID) {
		t.Fatalf("route.ManifestID = %q, want %q", route.ManifestID, dispatchJobManifestID(wantRouteID))
	}
	if len(route.Stops) != 2 {
		t.Fatalf("len(route.Stops) = %d, want 2", len(route.Stops))
	}
	if route.Stops[0].Sequence != 1 || route.Stops[1].Sequence != 2 {
		t.Fatalf("route stop sequences = %d,%d, want 1,2", route.Stops[0].Sequence, route.Stops[1].Sequence)
	}
	if len(projection.Unassigned) != 1 || projection.Unassigned[0].NodeUUID != "node-z" {
		t.Fatalf("projection.Unassigned = %#v, want node-z", projection.Unassigned)
	}
	if projection.ObjectiveCostKM != 6.54 {
		t.Fatalf("projection.ObjectiveCostKM = %v, want 6.54", projection.ObjectiveCostKM)
	}
}

func TestBuildDispatchJobProjection_AppliedVRPStillReady(t *testing.T) {
	now := time.Now().UTC()
	payload, err := json.Marshal(contract.OptimizationJobEnvelope{
		V:          "v1",
		JobID:      "job-applied",
		SupplierID: "supplier-1",
		JobType:    contract.OptimizationJobTypeAutoDispatch,
		SolverType: contract.OptimizationSolverTypeVRP,
		Status:     contract.OptimizationJobStatusSolved,
		VRP: &contract.VRPJobPayload{
			DepotNodeUUID: "warehouse:wh-1",
			Depot:         &contract.VRPDepotPayload{NodeUUID: "warehouse:wh-1", Label: "Warehouse", Lat: 41.2995, Lng: 69.2401},
			Vehicles: []contract.VRPVehiclePayload{{
				VehicleUUID:  "veh-1",
				DriverUUID:   "drv-1",
				DriverName:   "Driver One",
				VehicleType:  "TRUCK",
				VehicleClass: "CLASS_A",
				CapacityVU:   40,
			}},
			Nodes: []contract.VRPNodeProjectionPayload{{
				NodeUUID: "node-a", OrderID: "ORD-A", RetailerID: "RET-A", RetailerName: "Retailer A", Lat: 41.31, Lng: 69.25, Amount: 120000, DemandVU: 10,
			}},
		},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	resultPayload, err := json.Marshal(contract.OptimizationSolvedEvent{
		JobID:      "job-applied",
		TraceID:    "trace-1",
		SupplierID: "supplier-1",
		SolverType: contract.OptimizationSolverTypeVRP,
		Status:     contract.OptimizationSolverStatusOptimal,
		MatrixSize: 1,
		ProducedAt: now.Format(time.RFC3339Nano),
		VRP: &contract.VRPResultEnvelope{
			Meta: contract.SolverMetadata{ScaleFactor: 10000},
			Routes: []contract.VehicleRouteEnvelope{{
				VehicleUUID:      "veh-1",
				DriverUUID:       "drv-1",
				OrderedNodeUUIDs: []string{"node-a"},
				LoadScaled:       100000,
			}},
		},
	})
	if err != nil {
		t.Fatalf("marshal result payload: %v", err)
	}
	appliedAt := now
	projection, err := buildDispatchJobProjection(optimizationjobs.Job{
		JobID:         "job-applied",
		SupplierID:    "supplier-1",
		SolverType:    optimizationjobs.SolverTypeVRP,
		Status:        optimizationjobs.StatusApplied,
		Payload:       payload,
		ResultPayload: resultPayload,
		RequestedAt:   now.Add(-time.Minute),
		UpdatedAt:     now,
		AppliedAt:     &appliedAt,
	})
	if err != nil {
		t.Fatalf("buildDispatchJobProjection() error = %v", err)
	}
	if !projection.Ready {
		t.Fatal("projection.Ready = false, want true")
	}
	if projection.Status != string(optimizationjobs.StatusApplied) {
		t.Fatalf("projection.Status = %q, want %q", projection.Status, optimizationjobs.StatusApplied)
	}
	if len(projection.Routes) != 1 {
		t.Fatalf("len(projection.Routes) = %d, want 1", len(projection.Routes))
	}
}

func seedDispatchJobMutation(t *testing.T, supplierID string, jobID string, status optimizationjobs.Status, requestedAt time.Time, startedAt *time.Time, completedAt *time.Time) *spanner.Mutation {
	t.Helper()
	payload, err := json.Marshal(contract.OptimizationJobEnvelope{
		V:                 "v1",
		JobID:             jobID,
		SupplierID:        supplierID,
		JobType:           contract.OptimizationJobTypeAutoDispatch,
		SolverType:        contract.OptimizationSolverTypeVRP,
		TraceID:           "trace-" + jobID,
		SourceEventType:   "OPTIMIZATION_QUEUED",
		DispatchTimestamp: requestedAt.Format(time.RFC3339Nano),
		Status:            contract.OptimizationJobStatus(status),
		MatrixSize:        2,
		VRP:               &contract.VRPJobPayload{DepotNodeUUID: "depot-" + supplierID},
	})
	if err != nil {
		t.Fatalf("marshal seed payload for %s: %v", jobID, err)
	}

	job, err := optimizationjobs.New(optimizationjobs.CreateParams{
		JobID:           jobID,
		SupplierID:      supplierID,
		JobType:         optimizationjobs.JobTypeAutoDispatch,
		SolverType:      optimizationjobs.SolverTypeVRP,
		TraceID:         "trace-" + jobID,
		SourceEventType: "OPTIMIZATION_QUEUED",
		Payload:         payload,
		RequestedAt:     requestedAt,
	})
	if err != nil {
		t.Fatalf("optimizationjobs.New(%s): %v", jobID, err)
	}
	job.Status = status
	job.StartedAt = startedAt
	job.CompletedAt = completedAt
	job.UpdatedAt = requestedAt.Add(30 * time.Second)
	return optimizationjobs.InsertMutation(job)
}

func supplierDispatchJobsTestDatabase() string {
	if direct := os.Getenv("SPANNER_DATABASE_URI"); direct != "" {
		return direct
	}
	project := os.Getenv("SPANNER_PROJECT")
	instance := os.Getenv("SPANNER_INSTANCE")
	database := os.Getenv("SPANNER_DATABASE")
	if project != "" && instance != "" && database != "" {
		return fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
	}
	return defaultSupplierDispatchJobsTestDatabase
}

func ensureSupplierDispatchJobsSpannerReachable(t *testing.T) {
	t.Helper()
	host := spannerDispatchJobsEmulatorHost()
	conn, err := net.DialTimeout("tcp", host, 1500*time.Millisecond)
	if err != nil {
		t.Skipf("spanner emulator is not reachable at %s: %v", host, err)
		return
	}
	_ = conn.Close()
}

func spannerDispatchJobsEmulatorHost() string {
	host := os.Getenv("SPANNER_EMULATOR_HOST")
	if host != "" {
		return host
	}
	return "localhost:9010"
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func startDispatchJobsMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("miniredis is not permitted in this environment: %v", err)
		}
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	return mr
}
