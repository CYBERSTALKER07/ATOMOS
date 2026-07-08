//
//  SyncService.swift
//  driverappios
//

import Foundation

// MARK: - Sync Result

struct SyncResult: Codable {
    let status: String
    let processed: [String]   // order IDs confirmed
    let skipped: Int
}

// MARK: - Sync Delivery DTO (Sendable snapshot of OfflineDelivery)

struct SyncDeliveryDTO: Sendable {
    let orderId: String
    let signature: String
    let timestamp: Double
    let status: String
}

// MARK: - Protocol

protocol SyncServiceProtocol {
    /// POST /v1/sync/batch { "driver_id", "deliveries": [...] }
    func uploadBatch(driverId: String, deliveries: [SyncDeliveryDTO], bearerToken: String) async throws -> SyncResult
}

