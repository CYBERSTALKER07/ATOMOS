import Foundation
import Network

enum FactoryRealtimeStatus: Equatable {
    case idle
    case connecting
    case live
    case reconnecting
    case offline
}

enum FactoryRealtimeEventType: String {
    case supplyRequestUpdate = "FACTORY_SUPPLY_REQUEST_UPDATE"
    case transferUpdate = "FACTORY_TRANSFER_UPDATE"
    case manifestUpdate = "FACTORY_MANIFEST_UPDATE"
    case outboxFailed = "FACTORY_OUTBOX_FAILED"
}

struct FactoryLiveEvent: Decodable {
    let type: String
    let timestamp: String?
    let eventId: String?

    enum CodingKeys: String, CodingKey {
        case type
        case timestamp
        case eventId = "event_id"
        case id
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        type = try c.decode(String.self, forKey: .type)
        timestamp = try c.decodeIfPresent(String.self, forKey: .timestamp)
        eventId = try c.decodeIfPresent(String.self, forKey: .eventId)
            ?? c.decodeIfPresent(String.self, forKey: .id)
    }

    var eventType: FactoryRealtimeEventType? {
        FactoryRealtimeEventType(rawValue: type)
    }
}

final class FactoryRealtimeClient {
    private let session: URLSession
    private let decoder = JSONDecoder()
    private let pathMonitor = NWPathMonitor()
    private let workQueue = DispatchQueue(label: "factory.realtime.client")

    private var task: URLSessionWebSocketTask?
    private var heartbeatTask: Task<Void, Never>?
    private var reconnectWorkItem: DispatchWorkItem?
    private var reconnectAttempt = 0
    private var hasConnectedOnce = false
    private var closed = true
    private var networkAvailable = true
    private var lastEventId: String?
    private var stateHandler: (@MainActor (FactoryRealtimeStatus) -> Void)?
    private var eventHandler: (@MainActor (FactoryLiveEvent) -> Void)?
    private var reconnectHandler: (@MainActor () -> Void)?

    init(session: URLSession = .shared) {
        self.session = session
        pathMonitor.pathUpdateHandler = { [weak self] path in
            guard let self else { return }
            networkAvailable = path.status == .satisfied
            guard !closed else { return }
            if networkAvailable {
                if task == nil {
                    openSocket(isReconnect: reconnectAttempt > 0)
                }
            } else {
                reconnectWorkItem?.cancel()
                task?.cancel(with: .goingAway, reason: nil)
                task = nil
                publish(.offline)
            }
        }
        pathMonitor.start(queue: workQueue)
    }

    deinit {
        pathMonitor.cancel()
        reconnectWorkItem?.cancel()
    }

    func connect(
        onStateChange: @escaping @MainActor (FactoryRealtimeStatus) -> Void,
        onEvent: @escaping @MainActor (FactoryLiveEvent) -> Void,
        onReconnect: @escaping @MainActor () -> Void = {}
    ) {
        stateHandler = onStateChange
        eventHandler = onEvent
        reconnectHandler = onReconnect
        closed = false
        reconnectAttempt = 0
        hasConnectedOnce = false
        openSocket(isReconnect: false)
    }

    func disconnect() {
        closed = true
        heartbeatTask?.cancel()
        heartbeatTask = nil
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

            self.heartbeatTask?.cancel()
            self.task?.cancel(with: .goingAway, reason: nil)
            self.publish(isReconnect ? .reconnecting : .connecting)

            let wsURL = APIClient.shared.factoryWebSocketURL(token: token, lastEventId: self.lastEventId)
            var wsRequest = URLRequest(url: wsURL)
            wsRequest.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
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
                    self.startHeartbeat()
                    if wasReconnect, let reconnectHandler = self.reconnectHandler {
                        Task { @MainActor in
                            await FactorySessionReconcile.run()
                            reconnectHandler()
                        }
                    }
                }
            }
            self.receiveLoop()
        }
    }

    private func startHeartbeat() {
        heartbeatTask?.cancel()
        heartbeatTask = Task { [weak self] in
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 25_000_000_000)
                guard !Task.isCancelled, let self, !self.closed, let socket = self.task else { return }
                socket.sendPing { [weak self] error in
                    if error != nil {
                        self?.handleSocketDrop()
                    }
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
                if let payload,
                   let event = try? self.decoder.decode(FactoryLiveEvent.self, from: payload) {
                    if let eid = event.eventId?.trimmingCharacters(in: .whitespacesAndNewlines), !eid.isEmpty {
                        self.lastEventId = eid
                    }
                    if let eventHandler = self.eventHandler {
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
        heartbeatTask?.cancel()
        heartbeatTask = nil
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

    private func publish(_ status: FactoryRealtimeStatus) {
        guard let stateHandler else { return }
        Task { @MainActor in
            stateHandler(status)
        }
    }
}
