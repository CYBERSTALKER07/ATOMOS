import SwiftUI

struct OrdersView: View {
    var initialState: String? = nil
    @State private var vm: OrdersViewModel

    init(initialState: String? = nil) {
        self.initialState = initialState
        _vm = State(initialValue: OrdersViewModel(initialState: initialState))
    }

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            Picker("Queue", selection: $vm.hubTab) {
                ForEach(OrdersHubTab.allCases) { tab in
                    Text(tab.rawValue).tag(tab)
                }
            }
            .pickerStyle(.segmented)
            .padding(.horizontal)
            .padding(.vertical, LabTheme.spacingSM)
            .onChange(of: vm.hubTab) { _, _ in
                PegasusHaptics.selection()
            }

            OrdersList(
                hubTab: vm.hubTab,
                loading: vm.loading,
                error: vm.error,
                orders: vm.orders,
                preorders: vm.preorders,
                onRetry: {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                },
                onProposeActive: { vm.beginPropose(orderId: $0) },
                onRejectActive: { vm.beginReject(orderId: $0) },
                onProposePreorder: { vm.beginPropose(orderId: $0) },
                onRejectPreorder: { vm.beginReject(orderId: $0) }
            )
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.orders")
        .navigationDestination(for: String.self) { orderId in
            OrderDetailView(orderId: orderId)
        }
        .toolbar {
            if vm.hubTab == .active {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        ForEach(vm.filterStates, id: \.self) { state in
                            Button {
                                PegasusHaptics.selection()
                                vm.selectedState = state
                            } label: {
                                if state == vm.selectedState {
                                    Label(state, systemImage: "checkmark")
                                } else {
                                    Text(state)
                                }
                            }
                        }
                    } label: {
                        Label("mobile_warehouse.ui.filter", systemImage: "line.3.horizontal.decrease.circle")
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Filter orders by state")
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh orders")
            }
        }
        .onChange(of: initialState) { _, newValue in
            vm.applyInitialState(newValue)
        }
        .task(id: "\(vm.hubTab.rawValue)-\(vm.selectedState)") {
            await vm.load()
        }
        .refreshable {
            await vm.load()
        }
        .sheet(item: $vm.proposeSheetOrder) { target in
            NavigationStack {
                Form {
                    DatePicker("New delivery date", selection: $vm.proposeDate, displayedComponents: .date)
                    TextField("supplier_portal.orders.order_ops_actions.text.reason_required", text: $vm.reasonInput, axis: .vertical)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("mobile_warehouse.ui.propose_new_delivery_date")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") {
                            vm.cancelPropose()
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_warehouse.ui.notify_retailer") {
                            PegasusHaptics.medium()
                            Task { await vm.submitPropose(orderId: target.id) }
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.trimmedReason.isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .alert("Cancel order", isPresented: $vm.showRejectAlert) {
            TextField("supplier_portal.orders.order_ops_actions.text.reason_required", text: $vm.reasonInput)
            Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) {
                PegasusHaptics.warning()
                Task { await vm.submitReject() }
            }
            Button("common.action.cancel", role: .cancel) {
                vm.cancelReject()
            }
        } message: {
            Text("mobile_warehouse.ui.the_retailer_will_be_notified")
        }
        .overlay(alignment: .bottom) {
            if let statusMessage = vm.statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .padding(8)
                    .background(.ultraThinMaterial)
                    .clipShape(Capsule())
                    .padding()
                    .task(id: statusMessage) {
                        await vm.autoClearStatusMessage()
                    }
            }
        }
    }
}
