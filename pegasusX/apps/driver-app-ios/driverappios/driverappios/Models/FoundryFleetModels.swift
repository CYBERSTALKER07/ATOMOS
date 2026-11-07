//
//  FoundryFleetModels.swift
//  driverappios
//

import Foundation

public struct DVIRChecklist8Point: Codable, Sendable, Equatable {
    public var steeringAndHornOk: Bool
    public var serviceAndParkingBrakesOk: Bool
    public var tiresTreadAndPressureOk: Bool
    public var lightsAndHazardsOk: Bool
    public var wipersAndFluidsOk: Bool
    public var emergencyKitAndExtinguisherOk: Bool
    public var cargoBoxSanitationOk: Bool
    public var reeferTempCalibratedOk: Bool

    public init(
        steeringAndHornOk: Bool = true,
        serviceAndParkingBrakesOk: Bool = true,
        tiresTreadAndPressureOk: Bool = true,
        lightsAndHazardsOk: Bool = true,
        wipersAndFluidsOk: Bool = true,
        emergencyKitAndExtinguisherOk: Bool = true,
        cargoBoxSanitationOk: Bool = true,
        reeferTempCalibratedOk: Bool = true
    ) {
        self.steeringAndHornOk = steeringAndHornOk
        self.serviceAndParkingBrakesOk = serviceAndParkingBrakesOk
        self.tiresTreadAndPressureOk = tiresTreadAndPressureOk
        self.lightsAndHazardsOk = lightsAndHazardsOk
        self.wipersAndFluidsOk = wipersAndFluidsOk
        self.emergencyKitAndExtinguisherOk = emergencyKitAndExtinguisherOk
        self.cargoBoxSanitationOk = cargoBoxSanitationOk
        self.reeferTempCalibratedOk = reeferTempCalibratedOk
    }

    enum CodingKeys: String, CodingKey {
        case steeringAndHornOk = "steering_and_horn_ok"
        case serviceAndParkingBrakesOk = "service_and_parking_brakes_ok"
        case tiresTreadAndPressureOk = "tires_tread_and_pressure_ok"
        case lightsAndHazardsOk = "lights_and_hazards_ok"
        case wipersAndFluidsOk = "wipers_and_fluids_ok"
        case emergencyKitAndExtinguisherOk = "emergency_kit_and_extinguisher_ok"
        case cargoBoxSanitationOk = "cargo_box_sanitation_ok"
        case reeferTempCalibratedOk = "reefer_temp_calibrated_ok"
    }
}

public struct FleetDVIRInspection: Codable, Sendable, Identifiable {
    public var id: String { inspectionId ?? UUID().uuidString }
    public let inspectionId: String?
    public let supplierId: String
    public let vehicleId: String
    public let driverId: String
    public let odometerKm: Int64
    public let requiresColdChain: Bool
    public let reeferTempCelsius: Double?
    public let checklist: DVIRChecklist8Point
    public let failedPoints: [String]
    public let criticalDefectFound: Bool
    public let redTagApplied: Bool
    public let status: String
    public let maintenanceWorkOrderId: String?
    public let createdAt: String?

    public init(
        inspectionId: String? = nil,
        supplierId: String,
        vehicleId: String,
        driverId: String,
        odometerKm: Int64 = 0,
        requiresColdChain: Bool = false,
        reeferTempCelsius: Double? = nil,
        checklist: DVIRChecklist8Point = DVIRChecklist8Point(),
        failedPoints: [String] = [],
        criticalDefectFound: Bool = false,
        redTagApplied: Bool = false,
        status: String = "CLEARED_FOR_DISPATCH",
        maintenanceWorkOrderId: String? = nil,
        createdAt: String? = nil
    ) {
        self.inspectionId = inspectionId
        self.supplierId = supplierId
        self.vehicleId = vehicleId
        self.driverId = driverId
        self.odometerKm = odometerKm
        self.requiresColdChain = requiresColdChain
        self.reeferTempCelsius = reeferTempCelsius
        self.checklist = checklist
        self.failedPoints = failedPoints
        self.criticalDefectFound = criticalDefectFound
        self.redTagApplied = redTagApplied
        self.status = status
        self.maintenanceWorkOrderId = maintenanceWorkOrderId
        self.createdAt = createdAt
    }

    enum CodingKeys: String, CodingKey {
        case inspectionId = "inspection_id"
        case supplierId = "supplier_id"
        case vehicleId = "vehicle_id"
        case driverId = "driver_id"
        case odometerKm = "odometer_km"
        case requiresColdChain = "requires_cold_chain"
        case reeferTempCelsius = "reefer_temp_celsius"
        case checklist
        case failedPoints = "failed_points"
        case criticalDefectFound = "critical_defect_found"
        case redTagApplied = "red_tag_applied"
        case status
        case maintenanceWorkOrderId = "maintenance_work_order_id"
        case createdAt = "created_at"
    }
}

public struct YardGateCheckEvent: Codable, Sendable, Identifiable {
    public var id: String { eventId ?? UUID().uuidString }
    public let eventId: String?
    public let depotId: String
    public let supplierId: String
    public let vehicleId: String
    public let licensePlate: String
    public let direction: String
    public let vehiclePurpose: String
    public let boltSealBarcode: String?
    public let sealIntact: Bool
    public let assignedBaySlot: String?
    public let checkedAt: String?

    public init(
        eventId: String? = nil,
        depotId: String,
        supplierId: String,
        vehicleId: String,
        licensePlate: String,
        direction: String,
        vehiclePurpose: String,
        boltSealBarcode: String? = nil,
        sealIntact: Bool = true,
        assignedBaySlot: String? = nil,
        checkedAt: String? = nil
    ) {
        self.eventId = eventId
        self.depotId = depotId
        self.supplierId = supplierId
        self.vehicleId = vehicleId
        self.licensePlate = licensePlate
        self.direction = direction
        self.vehiclePurpose = vehiclePurpose
        self.boltSealBarcode = boltSealBarcode
        self.sealIntact = sealIntact
        self.assignedBaySlot = assignedBaySlot
        self.checkedAt = checkedAt
    }

    enum CodingKeys: String, CodingKey {
        case eventId = "event_id"
        case depotId = "depot_id"
        case supplierId = "supplier_id"
        case vehicleId = "vehicle_id"
        case licensePlate = "license_plate"
        case direction
        case vehiclePurpose = "vehicle_purpose"
        case boltSealBarcode = "bolt_seal_barcode"
        case sealIntact = "seal_intact"
        case assignedBaySlot = "assigned_bay_slot"
        case checkedAt = "checked_at"
    }
}

public struct RescueToteItem: Codable, Sendable, Identifiable {
    public var id: String { toteId }
    public let toteId: String
    public let stopSequence: Int64
    public let isColdChain: Bool
    public let expectedCashMinor: Int64
    public let thermalExcursionMinutes: Int
    public let maxAllowedExcursionMins: Int

    public init(
        toteId: String,
        stopSequence: Int64,
        isColdChain: Bool = false,
        expectedCashMinor: Int64 = 0,
        thermalExcursionMinutes: Int = 0,
        maxAllowedExcursionMins: Int = 30
    ) {
        self.toteId = toteId
        self.stopSequence = stopSequence
        self.isColdChain = isColdChain
        self.expectedCashMinor = expectedCashMinor
        self.thermalExcursionMinutes = thermalExcursionMinutes
        self.maxAllowedExcursionMins = maxAllowedExcursionMins
    }

    enum CodingKeys: String, CodingKey {
        case toteId = "tote_id"
        case stopSequence = "stop_sequence"
        case isColdChain = "is_cold_chain"
        case expectedCashMinor = "expected_cash_minor"
        case thermalExcursionMinutes = "thermal_excursion_minutes"
        case maxAllowedExcursionMins = "max_allowed_excursion_mins"
    }
}

public struct MidRouteRescueRequest: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let originalRouteId: String
    public let brokenVehicleId: String
    public let originalDriverId: String
    public let rescueVehicleId: String
    public let rescueDriverId: String
    public let lastCompletedStopSeq: Int64
    public let newRescueBoltSeal: String
    public let totes: [RescueToteItem]

    public init(
        cellId: String = "cell-uz",
        supplierId: String,
        originalRouteId: String,
        brokenVehicleId: String,
        originalDriverId: String,
        rescueVehicleId: String,
        rescueDriverId: String,
        lastCompletedStopSeq: Int64,
        newRescueBoltSeal: String,
        totes: [RescueToteItem]
    ) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.originalRouteId = originalRouteId
        self.brokenVehicleId = brokenVehicleId
        self.originalDriverId = originalDriverId
        self.rescueVehicleId = rescueVehicleId
        self.rescueDriverId = rescueDriverId
        self.lastCompletedStopSeq = lastCompletedStopSeq
        self.newRescueBoltSeal = newRescueBoltSeal
        self.totes = totes
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case originalRouteId = "original_route_id"
        case brokenVehicleId = "broken_vehicle_id"
        case originalDriverId = "original_driver_id"
        case rescueVehicleId = "rescue_vehicle_id"
        case rescueDriverId = "rescue_driver_id"
        case lastCompletedStopSeq = "last_completed_stop_seq"
        case newRescueBoltSeal = "new_rescue_bolt_seal"
        case totes
    }
}

public struct MidRouteRescueResult: Codable, Sendable {
    public let rescueRouteId: String
    public let originalDriverLiabilityMinor: Int64
    public let rescueDriverLiabilityMinor: Int64
    public let transferredToteIds: [String]
    public let quarantinedSpoiledToteIds: [String]
    public let newRescueBoltSeal: String

    enum CodingKeys: String, CodingKey {
        case rescueRouteId = "rescue_route_id"
        case originalDriverLiabilityMinor = "original_driver_liability_minor"
        case rescueDriverLiabilityMinor = "rescue_driver_liability_minor"
        case transferredToteIds = "transferred_tote_ids"
        case quarantinedSpoiledToteIds = "quarantined_spoiled_tote_ids"
        case newRescueBoltSeal = "new_rescue_bolt_seal"
    }
}

public struct PredictiveReeferRequest: Codable, Sendable {
    public let telemetryDeviceId: String
    public let currentAmbientCelsius: Double
    public let compressorCurrentAmps: Double
    public let cargoTempCelsius: Double

    public init(
        telemetryDeviceId: String,
        currentAmbientCelsius: Double,
        compressorCurrentAmps: Double,
        cargoTempCelsius: Double
    ) {
        self.telemetryDeviceId = telemetryDeviceId
        self.currentAmbientCelsius = currentAmbientCelsius
        self.compressorCurrentAmps = compressorCurrentAmps
        self.cargoTempCelsius = cargoTempCelsius
    }

    enum CodingKeys: String, CodingKey {
        case telemetryDeviceId = "telemetry_device_id"
        case currentAmbientCelsius = "current_ambient_celsius"
        case compressorCurrentAmps = "compressor_current_amps"
        case cargoTempCelsius = "cargo_temp_celsius"
    }
}

public struct PredictiveReeferResponse: Codable, Sendable {
    public let riskScore: Double
    public let isFailurePredicted: Bool
    public let estimatedTimeToCriticalFailureMinutes: Int
    public let recommendations: [String]

    enum CodingKeys: String, CodingKey {
        case riskScore = "risk_score"
        case isFailurePredicted = "is_failure_predicted"
        case estimatedTimeToCriticalFailureMinutes = "estimated_time_to_critical_failure_minutes"
        case recommendations
    }
}

public struct DispatchCheckResponse: Codable, Sendable {
    public let vehicleId: String
    public let isDispatchAllowed: Bool
    public let redTagQuarantined: Bool
    public let reason: String?

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case isDispatchAllowed = "is_dispatch_allowed"
        case redTagQuarantined = "red_tag_quarantined"
        case reason
    }
}
