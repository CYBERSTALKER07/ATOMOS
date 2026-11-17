import Foundation
import Observation

@Observable
@MainActor
final class VehicleDetailViewModel {
    typealias FetchVehicle = @MainActor (String) async throws -> VehicleDetailResponse
    typealias UpdateAvailability = @MainActor (String, Bool, String?, String?) async throws -> VehicleMutationResponse

    var vehicle: Vehicle?
    var loading = true
    var error: String?
    var mutating = false
    var selectedReason = "MANUAL_HOLD"
    var customNote = ""

    private let fetchVehicle: FetchVehicle
    private let updateAvailabilityFn: UpdateAvailability

    init(
        fetchVehicle: FetchVehicle? = nil,
        updateAvailability: UpdateAvailability? = nil
    ) {
        self.fetchVehicle = fetchVehicle ?? { vehicleId in
            try await WarehouseService.vehicle(id: vehicleId)
        }
        self.updateAvailabilityFn = updateAvailability ?? { vehicleId, isActive, reason, note in
            try await WarehouseService.updateVehicleAvailability(
                vehicleId: vehicleId,
                isActive: isActive,
                unavailableReason: reason,
                unavailableNote: note
            )
        }
    }

    func load(vehicleId: String, silent: Bool = false) async {
        if !silent && vehicle == nil { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let resp = try await fetchVehicle(vehicleId)
            vehicle = resp.vehicle
            if let reason = resp.vehicle.unavailableReason, !reason.isEmpty {
                selectedReason = reason
            } else {
                selectedReason = "MANUAL_HOLD"
            }
            customNote = resp.vehicle.unavailableNote ?? ""
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func updateAvailability(vehicleId: String, isActive: Bool, reason: String? = nil, note: String? = nil) async {
        mutating = true
        error = nil
        defer { mutating = false }
        do {
            _ = try await updateAvailabilityFn(vehicleId, isActive, reason, note)
            PegasusHaptics.success()
            await load(vehicleId: vehicleId)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
