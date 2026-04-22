//
//  TelemetryService.swift
//  driverappios
//

import Foundation

// MARK: - Protocol

protocol TelemetryServiceProtocol: AnyObject {
    /// Connect to ws://{host}:8080/ws/telemetry?role=DRIVER
    func connect(url: URL) async
    /// Send JSON TelemetryPayload every time device moves ≥10m
    func send(payload: TelemetryPayload)
    /// Disconnect WebSocket
    func disconnect()
    /// Whether the WebSocket is currently connected
    var isConnected: Bool { get }
}

// MARK: - Stub Implementation

final class TelemetryServiceStub: TelemetryServiceProtocol {

    static let shared = TelemetryServiceStub()

    private(set) var isConnected: Bool = false

    func connect(url: URL) async {
        try? await Task.sleep(nanoseconds: 300_000_000)
        isConnected = true
    }

    func send(payload: TelemetryPayload) {
        // Stub: no-op. In production, encode and send via URLSessionWebSocketTask.
    }

    func disconnect() {
        isConnected = false
    }
}
