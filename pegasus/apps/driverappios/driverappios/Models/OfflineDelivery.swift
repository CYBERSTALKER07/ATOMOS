//
//  OfflineDelivery.swift
//  driverappios
//

import Foundation
import SwiftData

@Model
class OfflineDelivery {
    var orderId: String
    var signature: String        // SHA-256 hex or scanned token
    var timestamp: Double        // Unix ms
    var status: String           // "DELIVERED" | "REJECTED_DAMAGED"
    var syncStatus: String       // "PENDING" — deleted after sync
    var deliverySessionId: String
    var action: String
    var seq: Int
    var idempotencyKey: String

    init(
        orderId: String,
        signature: String,
        timestamp: Double,
        status: String,
        deliverySessionId: String,
        action: String,
        seq: Int,
        idempotencyKey: String
    ) {
        self.orderId = orderId
        self.signature = signature
        self.timestamp = timestamp
        self.status = status
        self.syncStatus = "PENDING"
        self.deliverySessionId = deliverySessionId
        self.action = action
        self.seq = seq
        self.idempotencyKey = idempotencyKey
    }
}
