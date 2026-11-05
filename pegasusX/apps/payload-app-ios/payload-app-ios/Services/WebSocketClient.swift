//
//  WebSocketClient.swift
//  payload-app-ios
//
//  URLSessionWebSocketTask wrapper for /v1/ws with Authorization: Bearer.
//  Auto-reconnects every 3 s while a token is present. Emits notification
//  frames plus PAYLOAD_SYNC refresh frames. Mirrors Android `PayloadWebSocket`.
//

import Foundation

@MainActor
@Observable
final class WebSocketClient {
    private(set) var online = false

    /// Called for every notification frame and PAYLOAD_SYNC refresh frame.
    var onFrame: ((WsMessage) -> Void)?
    /// Called every time the socket transitions from offline → online (after reconnect).
    var onReconnect: (() -> Void)?

    private var task: URLSessionWebSocketTask?
    private var session: URLSession
    private var token: String?
    private var lastEventId: String?
    private var receiveTask: Task<Void, Never>?
    private var heartbeatTask: Task<Void, Never>?
    private var syncCoalesceTask: Task<Void, Never>?
    private var reconnectTask: Task<Void, Never>?
    private var reconnectAttempt = 0
    private let decoder = JSONDecoder()

    init() {
        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: cfg)
    }

    func connect(token: String) {
        self.token = token
        reconnectAttempt = 0
        openSocket()
    }

    func disconnect() {
        token = nil
        syncCoalesceTask?.cancel(); syncCoalesceTask = nil
        heartbeatTask?.cancel(); heartbeatTask = nil
        reconnectTask?.cancel(); reconnectTask = nil
        receiveTask?.cancel(); receiveTask = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        online = false
    }

    private func openSocket() {
        guard let token else { return }
        let base = APIClient.shared.wsBaseURL
        var components = URLComponents(string: "\(base)/v1/ws")
        if let lastId = lastEventId, !lastId.isEmpty {
            components?.queryItems = [URLQueryItem(name: "last_event_id", value: lastId)]
        }
        guard let url = components?.url ?? URL(string: "\(base)/v1/ws") else { return }
        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        if let lastId = lastEventId, !lastId.isEmpty {
            request.setValue(lastId, forHTTPHeaderField: "Last-Event-ID")
        }
        let t = session.webSocketTask(with: request)
        task = t
        t.resume()
        reconnectAttempt = 0
        online = true
        onReconnect?()
        startHeartbeat(t)
        startReceiving(t)
    }

    private func startHeartbeat(_ t: URLSessionWebSocketTask) {
        heartbeatTask?.cancel()
        heartbeatTask = Task { @MainActor [weak self] in
            while !Task.isCancelled, let self, self.online {
                try? await Task.sleep(nanoseconds: 25_000_000_000)
                guard !Task.isCancelled, self.online else { return }
                t.sendPing { [weak self] error in
                    guard error != nil else { return }
                    Task { @MainActor in
                        self?.handleDisconnect()
                    }
                }
            }
        }
    }

    private func startReceiving(_ t: URLSessionWebSocketTask) {
        receiveTask?.cancel()
        receiveTask = Task { @MainActor [weak self] in
            while !Task.isCancelled, let self {
                do {
                    let msg = try await t.receive()
                    let text: String? = {
                        switch msg {
                        case .string(let s): return s
                        case .data(let d): return String(data: d, encoding: .utf8)
                        @unknown default: return nil
                        }
                    }()
                    guard let text, let data = text.data(using: .utf8) else { continue }
                    if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                       let eid = (json["event_id"] as? String ?? json["id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines),
                       !eid.isEmpty {
                        self.lastEventId = eid
                    }
                    if let frame = try? self.decoder.decode(WsMessage.self, from: data) {
                        let hasContent = !(frame.title ?? "").isEmpty || !(frame.body ?? "").isEmpty
                        let isPayloadSync = frame.type == "PAYLOAD_SYNC" || frame.type == "WS_RESYNC_REQUIRED" || (frame.type?.hasPrefix("MANIFEST_") ?? false)
                        if hasContent {
                            self.onFrame?(frame)
                        } else if isPayloadSync {
                            self.emitCoalescedSync(frame)
                        }
                    }
                } catch {
                    self.handleDisconnect()
                    return
                }
            }
        }
    }

    private func emitCoalescedSync(_ frame: WsMessage) {
        syncCoalesceTask?.cancel()
        syncCoalesceTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: 180_000_000)
            guard !Task.isCancelled, let self else { return }
            self.onFrame?(frame)
        }
    }

    private func handleDisconnect() {
        online = false
        heartbeatTask?.cancel(); heartbeatTask = nil
        task = nil
        guard token != nil else { return }
        reconnectAttempt += 1
        let delaySeconds = reconnectDelaySeconds(attempt: reconnectAttempt - 1, base: 3, maxDelay: 60)
        reconnectTask?.cancel()
        reconnectTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: UInt64(delaySeconds * 1_000_000_000))
            guard !Task.isCancelled, let self, self.token != nil else { return }
            self.openSocket()
        }
    }
}

private func reconnectDelaySeconds(attempt: Int, base: TimeInterval, maxDelay: TimeInterval) -> TimeInterval {
    let capped = min(Swift.max(attempt, 0), 10)
    let exp = min(base * pow(2.0, Double(capped)), maxDelay)
    return exp + Double.random(in: 0...(exp / 2))
}
