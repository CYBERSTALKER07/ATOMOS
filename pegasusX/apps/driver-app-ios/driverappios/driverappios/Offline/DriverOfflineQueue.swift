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
        method: String = "POST",
        capturedLat: Double? = nil,
        capturedLng: Double? = nil,
        capturedAtMs: Double? = nil
    ) {
        guard let context else { return }
        let ep = DriverOfflineActionCatalog.normalize(endpoint)
        guard DriverOfflineActionCatalog.isOfflineEligible(ep) else { return }
        let fromBody = Self.coords(from: bodyJSON)
        let lat = capturedLat ?? fromBody.lat
        let lng = capturedLng ?? fromBody.lng
        let at = capturedAtMs ?? ((lat != nil && lng != nil) ? Date().timeIntervalSince1970 * 1000 : nil)
        let action = QueuedDriverAction(
            id: idempotencyKey,
            endpoint: ep,
            method: method,
            bodyJSON: Self.ensureCoords(in: bodyJSON, lat: lat, lng: lng),
            priority: DriverOfflineActionCatalog.priority(for: ep),
            clientTimestampIso: clientTimestampIso,
            orderId: orderId.isEmpty ? (extractOrderId(bodyJSON) ?? "") : orderId,
            capturedLat: lat,
            capturedLng: lng,
            capturedAtMs: at
        )
        context.insert(action)
        try? context.save()
        print("[DriverOfflineQueue] enqueued \(ep) order=\(action.orderId) lat=\(String(describing: lat))")
    }

    func enqueueJSONObject(
        endpoint: String,
        body: [String: Any],
        idempotencyKey: String,
        orderId: String = "",
        clientTimestampIso: String = DriverOfflineActionCatalog.nowIso(),
        capturedLat: Double? = nil,
        capturedLng: Double? = nil
    ) {
        var payload = body
        if payload["client_timestamp"] == nil {
            payload["client_timestamp"] = clientTimestampIso
        }
        let lat = capturedLat ?? (payload["latitude"] as? Double) ?? (payload["latitude"] as? NSNumber)?.doubleValue
        let lng = capturedLng ?? (payload["longitude"] as? Double) ?? (payload["longitude"] as? NSNumber)?.doubleValue
        guard let data = try? JSONSerialization.data(withJSONObject: payload),
              let json = String(data: data, encoding: .utf8) else { return }
        enqueue(
            endpoint: endpoint,
            bodyJSON: json,
            idempotencyKey: idempotencyKey,
            orderId: orderId,
            clientTimestampIso: clientTimestampIso,
            capturedLat: lat,
            capturedLng: lng
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
                let body = try await resolvePodLocalFiles(
                    endpoint: action.endpoint,
                    bodyJSON: action.bodyJSONForFlush(),
                    orderId: action.orderId
                )
                guard let body else {
                    recordAttempt(action, error: "pod_upload_pending")
                    continue
                }
                let (status, _) = try await api.rawRequest(
                    endpoint: action.endpoint,
                    method: action.method,
                    body: body,
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

    /// Upload local PoD JPEGs before credit / shop-closed flush. Returns nil to keep pending.
    private func resolvePodLocalFiles(endpoint: String, bodyJSON: String, orderId: String) async throws -> String? {
        let ep = DriverOfflineActionCatalog.normalize(endpoint)
        guard ep == DriverOfflineActionCatalog.credit || ep == DriverOfflineActionCatalog.shopClosed else {
            return bodyJSON
        }
        guard let data = bodyJSON.data(using: .utf8),
              var obj = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return bodyJSON
        }
        let photoLocal = (obj["photo_local_path"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let sigLocal = (obj["signature_local_path"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let photoProof = (obj["photo_proof_url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let photoUrl = (obj["photo_url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let sigUrl = (obj["signature_url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

        if !photoLocal.isEmpty && photoProof.isEmpty && photoUrl.isEmpty {
            let bytes = try MediaUploadService.readLocalJPEG(photoLocal)
            let url = try await MediaUploadService.uploadJPEGData(bytes, purpose: "credit_proof", orderId: orderId)
            if ep == DriverOfflineActionCatalog.credit {
                obj["photo_proof_url"] = url
            } else {
                obj["photo_url"] = url
            }
            obj.removeValue(forKey: "photo_local_path")
        }
        if !sigLocal.isEmpty && sigUrl.isEmpty {
            let bytes = try MediaUploadService.readLocalJPEG(sigLocal)
            let url = try await MediaUploadService.uploadJPEGData(bytes, purpose: "credit_proof", orderId: orderId)
            obj["signature_url"] = url
            obj.removeValue(forKey: "signature_local_path")
        }

        let proofAfter = (obj["photo_proof_url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let photoAfter = (obj["photo_url"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if ep == DriverOfflineActionCatalog.credit && proofAfter.isEmpty {
            return nil
        }
        if ep == DriverOfflineActionCatalog.shopClosed && photoAfter.isEmpty {
            return nil
        }
        guard let out = try? JSONSerialization.data(withJSONObject: obj),
              let str = String(data: out, encoding: .utf8) else {
            return nil
        }
        return str
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

    private static func coords(from json: String) -> (lat: Double?, lng: Double?) {
        guard let data = json.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return (nil, nil)
        }
        let lat = (obj["latitude"] as? Double) ?? (obj["latitude"] as? NSNumber)?.doubleValue
        let lng = (obj["longitude"] as? Double) ?? (obj["longitude"] as? NSNumber)?.doubleValue
        return (lat, lng)
    }

    private static func ensureCoords(in json: String, lat: Double?, lng: Double?) -> String {
        guard let lat, let lng,
              let data = json.data(using: .utf8),
              var obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return json
        }
        if obj["latitude"] == nil { obj["latitude"] = lat }
        if obj["longitude"] == nil { obj["longitude"] = lng }
        guard let out = try? JSONSerialization.data(withJSONObject: obj),
              let str = String(data: out, encoding: .utf8) else { return json }
        return str
    }

    private func isoToEpoch(_ iso: String) -> TimeInterval? {
        ISO8601DateFormatter().date(from: iso)?.timeIntervalSince1970
    }

    private func isoToEpochMs(_ iso: String) -> Double? {
        guard let epoch = isoToEpoch(iso) else { return nil }
        return epoch * 1000
    }
}
