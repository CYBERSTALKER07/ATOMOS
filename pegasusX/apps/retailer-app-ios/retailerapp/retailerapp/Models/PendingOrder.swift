import Foundation
import SwiftData

@Model
final class PendingOrder {
    static let statusPending = "PENDING"
    static let statusAwaitingOwnerApproval = "AWAITING_OWNER_APPROVAL"
    static let statusInProgress = "IN_PROGRESS"
    static let statusRateLimited = "RATE_LIMITED"
    static let statusAuthRequired = "AUTH_REQUIRED"
    static let statusCompleted = "COMPLETED"
    static let statusFailed = "FAILED"

    var endpoint: String = "/v1/checkout/unified"
    var method: String = "POST"
    var payloadJson: String
    var idempotencyKey: String = ""
    var createdAt: Date
    var retryCount: Int
    var lastError: String?
    var status: String = "PENDING"

    init(
        payloadJson: String,
        endpoint: String = "/v1/checkout/unified",
        method: String = "POST",
        idempotencyKey: String,
        status: String = PendingOrder.statusPending
    ) {
        self.endpoint = endpoint
        self.method = method
        self.payloadJson = payloadJson
        self.idempotencyKey = idempotencyKey
        self.createdAt = Date()
        self.retryCount = 0
        self.lastError = nil
        self.status = status
    }
}
