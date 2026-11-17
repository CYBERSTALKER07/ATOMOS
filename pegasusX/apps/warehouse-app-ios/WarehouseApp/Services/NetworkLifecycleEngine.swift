import Foundation
import Network
import Observation

public enum NetworkState: String, Sendable {
    case connected
    case connecting
    case offline
    case flapping
    case captiveBlocked
}

@Observable
@MainActor
public final class NetworkLifecycleEngine {
    public static let shared = NetworkLifecycleEngine()

    public private(set) var isOnline: Bool = true
    public private(set) var state: NetworkState = .connected
    public private(set) var syncEpoch: Int = 0

    private let monitor: NWPathMonitor
    private let queue = DispatchQueue(label: "com.pegasusx.warehouse.network-lifecycle", qos: .utility)
    private var transitionTimestamps: [Date] = []
    private var flappingTask: Task<Void, Never>?
    private var isTesting = false

    private init() {
        self.monitor = NWPathMonitor()
        self.monitor.pathUpdateHandler = { [weak self] path in
            Task { @MainActor [weak self] in
                guard let self = self, !self.isTesting else { return }
                self.handlePathUpdate(path)
            }
        }
        self.monitor.start(queue: queue)
    }

    private func handlePathUpdate(_ path: NWPath) {
        let online = path.status == .satisfied
        recordTransition(online: online)
    }

    private func recordTransition(online: Bool) {
        let now = Date()
        transitionTimestamps = transitionTimestamps.filter { now.timeIntervalSince($0) < 30.0 }
        transitionTimestamps.append(now)

        if transitionTimestamps.count >= 3 {
            state = .flapping
            isOnline = false

            flappingTask?.cancel()
            flappingTask = Task { [weak self] in
                try? await Task.sleep(nanoseconds: 15_000_000_000)
                guard !Task.isCancelled else { return }
                await MainActor.run {
                    guard let self = self else { return }
                    self.state = online ? .connected : .offline
                    self.isOnline = online
                    if online {
                        Task { await self.triggerSyncEpoch() }
                    }
                }
            }
            return
        }

        if online {
            state = .connected
            isOnline = true
            let jitterNs = UInt64.random(in: 200_000_000...800_000_000)
            Task { [weak self] in
                try? await Task.sleep(nanoseconds: jitterNs)
                guard !Task.isCancelled else { return }
                await self?.triggerSyncEpoch()
            }
        } else {
            state = .offline
            isOnline = false
        }
    }

    public func triggerSyncEpoch() async {
        syncEpoch += 1
    }

    public func waitForOnline() async {
        while !isOnline {
            try? await Task.sleep(nanoseconds: 500_000_000)
        }
    }

    // MARK: - Testing Hooks
    public func resetForTesting() {
        isTesting = true
        flappingTask?.cancel()
        flappingTask = nil
        transitionTimestamps.removeAll()
        isOnline = true
        state = .connected
        syncEpoch = 0
    }

    public func setOnlineForTesting(_ online: Bool) {
        isTesting = true
        recordTransition(online: online)
    }
}
