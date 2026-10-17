import Foundation
import SwiftData

@Model
final class PendingOrder {
    var endpoint: String = "/v1/checkout/unified"
    var method: String = "POST"
    var payloadJson: String
    var idempotencyKey: String = ""
    var createdAt: Date
    var retryCount: Int
    var lastError: String?

    init(
        payloadJson: String,
        endpoint: String = "/v1/checkout/unified",
        method: String = "POST",
        idempotencyKey: String
    ) {
        self.endpoint = endpoint
        self.method = method
        self.payloadJson = payloadJson
        self.idempotencyKey = idempotencyKey
        self.createdAt = Date()
        self.retryCount = 0
        self.lastError = nil
    }
}
