//
//  WebSocketClient.swift
//  payload-app-ios
//
//  URLSessionWebSocketTask wrapper for /v1/ws/payloader. Auto-reconnects every
//  3 s while a token is present. Emits raw `WsMessage` frames; the VM filters
//  empty (keep-alive) frames. Mirrors Android `PayloadWebSocket`.
//

import Foundation

@MainActor
@Observable
final class WebSocketClient {
    private(set) var online = false

    /// Called for every notification frame (frames whose title or body is non-empty).
    var onFrame: ((WsMessage) -> Void)?
    /// Called every time the socket transitions from offline → online (after reconnect).
    var onReconnect: (() -> Void)?

    private var task: URLSessionWebSocketTask?
    private var session: URLSession
    private var token: String?
    private var receiveTask: Task<Void, Never>?
    private var reconnectTask: Task<Void, Never>?

    init() {
        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: cfg)
    }

    func connect(token: String) {
        self.token = token
        openSocket()
    }

    func disconnect() {
        token = nil
        reconnectTask?.cancel(); reconnectTask = nil
        receiveTask?.cancel(); receiveTask = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        online = false
    }

    private func openSocket() {
        guard let token else { return }
        let base = APIClient.shared.wsBaseURL
        let encoded = token.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? token
        guard let url = URL(string: "\(base)/v1/ws/payloader?token=\(encoded)") else { return }
        let t = session.webSocketTask(with: url)
        task = t
        t.resume()
        online = true
        onReconnect?()
        startReceiving(t)
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
                    if let frame = try? JSONDecoder().decode(WsMessage.self, from: data) {
                        let hasContent = !(frame.title ?? "").isEmpty || !(frame.body ?? "").isEmpty
                        if hasContent { self.onFrame?(frame) }
                    }
                } catch {
                    self.handleDisconnect()
                    return
                }
            }
        }
    }

    private func handleDisconnect() {
        online = false
        task = nil
        guard token != nil else { return }
        reconnectTask?.cancel()
        reconnectTask = Task { @MainActor [weak self] in
            try? await Task.sleep(nanoseconds: 3_000_000_000)
            guard !Task.isCancelled, let self, self.token != nil else { return }
            self.openSocket()
        }
    }
}
