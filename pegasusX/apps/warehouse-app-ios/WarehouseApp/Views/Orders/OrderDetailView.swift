import SwiftUI

struct OrderDetailView: View {
    let orderId: String
    @Environment(\.openURL) private var openURL
    @State private var vm: OrderDetailViewModel

    init(orderId: String) {
        self.orderId = orderId
        _vm = State(initialValue: OrderDetailViewModel(orderId: orderId))
    }

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error {
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
            } else if let order = vm.order {
                let showReceipt = ["COMPLETED", "FISCALIZING", "FISCAL_FAILED"].contains(order.state.uppercased())
                ResponsiveGridContentWrapper {
                    Section("Summary") {
                        LabeledContent("State", value: order.state)
                        LabeledContent("Total", value: "\(order.totalUzs.formatted()) \(packCurrency(MarketPackStore.pack))")
                        LabeledContent("Retailer", value: order.retailerName.isEmpty ? "—" : order.retailerName)
                        LabeledContent("Order ID", value: order.orderId)
                            .font(.caption.monospaced())
                        if showReceipt {
                            Button("mobile_warehouse.ui.view_pegasus_receipt") {
                                PegasusHaptics.light()
                                Task {
                                    if let url = await vm.resolveReceiptURL() {
                                        openURL(url)
                                    }
                                }
                            }
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                            .accessibilityLabel("View Pegasus fiscal receipt")
                        }
                    }
                    OrderOpsActions(
                        state: order.state,
                        proposeDate: $vm.proposeDate,
                        reasonInput: $vm.reasonInput,
                        showProposeSheet: $vm.showProposeSheet,
                        pendingAction: $vm.pendingAction,
                        mutating: vm.mutating,
                        onLoadRecommendations: {
                            PegasusHaptics.light()
                            Task { await vm.loadRecommendations() }
                        }
                    )
                    OrderLineItems(lineItems: order.lineItems)
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("mobile_warehouse.ui.order_detail")
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
                .accessibilityLabel("Refresh order details")
            }
        }
        .task(id: orderId) {
            vm.updateOrderIdIfNeeded(orderId)
            await vm.load()
        }
        .sheet(isPresented: $vm.showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("Proposed delivery date", selection: $vm.proposeDate, displayedComponents: .date)
                    Section {
                        TextField("supplier_portal.admin.control_center.field.reason", text: $vm.reasonInput, axis: .vertical)
                            .lineLimit(3...5)
                    } footer: {
                        Text("mobile_warehouse.ui.the_retailer_is_notified_and_can_accept_or_reject_the_new_date")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("mobile_warehouse.ui.propose_new_date")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { vm.showProposeSheet = false }
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_warehouse.ui.send") {
                            PegasusHaptics.medium()
                            Task { await vm.submitPropose() }
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.mutating || vm.trimmedReason.isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .sheet(isPresented: $vm.showReassignSheet) {
            NavigationStack {
                Form {
                    Section("Select Vehicle") {
                        ForEach(vm.recommendations, id: \.driverId) { rec in
                            Button {
                                vm.selectRecommendation(rec)
                            } label: {
                                HStack {
                                    VStack(alignment: .leading) {
                                        Text(rec.driverName).foregroundStyle(.primary)
                                        Text(L10n.format("mobile_warehouse.ui.vehicleclass_licenseplate", "\(rec.vehicleClass)", "\(rec.licensePlate)"))
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    if vm.selectedDriverId == rec.driverId {
                                        Image(systemName: "checkmark").foregroundStyle(.blue)
                                    }
                                }
                                .frame(minHeight: 44)
                                .contentShape(Rectangle())
                            }
                            .buttonStyle(.plain)
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Driver \(rec.driverName), vehicle \(rec.licensePlate)\(vm.selectedDriverId == rec.driverId ? ", selected" : "")")
                        }
                    }
                    Section("Options") {
                        Toggle("Partial Reassignment", isOn: $vm.isPartialReassign)
                            .frame(minHeight: 44)
                        TextField("warehouse_portal.inventory.text.reason_optional", text: $vm.reasonInput, axis: .vertical)
                            .lineLimit(2...4)
                    }
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("supplier_portal.orders.re_dispatch_dialog.text.reassign_order")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { vm.showReassignSheet = false }
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_warehouse.ui.confirm") {
                            PegasusHaptics.medium()
                            Task { await vm.submitReassign() }
                        }
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                        .disabled(vm.selectedDriverId == nil || vm.mutating)
                    }
                }
            }
            .presentationDetents([.medium, .large])
        }
        .alert(item: $vm.pendingAction) { action in
            Alert(
                title: Text(vm.alertTitle(for: action)),
                message: Text(vm.reasonPrompt(for: action)),
                primaryButton: .default(Text(vm.confirmLabel(for: action))) {
                    PegasusHaptics.medium()
                    Task { await vm.runMutation(action) }
                },
                secondaryButton: .cancel()
            )
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
