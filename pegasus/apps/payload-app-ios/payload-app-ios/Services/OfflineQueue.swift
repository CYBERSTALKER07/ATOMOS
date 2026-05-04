//
//  OfflineQueue.swift
//  payload-app-ios
//
//  Persistent JSON-backed queue for actions taken while the WebSocket /
//  network is offline. Mirrors the Expo `offline_queue` SecureStore key and
//  the Android `payloader_offline_queue` EncryptedSharedPreferences entry.
//  Only inject-order is enqueued today; the structure supports arbitrary
//  endpoint+method+body to match the Android implementation.
//

import Foundation

@MainActor
final class OfflineQueue {
    static let shared = OfflineQueue()
    private let key = "payload_offline_queue"
    private let encoder = JSONEncoder()
    private let decoder = JSONDecoder()

    func read() -> [QueuedAction] {
        guard let data = UserDefaults.standard.data(forKey: key) else { return [] }
        return (try? decoder.decode([QueuedAction].self, from: data)) ?? []
    }

    func write(_ items: [QueuedAction]) {
        if items.isEmpty {
            UserDefaults.standard.removeObject(forKey: key)
            return
        }
        if let data = try? encoder.encode(items) {
            UserDefaults.standard.set(data, forKey: key)
        }
    }

    func enqueue(_ action: QueuedAction) {
        var items = read()
        items.append(action)
        write(items)
    }

    /// Replays every queued action against the live API. Returns
    /// (sentCount, remainingCount). Items that fail with a 5xx are kept;
    /// 4xx (other than 408/429) are dropped so a poison-pill cannot block
    /// the rest of the queue.
    func flush(api: APIClient) async -> (Int, Int) {
        let items = read()
        if items.isEmpty { return (0, 0) }
        var sent = 0
        var remaining: [QueuedAction] = []
        for action in items {
            do {
                let (status, _) = try await api.rawRequest(endpoint: action.endpoint,
                                                            method: action.method,
                                                            body: action.body,
                                                            idempotencyKey: action.id)
                if (200...299).contains(status) || status == 409 {
                    sent += 1
                } else if status == 408 || status == 429 || status >= 500 {
                    remaining.append(action)
                } else {
                    sent += 1 // 4xx other than retry-eligible: drop to avoid blocking.
                }
            } catch {
                remaining.append(action)
            }
        }
        write(remaining)
        return (sent, remaining.count)
    }
}
