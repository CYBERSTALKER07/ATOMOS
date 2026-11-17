import Foundation
import Observation

@Observable
@MainActor
final class FleetViewModel {
    var segment = 0
    var drivers: [FleetDriver] = []
    var vehicles: [FleetVehicle] = []
    var loading = true
    var error: String?

    private var loadGeneration = 0

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !drivers.isEmpty || !vehicles.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            async let d = SupplierService.fleetDrivers()
            async let v = SupplierService.fleetVehicles()
            let loadedDrivers = try await d
            let loadedVehicles = try await v
            guard generation == loadGeneration else { return }
            drivers = loadedDrivers
            vehicles = loadedVehicles
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }
}
