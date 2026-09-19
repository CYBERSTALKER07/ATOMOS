import Foundation
import SwiftData
import PegasusKit

@Model
final class QueuedDriverAction {
    @Attribute(.unique) var id: String
    var endpoint: String
    var method: String
    var bodyJSON: String
    var priority: Int
    var clientTimestampIso: String
    var attemptCount: Int
    var lastError: String
    var status: String
    var orderId: String
    var createdAt: Double
    /// Capture-time GPS — replay on flush; never replace with live location (§8.8).
    var capturedLat: Double?
    var capturedLng: Double?
    var capturedAtMs: Double?

    init(
        id: String,
        endpoint: String,
        method: String = "POST",
        bodyJSON: String,
        priority: Int,
        clientTimestampIso: String,
        orderId: String = "",
        attemptCount: Int = 0,
        lastError: String = "",
        status: String = DriverOfflineActionStatus.pending.rawValue,
        createdAt: Double = Date().timeIntervalSince1970 * 1000,
        capturedLat: Double? = nil,
        capturedLng: Double? = nil,
        capturedAtMs: Double? = nil
    ) {
        self.id = id
        self.endpoint = endpoint
        self.method = method
        self.bodyJSON = bodyJSON
        self.priority = priority
        self.clientTimestampIso = clientTimestampIso
        self.attemptCount = attemptCount
        self.lastError = lastError
        self.status = status
        self.orderId = orderId
        self.createdAt = createdAt
        self.capturedLat = capturedLat
        self.capturedLng = capturedLng
        self.capturedAtMs = capturedAtMs
    }

    func bodyJSONForFlush() -> String {
        QueuedMutationRecord(
            id: id,
            endpoint: endpoint,
            method: method,
            payloadJSON: bodyJSON,
            idempotencyKey: id,
            capturedLat: capturedLat,
            capturedLng: capturedLng,
            capturedAtMs: capturedAtMs,
            clientTimestampIso: clientTimestampIso,
            attemptCount: attemptCount,
            lastError: lastError,
            status: status,
            orderId: orderId,
            priority: priority,
            createdAtMs: createdAt
        ).payloadJSONWithCapturedCoords()
    }
}
