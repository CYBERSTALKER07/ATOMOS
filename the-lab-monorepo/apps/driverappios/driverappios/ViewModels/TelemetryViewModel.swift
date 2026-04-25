//
//  TelemetryViewModel.swift
//  driverappios
//

import CoreLocation
import Network
import SwiftUI

@Observable
@MainActor
final class TelemetryViewModel {

    // MARK: - State

    var isLive = false

    private let service: TelemetryServiceProtocol
    private let driverId: String
    private let monitor = NWPathMonitor()
    private let monitorQueue = DispatchQueue(label: "telemetry.network.monitor")
    private var lastSentLocation: CLLocationCoordinate2D?

    // MARK: - Init

    convenience init() {
        self.init(service: TelemetryServiceLive.shared, driverId: TokenStore.shared.userId ?? "")
    }

    init(
        service: TelemetryServiceProtocol,
        driverId: String
    ) {
        self.service = service
        self.driverId = driverId
    }

    // MARK: - Connect / Disconnect

    func start() async {
        guard let token = TokenStore.shared.token,
              let encodedToken = token.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed),
              let url = URL(string: "\(APIClient.shared.apiBaseURL.replacingOccurrences(of: "http", with: "ws"))/ws/telemetry?token=\(encodedToken)") else {
            isLive = false
            return
        }
        await service.connect(url: url)
        isLive = service.isConnected

        // Start network monitoring for auto-sync
        monitor.pathUpdateHandler = { [weak self] path in
            guard let self else { return }
            Task { @MainActor [weak self] in
                guard let self else { return }
                self.isLive = path.status == .satisfied && self.service.isConnected
                // Flush queued offline deliveries when connectivity restores
                if path.status == .satisfied {
                    await FleetServiceLive.shared.flushOfflineQueue()
                }
            }
        }
        monitor.start(queue: monitorQueue)
    }

    func stop() {
        service.disconnect()
        monitor.cancel()
        isLive = false
    }

    // MARK: - Send Telemetry

    func sendLocation(_ coordinate: CLLocationCoordinate2D, accuracy: Double?) {
        // Only send if moved ≥ 10m from last sent location
        if let last = lastSentLocation {
            let dist = haversineDistance(from: last, to: coordinate)
            guard dist >= 10 else { return }
        }

        lastSentLocation = coordinate

        let payload = TelemetryPayload(
            driver_id: driverId,
            latitude: coordinate.latitude,
            longitude: coordinate.longitude,
            accuracy: accuracy,
            timestamp: Date().timeIntervalSince1970 * 1000
        )
        service.send(payload: payload)
    }
}
