import Foundation

struct EmergencyTransferRequest: Encodable {
    let totalVolumeVu: Double
    let notes: String?

    enum CodingKeys: String, CodingKey {
        case totalVolumeVu = "total_volume_vu"
        case notes
    }
}

struct ForceReceiveRequest: Encodable {
    let factoryId: String?
    let totalVolumeVu: Double
    let notes: String?

    enum CodingKeys: String, CodingKey {
        case factoryId = "factory_id"
        case totalVolumeVu = "total_volume_vu"
        case notes
    }
}

struct TransferMutationResponse: Decodable {
    let transferId: String
    let state: String
    let notes: String?

    enum CodingKeys: String, CodingKey {
        case transferId = "transfer_id"
        case state, notes
    }
}

struct ReplenishmentInsight: Decodable, Identifiable {
    let id: String
    let warehouseId: String
    let warehouseName: String
    let productId: String
    let productName: String
    let urgency: String
    let currentStock: Int64
    let avgDailyVelocity: Double
    let daysUntilStockout: Int
    let reorderQuantity: Int64
    let status: String
    let createdAt: String
    let reasonCode: String?
    let demandBreakdown: [String: AnyCodable]?

    enum CodingKeys: String, CodingKey {
        case id
        case warehouseId = "warehouse_id"
        case warehouseName = "warehouse_name"
        case productId = "product_id"
        case productName = "product_name"
        case urgency
        case currentStock = "current_stock"
        case avgDailyVelocity = "avg_daily_velocity"
        case daysUntilStockout = "days_until_stockout"
        case reorderQuantity = "reorder_quantity"
        case status
        case createdAt = "created_at"
        case reasonCode = "reason_code"
        case demandBreakdown = "demand_breakdown"
    }
}

struct ReplenishmentInsightsResponse: Decodable {
    let insights: [ReplenishmentInsight]
    let data: [ReplenishmentInsight]?

    var rows: [ReplenishmentInsight] {
        if !insights.isEmpty { return insights }
        return data ?? []
    }
}

struct ReplenishmentInsightActionResponse: Decodable {
    let insightId: String
    let status: String
    let transferId: String?

    enum CodingKeys: String, CodingKey {
        case insightId = "insight_id"
        case status
        case transferId = "transfer_id"
    }
}

struct DispatchSettingsResponse: Decodable {
    let warehouseId: String
    let autoDispatchEnabled: Bool

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case autoDispatchEnabled = "auto_dispatch_enabled"
    }
}

struct DispatchSettingsPatchRequest: Encodable {
    let autoDispatchEnabled: Bool

    enum CodingKeys: String, CodingKey {
        case autoDispatchEnabled = "auto_dispatch_enabled"
    }
}

struct OpsFinancialsResponse: Decodable {
    let warehouseId: String
    let period: String
    let currency: String
    let totalRevenue: Int64
    let completedOrders: Int64
    let avgOrderValue: Int64
    let platformFee: Int64
    let netPayout: Int64
    let cashPending: Int64
    let cashCollected: Int64

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case period, currency
        case totalRevenue = "total_revenue"
        case completedOrders = "completed_orders"
        case avgOrderValue = "avg_order_value"
        case platformFee = "platform_fee"
        case netPayout = "net_payout"
        case cashPending = "cash_pending"
        case cashCollected = "cash_collected"
    }
}

struct WarehouseOrderMutationRequest: Encodable {
    let reason: String?
}

struct WarehouseProposeDeliveryRequest: Encodable {
    let proposedDeliveryDate: String
    let reason: String

    enum CodingKeys: String, CodingKey {
        case proposedDeliveryDate = "proposed_delivery_date"
        case reason
    }
}

struct WarehouseOrderMutationResponse: Decodable {
    let orderId: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case status
    }
}

struct RefreshTokenRequest: Encodable {
    let refreshToken: String

    enum CodingKeys: String, CodingKey {
        case refreshToken = "refresh_token"
    }
}

struct RouteCoordinateWire: Decodable {
    let lat: Double
    let lng: Double
}

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

struct WarehouseDriverLocationWire: Decodable {
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

    var resolvedLatitude: Double { latitude != 0 ? latitude : lat }
    var resolvedLongitude: Double { longitude != 0 ? longitude : lng }
}

struct WarehouseFleetLiveRoute: Decodable, Identifiable {
    var id: String { manifestId }
    let manifestId: String
    let routeId: String
    let driverId: String
    let driverName: String?
    let manifestState: String
    let routeGeometry: RouteGeometryWire?
    let driverLocation: WarehouseDriverLocationWire?
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

struct WarehouseFleetLiveMapResponse: Decodable {
    let routes: [WarehouseFleetLiveRoute]
    let warehouseId: String
    let fetchedAt: String

    enum CodingKeys: String, CodingKey {
        case routes
        case warehouseId = "warehouse_id"
        case fetchedAt = "fetched_at"
    }
}

struct BroadcastTemplate: Decodable, Identifiable {
    let id: String
    let category: String
    let title: String
    let body: String
    let defaultRole: String
    let scope: String
    let source: String?
    let warehouseId: String?
    let placeholderKeys: [String]?

    enum CodingKeys: String, CodingKey {
        case id, category, title, body, scope, source
        case defaultRole = "default_role"
        case warehouseId = "warehouse_id"
        case placeholderKeys = "placeholder_keys"
    }
}

struct BroadcastTemplatesResponse: Decodable {
    let templates: [BroadcastTemplate]
}

struct WarehouseBroadcastRequest: Encodable {
    let title: String
    let body: String
    let role: String?
}

struct WarehouseBroadcastResponse: Decodable {
    let status: String
    let warehouseId: String
    let supplierId: String

    enum CodingKeys: String, CodingKey {
        case status
        case warehouseId = "warehouse_id"
        case supplierId = "supplier_id"
    }
}

struct WarehouseBroadcastTemplateCreateRequest: Encodable {
    let title: String
    let body: String
    let defaultRole: String?
    let category: String?

    enum CodingKeys: String, CodingKey {
        case title, body, category
        case defaultRole = "default_role"
    }
}

struct BroadcastTemplateDeleteResponse: Decodable {
    let status: String
    let templateId: String

    enum CodingKeys: String, CodingKey {
        case status
        case templateId = "template_id"
    }
}

struct RetailerOverridePreview: Decodable {
    let retailersOnSkuCount: Int
    let activeOverrideCount: Int
    let catalogListPrice: Int64
    let marginDeltaPerUnit: Int64
    let marginEstimateLabel: String
    let affectedRetailerIds: [String]?
    let readOnly: Bool?

    enum CodingKeys: String, CodingKey {
        case retailersOnSkuCount = "retailers_on_sku_count"
        case activeOverrideCount = "active_override_count"
        case catalogListPrice = "catalog_list_price"
        case marginDeltaPerUnit = "margin_delta_per_unit"
        case marginEstimateLabel = "margin_estimate_label"
        case affectedRetailerIds = "affected_retailer_ids"
        case readOnly = "read_only"
    }
}

struct RetailerOverridePreviewRequest: Encodable {
    let retailerId: String?
    let productId: String?
    let skuId: String?
    let proposedPrice: Int64

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case productId = "product_id"
        case skuId = "sku_id"
        case proposedPrice = "proposed_price"
    }
}

struct WarehousePulseEvent: Decodable, Identifiable {
    let id: String
    let kind: String
    let title: String
    let description: String?
    let occurredAt: String
    let deepLink: String?
    let orderId: String?
    let manifestId: String?

    enum CodingKeys: String, CodingKey {
        case id, kind, title, description
        case occurredAt = "occurred_at"
        case deepLink = "deep_link"
        case orderId = "order_id"
        case manifestId = "manifest_id"
    }
}

struct WarehousePulseResponse: Decodable {
    let events: [WarehousePulseEvent]
    let fetchedAt: String

    enum CodingKeys: String, CodingKey {
        case events
        case fetchedAt = "fetched_at"
    }
}

enum HandoffPulseSupport {
    private static let handoffKinds: Set<String> = [
        "PREORDER",
        "ORDER_ACCEPTED",
        "ORDER_DISPATCHED",
        "MANIFEST_SEALED",
        "MANIFEST_DISPATCHED",
        "DISPATCH",
    ]

    static func filter(_ events: [WarehousePulseEvent]) -> [WarehousePulseEvent] {
        events.filter(isHandoff)
    }

    private static func isHandoff(_ event: WarehousePulseEvent) -> Bool {
        let haystack = "\(event.kind) \(event.title)".uppercased()
        if handoffKinds.contains(event.kind.uppercased()) { return true }
        return haystack.range(of: "PREORDER|ACCEPT|DISPATCH|SEAL|MANIFEST", options: .regularExpression) != nil
    }
}

struct RecommendReassignRequest: Encodable {
    let orderId: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
    }
}

struct TruckRecommendation: Decodable {
    let driverId: String
    let driverName: String
    let vehicleId: String
    let vehicleClass: String
    let licensePlate: String
    let score: Double
    let reason: String
    let toRoute: String

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case driverName = "driver_name"
        case vehicleId = "vehicle_id"
        case vehicleClass = "vehicle_class"
        case licensePlate = "license_plate"
        case score
        case reason
        case toRoute = "to_route"
    }
}

struct RecommendReassignResponse: Decodable {
    let orderId: String
    let retailerName: String
    let orderVolumeVu: Double
    let currentDriver: String
    let recommendations: [TruckRecommendation]

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case retailerName = "retailer_name"
        case orderVolumeVu = "order_volume_vu"
        case currentDriver = "current_driver"
        case recommendations
    }
}

struct ReassignOrderRequest: Encodable {
    let orderId: String
    let toDriverId: String
    let reason: String
    let toManifestId: String?
    let isPartial: Bool

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case toDriverId = "to_driver_id"
        case reason
        case toManifestId = "to_manifest_id"
        case isPartial = "is_partial"
    }
}
