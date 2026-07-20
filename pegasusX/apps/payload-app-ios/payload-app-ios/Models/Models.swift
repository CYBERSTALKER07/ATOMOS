//
//  Models.swift
//  payload-app-ios
//
//  Wire-format DTOs for every endpoint the Expo payload-terminal calls.
//  snake_case fields → CodingKeys; APIClient uses .convertFromSnakeCase by default.
//

import Foundation

// MARK: - Auth

struct LoginRequest: Encodable {
    var phone: String = ""
    var pin: String = ""
    var idToken: String = ""

    enum CodingKeys: String, CodingKey {
        case phone, pin
        case idToken = "id_token"
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        if !phone.isEmpty { try container.encode(phone, forKey: .phone) }
        if !pin.isEmpty { try container.encode(pin, forKey: .pin) }
        if !idToken.isEmpty { try container.encode(idToken, forKey: .idToken) }
    }
}

struct LoginResponse: Decodable {
    let token: String
    let refreshToken: String?
    let workerId: String
    let supplierId: String
    let role: String
    let name: String
    let warehouseId: String
    let warehouseName: String
    let warehouseLat: Double
    let warehouseLng: Double
    let firebaseToken: String?
}

struct RefreshTokenRequest: Encodable {
    let refreshToken: String

    enum CodingKeys: String, CodingKey {
        case refreshToken = "refresh_token"
    }
}

struct CatalogBarcodeLookup: Decodable {
    let productId: String?
    let skuId: String?
    let name: String?
    let barcode: String?
}

struct RefreshTokenResponse: Decodable {
    let token: String
    let refreshToken: String?
}

// MARK: - Trucks / Manifest

/// Wire format: bare JSON array of {id, label, license_plate, vehicle_class}.
struct Truck: Decodable, Identifiable {
    let id: String
    let label: String?
    let licensePlate: String?
    let vehicleClass: String?
}

struct LiveOrderItem: Decodable, Identifiable {
    let lineItemId: String
    let skuId: String
    let skuName: String
    let quantity: Int
    let unitPrice: Int64?
    let status: String?
    var id: String { lineItemId }
}

struct LiveOrder: Decodable, Identifiable {
    let orderId: String
    let retailerId: String?
    let amount: Int64?
    let paymentGateway: String?
    let state: String
    let routeId: String?
    let warehouseId: String?
    let items: [LiveOrderItem]?
    var id: String { orderId }
}

struct Manifest: Decodable, Identifiable {
    let manifestId: String
    let truckId: String?
    let driverId: String?
    let state: String   // DRAFT | LOADING | SEALED | DISPATCHED | COMPLETED
    let totalVolumeVu: Double?
    let maxVolumeVu: Double?
    let stopCount: Int?
    let regionCode: String?
    let sealedAt: String?
    let dispatchedAt: String?
    let createdAt: String?
    /// Hydrated by the detail endpoint only — Phase 4 wires this.
    let orders: [LiveOrder]?
    let overflowCount: Int?
    var id: String { manifestId }
}

struct ManifestsResponse: Decodable {
    let manifests: [Manifest]
}

// MARK: - Seal / Exception

/// Backend: order/service.go::PayloadSealRequest → {order_id, terminal_id, manifest_cleared}.
/// terminal_id is the active vehicle/truck id (Expo passes activeTruck).
struct SealOrderRequest: Encodable {
    let orderId: String
    let terminalId: String
    let manifestCleared: Bool
    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case terminalId = "terminal_id"
        case manifestCleared = "manifest_cleared"
    }
}

struct SealOrderResponse: Decodable {
    let status: String?
    let dispatchCode: String
    let orderId: String
}

struct SealManifestResponse: Decodable {
    let status: String?
    let stopCount: Int?
    let volumeVu: Double?
    let maxVu: Double?
}

struct SealCompletedManifestsRequest: Encodable {
    let manifestIds: [String]
    enum CodingKeys: String, CodingKey { case manifestIds = "manifest_ids" }
}

struct SealCompletedManifestsResponse: Decodable {
    let status: String?
    let sealedCount: Int?
    let results: [SealCompletedManifestResult]?
    enum CodingKeys: String, CodingKey {
        case status
        case sealedCount = "sealed_count"
        case results
    }
}

struct SealCompletedManifestResult: Decodable {
    let manifestId: String?
    let status: String?
    let explain: StatusExplain?
    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case status
        case explain
    }
}

struct ManifestExceptionRequest: Encodable {
    let manifestId: String
    let orderId: String
    let reason: String  // OVERFLOW | DAMAGED | MANUAL
    let metadata: String

    enum CodingKeys: String, CodingKey {
        case manifestId = "manifest_id"
        case orderId = "order_id"
        case reason, metadata
    }
}

struct ManifestExceptionResponse: Decodable {
    let status: String?
    let escalated: Bool?
    let overflowCount: Int?
}

struct ManifestExceptionRow: Decodable, Identifiable {
    let exceptionId: String
    let manifestId: String
    let orderId: String
    let reason: String
    let attemptCount: Int
    let escalated: Bool
    let createdAt: String
    var id: String { exceptionId }
    enum CodingKeys: String, CodingKey {
        case reason, escalated
        case exceptionId = "exception_id"
        case manifestId = "manifest_id"
        case orderId = "order_id"
        case attemptCount = "attempt_count"
        case createdAt = "created_at"
    }
}

struct ManifestExceptionsResponse: Decodable {
    let exceptions: [ManifestExceptionRow]
}

// MARK: - Inject / Reassign

struct InjectOrderRequest: Encodable {
    let orderId: String
    enum CodingKeys: String, CodingKey { case orderId = "order_id" }
}

struct RecommendReassignRequest: Encodable {
    let orderId: String
    enum CodingKeys: String, CodingKey { case orderId = "order_id" }
}

struct TruckRecommendation: Decodable, Identifiable {
    let driverId: String
    let driverName: String?
    let vehicleId: String?
    let vehicleClass: String?
    let licensePlate: String?
    let maxVolumeVu: Double?
    let usedVolumeVu: Double?
    let freeVolumeVu: Double?
    let distanceKm: Double?
    let orderCount: Int?
    let truckStatus: String?
    let score: Double?
    let recommendation: String?
    var id: String { driverId }
}

struct RecommendReassignResponse: Decodable {
    let orderId: String?
    let retailerName: String?
    let orderVolumeVu: Double?
    let currentDriver: String?
    let recommendations: [TruckRecommendation]
}

/// In this codebase RouteId == DriverId; payload terminal sends the recommended driver_id.
struct FleetReassignRequest: Encodable {
    let orderIds: [String]
    let newRouteId: String
    enum CodingKeys: String, CodingKey {
        case orderIds = "order_ids"
        case newRouteId = "new_route_id"
    }
}

struct ReassignConflict: Decodable, Identifiable {
    let orderId: String
    let reason: String?
    var id: String { orderId }
}

struct FleetReassignResponse: Decodable {
    let conflicts: [ReassignConflict]?
    let total: Int?
    let reassigned: Int?
    let newRouteId: String?
}

// MARK: - Missing Items (Edge 33)

struct MissingItemEntry: Encodable {
    let lineItemId: String
    let quantity: Int
    enum CodingKeys: String, CodingKey {
        case lineItemId = "line_item_id"
        case quantity
    }
}

struct MissingItemsRequest: Encodable {
    let orderId: String
    let missingItems: [MissingItemEntry]
    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case missingItems = "missing_items"
    }
}

// MARK: - FCM

struct DeviceTokenRequest: Encodable {
    let token: String
    let platform: String   // "IOS"
}

// MARK: - Pulse / explain / handoff

struct PulseEvent: Decodable, Identifiable {
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

struct PulseResponse: Decodable {
    let events: [PulseEvent]
    let fetchedAt: String

    enum CodingKeys: String, CodingKey {
        case events
        case fetchedAt = "fetched_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        events = try c.decodeIfPresent([PulseEvent].self, forKey: .events) ?? []
        fetchedAt = try c.decodeIfPresent(String.self, forKey: .fetchedAt) ?? ""
    }
}

struct StatusExplain: Decodable, Equatable {
    let code: String
    let title: String
    let summary: String
    let nextSteps: [String]?
    let deepLink: String?
    let recoverable: Bool

    enum CodingKeys: String, CodingKey {
        case code, title, summary, recoverable
        case nextSteps = "next_steps"
        case deepLink = "deep_link"
    }
}

struct HandoffCardMetadata: Decodable, Equatable {
    let kind: String
    let title: String
    let subtitle: String?
    let primaryCta: String?
    let primaryLink: String?
    let entityType: String?
    let entityId: String?
    let fields: [String: String]?

    enum CodingKeys: String, CodingKey {
        case kind, title, subtitle, fields
        case primaryCta = "primary_cta"
        case primaryLink = "primary_link"
        case entityType = "entity_type"
        case entityId = "entity_id"
    }
}

struct ApiErrorBody: Decodable {
    let error: String?
    let detail: String?
    let message: String?
    let explain: StatusExplain?
}

enum ApiExplainParser {
    static func parse(from data: Data) -> (message: String, explain: StatusExplain?)? {
        guard let body = try? JSONDecoder().decode(ApiErrorBody.self, from: data) else { return nil }
        let message = body.message ?? body.detail ?? body.error ?? body.explain?.summary
        guard let message, !message.isEmpty else { return nil }
        return (message, body.explain)
    }
}

// MARK: - Notifications

/// Wire shape: notifications/inbox.go::NotificationItem.
/// `read_at` is null when unread (RFC3339 string when read).
struct NotificationItem: Decodable, Identifiable {
    let notificationId: String
    let type: String
    let title: String
    let body: String
    let payload: String?
    let channel: String
    let readAt: String?
    let createdAt: String
    let handoffMetadata: HandoffCardMetadata?
    var id: String { notificationId }
    var isUnread: Bool { (readAt ?? "").isEmpty }

    enum CodingKeys: String, CodingKey {
        case type, title, body, payload, channel
        case notificationId = "notification_id"
        case readAt = "read_at"
        case createdAt = "created_at"
        case handoffMetadata = "handoff_metadata"
    }
}

struct NotificationsResponse: Decodable {
    let notifications: [NotificationItem]
    let unreadCount: Int
    let total: Int
    let limit: Int
}

struct MarkReadRequest: Encodable {
    let notificationIds: [String]?
    let markAll: Bool?
    enum CodingKeys: String, CodingKey {
        case notificationIds = "notification_ids"
        case markAll = "mark_all"
    }
}

struct ClientPolicyResponse: Decodable {
    let role: String
    let outdated: Bool
    let forceUpdate: Bool
    let updateDeferred: Bool?
    let minimumVersion: String
    let recommendedVersion: String?
    let updateURL: String?
    let deferReason: String?

    enum CodingKeys: String, CodingKey {
        case role
        case outdated
        case forceUpdate = "force_update"
        case updateDeferred = "update_deferred"
        case minimumVersion = "minimum_version"
        case recommendedVersion = "recommended_version"
        case updateURL = "update_url"
        case deferReason = "defer_reason"
    }
}

// MARK: - Inbound returns gate

struct InboundReturnRow: Decodable, Identifiable {
    var id: String { returnId }
    let returnId: String
    let orderId: String
    let skuId: String
    let productName: String
    let barcode: String?
    let expectedQty: Int
    let receivedQty: Int
    let reason: String
    let physicalStatus: String
    let driverName: String
    let suggestedDisposition: String

    enum CodingKeys: String, CodingKey {
        case returnId = "return_id"
        case orderId = "order_id"
        case skuId = "sku_id"
        case productName = "product_name"
        case barcode
        case expectedQty = "expected_qty"
        case receivedQty = "received_qty"
        case reason
        case physicalStatus = "physical_status"
        case driverName = "driver_name"
        case suggestedDisposition = "suggested_disposition"
    }
}

struct InboundReturnListResponse: Decodable {
    let data: [InboundReturnRow]
}

struct InboundSessionResponse: Decodable {
    let sessionId: String
}

struct InboundScanResponse: Decodable {
    let matched: Bool
    let returnId: String?
    let variance: Bool
    let message: String?
}

struct InboundHistoryResponse: Decodable {
    let data: [InboundReturnRow]
}

// MARK: - WebSocket frame

/// Backend `kafka/notification_dispatcher.go` pushes a flat
/// `{type, title, body, channel}` envelope. Treat any frame with title or
/// body as a notification regardless of `type` literal.
struct WsMessage: Decodable {
    let type: String?
    let title: String?
    let body: String?
    let channel: String?
    let manifestId: String?
    let warehouseId: String?
    let reason: String?
    let timestamp: String?

    enum CodingKeys: String, CodingKey {
        case type, title, body, channel, reason, timestamp
        case manifestId = "manifest_id"
        case warehouseId = "warehouse_id"
    }
}

// MARK: - Offline queue (for inject-order while WebSocket is disconnected)

struct QueuedAction: Codable, Identifiable {
    let id: String
    let endpoint: String
    let method: String
    let body: String
    let createdAt: Double
    enum CodingKeys: String, CodingKey {
        case id, endpoint, method, body
        case createdAt = "created_at"
    }
}

// MARK: - Generic

struct StatusResponse: Decodable { let status: String? }
