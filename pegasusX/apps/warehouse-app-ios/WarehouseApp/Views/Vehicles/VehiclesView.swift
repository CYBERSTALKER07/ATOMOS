import SwiftUI

struct VehiclesView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = VehiclesViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.vehicles.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.vehicles.isEmpty {
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
            } else if vm.vehicles.isEmpty {
                ContentUnavailableView(
                    "No Trucks",
                    systemImage: "truck.box",
                    description: Text("mobile_warehouse.ui.add_a_truck_to_get_started")
                )
            } else {
                VehiclesList(vehicles: vm.vehicles)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.trucks")
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
                .accessibilityLabel("Refresh trucks")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    vm.prepareCreateSheet()
                } label: {
                    Label("mobile_warehouse.ui.add_truck", systemImage: "plus")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Add truck")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
        .sheet(isPresented: $vm.showCreate) {
            CreateVehicleSheet(vm: vm)
        }
    }
}

private struct CreateVehicleSheet: View {
    @Bindable var vm: VehiclesViewModel
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub

    var body: some View {
        NavigationStack {
            Form {
                TextField("mobile_warehouse.ui.label", text: $vm.draftLabel)
                TextField("mobile_warehouse.ui.license_plate", text: $vm.draftPlate)
                Section("Vehicle Class") {
                    Picker("Class", selection: $vm.selectedClass) {
                        ForEach(vm.vehicleClasses, id: \.0) { cls, cap in
                            Text(L10n.format("mobile_warehouse.ui.cls_cap", "\(cls)", "\(cap)")).tag(cls)
                        }
                    }
                    .pickerStyle(.segmented)
                    .onChange(of: vm.selectedClass) { _, _ in
                        PegasusHaptics.selection()
                    }
                }
                if let error = vm.createError {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_warehouse.ui.add_truck")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_warehouse.ui.create") {
                        Task {
                            if await vm.createVehicle() {
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
