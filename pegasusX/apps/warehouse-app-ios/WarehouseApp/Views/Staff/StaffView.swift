import SwiftUI

struct StaffView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = StaffViewModel()

    var body: some View {
        @Bindable var vm = vm
        StaffList(
            staff: vm.staff,
            loading: vm.loading,
            error: vm.error,
            onRetry: {
                PegasusHaptics.light()
                Task { await vm.load() }
            }
        )
        .background(LabTheme.background)
        .navigationTitle("portal.nav.staff")
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
                .accessibilityLabel("Refresh staff")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    vm.prepareCreateSheet()
                } label: {
                    Label("mobile_warehouse.ui.add_staff", systemImage: "plus")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Add staff")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
        .sheet(isPresented: $vm.showCreate) {
            CreateStaffSheet(vm: vm)
        }
        .alert("Staff Created", isPresented: $vm.showPinAlert) {
            Button("warehouse_portal.kpi_stat_card.text.done") {
                vm.acknowledgePin()
            }
        } message: {
            Text(L10n.format("mobile_warehouse.ui.one_time_pin_createdpin_nsave_it_now", "\(vm.createdPin ?? "")"))
        }
    }
}

private struct CreateStaffSheet: View {
    @Bindable var vm: StaffViewModel
    @Environment(\.dismiss) private var dismiss
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub

    var body: some View {
        NavigationStack {
            Form {
                TextField("retailer_desktop.pos.text.name", text: $vm.draftName)
                TextField("common.field.phone", text: $vm.draftPhone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                TextField("warehouse_portal.staff.text.role", text: $vm.draftRole)
                if let error = vm.createError {
                    Text(error).foregroundStyle(.red).font(.caption)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_warehouse.ui.add_staff")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_warehouse.ui.create") {
                        Task {
                            if await vm.createStaff() {
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
