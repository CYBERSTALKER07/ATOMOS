import Foundation
import SwiftData

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
        createdAt: Double = Date().timeIntervalSince1970 * 1000
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
    }
}
