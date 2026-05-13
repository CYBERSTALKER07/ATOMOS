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
    private var pingTask: Task<Void, Never>?
    private var reconnectTask: Task<Void, Never>?
    private let session = URLSession(configuration: .default)
    private let encoder = JSONEncoder()

    private(set) var isConnected: Bool = false
    private var reconnectURL: URL?
    private var reconnectAttempt: Int = 0
    private var intentionalDisconnect = false
    private static let maxReconnectDelay: UInt64 = 60_000_000_000 // 60s
    private static let baseReconnectDelay: UInt64 = 5_000_000_000  // 5s
    private static let pingInterval: UInt64 = 30_000_000_000 // 30s

    deinit {
        disconnect(intentional: true)
    }

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
        reconnectTask = nil

        startPingLoop()
    }

    private func startPingLoop() {
        pingTask?.cancel()
        pingTask = Task { [weak self] in
            guard let self else { return }
            while !Task.isCancelled {
                try? await Task.sleep(nanoseconds: Self.pingInterval)
                if Task.isCancelled { return }
                guard self.isConnected else { return }
                let ok = await self.sendPing()
                if !ok {
                    self.handleDisconnect()
                    return
                }
            }
        }
    }

    private func sendPing() async -> Bool {
        guard let task = webSocketTask else { return false }
        return await withCheckedContinuation { continuation in
            task.sendPing { error in
                continuation.resume(returning: error == nil)
            }
        }
    }

    private func handleDisconnect() {
        guard !intentionalDisconnect else { return }
        if !isConnected && webSocketTask == nil { return }
        isConnected = false
        pingTask?.cancel()
        pingTask = nil
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        scheduleReconnect()
    }

    private func scheduleReconnect() {
        guard let url = reconnectURL, !intentionalDisconnect else { return }
        guard reconnectTask == nil else { return }

        reconnectAttempt += 1
        let exponent = min(reconnectAttempt - 1, 4)
        let baseDelay = min(
            Self.baseReconnectDelay * UInt64(1 << exponent),
            Self.maxReconnectDelay
        )
        let jitterFactor = Double.random(in: 0.85...1.15)
        let jitteredDelay = min(
            UInt64(Double(baseDelay) * jitterFactor),
            Self.maxReconnectDelay
        )

        reconnectTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: jitteredDelay)
            guard let self, !self.intentionalDisconnect else { return }
            self.reconnectTask = nil
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
                self?.handleDisconnect()
            }
        }
    }

    // MARK: - Disconnect

    func disconnect() {
        disconnect(intentional: true)
    }

    private func disconnect(intentional: Bool) {
        if intentional { intentionalDisconnect = true }
        pingTask?.cancel()
        pingTask = nil
        reconnectTask?.cancel()
        reconnectTask = nil
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        isConnected = false
    }
}
