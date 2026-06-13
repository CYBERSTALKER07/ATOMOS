import Foundation

// MARK: - Exceptions

struct SupplierExceptionRow: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let kind: String
    let status: String
    let retailerId: String?
    let note: String?
    let manifestId: String?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case kind, status
        case retailerId = "retailer_id"
        case note
        case manifestId = "manifest_id"
        case updatedAt = "updated_at"
    }
}

struct SupplierExceptionsResponse: Decodable {
    let exceptions: [SupplierExceptionRow]
}

// MARK: - Manifests

struct SupplierManifestRow: Decodable, Identifiable {
    var id: String { manifestId }
    let manifestId: String
    let status: String
    let state: String
    let ordersCount: Int
    let driverId: String?
    let driverName: String
    let vehicleId: String?
    let vehiclePlate: String?
    let truckId: String?
    let totalVu: Int64
    let totalVolumeVu: Double?
    let maxVolumeVu: Double?
    let stopCount: Int
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case status, state
        case ordersCount = "orders_count"
        case driverId = "driver_id"
        case driverName = "driver_name"
        case vehicleId = "vehicle_id"
        case vehiclePlate = "vehicle_plate"
        case truckId = "truck_id"
        case totalVu = "total_vu"
        case totalVolumeVu = "total_volume_vu"
        case maxVolumeVu = "max_volume_vu"
        case stopCount = "stop_count"
        case updatedAt = "updated_at"
    }
}

struct SupplierManifestsResponse: Decodable {
    let manifests: [SupplierManifestRow]
}

// MARK: - Dispatch

struct SupplierDispatchPreview: Decodable {
    let pendingCount: Int?
    let availableDriverCount: Int?
    let undispatchedOrderCount: Int?

    enum CodingKeys: String, CodingKey {
        case pendingCount = "pending_count"
        case availableDriverCount = "available_driver_count"
        case undispatchedOrderCount = "undispatched_orders"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        pendingCount = try container.decodeIfPresent(Int.self, forKey: .pendingCount)
        availableDriverCount = try container.decodeIfPresent(Int.self, forKey: .availableDriverCount)
        if let count = try? container.decodeIfPresent(Int.self, forKey: .undispatchedOrderCount) {
            undispatchedOrderCount = count
        } else if let array = try? container.decodeIfPresent([String].self, forKey: .undispatchedOrderCount) {
            undispatchedOrderCount = array.count
        } else {
            undispatchedOrderCount = nil
        }
    }
}

// MARK: - Pricing / topology / lanes

struct SupplierPricingRule: Decodable {
    let supplierId: String
    let baseMarkupBps: Int
    let retailerDiscountBps: Int
    let minMarginBps: Int
    let currency: String
    let ruleVersion: Int
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case baseMarkupBps = "base_markup_bps"
        case retailerDiscountBps = "retailer_discount_bps"
        case minMarginBps = "min_margin_bps"
        case currency
        case ruleVersion = "rule_version"
        case updatedAt = "updated_at"
    }
}

struct SupplierPricingRuleUpdateRequest: Encodable {
    let baseMarkupBps: Int
    let retailerDiscountBps: Int
    let minMarginBps: Int
    let currency: String?

    enum CodingKeys: String, CodingKey {
        case baseMarkupBps = "base_markup_bps"
        case retailerDiscountBps = "retailer_discount_bps"
        case minMarginBps = "min_margin_bps"
        case currency
    }
}

struct SupplierTopologyWarehouse: Decodable, Identifiable {
    var id: String { warehouseId }
    let warehouseId: String
    let name: String
    let lat: Double
    let lng: Double

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case name, lat, lng
    }
}

struct SupplierTopologyFactory: Decodable, Identifiable {
    var id: String { factoryId }
    let factoryId: String
    let name: String
    let lat: Double
    let lng: Double

    enum CodingKeys: String, CodingKey {
        case factoryId = "factory_id"
        case name, lat, lng
    }
}

struct SupplierTopologyResponse: Decodable {
    let supplierId: String
    let warehouses: [SupplierTopologyWarehouse]
    let factories: [SupplierTopologyFactory]
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case supplierId = "supplier_id"
        case warehouses, factories
        case updatedAt = "updated_at"
    }
}

struct SupplierSupplyLaneRow: Decodable, Identifiable {
    var id: String { laneId }
    let laneId: String
    let name: String
    let warehouseId: String
    let h3Cells: Int
    let drivers: Int
    let ordersToday: Int
    let capacity: Int
    let utilizationPct: Double

    enum CodingKeys: String, CodingKey {
        case laneId = "lane_id"
        case name
        case warehouseId = "warehouse_id"
        case h3Cells = "h3_cells"
        case drivers
        case ordersToday = "orders_today"
        case capacity
        case utilizationPct = "utilization_pct"
    }
}

struct SupplierSupplyLanesResponse: Decodable {
    let lanes: [SupplierSupplyLaneRow]
}

// MARK: - Shop closed / negotiations

struct ShopClosedAttemptRow: Decodable, Identifiable {
    var id: String { attemptId }
    let attemptId: String
    let orderId: String
    let driverId: String
    let retailerId: String
    let resolution: String
    let createdAt: String
    let updatedAt: String?

    enum CodingKeys: String, CodingKey {
        case attemptId = "attempt_id"
        case orderId = "order_id"
        case driverId = "driver_id"
        case retailerId = "retailer_id"
        case resolution
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct ShopClosedActiveResponse: Decodable {
    let data: [ShopClosedAttemptRow]
}

struct NegotiationProposalItem: Decodable {
    let skuId: String
    let originalQty: Int
    let proposedQty: Int

    enum CodingKeys: String, CodingKey {
        case skuId = "sku_id"
        case originalQty = "original_qty"
        case proposedQty = "proposed_qty"
    }
}

struct NegotiationProposalRow: Decodable, Identifiable {
    var id: String { proposalId }
    let proposalId: String
    let orderId: String
    let driverId: String
    let items: [NegotiationProposalItem]
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case proposalId = "proposal_id"
        case orderId = "order_id"
        case driverId = "driver_id"
        case items
        case createdAt = "created_at"
    }
}

struct NegotiationPendingResponse: Decodable {
    let data: [NegotiationProposalRow]
}

struct ShopClosedResolveRequest: Encodable {
    let attemptId: String
    let action: String

    enum CodingKeys: String, CodingKey {
        case attemptId = "attempt_id"
        case action
    }
}

struct NegotiationResolveRequest: Encodable {
    let proposalId: String
    let action: String
    let resolution: String?

    enum CodingKeys: String, CodingKey {
        case proposalId = "proposal_id"
        case action, resolution
    }
}

struct NegotiationResolveResponse: Decodable {
    let status: String
    let proposalId: String
    let orderId: String

    enum CodingKeys: String, CodingKey {
        case status
        case proposalId = "proposal_id"
        case orderId = "order_id"
    }
}

// MARK: - Fleet orders / treasury / ws

struct SupplierFleetOrderRow: Decodable, Identifiable {
    var id: String { orderId }
    let orderId: String
    let retailerId: String?
    let driverId: String?
    let status: String
    let state: String?
    let routeId: String?
    let totalMinor: Int64?
    let currency: String?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case id
        case orderId = "order_id"
        case retailerId = "retailer_id"
        case driverId = "driver_id"
        case status, state
        case routeId = "route_id"
        case totalMinor = "total_minor"
        case currency
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = try c.decodeIfPresent(String.self, forKey: .orderId)
            ?? c.decode(String.self, forKey: .id)
        retailerId = try c.decodeIfPresent(String.self, forKey: .retailerId)
        driverId = try c.decodeIfPresent(String.self, forKey: .driverId)
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state)
        routeId = try c.decodeIfPresent(String.self, forKey: .routeId)
        totalMinor = try c.decodeIfPresent(Int64.self, forKey: .totalMinor)
        currency = try c.decodeIfPresent(String.self, forKey: .currency)
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

// MARK: - Fleet live map

struct RouteGeometryWire: Decodable {
    let routeId: String?
    let encodedPolyline: String?
    let coordinates: [RouteCoordinateWire]
    let source: String
    let stopCount: Int?

    enum CodingKeys: String, CodingKey {
        case routeId = "route_id"
        case encodedPolyline = "encoded_polyline"
        case coordinates, source
        case stopCount = "stop_count"
    }
}

struct RouteCoordinateWire: Decodable {
    let lat: Double
    let lng: Double
}

struct SupplierDriverLocationWire: Decodable {
    let driverId: String
    let supplierId: String?
    let lat: Double
    let lng: Double
    let latitude: Double
    let longitude: Double
    let reportedAt: String
    let receivedAt: String
    let staleAfterSeconds: Int

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case supplierId = "supplier_id"
        case lat, lng, latitude, longitude
        case reportedAt = "reported_at"
        case receivedAt = "received_at"
        case staleAfterSeconds = "stale_after_seconds"
    }
}

struct SupplierFleetLiveRoute: Decodable, Identifiable {
    var id: String { manifestId }
    let manifestId: String
    let routeId: String
    let driverId: String
    let driverName: String?
    let manifestState: String
    let routeGeometry: RouteGeometryWire?
    let driverLocation: SupplierDriverLocationWire?
    let liveLocationAvailable: Bool
    let locationStale: Bool?

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case routeId = "route_id"
        case driverId = "driver_id"
        case driverName = "driver_name"
        case manifestState = "manifest_state"
        case routeGeometry = "route_geometry"
        case driverLocation = "driver_location"
        case liveLocationAvailable = "live_location_available"
        case locationStale = "location_stale"
    }
}

struct SupplierFleetLiveMapResponse: Decodable {
    let routes: [SupplierFleetLiveRoute]
    let fetchedAt: String

    enum CodingKeys: String, CodingKey {
        case routes
        case fetchedAt = "fetched_at"
    }
}

struct SupplierWsSessionResponse: Decodable {
    let token: String
    let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case token
        case expiresAt = "expires_at"
    }
}

struct PaymentLedgerEntry: Decodable, Identifiable {
    var id: String { ledgerEntryId }
    let ledgerEntryId: String
    let orderId: String?
    let gateway: String
    let entryType: String
    let amountMinor: Int64
    let currency: String
    let occurredAt: String

    enum CodingKeys: String, CodingKey {
        case ledgerEntryId = "ledger_entry_id"
        case orderId = "order_id"
        case gateway
        case entryType = "entry_type"
        case amountMinor = "amount_minor"
        case currency
        case occurredAt = "occurred_at"
    }
}

struct PaymentLedgerResponse: Decodable {
    let items: [PaymentLedgerEntry]
    let count: Int
    let limit: Int
    let supplierId: String

    enum CodingKeys: String, CodingKey {
        case items, count, limit
        case supplierId = "supplier_id"
    }
}

struct SupplierReplenishmentTriggerResponse: Decodable {
    let status: String
    let requestId: String
    let warehouseId: String

    enum CodingKeys: String, CodingKey {
        case status
        case requestId = "request_id"
        case warehouseId = "warehouse_id"
    }
}

struct SupplierActivityEvent: Decodable, Identifiable {
    let id: String
    let type: String
    let timestamp: String
    let description: String
    let orderId: String?
    let manifestId: String?

    enum CodingKeys: String, CodingKey {
        case id, type, timestamp, description
        case orderId = "order_id"
        case manifestId = "manifest_id"
    }
}

struct SupplierActivityResponse: Decodable {
    let events: [SupplierActivityEvent]
}

struct FleetDriverCreateRequest: Encodable {
    let name: String
    let phone: String
    let pin: String
    let homeNodeType: String
    let homeNodeId: String
    let vehicleId: String?
    let isActive: Bool?

    enum CodingKeys: String, CodingKey {
        case name, phone, pin
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
        case vehicleId = "vehicle_id"
        case isActive = "is_active"
    }
}

struct FleetVehicleCreateRequest: Encodable {
    let label: String?
    let licensePlate: String
    let homeNodeType: String
    let homeNodeId: String
    let isActive: Bool?

    enum CodingKeys: String, CodingKey {
        case label
        case licensePlate = "license_plate"
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
        case isActive = "is_active"
    }
}

struct SupplierProfileUpdateRequest: Encodable {
    let legalName: String?
    let contactName: String?
    let email: String?
    let phone: String?
    let country: String?

    enum CodingKeys: String, CodingKey {
        case legalName = "legal_name"
        case contactName = "contact_name"
        case email, phone, country
    }
}

// MARK: - Finance reconciliation

struct SettlementCurrencyTotal: Decodable {
    let currency: String
    let entryCount: Int
    let amountMinorTotal: Int64

    enum CodingKeys: String, CodingKey {
        case currency
        case entryCount = "entry_count"
        case amountMinorTotal = "amount_minor_total"
    }
}

struct SettlementAuthorityRow: Decodable, Identifiable {
    var id: String { "\(gateway)-\(entryType)-\(currency)" }
    let gateway: String
    let entryType: String
    let currency: String
    let entryCount: Int
    let amountMinorTotal: Int64
    let firstOccurredAt: String
    let lastOccurredAt: String

    enum CodingKeys: String, CodingKey {
        case gateway
        case entryType = "entry_type"
        case currency
        case entryCount = "entry_count"
        case amountMinorTotal = "amount_minor_total"
        case firstOccurredAt = "first_occurred_at"
        case lastOccurredAt = "last_occurred_at"
    }
}

struct SettlementAuthorityResponse: Decodable {
    let items: [SettlementAuthorityRow]
    let count: Int
    let supplierId: String
    let entryCountTotal: Int
    let totalsByCurrency: [SettlementCurrencyTotal]

    enum CodingKeys: String, CodingKey {
        case items, count
        case supplierId = "supplier_id"
        case entryCountTotal = "entry_count_total"
        case totalsByCurrency = "totals_by_currency"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        items = try c.decodeIfPresent([SettlementAuthorityRow].self, forKey: .items) ?? []
        count = try c.decodeIfPresent(Int.self, forKey: .count) ?? 0
        supplierId = try c.decodeIfPresent(String.self, forKey: .supplierId) ?? ""
        entryCountTotal = try c.decodeIfPresent(Int.self, forKey: .entryCountTotal) ?? 0
        totalsByCurrency = try c.decodeIfPresent([SettlementCurrencyTotal].self, forKey: .totalsByCurrency) ?? []
    }
}

struct ReconciliationMismatchRow: Decodable, Identifiable {
    var id: String { "\(gateway)-\(currency)" }
    let gateway: String
    let currency: String
    let netAmountMinor: Int64
    let creditAmountMinorTotal: Int64
    let debitAmountMinorTotal: Int64
    let entryCountTotal: Int
    let lastOccurredAt: String

    enum CodingKeys: String, CodingKey {
        case gateway, currency
        case netAmountMinor = "net_amount_minor"
        case creditAmountMinorTotal = "credit_amount_minor_total"
        case debitAmountMinorTotal = "debit_amount_minor_total"
        case entryCountTotal = "entry_count_total"
        case lastOccurredAt = "last_occurred_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        gateway = try c.decodeIfPresent(String.self, forKey: .gateway) ?? ""
        currency = try c.decodeIfPresent(String.self, forKey: .currency) ?? ""
        netAmountMinor = try c.decodeIfPresent(Int64.self, forKey: .netAmountMinor) ?? 0
        creditAmountMinorTotal = try c.decodeIfPresent(Int64.self, forKey: .creditAmountMinorTotal) ?? 0
        debitAmountMinorTotal = try c.decodeIfPresent(Int64.self, forKey: .debitAmountMinorTotal) ?? 0
        entryCountTotal = try c.decodeIfPresent(Int.self, forKey: .entryCountTotal) ?? 0
        lastOccurredAt = try c.decodeIfPresent(String.self, forKey: .lastOccurredAt) ?? ""
    }
}

// MARK: - AI recommendations

struct SupplierAIRecommendationEvidence: Decodable, Identifiable {
    var id: String { "\(label)-\(value)" }
    let label: String
    let value: String
    let href: String?
}

struct SupplierAIRecommendation: Decodable, Identifiable {
    var id: String { recommendationId }
    let recommendationId: String
    let aggregateId: String
    let aggregateType: String
    let action: String
    let status: String
    let score: Double
    let confidence: Double
    let source: String
    let explanation: String
    let reasonCodes: [String]
    let evidence: [SupplierAIRecommendationEvidence]
    let decision: String?
    let decisionNote: String?
    let decidedBy: String?
    let decidedAt: String?
    let generatedAt: String
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case recommendationId = "recommendation_id"
        case aggregateId = "aggregate_id"
        case aggregateType = "aggregate_type"
        case action, status, score, confidence, source, explanation
        case reasonCodes = "reason_codes"
        case evidence, decision
        case decisionNote = "decision_note"
        case decidedBy = "decided_by"
        case decidedAt = "decided_at"
        case generatedAt = "generated_at"
        case updatedAt = "updated_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        recommendationId = try c.decodeIfPresent(String.self, forKey: .recommendationId) ?? ""
        aggregateId = try c.decodeIfPresent(String.self, forKey: .aggregateId) ?? ""
        aggregateType = try c.decodeIfPresent(String.self, forKey: .aggregateType) ?? ""
        action = try c.decodeIfPresent(String.self, forKey: .action) ?? ""
        status = try c.decodeIfPresent(String.self, forKey: .status) ?? ""
        score = try c.decodeIfPresent(Double.self, forKey: .score) ?? 0
        confidence = try c.decodeIfPresent(Double.self, forKey: .confidence) ?? 0
        source = try c.decodeIfPresent(String.self, forKey: .source) ?? ""
        explanation = try c.decodeIfPresent(String.self, forKey: .explanation) ?? ""
        reasonCodes = try c.decodeIfPresent([String].self, forKey: .reasonCodes) ?? []
        evidence = try c.decodeIfPresent([SupplierAIRecommendationEvidence].self, forKey: .evidence) ?? []
        decision = try c.decodeIfPresent(String.self, forKey: .decision)
        decisionNote = try c.decodeIfPresent(String.self, forKey: .decisionNote)
        decidedBy = try c.decodeIfPresent(String.self, forKey: .decidedBy)
        decidedAt = try c.decodeIfPresent(String.self, forKey: .decidedAt)
        generatedAt = try c.decodeIfPresent(String.self, forKey: .generatedAt) ?? ""
        updatedAt = try c.decodeIfPresent(String.self, forKey: .updatedAt) ?? ""
    }
}

struct SupplierAIRecommendationsResponse: Decodable {
    let items: [SupplierAIRecommendation]
    let count: Int
    let limit: Int
    let status: String?
    let updatedAt: String

    enum CodingKeys: String, CodingKey {
        case items, count, limit, status
        case updatedAt = "updated_at"
    }
}

struct SupplierAIRecommendationDecisionRequest: Encodable {
    let recommendationId: String
    let decision: String
    let note: String?

    enum CodingKeys: String, CodingKey {
        case recommendationId = "recommendation_id"
        case decision, note
    }
}

struct SupplierAIRecommendationDecisionResponse: Decodable {
    let recommendation: SupplierAIRecommendation
}

struct ReconciliationMismatchResponse: Decodable {
    let items: [ReconciliationMismatchRow]
}

// MARK: - Org members

struct SupplierOrgMember: Decodable, Identifiable {
    var id: String { userId }
    let userId: String
    let name: String
    let phone: String
    let supplierRole: String
    let assignedWarehouseId: String?
    let assignedFactoryId: String?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case name, phone
        case supplierRole = "supplier_role"
        case assignedWarehouseId = "assigned_warehouse_id"
        case assignedFactoryId = "assigned_factory_id"
    }
}

struct SupplierOrgMembersResponse: Decodable {
    let items: [SupplierOrgMember]
}

struct SupplierOrgMemberCreateRequest: Encodable {
    let name: String
    let email: String?
    let phone: String
    let password: String
    let supplierRole: String
    let assignedWarehouseId: String?
    let assignedFactoryId: String?

    enum CodingKeys: String, CodingKey {
        case name, email, phone, password
        case supplierRole = "supplier_role"
        case assignedWarehouseId = "assigned_warehouse_id"
        case assignedFactoryId = "assigned_factory_id"
    }
}

struct ApproveEarlyCompleteRequest: Encodable {
    let driverId: String

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
    }
}
