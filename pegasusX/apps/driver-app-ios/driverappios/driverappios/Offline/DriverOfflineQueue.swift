import Foundation
import SwiftData

@MainActor
final class DriverOfflineQueue {
    static let shared = DriverOfflineQueue()

    private var container: ModelContainer?
    private var migrated = false

    private init() {}

    func attach(container: ModelContainer) {
        self.container = container
        migrateLegacyDeliveriesIfNeeded()
    }

    var context: ModelContext? { container?.mainContext }

    func enqueue(
        endpoint: String,
        bodyJSON: String,
        idempotencyKey: String,
        orderId: String = "",
        clientTimestampIso: String = DriverOfflineActionCatalog.nowIso(),
        method: String = "POST"
    ) {
        guard let context else { return }
        let ep = DriverOfflineActionCatalog.normalize(endpoint)
        guard DriverOfflineActionCatalog.isOfflineEligible(ep) else { return }
        let action = QueuedDriverAction(
            id: idempotencyKey,
            endpoint: ep,
            method: method,
            bodyJSON: bodyJSON,
            priority: DriverOfflineActionCatalog.priority(for: ep),
            clientTimestampIso: clientTimestampIso,
            orderId: orderId.isEmpty ? (extractOrderId(bodyJSON) ?? "") : orderId
        )
        context.insert(action)
        try? context.save()
        print("[DriverOfflineQueue] enqueued \(ep) order=\(action.orderId)")
    }

    func enqueueJSONObject(
        endpoint: String,
        body: [String: Any],
        idempotencyKey: String,
        orderId: String = "",
        clientTimestampIso: String = DriverOfflineActionCatalog.nowIso()
    ) {
        var payload = body
        if payload["client_timestamp"] == nil {
            payload["client_timestamp"] = clientTimestampIso
        }
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let json = String(data: data, encoding: .utf8) else { return }
        enqueue(
            endpoint: endpoint,
            bodyJSON: json,
            idempotencyKey: idempotencyKey,
            orderId: orderId,
            clientTimestampIso: clientTimestampIso
        )
    }

    func pending() -> [QueuedDriverAction] {
        guard let context else { return [] }
        let status = DriverOfflineActionStatus.pending.rawValue
        let predicate = #Predicate<QueuedDriverAction> { $0.status == status }
        var descriptor = FetchDescriptor<QueuedDriverAction>(
            predicate: predicate,
            sortBy: [SortDescriptor(\.priority), SortDescriptor(\.createdAt)]
        )
        return (try? context.fetch(descriptor)) ?? []
    }

    func dead() -> [QueuedDriverAction] {
        guard let context else { return [] }
        let status = DriverOfflineActionStatus.dead.rawValue
        let predicate = #Predicate<QueuedDriverAction> { $0.status == status }
        let descriptor = FetchDescriptor<QueuedDriverAction>(
            predicate: predicate,
            sortBy: [SortDescriptor(\.createdAt, order: .reverse)]
        )
        return (try? context.fetch(descriptor)) ?? []
    }

    func pendingCount() -> Int { pending().count }

    func delete(_ action: QueuedDriverAction) {
        guard let context else { return }
        context.delete(action)
        try? context.save()
    }

    func markDead(_ action: QueuedDriverAction, error: String) {
        action.status = DriverOfflineActionStatus.dead.rawValue
        action.lastError = error
        action.attemptCount += 1
        try? context?.save()
    }

    func recordAttempt(_ action: QueuedDriverAction, error: String) {
        action.attemptCount += 1
        action.lastError = error
        try? context?.save()
    }

    func clearDead() {
        for item in dead() { delete(item) }
    }

    /// Flush engine — protocol order + poison-pill handling.
    @discardableResult
    func flush(api: APIClient) async -> (sent: Int, remaining: Int) {
        await DriverSessionReconcile.run()
        migrateLegacyDeliveriesIfNeeded()
        var sent = 0
        let items = pending()
        guard !items.isEmpty else { return (0, 0) }

        // Deliver batch fast-path
        let delivers = items.filter { DriverOfflineActionCatalog.normalize($0.endpoint) == DriverOfflineActionCatalog.deliver }
        if !delivers.isEmpty {
            sent += await flushDeliverBatch(delivers, api: api)
        }

        for action in pending() {
            let ep = DriverOfflineActionCatalog.normalize(action.endpoint)
            if ep == DriverOfflineActionCatalog.proximity {
                let age = Date().timeIntervalSince1970 - (isoToEpoch(action.clientTimestampIso) ?? (action.createdAt / 1000))
                if age > DriverOfflineActionCatalog.proximityMaxAge {
                    recordAttempt(action, error: "proximity_timestamp_stale age=\(Int(age))s")
                    continue
                }
            }
            do {
                let (status, _) = try await api.rawRequest(
                    endpoint: action.endpoint,
                    method: action.method,
                    body: action.bodyJSON,
                    idempotencyKey: action.id
                )
                if DriverOfflineActionCatalog.isSuccessHTTP(status) {
                    delete(action)
                    sent += 1
                } else if DriverOfflineActionCatalog.isRetryableHTTP(status) {
                    recordAttempt(action, error: "http_\(status)")
                    if action.attemptCount >= DriverOfflineActionCatalog.maxAttempts {
                        markDead(action, error: "http_\(status)_max_attempts")
                    }
                } else {
                    markDead(action, error: "http_\(status)")
                }
            } catch {
                recordAttempt(action, error: error.localizedDescription)
                if action.attemptCount >= DriverOfflineActionCatalog.maxAttempts {
                    markDead(action, error: error.localizedDescription)
                }
            }
        }
        return (sent, pending().count)
    }

    private func flushDeliverBatch(_ actions: [QueuedDriverAction], api: APIClient) async -> Int {
        guard let driverId = TokenStore.shared.userId, !driverId.isEmpty,
              let bearer = TokenStore.shared.token, !bearer.isEmpty else { return 0 }
        let dtos: [SyncDeliveryDTO] = actions.compactMap { action in
            guard let data = action.bodyJSON.data(using: .utf8),
                  let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
                return SyncDeliveryDTO(
                    orderId: action.orderId,
                    signature: "",
                    timestamp: isoToEpochMs(action.clientTimestampIso) ?? action.createdAt,
                    status: "DELIVERED"
                )
            }
            let orderId = (obj["order_id"] as? String) ?? action.orderId
            let signature = (obj["signature"] as? String)
                ?? (obj["scanned_token"] as? String)
                ?? ""
            return SyncDeliveryDTO(
                orderId: orderId,
                signature: signature,
                timestamp: isoToEpochMs(action.clientTimestampIso) ?? action.createdAt,
                status: "DELIVERED"
            )
        }
        guard !dtos.isEmpty else { return 0 }
        guard let result = try? await SyncServiceLive.shared.uploadBatch(
            driverId: driverId,
            deliveries: dtos,
            bearerToken: bearer
        ) else { return 0 }
        let processed = Set(result.processed)
        var sent = 0
        for action in actions where processed.contains(action.orderId) {
            delete(action)
            sent += 1
        }
        return sent
    }

    private func migrateLegacyDeliveriesIfNeeded() {
        guard !migrated, let context else { return }
        migrated = true
        let descriptor = FetchDescriptor<OfflineDelivery>()
        let legacy = (try? context.fetch(descriptor)) ?? []
        for delivery in legacy where delivery.syncStatus == "PENDING" {
            let key = DriverIdempotency.deliver(orderId: delivery.orderId)
            let body: [String: Any] = [
                "order_id": delivery.orderId,
                "scanned_token": delivery.signature,
                "signature": delivery.signature,
                "timestamp": delivery.timestamp,
            ]
            if let data = try? JSONSerialization.data(withJSONObject: body),
               let json = String(data: data, encoding: .utf8) {
                enqueue(
                    endpoint: DriverOfflineActionCatalog.deliver,
                    bodyJSON: json,
                    idempotencyKey: key,
                    orderId: delivery.orderId,
                    clientTimestampIso: ISO8601DateFormatter().string(
                        from: Date(timeIntervalSince1970: delivery.timestamp / 1000)
                    )
                )
            }
            context.delete(delivery)
        }
        try? context.save()
    }

    private func extractOrderId(_ json: String) -> String? {
        guard let data = json.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return nil }
        return obj["order_id"] as? String
    }

    private func isoToEpoch(_ iso: String) -> TimeInterval? {
        ISO8601DateFormatter().date(from: iso)?.timeIntervalSince1970
    }

    private func isoToEpochMs(_ iso: String) -> Double? {
        guard let epoch = isoToEpoch(iso) else { return nil }
        return epoch * 1000
    }
}
