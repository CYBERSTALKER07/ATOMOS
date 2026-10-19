import Foundation
import SwiftData

/// Mirrors `PegasusKit.OfflineHttpSemantics` until payload ships an `.xcodeproj`
/// that can depend on `packages/mobile-ios-kit` (§8.8).
private enum PayloadOfflineHTTP {
    static let statusPending = "PENDING"

    static func normalizeEndpoint(_ endpoint: String) -> String {
        var ep = endpoint.trimmingCharacters(in: .whitespacesAndNewlines)
        if ep.hasPrefix("/") { ep.removeFirst() }
        if ep.hasPrefix("api/") { ep = String(ep.dropFirst(4)) }
        return ep
    }

    static func isRetryable(_ code: Int) -> Bool {
        code == 408 || code == 429 || (500...599).contains(code)
    }

    static func isSuccess(_ code: Int) -> Bool {
        (200...299).contains(code) || code == 409
    }
}

@Model
final class QueuedActionModel {
    @Attribute(.unique) var id: String
    var endpoint: String
    var method: String
    var body: String
    var createdAt: Double
    var capturedLat: Double?
    var capturedLng: Double?
    var capturedAtMs: Double?
    var attemptCount: Int
    var lastError: String
    var status: String

    init(
        id: String,
        endpoint: String,
        method: String,
        body: String,
        createdAt: Double,
        capturedLat: Double? = nil,
        capturedLng: Double? = nil,
        capturedAtMs: Double? = nil,
        attemptCount: Int = 0,
        lastError: String = "",
        status: String = PayloadOfflineHTTP.statusPending
    ) {
        self.id = id
        self.endpoint = PayloadOfflineHTTP.normalizeEndpoint(endpoint)
        self.method = method
        self.body = body
        self.createdAt = createdAt
        self.capturedLat = capturedLat
        self.capturedLng = capturedLng
        self.capturedAtMs = capturedAtMs
        self.attemptCount = attemptCount
        self.lastError = lastError
        self.status = status
    }

    func bodyForFlush() -> String {
        guard let lat = capturedLat, let lng = capturedLng,
              let data = body.data(using: .utf8),
              var obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return body
        }
        obj["latitude"] = lat
        obj["longitude"] = lng
        if let at = capturedAtMs { obj["captured_at_ms"] = at }
        guard let out = try? JSONSerialization.data(withJSONObject: obj),
              let str = String(data: out, encoding: .utf8) else { return body }
        return str
    }
}

@MainActor
final class OfflineQueue {
    static let shared = OfflineQueue()

    let container: ModelContainer
    let context: ModelContext

    private init() {
        do {
            let config = ModelConfiguration(isStoredInMemoryOnly: false)
            container = try ModelContainer(for: QueuedActionModel.self, configurations: config)
            context = container.mainContext
        } catch {
            fatalError("Failed to initialize SwiftData ModelContainer for OfflineQueue: \(error)")
        }
    }

    func read() -> [QueuedActionModel] {
        let descriptor = FetchDescriptor<QueuedActionModel>(sortBy: [SortDescriptor(\.createdAt)])
        return (try? context.fetch(descriptor)) ?? []
    }

    func enqueue(_ action: QueuedActionModel) {
        context.insert(action)
        try? context.save()
    }

    /// Replays every queued action against the live API. Returns
    /// (sentCount, remainingCount). Items that fail with a 5xx are kept;
    /// 4xx (other than 408/429) are dropped so a poison-pill cannot block
    /// the rest of the queue.
    func flush(api: APIClient) async -> (Int, Int) {
        let items = read()
        if items.isEmpty { return (0, 0) }
        var sent = 0
        var remainingCount = items.count

        for action in items {
            do {
                let (status, _) = try await api.rawRequest(
                    endpoint: action.endpoint,
                    method: action.method,
                    body: action.bodyForFlush(),
                    idempotencyKey: action.id
                )
                if PayloadOfflineHTTP.isSuccess(status) {
                    context.delete(action)
                    sent += 1
                    remainingCount -= 1
                } else if PayloadOfflineHTTP.isRetryable(status) {
                    action.attemptCount += 1
                    action.lastError = "http_\(status)"
                    try? context.save()
                } else {
                    context.delete(action)
                    sent += 1
                    remainingCount -= 1
                }
            } catch {
                action.attemptCount += 1
                action.lastError = error.localizedDescription
                try? context.save()
            }
        }
        return (sent, remainingCount)
    }
}
