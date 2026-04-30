//
//  ProfileService.swift
//  driverappios
//
//  Polls GET /v1/driver/profile every 60s.
//  When the backend returns a different vehicleId, updates TokenStore
//  which triggers @Observable reactivity across all SwiftUI views.
//

import Foundation

@MainActor
final class ProfileService {
    static let shared = ProfileService()

    private var pollingTask: Task<Void, Never>?
    private let baseInterval: UInt64 = 60_000_000_000 // 60 seconds in nanoseconds
    private var consecutiveFailures = 0

    private init() {}

    func startPolling() {
        guard pollingTask == nil else { return }
        consecutiveFailures = 0
        pollingTask = Task { [weak self] in
            while !Task.isCancelled {
                await self?.pollOnce()
                let base = self?.baseInterval ?? 60_000_000_000
                let failures = self?.consecutiveFailures ?? 0
                let backoff = min(base * UInt64(1 << min(failures, 4)), 5 * 60_000_000_000) // max 5min
                try? await Task.sleep(nanoseconds: backoff)
            }
        }
    }

    func stopPolling() {
        pollingTask?.cancel()
        pollingTask = nil
    }

    private func pollOnce() async {
        guard TokenStore.shared.isAuthenticated else { return }
        do {
            let profile = try await APIClient.shared.getDriverProfile()
            consecutiveFailures = 0
            applyIfChanged(profile)
        } catch {
            consecutiveFailures += 1
            print("[ProfileService] Poll failed (attempt \(consecutiveFailures)): \(error)")
        }
    }

    private func applyIfChanged(_ profile: DriverProfileResponse) {
        let store = TokenStore.shared

        // Only mutate when something actually changed to avoid unnecessary SwiftUI redraws
        if store.vehicleId != profile.vehicleId
            || store.vehicleClass != profile.vehicleClass
            || store.maxVolumeVU != profile.maxVolumeVU
            || store.vehicleType != profile.vehicleType
            || store.licensePlate != profile.licensePlate
            || store.warehouseId != profile.warehouseId
            || store.warehouseName != profile.warehouseName
            || store.warehouseLat != (profile.warehouseLat ?? 0)
            || store.warehouseLng != (profile.warehouseLng ?? 0)
        {
            store.vehicleId = profile.vehicleId
            store.vehicleClass = profile.vehicleClass
            store.maxVolumeVU = profile.maxVolumeVU
            store.vehicleType = profile.vehicleType
            store.licensePlate = profile.licensePlate
            store.warehouseId = profile.warehouseId
            store.warehouseName = profile.warehouseName
            store.warehouseLat = profile.warehouseLat ?? 0
            store.warehouseLng = profile.warehouseLng ?? 0

            store.persistVehicleInfo()
        }
    }
}
