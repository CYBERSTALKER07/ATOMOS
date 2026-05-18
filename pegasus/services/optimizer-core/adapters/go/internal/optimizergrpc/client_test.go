package optimizergrpc

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"optimizercoreadapter/internal/model"
	optimizercorepb "optimizercoreadapter/internal/optimizerpb"
)

const testBufConnSize = 1024 * 1024

type fakeOptimizerCoreService struct {
	optimizercorepb.UnimplementedOptimizerCoreServiceServer
	calculateRouteFn    func(context.Context, *optimizercorepb.VRPRequest) (*optimizercorepb.VRPResponse, error)
	resolveConstraintFn func(context.Context, *optimizercorepb.CPSATRequest) (*optimizercorepb.CPSATResponse, error)
}

func (s *fakeOptimizerCoreService) CalculateRoute(ctx context.Context, req *optimizercorepb.VRPRequest) (*optimizercorepb.VRPResponse, error) {
	if s.calculateRouteFn == nil {
		return nil, status.Error(codes.Unimplemented, "calculate route not implemented in test service")
	}
	return s.calculateRouteFn(ctx, req)
}

func (s *fakeOptimizerCoreService) ResolveConstraint(ctx context.Context, req *optimizercorepb.CPSATRequest) (*optimizercorepb.CPSATResponse, error) {
	if s.resolveConstraintFn == nil {
		return nil, status.Error(codes.Unimplemented, "resolve constraint not implemented in test service")
	}
	return s.resolveConstraintFn(ctx, req)
}

func TestCalculateRouteMapsProtoRoundTrip(t *testing.T) {
	client, cleanup := newBufConnClient(t, 200*time.Millisecond, &fakeOptimizerCoreService{
		calculateRouteFn: func(_ context.Context, req *optimizercorepb.VRPRequest) (*optimizercorepb.VRPResponse, error) {
			if req.GetMeta().GetJobId() != "job-vrp-1" {
				t.Fatalf("unexpected meta.job_id: %q", req.GetMeta().GetJobId())
			}
			if req.GetMeta().GetJobType() != optimizercorepb.JobType_JOB_TYPE_FLEET_ROUTING {
				t.Fatalf("unexpected meta.job_type: %v", req.GetMeta().GetJobType())
			}
			if req.GetDepotNodeUuid() != "depot" {
				t.Fatalf("unexpected depot_node_uuid: %q", req.GetDepotNodeUuid())
			}
			if len(req.GetDropOffNodeUuids()) != 2 || req.GetDropOffNodeUuids()[0] != "n1" || req.GetDropOffNodeUuids()[1] != "n2" {
				t.Fatalf("unexpected drop_off_node_uuids: %#v", req.GetDropOffNodeUuids())
			}
			if len(req.GetDistanceMatrixScaled()) != 3 {
				t.Fatalf("unexpected matrix row count: %d", len(req.GetDistanceMatrixScaled()))
			}

			return &optimizercorepb.VRPResponse{
				Meta:                req.GetMeta(),
				Feasible:            true,
				TimedOut:            false,
				ObjectiveCostScaled: 1234,
				Routes: []*optimizercorepb.VehicleRoute{{
					VehicleUuid:      "veh-1",
					DriverUuid:       "drv-1",
					OrderedNodeUuids: []string{"n1", "n2"},
					LoadScaled:       150,
					RouteCostScaled:  1234,
				}},
				UnassignedNodeUuids: []string{},
				Warnings:            []string{"best effort"},
			}, nil
		},
	})
	defer cleanup()

	result, err := client.CalculateRoute(context.Background(), &model.VRPRequestEnvelope{
		Meta: model.SolverMetadata{
			JobID:            "job-vrp-1",
			TraceID:          "trace-vrp-1",
			SupplierID:       "sup-1",
			SolverType:       model.SolverTypeVRP,
			ScaleFactor:      10_000,
			IdempotencyKey:   "idem-vrp-1",
			SourceEventType:  "OPTIMIZATION_REQUESTED",
			RequestedAtUnixM: 1710000000000,
		},
		DepotNodeUUID: "depot",
		IndexToUUID:   []string{"depot", "n1", "n2"},
		DistanceMatrixScaled: [][]int64{
			{0, 5, 8},
			{5, 0, 3},
			{8, 3, 0},
		},
		Vehicles: []model.VehicleEnvelope{{
			VehicleUUID:           "veh-1",
			DriverUUID:            "drv-1",
			CapacityScaled:        200,
			StartTimeWindowScaled: 0,
			EndTimeWindowScaled:   1000,
		}},
		NodeDemands:       []model.NodeDemandEnvelope{{NodeUUID: "n1", DemandScaled: 50}, {NodeUUID: "n2", DemandScaled: 100}},
		NodeTimeWindows:   []model.NodeTimeWindowEnvelope{{NodeUUID: "n1", StartTimeWindowScaled: 0, EndTimeWindowScaled: 1000}, {NodeUUID: "n2", StartTimeWindowScaled: 0, EndTimeWindowScaled: 1000}},
		SolverTimeLimitMs: 100,
		ReturnBestEffort:  true,
	})
	if err != nil {
		t.Fatalf("calculate route returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Feasible {
		t.Fatalf("expected feasible result, got: %+v", result)
	}
	if result.ObjectiveCostScaled != 1234 {
		t.Fatalf("unexpected objective cost: %d", result.ObjectiveCostScaled)
	}
	if len(result.Routes) != 1 {
		t.Fatalf("expected one route, got: %d", len(result.Routes))
	}
	if result.Routes[0].VehicleUUID != "veh-1" || result.Routes[0].DriverUUID != "drv-1" {
		t.Fatalf("unexpected route identity: %+v", result.Routes[0])
	}
	if got := result.Meta.SolverType; got != model.SolverTypeVRP {
		t.Fatalf("unexpected mapped solver type: %q", got)
	}
}

func TestResolveConstraintMapsInfeasibleResponse(t *testing.T) {
	client, cleanup := newBufConnClient(t, 200*time.Millisecond, &fakeOptimizerCoreService{
		resolveConstraintFn: func(_ context.Context, req *optimizercorepb.CPSATRequest) (*optimizercorepb.CPSATResponse, error) {
			if req.GetMeta().GetJobType() != optimizercorepb.JobType_JOB_TYPE_FACTORY_SCHEDULING {
				t.Fatalf("unexpected meta.job_type: %v", req.GetMeta().GetJobType())
			}
			if len(req.GetFactorySlots()) != 1 {
				t.Fatalf("unexpected factory slots: %d", len(req.GetFactorySlots()))
			}
			if len(req.GetManifestRequirements()) != 2 {
				t.Fatalf("unexpected manifest requirements: %d", len(req.GetManifestRequirements()))
			}
			return &optimizercorepb.CPSATResponse{
				Meta:                 req.GetMeta(),
				Feasible:             false,
				TimedOut:             false,
				ObjectiveScoreScaled: 42,
				Assignments: []*optimizercorepb.Assignment{{
					ManifestId:      "m-1",
					FactoryNodeUuid: "f-1",
					Assigned:        true,
				}},
				UnassignedManifestIds: []string{"m-2"},
				Warnings:              []string{"capacity shortfall"},
			}, nil
		},
	})
	defer cleanup()

	result, err := client.ResolveConstraint(context.Background(), &model.CPSATRequestEnvelope{
		Meta: model.SolverMetadata{
			JobID:            "job-cp-1",
			TraceID:          "trace-cp-1",
			SupplierID:       "sup-1",
			SolverType:       model.SolverTypeCPSAT,
			ScaleFactor:      10_000,
			IdempotencyKey:   "idem-cp-1",
			SourceEventType:  "OPTIMIZATION_REQUESTED",
			RequestedAtUnixM: 1710000000011,
		},
		FactorySlots: []model.FactorySlotEnvelope{{
			FactoryNodeUUID:    "f-1",
			SlotCapacityScaled: 100,
		}},
		ManifestRequirements: []model.ManifestRequirementEnvelope{
			{ManifestID: "m-1", RequiredCapacityScaled: 80, PriorityScoreScaled: 10, EligibleFactoryNodeUUIDs: []string{"f-1"}},
			{ManifestID: "m-2", RequiredCapacityScaled: 80, PriorityScoreScaled: 8, EligibleFactoryNodeUUIDs: []string{"f-1"}},
		},
		SolverTimeLimitMs: 200,
		ReturnBestEffort:  true,
		NumSearchWorkers:  2,
	})
	if err != nil {
		t.Fatalf("resolve constraint returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil cp-sat result")
	}
	if result.Feasible {
		t.Fatalf("expected infeasible result, got feasible: %+v", result)
	}
	if len(result.Assignments) != 1 || result.Assignments[0].ManifestID != "m-1" {
		t.Fatalf("unexpected assignments: %+v", result.Assignments)
	}
	if len(result.UnassignedManifestIDs) != 1 || result.UnassignedManifestIDs[0] != "m-2" {
		t.Fatalf("unexpected unassigned manifests: %+v", result.UnassignedManifestIDs)
	}
}

func TestCalculateRouteTimeoutReturnsDeadlineExceeded(t *testing.T) {
	client, cleanup := newBufConnClient(t, 25*time.Millisecond, &fakeOptimizerCoreService{
		calculateRouteFn: func(ctx context.Context, req *optimizercorepb.VRPRequest) (*optimizercorepb.VRPResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	defer cleanup()

	_, err := client.CalculateRoute(context.Background(), &model.VRPRequestEnvelope{
		Meta:          model.SolverMetadata{JobID: "job-timeout", SolverType: model.SolverTypeVRP},
		DepotNodeUUID: "depot",
		IndexToUUID:   []string{"depot", "node-1"},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if status.Code(err) != codes.DeadlineExceeded && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
}

func TestCalculateRouteSidecarUnavailableReturnsError(t *testing.T) {
	client, cleanup := newBufConnClient(t, 200*time.Millisecond, &fakeOptimizerCoreService{
		calculateRouteFn: func(_ context.Context, req *optimizercorepb.VRPRequest) (*optimizercorepb.VRPResponse, error) {
			return nil, status.Error(codes.Unavailable, "sidecar unavailable")
		},
	})
	defer cleanup()

	_, err := client.CalculateRoute(context.Background(), &model.VRPRequestEnvelope{
		Meta:          model.SolverMetadata{JobID: "job-crash", SolverType: model.SolverTypeVRP},
		DepotNodeUUID: "depot",
		IndexToUUID:   []string{"depot", "node-1"},
	})
	if err == nil {
		t.Fatal("expected unavailable error")
	}
	if status.Code(err) != codes.Unavailable && !strings.Contains(strings.ToLower(err.Error()), "unavailable") {
		t.Fatalf("expected unavailable transport error, got: %v", err)
	}
}

func newBufConnClient(t *testing.T, timeout time.Duration, service *fakeOptimizerCoreService) (*GRPCClient, func()) {
	t.Helper()

	listener := bufconn.Listen(testBufConnSize)
	grpcServer := grpc.NewServer()
	optimizercorepb.RegisterOptimizerCoreServiceServer(grpcServer, service)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return listener.Dial()
	}

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}

	client := &GRPCClient{
		conn:    conn,
		client:  optimizercorepb.NewOptimizerCoreServiceClient(conn),
		timeout: timeout,
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}
