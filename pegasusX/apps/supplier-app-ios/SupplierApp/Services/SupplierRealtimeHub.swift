import Foundation
import Observation

/// Cross-screen refresh epoch driven by WebSocket operational events.
/// Leading-edge + trailing-edge burst coalescing prevents Kafka/Redis event storms
/// from triggering redundant view reloads while preserving immediate first-event reactivity.
@Observable
final class SupplierRealtimeHub {
    var refreshEpoch: Int = 0
    var reconnectEpoch: Int = 0

    @ObservationIgnored private var lastBumpAt: Date = .distantPast
    @ObservationIgnored private var trailingTask: Task<Void, Never>?
    @ObservationIgnored private let coalesceWindow: TimeInterval = 0.20

    func bump() {
        let now = Date()
        if now.timeIntervalSince(lastBumpAt) >= coalesceWindow {
            lastBumpAt = now
            refreshEpoch += 1
        } else {
            trailingTask?.cancel()
            trailingTask = Task { @MainActor [weak self] in
                try? await Task.sleep(nanoseconds: 200_000_000)
                guard !Task.isCancelled, let self else { return }
                self.lastBumpAt = Date()
                self.refreshEpoch += 1
            }
        }
    }

    func bumpReconnect() {
        reconnectEpoch += 1
    }
}
