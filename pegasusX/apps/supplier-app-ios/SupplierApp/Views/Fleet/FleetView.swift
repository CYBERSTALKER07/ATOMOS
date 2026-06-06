import SwiftUI

struct FleetView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var segment = 0
    @State private var drivers: [FleetDriver] = []
    @State private var vehicles: [FleetVehicle] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Fleet", selection: $segment) {
                    Text("Drivers").tag(0)
                    Text("Vehicles").tag(1)
                }
                .pickerStyle(.segmented)
                .padding()

                Group {
                    if loading {
                        SupplierLoadingView(title: "Loading fleet…")
                    } else if let error {
                        SupplierErrorView(message: error) { Task { await load() } }
                    } else if segment == 0 {
                        fleetList(
                            emptyTitle: "No drivers",
                            emptyMessage: "Create drivers from the supplier portal or API."
                        ) {
                            ForEach(drivers) { driver in
                                DriverRow(driver: driver)
                            }
                        }
                    } else {
                        fleetList(
                            emptyTitle: "No vehicles",
                            emptyMessage: "Register trucks against your home nodes."
                        ) {
                            ForEach(vehicles) { vehicle in
                                VehicleRow(vehicle: vehicle)
                            }
                        }
                    }
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Fleet")
            .task { await load() }
            .refreshable { await load(silent: true) }
            .onChange(of: segment) { _, _ in }
        }
    }

    @ViewBuilder
    private func fleetList<Rows: View>(
        emptyTitle: String,
        emptyMessage: String,
        @ViewBuilder rows: () -> Rows
    ) -> some View {
        if (segment == 0 && drivers.isEmpty) || (segment == 1 && vehicles.isEmpty) {
            SupplierEmptyView(title: emptyTitle, message: emptyMessage)
        } else {
            List {
                rows()
            }
            .listStyle(.insetGrouped)
            .supplierReadableWidth()
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            async let d = SupplierService.fleetDrivers()
            async let v = SupplierService.fleetVehicles()
            drivers = try await d
            vehicles = try await v
            _ = try? await SupplierOperationsService.fleetOrders()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}

private struct DriverRow: View {
    let driver: FleetDriver

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack {
                Text(driver.name)
                    .font(.headline)
                Spacer()
                Circle()
                    .fill(driver.isActive ? SupplierTheme.success : .secondary)
                    .frame(width: 8, height: 8)
            }
            Text(driver.phone)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            Text("\(driver.homeNodeType) · \(driver.homeNodeId)")
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
    }
}

private struct VehicleRow: View {
    let vehicle: FleetVehicle

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(vehicle.licensePlate)
                .font(.headline)
            if let label = vehicle.label, !label.isEmpty {
                Text(label)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            Text("\(vehicle.homeNodeType) · \(vehicle.homeNodeId)")
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
    }
}
