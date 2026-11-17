import Foundation
import Network

enum WarehouseRealtimeStatus: Equatable {
    case idle
    case connecting
    case live
    case reconnecting
    case offline
}

final class WarehouseRealtimeClient {
    private let session: URLSession
    private let decoder = JSONDecoder()
    private let pathMonitor = NWPathMonitor()
    private let workQueue = DispatchQueue(label: "warehouse.realtime.client")

    private var task: URLSessionWebSocketTask?
    private var pingTask: Task<Void, Never>?
    private var reconnectWorkItem: DispatchWorkItem?
    private var reconnectAttempt = 0
    private var closed = true
    private var networkAvailable = true
    private var stateHandler: (@MainActor (WarehouseRealtimeStatus) -> Void)?
    private var eventHandler: (@MainActor (WarehouseLiveEvent) -> Void)?
    private var reconnectHandler: (@MainActor () -> Void)?
    private var hasConnectedOnce = false
    private var lastEventId: String?
    private static let pingIntervalNs: UInt64 = 25_000_000_000 // 25s heartbeat

    init(session: URLSession = .shared) {
        self.session = session
        pathMonitor.pathUpdateHandler = { [weak self] path in
            guard let self else { return }
            self.networkAvailable = path.status == .satisfied
            guard !self.closed else { return }
            if self.networkAvailable {
                if self.task == nil {
                    self.openSocket(isReconnect: self.reconnectAttempt > 0)
                }
            } else {
                self.reconnectWorkItem?.cancel()
                self.task?.cancel(with: .goingAway, reason: nil)
                self.task = nil
                self.publish(.offline)
            }
        }
        pathMonitor.start(queue: workQueue)
    }

    deinit {
        pathMonitor.cancel()
        pingTask?.cancel()
        reconnectWorkItem?.cancel()
    }

    func connect(
        onStateChange: @escaping @MainActor (WarehouseRealtimeStatus) -> Void,
        onEvent: @escaping @MainActor (WarehouseLiveEvent) -> Void,
        onReconnect: @escaping @MainActor () -> Void = {}
    ) {
        stateHandler = onStateChange
        eventHandler = onEvent
        reconnectHandler = onReconnect
        closed = false
        reconnectAttempt = 0
        openSocket(isReconnect: false)
    }

    func disconnect() {
        closed = true
        pingTask?.cancel()
        pingTask = nil
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        publish(.idle)
    }

    private func openSocket(isReconnect: Bool) {
        guard !closed else { return }
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil

        guard networkAvailable else {
            publish(.offline)
            return
        }

        Task { [weak self] in
            guard let self else { return }
            let token = await MainActor.run { TokenStore.shared.token }
            guard let token, !token.isEmpty else {
                self.publish(.offline)
                return
            }

            self.pingTask?.cancel()
            self.task?.cancel(with: .goingAway, reason: nil)
            self.publish(isReconnect ? .reconnecting : .connecting)

            var wsRequest = URLRequest(url: APIClient.shared.warehouseWebSocketURL(token: token, lastEventId: self.lastEventId))
            if let lastId = self.lastEventId, !lastId.isEmpty {
                wsRequest.setValue(lastId, forHTTPHeaderField: "Last-Event-ID")
            }
            let socketTask = self.session.webSocketTask(with: wsRequest)
            self.task = socketTask
            socketTask.resume()
            socketTask.sendPing { [weak self] error in
                guard let self else { return }
                if error != nil {
                    self.handleSocketDrop()
                } else {
                    let wasReconnect = self.hasConnectedOnce
                    self.hasConnectedOnce = true
                    self.reconnectAttempt = 0
                    self.publish(.live)
                    self.startHeartbeatLoop(for: socketTask)
                    if wasReconnect {
                        Task { await WarehouseSessionReconcile.run() }
                        if let reconnectHandler = self.reconnectHandler {
                            Task { @MainActor in reconnectHandler() }
                        }
                    }
                }
            }
            self.receiveLoop()
        }
    }

    private func startHeartbeatLoop(for socketTask: URLSessionWebSocketTask) {
        pingTask?.cancel()
        pingTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: Self.pingIntervalNs)
                guard !Task.isCancelled, let self, !self.closed, self.task === socketTask else { return }
                let alive = await withCheckedContinuation { continuation in
                    socketTask.sendPing { error in
                        continuation.resume(returning: error == nil)
                    }
                }
                if !alive {
                    self.handleSocketDrop()
                    return
                }
            }
        }
    }

    private func receiveLoop() {
        task?.receive { [weak self] result in
            guard let self else { return }
            guard !self.closed else { return }

            switch result {
            case .success(let message):
                let payload: Data?
                switch message {
                case .string(let text):
                    payload = text.data(using: .utf8)
                case .data(let data):
                    payload = data
                @unknown default:
                    payload = nil
                }

                self.publish(.live)
                if let payload {
                    if let rawObj = try? JSONSerialization.jsonObject(with: payload) as? [String: Any],
                       let eid = (rawObj["event_id"] as? String ?? rawObj["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
                       !eid.isEmpty {
                        self.lastEventId = eid
                    }
                    if let event = try? self.decoder.decode(WarehouseLiveEvent.self, from: payload),
                       let eventHandler = self.eventHandler {
                        Task { @MainActor in
                            eventHandler(event)
                        }
                    }
                }

                self.receiveLoop()
            case .failure:
                self.handleSocketDrop()
            }
        }
    }

    private func handleSocketDrop() {
        pingTask?.cancel()
        pingTask = nil
        task = nil
        scheduleReconnect()
    }

    private func scheduleReconnect() {
        guard !closed else { return }
        reconnectWorkItem?.cancel()

        guard networkAvailable else {
            publish(.offline)
            return
        }

        reconnectAttempt += 1
        publish(.reconnecting)

        let baseDelay = min(30.0, pow(2.0, Double(max(reconnectAttempt - 1, 0))))
        let delay = baseDelay + Double.random(in: 0...(baseDelay / 2))
        let workItem = DispatchWorkItem { [weak self] in
            self?.openSocket(isReconnect: true)
        }
        reconnectWorkItem = workItem
        workQueue.asyncAfter(deadline: .now() + delay, execute: workItem)
    }

    private func publish(_ status: WarehouseRealtimeStatus) {
        guard let stateHandler else { return }
        Task { @MainActor in
            stateHandler(status)
        }
    }
}