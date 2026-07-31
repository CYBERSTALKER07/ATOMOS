import SwiftUI

struct FleetView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
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
                        SupplierLoadingView(
                            title: "Loading fleet",
                            message: "Fetching drivers and vehicles for your nodes."
                        )
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
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink {
                        FleetLiveMapView()
                    } label: {
                        Label("Live map", systemImage: "map")
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load(silent: true) }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await load() }
            .refreshable { await load(silent: true) }
            .onChange(of: realtimeHub.refreshEpoch) { _, _ in
                Task { await load(silent: true) }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                Task { await load(silent: true) }
            }
        }
    }

    @ViewBuilder
    private func fleetList<Rows: View>(
        emptyTitle: String,
        emptyMessage: String,
        @ViewBuilder rows: @escaping () -> Rows
    ) -> some View {
        if (segment == 0 && drivers.isEmpty) || (segment == 1 && vehicles.isEmpty) {
            SupplierEmptyView(title: emptyTitle, message: emptyMessage)
        } else {
            ResponsiveGridContentWrapper {
                rows()
            }
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
                SupplierStatusBadge(
                    text: driver.isActive ? "ACTIVE" : "OFFLINE",
                    tint: driver.isActive ? SupplierTheme.live : SupplierTheme.secondaryLabel
                )
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
