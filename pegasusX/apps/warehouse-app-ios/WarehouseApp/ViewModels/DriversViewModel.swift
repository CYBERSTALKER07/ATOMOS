import Foundation
import Observation

@Observable
@MainActor
final class DriversViewModel {
    typealias FetchDrivers = @MainActor () async throws -> DriverListResponse
    typealias FetchVehicles = @MainActor () async throws -> VehicleListResponse
    typealias AssignDriver = @MainActor (String, String?) async throws -> AssignDriverVehicleResponse
    typealias CreateDriver = @MainActor (String, String) async throws -> CreateDriverResponse

    var drivers: [Driver] = []
    var vehicles: [Vehicle] = []
    var loading = true
    var error: String?
    var showCreate = false
    var createdPin: String?
    var showPinAlert = false
    var updatingDriverId: String?

    var draftName = ""
    var draftPhone = ""
    var submitting = false
    var createError: String?

    var canCreate: Bool {
        !submitting
            && !draftName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !draftPhone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private let fetchDrivers: FetchDrivers
    private let fetchVehicles: FetchVehicles
    private let assignDriverFn: AssignDriver
    private let createDriverFn: CreateDriver

    init(
        fetchDrivers: FetchDrivers? = nil,
        fetchVehicles: FetchVehicles? = nil,
        assignDriver: AssignDriver? = nil,
        createDriver: CreateDriver? = nil
    ) {
        self.fetchDrivers = fetchDrivers ?? {
            try await WarehouseService.drivers()
        }
        self.fetchVehicles = fetchVehicles ?? {
            try await WarehouseService.vehicles()
        }
        self.assignDriverFn = assignDriver ?? { driverId, vehicleId in
            try await WarehouseService.assignDriver(driverId: driverId, vehicleId: vehicleId)
        }
        self.createDriverFn = createDriver ?? { name, phone in
            try await WarehouseService.createDriver(name: name, phone: phone)
        }
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            async let driverResponse = fetchDrivers()
            async let vehicleResponse = fetchVehicles()
            let (driverResp, vehicleResp) = try await (driverResponse, vehicleResponse)
            drivers = driverResp.drivers
            vehicles = vehicleResp.vehicles
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func assign(driverId: String, vehicleId: String?) async {
        updatingDriverId = driverId
        error = nil
        defer { updatingDriverId = nil }
        do {
            _ = try await assignDriverFn(driverId, vehicleId)
            PegasusHaptics.success()
            await load()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func prepareCreateSheet() {
        draftName = ""
        draftPhone = ""
        createError = nil
        submitting = false
        showCreate = true
    }

    func createDriver() async -> Bool {
        let name = draftName.trimmingCharacters(in: .whitespacesAndNewlines)
        let phone = draftPhone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty, !phone.isEmpty else { return false }

        submitting = true
        createError = nil
        defer { submitting = false }

        do {
            let resp = try await createDriverFn(name, phone)
            showCreate = false
            createdPin = resp.pin
            showPinAlert = true
            PegasusHaptics.success()
            await load()
            return true
        } catch {
            createError = error.localizedDescription
            PegasusHaptics.error()
            return false
        }
    }

    func acknowledgePin() {
        showPinAlert = false
        createdPin = nil
    }

    func handleReconnectForAssignment() {
        if updatingDriverId != nil {
            updatingDriverId = nil
            error = "Connection restored — verify assignment status before retrying."
        }
    }

    func handleReconnectDuringCreate() {
        if submitting {
            submitting = false
            createError = "Connection restored — verify status before retrying."
        }
    }
}
