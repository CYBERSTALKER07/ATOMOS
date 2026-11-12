import Foundation
import Observation

@Observable
@MainActor
final class VehiclesViewModel {
    typealias FetchVehicles = @MainActor () async throws -> VehicleListResponse
    typealias CreateVehicle = @MainActor (String, String, String) async throws -> Vehicle

    let vehicleClasses = [
        ("CLASS_A", "50 VU"),
        ("CLASS_B", "150 VU"),
        ("CLASS_C", "400 VU"),
    ]

    var vehicles: [Vehicle] = []
    var loading = true
    var error: String?
    var showCreate = false

    var draftLabel = ""
    var draftPlate = ""
    var selectedClass = "CLASS_A"
    var submitting = false
    var createError: String?

    var canCreate: Bool {
        !submitting
            && !draftLabel.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !draftPlate.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private let fetchVehicles: FetchVehicles
    private let createVehicleFn: CreateVehicle

    init(
        fetchVehicles: FetchVehicles? = nil,
        createVehicle: CreateVehicle? = nil
    ) {
        self.fetchVehicles = fetchVehicles ?? {
            try await WarehouseService.vehicles()
        }
        self.createVehicleFn = createVehicle ?? { label, plate, vehicleClass in
            try await WarehouseService.createVehicle(
                label: label,
                licensePlate: plate,
                vehicleClass: vehicleClass
            )
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let resp = try await fetchVehicles()
            vehicles = resp.vehicles
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func prepareCreateSheet() {
        draftLabel = ""
        draftPlate = ""
        selectedClass = "CLASS_A"
        createError = nil
        submitting = false
        showCreate = true
    }

    func createVehicle() async -> Bool {
        let label = draftLabel.trimmingCharacters(in: .whitespacesAndNewlines)
        let plate = draftPlate.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !label.isEmpty, !plate.isEmpty else { return false }

        submitting = true
        createError = nil
        defer { submitting = false }

        do {
            _ = try await createVehicleFn(label, plate, selectedClass)
            showCreate = false
            PegasusHaptics.success()
            await load()
            return true
        } catch {
            createError = error.localizedDescription
            PegasusHaptics.error()
            return false
        }
    }

    func handleReconnectDuringCreate() {
        if submitting {
            submitting = false
            createError = "Connection restored — verify status before retrying."
        }
    }
}
