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
    private var reconnectWorkItem: DispatchWorkItem?
    private var reconnectAttempt = 0
    private var closed = true
    private var networkAvailable = true
    private var stateHandler: (@MainActor (FactoryRealtimeStatus) -> Void)?
    private var eventHandler: (@MainActor (FactoryLiveEvent) -> Void)?

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
        onEvent: @escaping @MainActor (FactoryLiveEvent) -> Void
    ) {
        stateHandler = onStateChange
        eventHandler = onEvent
        closed = false
        reconnectAttempt = 0
        openSocket(isReconnect: false)
    }

    func disconnect() {
        closed = true
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

            self.task?.cancel(with: .goingAway, reason: nil)
            self.publish(isReconnect ? .reconnecting : .connecting)

            let socketTask = self.session.webSocketTask(with: APIClient.shared.factoryWebSocketURL(token: token))
            self.task = socketTask
            socketTask.resume()
            socketTask.sendPing { [weak self] error in
                guard let self else { return }
                if error != nil {
                    self.handleSocketDrop()
                } else {
                    self.reconnectAttempt = 0
                    self.publish(.live)
                }
            }
            self.receiveLoop()
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
                   let event = try? self.decoder.decode(FactoryLiveEvent.self, from: payload),
                   let eventHandler = self.eventHandler {
                    Task { @MainActor in
                        eventHandler(event)
                    }
                }

                self.receiveLoop()
            case .failure:
                self.handleSocketDrop()
            }
        }
    }

    private func handleSocketDrop() {
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

        let delay = min(30.0, pow(2.0, Double(max(reconnectAttempt - 1, 0))))
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
