import SwiftUI

struct DriversView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = DriversViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.drivers.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.drivers.isEmpty {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.drivers.isEmpty {
                ContentUnavailableView(
                    "No Drivers",
                    systemImage: "person.badge.key",
                    description: Text("mobile_warehouse.ui.add_a_driver_to_get_started")
                )
            } else {
                DriversList(
                    drivers: vm.drivers,
                    vehicles: vm.vehicles,
                    updatingDriverId: vm.updatingDriverId,
                    onAssign: { driverId, vehicleId in
                        Task { await vm.assign(driverId: driverId, vehicleId: vehicleId) }
                    }
                )
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.drivers")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh drivers")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    vm.prepareCreateSheet()
                } label: {
                    Label("mobile_warehouse.ui.add_driver", systemImage: "plus")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Add driver")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            vm.handleReconnectForAssignment()
        }
        .sheet(isPresented: $vm.showCreate) {
            CreateDriverSheet(vm: vm)
        }
        .alert("Driver Created", isPresented: $vm.showPinAlert) {
            Button("warehouse_portal.kpi_stat_card.text.done") {
                vm.acknowledgePin()
            }
        } message: {
            Text(L10n.format("mobile_warehouse.ui.one_time_pin_createdpin_nsave_it_now", "\(vm.createdPin ?? "")"))
        }
    }
}

private struct CreateDriverSheet: View {
    @Bindable var vm: DriversViewModel
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $vm.draftName)
                TextField("common.field.phone", text: $vm.draftPhone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                if let error = vm.createError {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_warehouse.ui.add_driver")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_warehouse.ui.create") {
                        Task {
                            if await vm.createDriver() {
                                dismiss()
                            }
                        }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                    .disabled(!vm.canCreate)
                }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                vm.handleReconnectDuringCreate()
            }
        }
    }
}
