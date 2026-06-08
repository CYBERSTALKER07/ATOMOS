//
//  OfflineDeliveryStore.swift
//  driverappios
//

import Foundation
import SwiftData

/// CRUD operations for the offline delivery queue backed by SwiftData.
@MainActor
final class OfflineDeliveryStore {

    private let modelContext: ModelContext

    init(modelContext: ModelContext) {
        self.modelContext = modelContext
    }

    // MARK: - Create

    func enqueue(orderId: String, signature: String, status: String) {
        let delivery = OfflineDelivery(
            orderId: orderId,
            signature: signature,
            timestamp: Date().timeIntervalSince1970 * 1000,
            status: status
        )
        modelContext.insert(delivery)
        try? modelContext.save()
    }

    // MARK: - Read

    func fetchPending() -> [OfflineDelivery] {
        let predicate = #Predicate<OfflineDelivery> { $0.syncStatus == "PENDING" }
        let descriptor = FetchDescriptor<OfflineDelivery>(predicate: predicate)
        return (try? modelContext.fetch(descriptor)) ?? []
    }

    func fetchAll() -> [OfflineDelivery] {
        let descriptor = FetchDescriptor<OfflineDelivery>()
        return (try? modelContext.fetch(descriptor)) ?? []
    }

    // MARK: - Delete

    func delete(_ delivery: OfflineDelivery) {
        modelContext.delete(delivery)
        try? modelContext.save()
    }

    func deleteSynced(orderIds: [String]) {
        let pending = fetchPending()
        for delivery in pending where orderIds.contains(delivery.orderId) {
            modelContext.delete(delivery)
        }
        try? modelContext.save()
    }
}
