package optimizergrpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"optimizercoreadapter/internal/model"
	optimizercorepb "optimizercoreadapter/internal/optimizerpb"
	"optimizercoreadapter/internal/telemetry"
)

// SolverClient abstracts optimizer-core RPC calls so the worker can be tested
// without a live gRPC sidecar.
type SolverClient interface {
	CalculateRoute(ctx context.Context, req *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error)
	ResolveConstraint(ctx context.Context, req *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error)
	Close() error
}

type GRPCClient struct {
	conn    *grpc.ClientConn
	client  optimizercorepb.OptimizerCoreServiceClient
	timeout time.Duration
}

func New(address string, timeout time.Duration) (*GRPCClient, error) {
	if address == "" {
		return nil, fmt.Errorf("optimizer-core grpc address is required")
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial optimizer-core grpc %s: %w", address, err)
	}

	return &GRPCClient{
		conn:    conn,
		client:  optimizercorepb.NewOptimizerCoreServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (c *GRPCClient) CalculateRoute(ctx context.Context, req *model.VRPRequestEnvelope) (*model.VRPResultEnvelope, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("optimizer grpc client is not initialized")
	}
	if req == nil {
		return nil, errors.New("vrp request is required")
	}

	rpcReq, err := toProtoVRPRequest(req)
	if err != nil {
		return nil, fmt.Errorf("map vrp request: %w", err)
	}

	rpcCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	startedAt := time.Now()

	rpcResp, err := c.client.CalculateRoute(rpcCtx, rpcReq)
	if err != nil {
		return nil, fmt.Errorf("calculate route rpc: %w", err)
	}
	if rpcResp == nil {
		return nil, errors.New("calculate route rpc returned nil response")
	}

	result := fromProtoVRPResponse(rpcResp, req.Meta)
	telemetry.RecordSolverOutcome(string(req.Meta.SolverType), string(result.Status), time.Since(startedAt), result.MatrixSize)
	return result, nil
}

func (c *GRPCClient) ResolveConstraint(ctx context.Context, req *model.CPSATRequestEnvelope) (*model.CPSATResultEnvelope, error) {
	if c == nil || c.client == nil {
		return nil, errors.New("optimizer grpc client is not initialized")
	}
	if req == nil {
		return nil, errors.New("cp-sat request is required")
	}

	rpcReq, err := toProtoCPSATRequest(req)
	if err != nil {
		return nil, fmt.Errorf("map cp-sat request: %w", err)
	}

	rpcCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	startedAt := time.Now()

	rpcResp, err := c.client.ResolveConstraint(rpcCtx, rpcReq)
	if err != nil {
		return nil, fmt.Errorf("resolve constraint rpc: %w", err)
	}
	if rpcResp == nil {
		return nil, errors.New("resolve constraint rpc returned nil response")
	}

	result := fromProtoCPSATResponse(rpcResp, req.Meta)
	telemetry.RecordSolverOutcome(string(req.Meta.SolverType), string(result.Status), time.Since(startedAt), result.MatrixSize)
	return result, nil
}

func (c *GRPCClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func toProtoVRPRequest(req *model.VRPRequestEnvelope) (*optimizercorepb.VRPRequest, error) {
	if req.DepotNodeUUID == "" {
		return nil, errors.New("depot node uuid is required")
	}

	dropOffNodeUUIDs := deriveDropOffNodeUUIDs(req.DepotNodeUUID, req.IndexToUUID)

	distanceMatrixScaled := make([]*optimizercorepb.Int64Row, 0, len(req.DistanceMatrixScaled))
	for _, row := range req.DistanceMatrixScaled {
		values := make([]int64, len(row))
		copy(values, row)
		distanceMatrixScaled = append(distanceMatrixScaled, &optimizercorepb.Int64Row{Values: values})
	}

	vehicles := make([]*optimizercorepb.Vehicle, 0, len(req.Vehicles))
	for _, vehicle := range req.Vehicles {
		vehicles = append(vehicles, &optimizercorepb.Vehicle{
			VehicleUuid:           vehicle.VehicleUUID,
			DriverUuid:            vehicle.DriverUUID,
			CapacityScaled:        vehicle.CapacityScaled,
			StartTimeWindowScaled: vehicle.StartTimeWindowScaled,
			EndTimeWindowScaled:   vehicle.EndTimeWindowScaled,
		})
	}

	nodeDemands := make([]*optimizercorepb.NodeDemand, 0, len(req.NodeDemands))
	for _, demand := range req.NodeDemands {
		nodeDemands = append(nodeDemands, &optimizercorepb.NodeDemand{
			NodeUuid:     demand.NodeUUID,
			DemandScaled: demand.DemandScaled,
		})
	}

	nodeTimeWindows := make([]*optimizercorepb.NodeTimeWindow, 0, len(req.NodeTimeWindows))
	for _, window := range req.NodeTimeWindows {
		nodeTimeWindows = append(nodeTimeWindows, &optimizercorepb.NodeTimeWindow{
			NodeUuid:              window.NodeUUID,
			StartTimeWindowScaled: window.StartTimeWindowScaled,
			EndTimeWindowScaled:   window.EndTimeWindowScaled,
		})
	}

	return &optimizercorepb.VRPRequest{
		Meta:                 toProtoMetadata(req.Meta),
		DepotNodeUuid:        req.DepotNodeUUID,
		DropOffNodeUuids:     dropOffNodeUUIDs,
		DistanceMatrixScaled: distanceMatrixScaled,
		Vehicles:             vehicles,
		NodeDemands:          nodeDemands,
		NodeTimeWindows:      nodeTimeWindows,
		SolverTimeLimitMs:    req.SolverTimeLimitMs,
		ReturnBestEffort:     req.ReturnBestEffort,
	}, nil
}

func toProtoCPSATRequest(req *model.CPSATRequestEnvelope) (*optimizercorepb.CPSATRequest, error) {
	factorySlots := make([]*optimizercorepb.FactorySlot, 0, len(req.FactorySlots))
	for _, slot := range req.FactorySlots {
		factorySlots = append(factorySlots, &optimizercorepb.FactorySlot{
			FactoryNodeUuid:    slot.FactoryNodeUUID,
			SlotCapacityScaled: slot.SlotCapacityScaled,
		})
	}

	manifestRequirements := make([]*optimizercorepb.ManifestRequirement, 0, len(req.ManifestRequirements))
	for _, requirement := range req.ManifestRequirements {
		eligibleFactoryNodeUUIDs := make([]string, len(requirement.EligibleFactoryNodeUUIDs))
		copy(eligibleFactoryNodeUUIDs, requirement.EligibleFactoryNodeUUIDs)

		manifestRequirements = append(manifestRequirements, &optimizercorepb.ManifestRequirement{
			ManifestId:               requirement.ManifestID,
			RequiredCapacityScaled:   requirement.RequiredCapacityScaled,
			PriorityScoreScaled:      requirement.PriorityScoreScaled,
			EligibleFactoryNodeUuids: eligibleFactoryNodeUUIDs,
		})
	}

	return &optimizercorepb.CPSATRequest{
		Meta:                 toProtoMetadata(req.Meta),
		FactorySlots:         factorySlots,
		ManifestRequirements: manifestRequirements,
		SolverTimeLimitMs:    req.SolverTimeLimitMs,
		ReturnBestEffort:     req.ReturnBestEffort,
		NumSearchWorkers:     req.NumSearchWorkers,
	}, nil
}

func deriveDropOffNodeUUIDs(depotNodeUUID string, indexToUUID []string) []string {
	dropOffNodeUUIDs := make([]string, 0, len(indexToUUID))
	for index, nodeUUID := range indexToUUID {
		if nodeUUID == "" {
			continue
		}
		if index == 0 && nodeUUID == depotNodeUUID {
			continue
		}
		if nodeUUID == depotNodeUUID {
			continue
		}
		dropOffNodeUUIDs = append(dropOffNodeUUIDs, nodeUUID)
	}
	return dropOffNodeUUIDs
}

func fromProtoVRPResponse(resp *optimizercorepb.VRPResponse, fallback model.SolverMetadata) *model.VRPResultEnvelope {
	if resp == nil {
		return &model.VRPResultEnvelope{Meta: fallback}
	}

	routes := make([]model.VehicleRouteEnvelope, 0, len(resp.GetRoutes()))
	for _, route := range resp.GetRoutes() {
		if route == nil {
			continue
		}
		orderedNodeUUIDs := make([]string, len(route.GetOrderedNodeUuids()))
		copy(orderedNodeUUIDs, route.GetOrderedNodeUuids())

		routes = append(routes, model.VehicleRouteEnvelope{
			VehicleUUID:      route.GetVehicleUuid(),
			DriverUUID:       route.GetDriverUuid(),
			OrderedNodeUUIDs: orderedNodeUUIDs,
			LoadScaled:       route.GetLoadScaled(),
			RouteCostScaled:  route.GetRouteCostScaled(),
		})
	}

	unassignedNodeUUIDs := make([]string, len(resp.GetUnassignedNodeUuids()))
	copy(unassignedNodeUUIDs, resp.GetUnassignedNodeUuids())
	warnings := make([]string, len(resp.GetWarnings()))
	copy(warnings, resp.GetWarnings())

	return &model.VRPResultEnvelope{
		Meta:                fromProtoMetadata(resp.GetMeta(), fallback),
		Status:              fromProtoVRPStatus(resp),
		TimedOut:            resp.GetTimedOut(),
		MatrixSize:          vrpMatrixSize(resp),
		ObjectiveCostScaled: resp.GetObjectiveCostScaled(),
		Routes:              routes,
		UnassignedNodeUUIDs: unassignedNodeUUIDs,
		Warnings:            warnings,
	}
}

func fromProtoCPSATResponse(resp *optimizercorepb.CPSATResponse, fallback model.SolverMetadata) *model.CPSATResultEnvelope {
	if resp == nil {
		return &model.CPSATResultEnvelope{Meta: fallback}
	}

	assignments := make([]model.AssignmentEnvelope, 0, len(resp.GetAssignments()))
	for _, assignment := range resp.GetAssignments() {
		if assignment == nil {
			continue
		}
		assignments = append(assignments, model.AssignmentEnvelope{
			ManifestID:      assignment.GetManifestId(),
			FactoryNodeUUID: assignment.GetFactoryNodeUuid(),
			Assigned:        assignment.GetAssigned(),
		})
	}

	unassignedManifestIDs := make([]string, len(resp.GetUnassignedManifestIds()))
	copy(unassignedManifestIDs, resp.GetUnassignedManifestIds())
	warnings := make([]string, len(resp.GetWarnings()))
	copy(warnings, resp.GetWarnings())

	return &model.CPSATResultEnvelope{
		Meta:                  fromProtoMetadata(resp.GetMeta(), fallback),
		Status:                fromProtoCPSATStatus(resp),
		TimedOut:              resp.GetTimedOut(),
		MatrixSize:            cpsatMatrixSize(resp),
		ObjectiveScoreScaled:  resp.GetObjectiveScoreScaled(),
		Assignments:           assignments,
		UnassignedManifestIDs: unassignedManifestIDs,
		Warnings:              warnings,
	}
}

func fromProtoVRPStatus(resp *optimizercorepb.VRPResponse) model.SolverStatus {
	if resp == nil {
		return model.SolverStatusModelInvalid
	}
	if resp.GetMatrixSize() == 0 {
		if resp.GetTimedOut() || resp.GetFeasible() {
			return model.SolverStatusFeasible
		}
		return model.SolverStatusInfeasible
	}

	switch resp.GetStatus() {
	case optimizercorepb.SolverStatus_OPTIMAL:
		return model.SolverStatusOptimal
	case optimizercorepb.SolverStatus_FEASIBLE:
		return model.SolverStatusFeasible
	case optimizercorepb.SolverStatus_INFEASIBLE:
		return model.SolverStatusInfeasible
	case optimizercorepb.SolverStatus_MODEL_INVALID:
		return model.SolverStatusModelInvalid
	default:
		return model.SolverStatusModelInvalid
	}
}

func fromProtoCPSATStatus(resp *optimizercorepb.CPSATResponse) model.SolverStatus {
	if resp == nil {
		return model.SolverStatusModelInvalid
	}
	if resp.GetMatrixSize() == 0 {
		if resp.GetTimedOut() || resp.GetFeasible() {
			return model.SolverStatusFeasible
		}
		return model.SolverStatusInfeasible
	}

	switch resp.GetStatus() {
	case optimizercorepb.SolverStatus_OPTIMAL:
		return model.SolverStatusOptimal
	case optimizercorepb.SolverStatus_FEASIBLE:
		return model.SolverStatusFeasible
	case optimizercorepb.SolverStatus_INFEASIBLE:
		return model.SolverStatusInfeasible
	case optimizercorepb.SolverStatus_MODEL_INVALID:
		return model.SolverStatusModelInvalid
	default:
		return model.SolverStatusModelInvalid
	}
}

func vrpMatrixSize(resp *optimizercorepb.VRPResponse) int32 {
	if resp == nil {
		return 0
	}
	if size := resp.GetMatrixSize(); size > 0 {
		return size
	}
	return int32(len(resp.GetUnassignedNodeUuids()) + 1)
}

func cpsatMatrixSize(resp *optimizercorepb.CPSATResponse) int32 {
	if resp == nil {
		return 0
	}
	if size := resp.GetMatrixSize(); size > 0 {
		return size
	}
	return int32(len(resp.GetAssignments()))
}

func toProtoMetadata(meta model.SolverMetadata) *optimizercorepb.SolverMetadata {
	return &optimizercorepb.SolverMetadata{
		JobId:             meta.JobID,
		TraceId:           meta.TraceID,
		SupplierId:        meta.SupplierID,
		JobType:           toProtoJobType(meta.SolverType),
		ScaleFactor:       meta.ScaleFactor,
		IdempotencyKey:    meta.IdempotencyKey,
		SourceEventType:   meta.SourceEventType,
		RequestedAtUnixMs: meta.RequestedAtUnixM,
	}
}

func fromProtoMetadata(meta *optimizercorepb.SolverMetadata, fallback model.SolverMetadata) model.SolverMetadata {
	if meta == nil {
		return fallback
	}

	return model.SolverMetadata{
		JobID:            meta.GetJobId(),
		TraceID:          meta.GetTraceId(),
		SupplierID:       meta.GetSupplierId(),
		SolverType:       fromProtoJobType(meta.GetJobType(), fallback.SolverType),
		ScaleFactor:      meta.GetScaleFactor(),
		IdempotencyKey:   meta.GetIdempotencyKey(),
		SourceEventType:  meta.GetSourceEventType(),
		RequestedAtUnixM: meta.GetRequestedAtUnixMs(),
	}
}

func toProtoJobType(solverType model.SolverType) optimizercorepb.JobType {
	switch solverType {
	case model.SolverTypeVRP:
		return optimizercorepb.JobType_JOB_TYPE_FLEET_ROUTING
	case model.SolverTypeCPSAT:
		return optimizercorepb.JobType_JOB_TYPE_FACTORY_SCHEDULING
	default:
		return optimizercorepb.JobType_JOB_TYPE_UNSPECIFIED
	}
}

func fromProtoJobType(jobType optimizercorepb.JobType, fallback model.SolverType) model.SolverType {
	switch jobType {
	case optimizercorepb.JobType_JOB_TYPE_FLEET_ROUTING:
		return model.SolverTypeVRP
	case optimizercorepb.JobType_JOB_TYPE_FACTORY_SCHEDULING:
		return model.SolverTypeCPSAT
	default:
		if fallback == "" {
			return model.SolverTypeVRP
		}
		return fallback
	}
}
