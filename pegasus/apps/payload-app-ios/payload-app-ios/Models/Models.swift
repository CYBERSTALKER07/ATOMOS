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
    let phone: String
    let pin: String
}

struct LoginResponse: Decodable {
    let token: String
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
    var id: String { notificationId }
    var isUnread: Bool { (readAt ?? "").isEmpty }
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
