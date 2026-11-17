//
//  SyncQueueViewModel.swift
//  driverappios
//

import Foundation
import SwiftData
import Observation

@Observable
@MainActor
final class SyncQueueViewModel {

    var pending: [QueuedDriverAction] = []
    var dead: [QueuedDriverAction] = []
    var status: String?
    var flushing: Bool = false
    var escalating: Bool = false

    private let queue: DriverOfflineQueue
    private let escalateDeadLetters: ([APIClient.SyncEscalationItemRequest]) async throws -> APIClient.SyncEscalationResponse
    private let flushHandler: () async -> (sent: Int, remaining: Int)

    convenience init() {
        self.init(queue: DriverOfflineQueue.shared)
    }

    init(
        queue: DriverOfflineQueue,
        escalateDeadLetters: @escaping ([APIClient.SyncEscalationItemRequest]) async throws -> APIClient.SyncEscalationResponse = { items in
            try await APIClient.shared.escalateSyncDeadLetters(items: items)
        },
        flushHandler: (() async -> (sent: Int, remaining: Int))? = nil
    ) {
        self.queue = queue
        self.escalateDeadLetters = escalateDeadLetters
        self.flushHandler = flushHandler ?? { await queue.flush(api: .shared) }
    }

    func attach(container: ModelContainer) {
        queue.attach(container: container)
        reload()
    }

    func reload() {
        pending = queue.pending()
        dead = queue.dead()
    }

    func retryDeadItem(_ row: QueuedDriverAction) async {
        queue.requeue(row)
        status = "Re-queued \(row.endpoint) for immediate retry."
        reload()
        await flush()
    }

    func escalateDeadToSupport() async {
        guard !escalating else { return }
        let items = queue.dead()
        guard !items.isEmpty else { return }

        escalating = true
        defer { escalating = false }

        let payload = items.map { row in
            APIClient.SyncEscalationItemRequest(
                id: row.id,
                endpoint: row.endpoint,
                orderId: row.orderId.isEmpty ? nil : row.orderId,
                payloadJson: row.bodyJSON,
                lastError: row.lastError.isEmpty ? nil : row.lastError,
                attemptCount: row.attemptCount,
                clientTimestampIso: row.clientTimestampIso.isEmpty ? nil : row.clientTimestampIso
            )
        }

        do {
            let resp = try await escalateDeadLetters(payload)
            if resp.ok {
                queue.deleteActions(items)
                reload()
                let ref = (resp.escalationId?.isEmpty == false) ? resp.escalationId! : "recorded"
                status = "Escalated \(items.count) item(s) to Support Review (\(ref))."
            } else {
                status = "Support escalation not confirmed; local items preserved."
            }
        } catch {
            status = "Could not escalate to Support Review (\(error.localizedDescription)). Dead-letter records kept safely."
        }
    }

    func flush() async {
        flushing = true
        defer { flushing = false }
        let result = await flushHandler()
        status = "Sent \(result.sent), remaining \(result.remaining)"
        reload()
    }
}
