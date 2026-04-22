//
//  OfflineDelivery.swift
//  driverappios
//

import Foundation
import SwiftData

@Model
class OfflineDelivery {
    var orderId: String
    var signature: String        // SHA-256 hex
    var timestamp: Double        // Unix ms
    var status: String           // "DELIVERED" | "REJECTED_DAMAGED"
    var syncStatus: String       // "PENDING" — deleted after sync

    init(orderId: String, signature: String, timestamp: Double, status: String) {
        self.orderId = orderId
        self.signature = signature
        self.timestamp = timestamp
        self.status = status
        self.syncStatus = "PENDING"
    }
}
