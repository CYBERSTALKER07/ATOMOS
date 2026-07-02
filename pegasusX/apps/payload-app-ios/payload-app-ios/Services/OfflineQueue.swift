import Foundation
import SwiftData

@Model
final class QueuedActionModel {
    @Attribute(.unique) var id: String
    var endpoint: String
    var method: String
    var body: String
    var createdAt: Double

    init(id: String, endpoint: String, method: String, body: String, createdAt: Double) {
        self.id = id
        self.endpoint = endpoint
        self.method = method
        self.body = body
        self.createdAt = createdAt
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

    func write(_ items: [QueuedActionModel]) {
        // Not used directly in SwiftData pattern since we just insert/delete.
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
                let (status, _) = try await api.rawRequest(endpoint: action.endpoint,
                                                            method: action.method,
                                                            body: action.body,
                                                            idempotencyKey: action.id)
                if (200...299).contains(status) || status == 409 {
                    context.delete(action)
                    sent += 1
                    remainingCount -= 1
                } else if status == 408 || status == 429 || status >= 500 {
                    // keep
                } else {
                    context.delete(action)
                    sent += 1 // 4xx other than retry-eligible: drop to avoid blocking.
                    remainingCount -= 1
                }
            } catch {
                // keep
            }
        }
        try? context.save()
        return (sent, remainingCount)
    }
}
