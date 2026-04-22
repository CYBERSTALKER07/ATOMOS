import Foundation
import SwiftData

@Model
final class PendingOrder {
    var payloadJson: String
    var createdAt: Date
    var retryCount: Int

    init(payloadJson: String) {
        self.payloadJson = payloadJson
        self.createdAt = Date()
        self.retryCount = 0
    }
}
