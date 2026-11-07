import Foundation
import Observation

@Observable
final class RetailerRefreshCenter {
    static let shared = RetailerRefreshCenter()

    private(set) var refreshToken = 0
    private(set) var lastRefreshAt = Date.distantPast
    private var trailingTask: Task<Void, Never>?
    private let coalesceInterval: TimeInterval = 0.20

    private init() {}

    func trigger() {
        let now = Date()
        if now.timeIntervalSince(lastRefreshAt) >= coalesceInterval {
            trailingTask?.cancel()
            trailingTask = nil
            lastRefreshAt = now
            refreshToken &+= 1
            return
        }
        guard trailingTask == nil else { return }
        let remaining = max(0.05, coalesceInterval - now.timeIntervalSince(lastRefreshAt))
        trailingTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(remaining * 1_000_000_000))
            guard !Task.isCancelled, let self else { return }
            self.trailingTask = nil
            self.lastRefreshAt = Date()
            self.refreshToken &+= 1
        }
    }
}
