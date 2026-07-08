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

