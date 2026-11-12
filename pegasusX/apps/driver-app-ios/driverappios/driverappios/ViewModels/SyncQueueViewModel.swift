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

    private let queue: DriverOfflineQueue

    convenience init() {
        self.init(queue: DriverOfflineQueue.shared)
    }

    init(queue: DriverOfflineQueue) {
        self.queue = queue
    }

    func attach(container: ModelContainer) {
        queue.attach(container: container)
        reload()
    }

    func reload() {
        pending = queue.pending()
        dead = queue.dead()
    }

    func dismissAllDead() {
        queue.clearDead()
        reload()
    }

    func flush() async {
        flushing = true
        defer { flushing = false }
        let result = await queue.flush(api: .shared)
        status = "Sent \(result.sent), remaining \(result.remaining)"
        reload()
    }
}
