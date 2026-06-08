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

// MARK: - Stub Implementation

final class SyncServiceStub: SyncServiceProtocol {

    static let shared = SyncServiceStub()

    func uploadBatch(driverId: String, deliveries: [SyncDeliveryDTO], bearerToken: String) async throws -> SyncResult {
        try await Task.sleep(nanoseconds: 500_000_000)
        let ids = deliveries.map(\.orderId)
        return SyncResult(status: "OK", processed: ids, skipped: 0)
    }
}
