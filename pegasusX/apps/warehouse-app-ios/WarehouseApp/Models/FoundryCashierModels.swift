//
//  FoundryCashierModels.swift
//  WarehouseAppIOS
//

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

    public init(
        sessionId: String,
        supplierId: String,
        depotId: String,
        driverId: String,
        routeId: String,
        bagSealBarcode: String,
        attemptNumber: Int64,
        countedCashMinor: Int64,
        expectedCashMinor: Int64? = nil,
        varianceMinor: Int64? = nil,
        status: String,
        clearanceTokenHash: String? = nil,
        microSkimmingFlagged: Bool? = nil,
        createdAt: String,
        resolvedAt: String? = nil
    ) {
        self.sessionId = sessionId
        self.supplierId = supplierId
        self.depotId = depotId
        self.driverId = driverId
        self.routeId = routeId
        self.bagSealBarcode = bagSealBarcode
        self.attemptNumber = attemptNumber
        self.countedCashMinor = countedCashMinor
        self.expectedCashMinor = expectedCashMinor
        self.varianceMinor = varianceMinor
        self.status = status
        self.clearanceTokenHash = clearanceTokenHash
        self.microSkimmingFlagged = microSkimmingFlagged
        self.createdAt = createdAt
        self.resolvedAt = resolvedAt
    }

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
        terminalId: String = "POS-TASH-01",
        cardLastFour: String = "8821",
        authCode: String,
        paymentNetwork: String = "HUMO",
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

public struct DepotVaultLedgerEntry: Codable, Sendable, Identifiable {
    public var id: String { vaultEntryId }
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

    public init(
        vaultEntryId: String,
        depotId: String,
        supplierId: String,
        sessionId: String? = nil,
        citManifestId: String? = nil,
        entryType: String,
        amountMinor: Int64,
        runningBalanceMinor: Int64 = 0,
        courierBadgeId: String? = nil,
        sealBarcode: String? = nil,
        createdAt: String
    ) {
        self.vaultEntryId = vaultEntryId
        self.depotId = depotId
        self.supplierId = supplierId
        self.sessionId = sessionId
        self.citManifestId = citManifestId
        self.entryType = entryType
        self.amountMinor = amountMinor
        self.runningBalanceMinor = runningBalanceMinor
        self.courierBadgeId = courierBadgeId
        self.sealBarcode = sealBarcode
        self.createdAt = createdAt
    }

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

    public init(
        legalEntityTin: String,
        supplierId: String,
        calendarDate: String,
        cashMinor: Int64
    ) {
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

    public init(
        legalEntityTin: String,
        accumulatedCashMinor: Int64,
        ceilingMinor: Int64
    ) {
        self.legalEntityTin = legalEntityTin
        self.accumulatedCashMinor = accumulatedCashMinor
        self.ceilingMinor = ceilingMinor
    }

    enum CodingKeys: String, CodingKey {
        case legalEntityTin = "legal_entity_tin"
        case accumulatedCashMinor = "accumulated_cash_minor"
        case ceilingMinor = "ceiling_minor"
    }
}

// MARK: - Role 9: Yard Gate Check Event

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
