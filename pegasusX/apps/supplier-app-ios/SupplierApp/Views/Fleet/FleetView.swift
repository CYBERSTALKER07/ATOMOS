import SwiftUI

struct FleetView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var vm = FleetViewModel()

    var body: some View {
        VStack(spacing: 0) {
            Picker("mobile_supplier.chrome.fleet", selection: $vm.segment) {
                Text("portal.nav.drivers").tag(0)
                Text("supplier_portal.org_fleet.components.vehicle_form.text.vehicles").tag(1)
            }
            .pickerStyle(.segmented)
            .padding()
            .onChange(of: vm.segment) { _, _ in
                PegasusHaptics.selection()
            }

            Group {
                if vm.loading && vm.drivers.isEmpty && vm.vehicles.isEmpty {
                    SupplierLoadingView(
                        title: "mobile_supplier.loading.fleet",
                        message: "mobile_supplier.loading.fleet_message"
                    )
                } else if let error = vm.error, vm.drivers.isEmpty && vm.vehicles.isEmpty {
                    SupplierErrorView(message: error) { Task { await vm.load() } }
                } else if vm.segment == 0 {
                    fleetList(
                        emptyTitle: "mobile_supplier.empty.no_drivers",
                        emptyMessage: "mobile_supplier.empty.drivers_portal"
                    ) {
                        ForEach(vm.drivers) { driver in
                            DriverRow(driver: driver)
                        }
                    }
                } else {
                    fleetList(
                        emptyTitle: "mobile_supplier.empty.no_vehicles",
                        emptyMessage: "mobile_supplier.empty.vehicles_register"
                    ) {
                        ForEach(vm.vehicles) { vehicle in
                            VehicleRow(vehicle: vehicle)
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.fleet")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                NavigationLink {
                    FleetLiveMapView()
                } label: {
                    Image(systemName: "map")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("retailer_desktop.tracking.tracking_map.text.live_map"))
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load(silent: true) }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("portal.page.orders.action.refresh"))
            }
        }
        .task { await vm.load() }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(silent: true)
        }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
    }

    @ViewBuilder
    private func fleetList<Rows: View>(
        emptyTitle: String,
        emptyMessage: String,
        @ViewBuilder rows: @escaping () -> Rows
    ) -> some View {
        if (vm.segment == 0 && vm.drivers.isEmpty) || (vm.segment == 1 && vm.vehicles.isEmpty) {
            SupplierEmptyView(title: emptyTitle, message: emptyMessage)
        } else {
            ResponsiveGridContentWrapper {
                rows()
            }
            .supplierReadableWidth()
        }
    }
}
