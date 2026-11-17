import SwiftUI

struct PreordersView: View {
    @State private var vm = PreordersViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.rows.isEmpty {
                WarehouseLoadingView(title: "Loading pre-orders…", message: "Fetching scheduled deliveries")
            } else if vm.rows.isEmpty {
                if vm.hasBlockingError, let error = vm.statusMessage {
                    WarehouseErrorView(message: error, retry: {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    })
                } else {
                    WarehouseEmptyView(title: "No scheduled pre-orders", message: "No future orders waiting for delivery scheduling.")
                }
            } else {
                PreordersList(
                    rows: vm.rows,
                    acting: vm.acting,
                    onPropose: { row in
                        vm.beginPropose(row)
                    },
                    onReject: { row in
                        vm.beginReject(row)
                    }
                )
            }
        }
        .navigationTitle("portal.nav.preorders")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh pre-orders")
            }
        }
        .task {
            await vm.load()
        }
        .refreshable {
            await vm.load()
        }
        .sheet(item: $vm.activeSheet) { sheet in
            switch sheet {
            case .propose(let row):
                NavigationStack {
                    Form {
                        DatePicker("Proposed delivery date", selection: $vm.proposeDate, displayedComponents: .date)
                        Section("Reason for date change") {
                            TextField("supplier_portal.admin.control_center.field.reason", text: $vm.reasonInput, axis: .vertical)
                                .lineLimit(2...4)
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("mobile_warehouse.ui.propose_date")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.cancel") { vm.activeSheet = nil }
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("mobile_warehouse.ui.send") {
                                PegasusHaptics.medium()
                                Task { await vm.submitPropose(row) }
                            }
                            .disabled(vm.acting || vm.trimmedReason.isEmpty)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                        }
                    }
                }
                .presentationDetents([.medium, .large])
            case .reject(let row):
                NavigationStack {
                    Form {
                        Section("Rejection reason") {
                            TextField("supplier_portal.admin.control_center.field.reason", text: $vm.reasonInput, axis: .vertical)
                                .lineLimit(2...4)
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("mobile_warehouse.ui.reject_pre_order")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.cancel") { vm.activeSheet = nil }
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("mobile_warehouse.ui.reject", role: .destructive) {
                                PegasusHaptics.warning()
                                Task { await vm.submitReject(row) }
                            }
                            .disabled(vm.acting || vm.trimmedReason.isEmpty)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                        }
                    }
                }
                .presentationDetents([.medium])
            }
        }
        .safeAreaInset(edge: .bottom) {
            if let statusMessage = vm.statusMessage {
                Text(statusMessage)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 8)
                    .background(.bar)
            }
        }
    }
}
