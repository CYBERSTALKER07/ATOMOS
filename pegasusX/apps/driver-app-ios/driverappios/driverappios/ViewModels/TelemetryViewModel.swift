//
//  TelemetryViewModel.swift
//  driverappios
//

import CoreLocation
import Network
import SwiftUI
import Observation

@Observable
@MainActor
final class TelemetryViewModel {

    // MARK: - State

    var isLive = false
    var isCommandSocketLive = false

    private let service: TelemetryServiceProtocol
    private let driverId: String
    private var isTrackingNetwork = false
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
        let wsBase = APIClient.shared.resolvedBaseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")
        guard TokenStore.shared.token != nil,
              let url = URL(string: "\(wsBase)/v1/ws") else {
            isLive = false
            return
        }
        await service.connect(url: url)
        syncLiveFlags(networkSatisfied: NetworkLifecycleEngine.shared.isOnline)

        isTrackingNetwork = true
        trackNetworkState()
    }

    private func trackNetworkState() {
        guard isTrackingNetwork else { return }
        withObservationTracking {
            let online = NetworkLifecycleEngine.shared.isOnline
            self.syncLiveFlags(networkSatisfied: online)
        } onChange: {
            Task { @MainActor [weak self] in
                guard let self, self.isTrackingNetwork else { return }
                if NetworkLifecycleEngine.shared.isOnline {
                    await FleetServiceLive.shared.flushOfflineQueue()
                }
                self.trackNetworkState()
            }
        }
    }

    func stop() {
        isTrackingNetwork = false
        service.disconnect()
        isLive = false
        isCommandSocketLive = false
    }

    func syncLiveFlags(networkSatisfied: Bool = true) {
        isCommandSocketLive = DriverSocketState.shared.connectionState == .connected
        isLive = networkSatisfied && service.isConnected && isCommandSocketLive
    }

    // MARK: - Send Telemetry

    func sendLocation(_ coordinate: CLLocationCoordinate2D, accuracy: Double?) {
        // Filter out invalid or coarse cell-tower fixes before streaming to Kafka/Redis
        if let acc = accuracy, (acc < 0 || acc > 65) {
            return
        }
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
