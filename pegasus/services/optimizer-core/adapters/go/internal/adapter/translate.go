package adapter

import (
	"fmt"

	"optimizercoreadapter/internal/mapping"
	"optimizercoreadapter/internal/model"
	"optimizercoreadapter/internal/scaling"
)

const (
	defaultVRPTimeLimitMS   int64 = 2_000
	defaultCPSATTimeLimitMS int64 = 30_000
)

func BuildVRPRequest(job model.OptimizationJob, payload model.VRPPayload) (*model.VRPRequestEnvelope, error) {
	if payload.DepotNodeUUID == "" {
		return nil, fmt.Errorf("vrp payload missing depot_node_uuid")
	}

	orderedNodes := make([]string, 0, 1+len(payload.DropOffNodeUUIDs))
	orderedNodes = append(orderedNodes, payload.DepotNodeUUID)
	orderedNodes = append(orderedNodes, payload.DropOffNodeUUIDs...)

	uuidIndex, err := mapping.NewBidirectionalIndexMap(orderedNodes)
	if err != nil {
		return nil, fmt.Errorf("build uuid-index mapping: %w", err)
	}

	distanceMatrixScaled := scaling.ScaleMatrixFloat64(payload.DistanceMatrixKM)
	size := uuidIndex.Size()
	if len(distanceMatrixScaled) != size {
		return nil, fmt.Errorf("distance matrix row mismatch: got %d expected %d", len(distanceMatrixScaled), size)
	}
	for _, row := range distanceMatrixScaled {
		if len(row) != size {
			return nil, fmt.Errorf("distance matrix column mismatch: got %d expected %d", len(row), size)
		}
	}

	vehicles := make([]model.VehicleEnvelope, 0, len(payload.Vehicles))
	for _, vehicle := range payload.Vehicles {
		vehicles = append(vehicles, model.VehicleEnvelope{
			VehicleUUID:           vehicle.VehicleUUID,
			DriverUUID:            vehicle.DriverUUID,
			CapacityScaled:        scaling.ScaleFloat64(vehicle.CapacityVU),
			StartTimeWindowScaled: scaling.ScaleFloat64(vehicle.StartWindowHours),
			EndTimeWindowScaled:   scaling.ScaleFloat64(vehicle.EndWindowHours),
		})
	}

	nodeDemands := make([]model.NodeDemandEnvelope, 0, len(payload.NodeDemands))
	for _, demand := range payload.NodeDemands {
		nodeDemands = append(nodeDemands, model.NodeDemandEnvelope{
			NodeUUID:     demand.NodeUUID,
			DemandScaled: scaling.ScaleFloat64(demand.DemandVU),
		})
	}

	nodeTimeWindows := make([]model.NodeTimeWindowEnvelope, 0, len(payload.NodeTimeWindows))
	for _, window := range payload.NodeTimeWindows {
		nodeTimeWindows = append(nodeTimeWindows, model.NodeTimeWindowEnvelope{
			NodeUUID:              window.NodeUUID,
			StartTimeWindowScaled: scaling.ScaleFloat64(window.StartWindowHours),
			EndTimeWindowScaled:   scaling.ScaleFloat64(window.EndWindowHours),
		})
	}

	timeLimitMS := payload.SolverTimeLimitMs
	if timeLimitMS <= 0 {
		timeLimitMS = defaultVRPTimeLimitMS
	}

	meta := model.SolverMetadata{
		JobID:            job.JobID,
		TraceID:          job.TraceID,
		SupplierID:       job.SupplierID,
		SolverType:       job.SolverType,
		ScaleFactor:      scaling.Factor,
		IdempotencyKey:   job.IdempotencyKey,
		SourceEventType:  job.SourceEventType,
		RequestedAtUnixM: job.RequestedAtUnixM,
	}

	return &model.VRPRequestEnvelope{
		Meta:                 meta,
		DepotNodeUUID:        payload.DepotNodeUUID,
		IndexToUUID:          uuidIndex.Ordered(),
		UUIDToIndex:          uuidIndex.Reverse(),
		DistanceMatrixScaled: distanceMatrixScaled,
		Vehicles:             vehicles,
		NodeDemands:          nodeDemands,
		NodeTimeWindows:      nodeTimeWindows,
		SolverTimeLimitMs:    timeLimitMS,
		ReturnBestEffort:     true,
	}, nil
}

func BuildCPSATRequest(job model.OptimizationJob, payload model.CPSATPayload) (*model.CPSATRequestEnvelope, error) {
	factoryIDs := make([]string, 0, len(payload.FactorySlots))
	for _, slot := range payload.FactorySlots {
		factoryIDs = append(factoryIDs, slot.FactoryNodeUUID)
	}

	if _, err := mapping.NewBidirectionalIndexMap(factoryIDs); err != nil {
		return nil, fmt.Errorf("build factory uuid-index mapping: %w", err)
	}

	manifestIDs := make([]string, 0, len(payload.ManifestRequirements))
	for _, requirement := range payload.ManifestRequirements {
		manifestIDs = append(manifestIDs, requirement.ManifestID)
	}

	if _, err := mapping.NewBidirectionalIndexMap(manifestIDs); err != nil {
		return nil, fmt.Errorf("build manifest uuid-index mapping: %w", err)
	}

	factorySlots := make([]model.FactorySlotEnvelope, 0, len(payload.FactorySlots))
	for _, slot := range payload.FactorySlots {
		factorySlots = append(factorySlots, model.FactorySlotEnvelope{
			FactoryNodeUUID:    slot.FactoryNodeUUID,
			SlotCapacityScaled: scaling.ScaleFloat64(slot.SlotCapacity),
		})
	}

	manifestRequirements := make([]model.ManifestRequirementEnvelope, 0, len(payload.ManifestRequirements))
	for _, requirement := range payload.ManifestRequirements {
		manifestRequirements = append(manifestRequirements, model.ManifestRequirementEnvelope{
			ManifestID:               requirement.ManifestID,
			RequiredCapacityScaled:   scaling.ScaleFloat64(requirement.RequiredCapacity),
			PriorityScoreScaled:      scaling.ScaleFloat64(requirement.PriorityScore),
			EligibleFactoryNodeUUIDs: requirement.EligibleFactoryNodeUUIDs,
		})
	}

	timeLimitMS := payload.SolverTimeLimitMs
	if timeLimitMS <= 0 {
		timeLimitMS = defaultCPSATTimeLimitMS
	}

	meta := model.SolverMetadata{
		JobID:            job.JobID,
		TraceID:          job.TraceID,
		SupplierID:       job.SupplierID,
		SolverType:       job.SolverType,
		ScaleFactor:      scaling.Factor,
		IdempotencyKey:   job.IdempotencyKey,
		SourceEventType:  job.SourceEventType,
		RequestedAtUnixM: job.RequestedAtUnixM,
	}

	return &model.CPSATRequestEnvelope{
		Meta:                 meta,
		FactorySlots:         factorySlots,
		ManifestRequirements: manifestRequirements,
		SolverTimeLimitMs:    timeLimitMS,
		ReturnBestEffort:     true,
		NumSearchWorkers:     payload.NumSearchWorkers,
	}, nil
}
