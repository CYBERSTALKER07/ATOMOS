//
//  TelemetryServiceLive.swift
//  driverappios
//
//  Real WebSocket telemetry service matching Android's TelemetryService.
//

import Foundation

final class TelemetryServiceLive: TelemetryServiceProtocol {

    static let shared = TelemetryServiceLive()

    private var webSocketTask: URLSessionWebSocketTask?
    private let session = URLSession(configuration: .default)
    private let encoder = JSONEncoder()

    private(set) var isConnected: Bool = false
    private var reconnectURL: URL?
    private var reconnectAttempt: Int = 0
    private var intentionalDisconnect = false
    private static let maxReconnectDelay: UInt64 = 60_000_000_000 // 60s
    private static let baseReconnectDelay: UInt64 = 5_000_000_000  // 5s

    // MARK: - Connect

    func connect(url: URL) async {
        intentionalDisconnect = false
        reconnectURL = url
        reconnectAttempt = 0
        await establishConnection(url: url)
    }

    private func establishConnection(url: URL) async {
        disconnect(intentional: false)

        webSocketTask = session.webSocketTask(with: url)
        webSocketTask?.resume()
        isConnected = true
        reconnectAttempt = 0

        // Keep-alive ping loop + reconnect on failure
        Task { [weak self] in
            while self?.isConnected == true {
                try? await Task.sleep(nanoseconds: 30_000_000_000) // 30s
                self?.webSocketTask?.sendPing { error in
                    if error != nil {
                        Task { @MainActor in
                            self?.handleDisconnect()
                        }
                    }
                }
            }
        }
    }

    private func handleDisconnect() {
        guard !intentionalDisconnect else { return }
        isConnected = false
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        scheduleReconnect()
    }

    private func scheduleReconnect() {
        guard let url = reconnectURL, !intentionalDisconnect else { return }
        reconnectAttempt += 1
        let delay = min(
            Self.baseReconnectDelay * UInt64(1 << min(reconnectAttempt - 1, 4)),
            Self.maxReconnectDelay
        )
        Task { [weak self] in
            try? await Task.sleep(nanoseconds: delay)
            guard let self, !self.intentionalDisconnect else { return }
            await self.establishConnection(url: url)
        }
    }

    // MARK: - Send Telemetry

    func send(payload: TelemetryPayload) {
        guard isConnected, let task = webSocketTask else { return }

        guard let data = try? encoder.encode(payload) else { return }
        let message = URLSessionWebSocketTask.Message.data(data)

        task.send(message) { [weak self] error in
            if error != nil {
                Task { @MainActor in
                    self?.handleDisconnect()
                }
            }
        }
    }

    // MARK: - Disconnect

    func disconnect() {
        disconnect(intentional: true)
    }

    private func disconnect(intentional: Bool) {
        if intentional { intentionalDisconnect = true }
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        isConnected = false
    }
}
