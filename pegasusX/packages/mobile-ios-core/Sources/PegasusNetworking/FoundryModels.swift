import Foundation

// MARK: - Role 7: Cashier & AML Vault Models

public struct StartCashierSessionRequest: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let depotId: String
    public let driverId: String
    public let routeId: String
    public let bagSealBarcode: String
    public let expectedCashMinor: Int64

    public init(
        cellId: String = "cell-uz",
        supplierId: String,
        depotId: String,
        driverId: String,
        routeId: String,
        bagSealBarcode: String,
        expectedCashMinor: Int64
    ) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.depotId = depotId
        self.driverId = driverId
        self.routeId = routeId
        self.bagSealBarcode = bagSealBarcode
        self.expectedCashMinor = expectedCashMinor
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case depotId = "depot_id"
        case driverId = "driver_id"
        case routeId = "route_id"
        case bagSealBarcode = "bag_seal_barcode"
        case expectedCashMinor = "expected_cash_minor"
    }
}

public struct CashierSessionView: Codable, Sendable, Identifiable {
    public var id: String { sessionId }
    public let sessionId: String
    public let supplierId: String
    public let depotId: String
    public let driverId: String
    public let routeId: String
    public let bagSealBarcode: String
    public let attemptNumber: Int64
    public let countedCashMinor: Int64
    public let expectedCashMinor: Int64?
    public let varianceMinor: Int64?
    public let status: String
    public let clearanceTokenHash: String?
    public let microSkimmingFlagged: Bool?
    public let createdAt: String
    public let resolvedAt: String?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case supplierId = "supplier_id"
        case depotId = "depot_id"
        case driverId = "driver_id"
        case routeId = "route_id"
        case bagSealBarcode = "bag_seal_barcode"
        case attemptNumber = "attempt_number"
        case countedCashMinor = "counted_cash_minor"
        case expectedCashMinor = "expected_cash_minor"
        case varianceMinor = "variance_minor"
        case status
        case clearanceTokenHash = "clearance_token_hash"
        case microSkimmingFlagged = "micro_skimming_flagged"
        case createdAt = "created_at"
        case resolvedAt = "resolved_at"
    }
}

public struct BlindCountRequest: Codable, Sendable {
    public let denominationsUzs: [String: Int64]
    public let toleranceThresholdMinor: Int64

    public init(denominationsUzs: [String: Int64], toleranceThresholdMinor: Int64 = 500000) {
        self.denominationsUzs = denominationsUzs
        self.toleranceThresholdMinor = toleranceThresholdMinor
    }

    enum CodingKeys: String, CodingKey {
        case denominationsUzs = "denominations_uzs"
        case toleranceThresholdMinor = "tolerance_threshold_minor"
    }
}

public struct RRNTenderReclassificationRequest: Codable, Sendable {
    public let sessionId: String
    public let orderId: String
    public let disputedCashMinor: Int64
    public let gatewayRrn: String
    public let terminalId: String
    public let cardLastFour: String
    public let authCode: String
    public let paymentNetwork: String
    public let rrnCapturedAt: String
    public let podGeofenceTimestamp: String

    public init(
        sessionId: String,
        orderId: String,
        disputedCashMinor: Int64,
        gatewayRrn: String,
        terminalId: String,
        cardLastFour: String,
        authCode: String,
        paymentNetwork: String,
        rrnCapturedAt: String,
        podGeofenceTimestamp: String
    ) {
        self.sessionId = sessionId
        self.orderId = orderId
        self.disputedCashMinor = disputedCashMinor
        self.gatewayRrn = gatewayRrn
        self.terminalId = terminalId
        self.cardLastFour = cardLastFour
        self.authCode = authCode
        self.paymentNetwork = paymentNetwork
        self.rrnCapturedAt = rrnCapturedAt
        self.podGeofenceTimestamp = podGeofenceTimestamp
    }

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case orderId = "order_id"
        case disputedCashMinor = "disputed_cash_minor"
        case gatewayRrn = "gateway_rrn"
        case terminalId = "terminal_id"
        case cardLastFour = "card_last_four"
        case authCode = "auth_code"
        case paymentNetwork = "payment_network"
        case rrnCapturedAt = "rrn_captured_at"
        case podGeofenceTimestamp = "pod_geofence_timestamp"
    }
}

public struct CITHandoverRequest: Codable, Sendable {
    public let depotId: String
    public let supplierId: String
    public let sessionIds: [String]
    public let courierBadgeId: String
    public let citSealBarcode: String
    public let cashMinor: Int64

    public init(
        depotId: String,
        supplierId: String,
        sessionIds: [String],
        courierBadgeId: String,
        citSealBarcode: String,
        cashMinor: Int64
    ) {
        self.depotId = depotId
        self.supplierId = supplierId
        self.sessionIds = sessionIds
        self.courierBadgeId = courierBadgeId
        self.citSealBarcode = citSealBarcode
        self.cashMinor = cashMinor
    }

    enum CodingKeys: String, CodingKey {
        case depotId = "depot_id"
        case supplierId = "supplier_id"
        case sessionIds = "session_ids"
        case courierBadgeId = "courier_badge_id"
        case citSealBarcode = "cit_seal_barcode"
        case cashMinor = "cash_minor"
    }
}

public struct DepotVaultLedgerEntry: Codable, Sendable {
    public let vaultEntryId: String
    public let depotId: String
    public let supplierId: String
    public let sessionId: String?
    public let citManifestId: String?
    public let entryType: String
    public let amountMinor: Int64
    public let runningBalanceMinor: Int64
    public let courierBadgeId: String?
    public let sealBarcode: String?
    public let createdAt: String

    enum CodingKeys: String, CodingKey {
        case vaultEntryId = "vault_entry_id"
        case depotId = "depot_id"
        case supplierId = "supplier_id"
        case sessionId = "session_id"
        case citManifestId = "cit_manifest_id"
        case entryType = "entry_type"
        case amountMinor = "amount_minor"
        case runningBalanceMinor = "running_balance_minor"
        case courierBadgeId = "courier_badge_id"
        case sealBarcode = "seal_barcode"
        case createdAt = "created_at"
    }
}

public struct AMLCheckRequest: Codable, Sendable {
    public let legalEntityTin: String
    public let supplierId: String
    public let calendarDate: String
    public let cashMinor: Int64

    public init(legalEntityTin: String, supplierId: String, calendarDate: String, cashMinor: Int64) {
        self.legalEntityTin = legalEntityTin
        self.supplierId = supplierId
        self.calendarDate = calendarDate
        self.cashMinor = cashMinor
    }

    enum CodingKeys: String, CodingKey {
        case legalEntityTin = "legal_entity_tin"
        case supplierId = "supplier_id"
        case calendarDate = "calendar_date"
        case cashMinor = "cash_minor"
    }
}

public struct AMLCheckResponse: Codable, Sendable {
    public let legalEntityTin: String
    public let accumulatedCashMinor: Int64
    public let ceilingMinor: Int64

    enum CodingKeys: String, CodingKey {
        case legalEntityTin = "legal_entity_tin"
        case accumulatedCashMinor = "accumulated_cash_minor"
        case ceilingMinor = "ceiling_minor"
    }
}

// MARK: - Role 9: Fleet, Yard & Maintenance Models

public struct DVIRChecklist8Point: Codable, Sendable {
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

public struct FleetDVIRInspection: Codable, Sendable {
    public let inspectionId: String?
    public let cellId: String
    public let supplierId: String
    public let vehicleId: String
    public let driverId: String
    public let inspectionType: String
    public let requiresColdChain: Bool
    public let checklist: DVIRChecklist8Point
    public let failedPoints: [String]?
    public let criticalDefectFound: Bool
    public let redTagApplied: Bool
    public let odometerKm: Int64
    public let reeferTempCelsius: Double
    public let status: String?
    public let maintenanceWorkOrderId: String?
    public let preventativeServiceDue: Bool
    public let createdAt: String?

    public init(
        inspectionId: String? = nil,
        cellId: String = "cell-uz",
        supplierId: String,
        vehicleId: String,
        driverId: String,
        inspectionType: String = "PRE_TRIP",
        requiresColdChain: Bool = false,
        checklist: DVIRChecklist8Point = DVIRChecklist8Point(),
        failedPoints: [String]? = nil,
        criticalDefectFound: Bool = false,
        redTagApplied: Bool = false,
        odometerKm: Int64 = 0,
        reeferTempCelsius: Double = 0.0,
        status: String? = nil,
        maintenanceWorkOrderId: String? = nil,
        preventativeServiceDue: Bool = false,
        createdAt: String? = nil
    ) {
        self.inspectionId = inspectionId
        self.cellId = cellId
        self.supplierId = supplierId
        self.vehicleId = vehicleId
        self.driverId = driverId
        self.inspectionType = inspectionType
        self.requiresColdChain = requiresColdChain
        self.checklist = checklist
        self.failedPoints = failedPoints
        self.criticalDefectFound = criticalDefectFound
        self.redTagApplied = redTagApplied
        self.odometerKm = odometerKm
        self.reeferTempCelsius = reeferTempCelsius
        self.status = status
        self.maintenanceWorkOrderId = maintenanceWorkOrderId
        self.preventativeServiceDue = preventativeServiceDue
        self.createdAt = createdAt
    }

    enum CodingKeys: String, CodingKey {
        case inspectionId = "inspection_id"
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case vehicleId = "vehicle_id"
        case driverId = "driver_id"
        case inspectionType = "inspection_type"
        case requiresColdChain = "requires_cold_chain"
        case checklist
        case failedPoints = "failed_points"
        case criticalDefectFound = "critical_defect_found"
        case redTagApplied = "red_tag_applied"
        case odometerKm = "odometer_km"
        case reeferTempCelsius = "reefer_temp_celsius"
        case status
        case maintenanceWorkOrderId = "maintenance_work_order_id"
        case preventativeServiceDue = "preventative_service_due"
        case createdAt = "created_at"
    }
}

public struct YardGateCheckEvent: Codable, Sendable {
    public let gateEventId: String?
    public let depotId: String
    public let supplierId: String
    public let vehicleId: String
    public let licensePlate: String
    public let direction: String
    public let vehiclePurpose: String
    public let boltSealBarcode: String?
    public let sealIntact: Bool
    public let assignedBaySlot: String?
    public let createdAt: String?

    public init(
        gateEventId: String? = nil,
        depotId: String,
        supplierId: String,
        vehicleId: String,
        licensePlate: String,
        direction: String,
        vehiclePurpose: String,
        boltSealBarcode: String? = nil,
        sealIntact: Bool = true,
        assignedBaySlot: String? = nil,
        createdAt: String? = nil
    ) {
        self.gateEventId = gateEventId
        self.depotId = depotId
        self.supplierId = supplierId
        self.vehicleId = vehicleId
        self.licensePlate = licensePlate
        self.direction = direction
        self.vehiclePurpose = vehiclePurpose
        self.boltSealBarcode = boltSealBarcode
        self.sealIntact = sealIntact
        self.assignedBaySlot = assignedBaySlot
        self.createdAt = createdAt
    }

    enum CodingKeys: String, CodingKey {
        case gateEventId = "gate_event_id"
        case depotId = "depot_id"
        case supplierId = "supplier_id"
        case vehicleId = "vehicle_id"
        case licensePlate = "license_plate"
        case direction
        case vehiclePurpose = "vehicle_purpose"
        case boltSealBarcode = "bolt_seal_barcode"
        case sealIntact = "seal_intact"
        case assignedBaySlot = "assigned_bay_slot"
        case createdAt = "created_at"
    }
}

public struct PredictiveReeferRequest: Codable, Sendable {
    public let vehicleId: String
    public let coolingRateDegPerMin: Double
    public let minRequiredRateDegPerMin: Double

    public init(vehicleId: String, coolingRateDegPerMin: Double, minRequiredRateDegPerMin: Double = 0.25) {
        self.vehicleId = vehicleId
        self.coolingRateDegPerMin = coolingRateDegPerMin
        self.minRequiredRateDegPerMin = minRequiredRateDegPerMin
    }

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case coolingRateDegPerMin = "cooling_rate_deg_per_min"
        case minRequiredRateDegPerMin = "min_required_rate_deg_per_min"
    }
}

public struct PredictiveReeferResponse: Codable, Sendable {
    public let vehicleId: String
    public let preFailureMaintenanceHold: Bool

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case preFailureMaintenanceHold = "pre_failure_maintenance_hold"
    }
}

public struct DispatchCheckResponse: Codable, Sendable {
    public let vehicleId: String
    public let dispatchAuthorized: Bool

    enum CodingKeys: String, CodingKey {
        case vehicleId = "vehicle_id"
        case dispatchAuthorized = "dispatch_authorized"
    }
}

public struct RescueToteItem: Codable, Sendable {
    public let toteId: String
    public let stopSequence: Long
    public let isColdChain: Bool
    public let expectedCashMinor: Int64
    public let thermalExcursionMinutes: Int64
    public let maxAllowedExcursionMins: Int64

    public init(
        toteId: String,
        stopSequence: Long,
        isColdChain: Bool = false,
        expectedCashMinor: Int64 = 0,
        thermalExcursionMinutes: Int64 = 0,
        maxAllowedExcursionMins: Int64 = 30
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

public typealias Long = Int64

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

// MARK: - Foundry Triad Models

public struct SearchAroundRequest: Codable, Sendable {
    public let cellId: String
    public let startObjectType: String
    public let startObjectIds: [String]
    public let allowedLinkTypes: [String]
    public let maxHops: Int

    public init(
        cellId: String = "cell-uz",
        startObjectType: String,
        startObjectIds: [String],
        allowedLinkTypes: [String] = [],
        maxHops: Int = 3
    ) {
        self.cellId = cellId
        self.startObjectType = startObjectType
        self.startObjectIds = startObjectIds
        self.allowedLinkTypes = allowedLinkTypes
        self.maxHops = maxHops
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case startObjectType = "start_object_type"
        case startObjectIds = "start_object_ids"
        case allowedLinkTypes = "allowed_link_types"
        case maxHops = "max_hops"
    }
}

public struct SearchAroundResponse: Codable, Sendable {
    public let objects: [OntologyObjectDto]
}

public struct OntologyObjectDto: Codable, Sendable {
    public let cellId: String
    public let objectType: String
    public let objectId: String
    public let supplierId: String
    public let version: Int64
    public let securityMarkings: [String]
    public let validFrom: String?
    public let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case objectType = "object_type"
        case objectId = "object_id"
        case supplierId = "supplier_id"
        case version
        case securityMarkings = "security_markings"
        case validFrom = "valid_from"
        case updatedAt = "updated_at"
    }
}

public struct LotRecallRequest: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let rawMaterialLotId: String

    public init(cellId: String = "cell-uz", supplierId: String, rawMaterialLotId: String) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.rawMaterialLotId = rawMaterialLotId
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case rawMaterialLotId = "raw_material_lot_id"
    }
}

public struct RecallBlastRadiusReport: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let contaminatedRawMaterialLot: String
    public let affectedWorkOrders: [String]
    public let affectedFinishedLots: [String]
    public let quarantinedSsccPallets: [String]
    public let quarantinedWarehouseTotes: [String]
    public let inTransitDriverDropBlocks: [String]
    public let retailerReversePickupOrders: [String]
    public let impactedSoliqFacturas: [String]

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case contaminatedRawMaterialLot = "contaminated_raw_material_lot"
        case affectedWorkOrders = "affected_work_orders"
        case affectedFinishedLots = "affected_finished_lots"
        case quarantinedSsccPallets = "quarantined_sscc_pallets"
        case quarantinedWarehouseTotes = "quarantined_warehouse_totes"
        case inTransitDriverDropBlocks = "in_transit_driver_drop_blocks"
        case retailerReversePickupOrders = "retailer_reverse_pickup_orders"
        case impactedSoliqFacturas = "impacted_soliq_facturas"
    }
}

public struct ActionExecutionRequest: Codable, Sendable {
    public let actionName: String
    public let actionType: String
    public let scenarioId: String?
    public let actorId: String
    public let actorRole: String
    public let targetObjectType: String
    public let targetObjectId: String
    public let idempotencyKey: String
    public let cellId: String
    public let supplierId: String

    public init(
        actionName: String,
        actionType: String,
        scenarioId: String? = nil,
        actorId: String,
        actorRole: String,
        targetObjectType: String,
        targetObjectId: String,
        idempotencyKey: String,
        cellId: String = "cell-uz",
        supplierId: String
    ) {
        self.actionName = actionName
        self.actionType = actionType
        self.scenarioId = scenarioId
        self.actorId = actorId
        self.actorRole = actorRole
        self.targetObjectType = targetObjectType
        self.targetObjectId = targetObjectId
        self.idempotencyKey = idempotencyKey
        self.cellId = cellId
        self.supplierId = supplierId
    }

    enum CodingKeys: String, CodingKey {
        case actionName = "action_name"
        case actionType = "action_type"
        case scenarioId = "scenario_id"
        case actorId = "actor_id"
        case actorRole = "actor_role"
        case targetObjectType = "target_object_type"
        case targetObjectId = "target_object_id"
        case idempotencyKey = "idempotency_key"
        case cellId = "cell_id"
        case supplierId = "supplier_id"
    }
}

public struct ActionExecutionRecord: Codable, Sendable {
    public let executionId: String
    public let actionName: String
    public let actionType: String
    public let scenarioId: String?
    public let actorId: String
    public let actorRole: String
    public let targetObjectType: String
    public let targetObjectId: String
    public let idempotencyKey: String
    public let executedAt: String
    public let status: String
    public let cellId: String
    public let supplierId: String

    enum CodingKeys: String, CodingKey {
        case executionId = "execution_id"
        case actionName = "action_name"
        case actionType = "action_type"
        case scenarioId = "scenario_id"
        case actorId = "actor_id"
        case actorRole = "actor_role"
        case targetObjectType = "target_object_type"
        case targetObjectId = "target_object_id"
        case idempotencyKey = "idempotency_key"
        case executedAt = "executed_at"
        case status
        case cellId = "cell_id"
        case supplierId = "supplier_id"
    }
}

public struct ScenarioPatchRequest: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let objectType: String
    public let objectId: String
    public let operation: String

    public init(cellId: String = "cell-uz", supplierId: String, objectType: String, objectId: String, operation: String) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.objectType = objectType
        self.objectId = objectId
        self.operation = operation
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case objectType = "object_type"
        case objectId = "object_id"
        case operation
    }
}

public struct ScenarioCommitResult: Codable, Sendable {
    public let scenarioId: String
    public let committedObjects: Int
    public let conflicts: [ScenarioWatermarkConflict]

    enum CodingKeys: String, CodingKey {
        case scenarioId = "scenario_id"
        case committedObjects = "committed_objects"
        case conflicts
    }
}

public struct ScenarioWatermarkConflict: Codable, Sendable {
    public let cellId: String
    public let objectType: String
    public let objectId: String
    public let baseVersion: Int64
    public let liveVersion: Int64

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case objectType = "object_type"
        case objectId = "object_id"
        case baseVersion = "base_version"
        case liveVersion = "live_version"
    }
}
