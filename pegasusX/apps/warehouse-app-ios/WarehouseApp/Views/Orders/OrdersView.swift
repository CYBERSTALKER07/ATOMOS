import SwiftUI



struct OrdersView: View {
    @State private var hubTab: OrdersHubTab = .active
    @State private var orders: [Order] = []
    @State private var preorders: [WarehousePreorderRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedState = "ALL"
    @State private var proposeTarget: String?
    @State private var proposeDate = Date()
    @State private var rejectTarget: String?
    @State private var reasonInput = ""
    @State private var statusMessage: String?

    private let states = ["ALL", "PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "COMPLETED", "CANCELLED"]

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Queue", selection: $hubTab) {
                    ForEach(OrdersHubTab.allCases) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.vertical, LabTheme.spacingSM)

                OrdersList(
                    hubTab: hubTab,
                    loading: loading,
                    error: error,
                    orders: orders,
                    preorders: preorders,
                    onRetry: { load() },
                    onProposeActive: { proposeTarget = $0; proposeDate = Date(); reasonInput = "" },
                    onRejectActive: { rejectTarget = $0; reasonInput = "" },
                    onProposePreorder: { proposeTarget = $0; proposeDate = Date(); reasonInput = "" },
                    onRejectPreorder: { rejectTarget = $0; reasonInput = "" }
                )
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.orders")
            .navigationDestination(for: String.self) { orderId in
                OrderDetailView(orderId: orderId)
            }
            .toolbar {
                if hubTab == .active {
                    ToolbarItem(placement: .topBarTrailing) {
                        Menu {
                            ForEach(states, id: \.self) { state in
                                Button {
                                    selectedState = state
                                } label: {
                                    if state == selectedState {
                                        Label(state, systemImage: "checkmark")
                                    } else {
                                        Text(state)
                                    }
                                }
                            }
                        } label: {
                            Label("mobile_warehouse.ui.filter", systemImage: "line.3.horizontal.decrease.circle")
                        }
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task(id: "\(hubTab)-\(selectedState)") { load() }
            .refreshable { load() }
            .onChange(of: hubTab) { load() }
            .sheet(item: Binding(
                get: { proposeTarget.map { ProposeSheetOrder(id: $0) } },
                set: { proposeTarget = $0?.id },
            )) { target in
                NavigationStack {
                    Form {
                        DatePicker("New delivery date", selection: $proposeDate, displayedComponents: .date)
                        TextField("supplier_portal.orders.order_ops_actions.text.reason_required", text: $reasonInput, axis: .vertical)
                    }
                    .navigationTitle("mobile_warehouse.ui.propose_new_delivery_date")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.cancel") {
                                proposeTarget = nil
                                reasonInput = ""
                            }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("mobile_warehouse.ui.notify_retailer") {
                                Task { await submitPropose(orderId: target.id) }
                            }
                            .disabled(reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        }
                    }
                }
                .presentationDetents([.medium])
            }
            .alert("Cancel order", isPresented: Binding(
                get: { rejectTarget != nil },
                set: { if !$0 { rejectTarget = nil; reasonInput = "" } },
            )) {
                TextField("supplier_portal.orders.order_ops_actions.text.reason_required", text: $reasonInput)
                Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) {
                    guard let orderId = rejectTarget, !reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
                    Task {
                        do {
                            let isPreorder = preorders.contains { $0.orderId == orderId }
                            if isPreorder {
                                _ = try await WarehouseOperationsService.rejectPreorder(orderId: orderId, reason: reasonInput)
                            } else {
                                _ = try await WarehouseOperationsService.rejectOrder(orderId: orderId, reason: reasonInput)
                            }
                            statusMessage = "Order cancelled · retailer notified"
                            rejectTarget = nil
                            reasonInput = ""
                            load()
                        } catch {
                            statusMessage = error.localizedDescription
                        }
                    }
                }
                Button("common.action.cancel", role: .cancel) { rejectTarget = nil; reasonInput = "" }
            } message: {
                Text("mobile_warehouse.ui.the_retailer_will_be_notified")
            }
            .overlay(alignment: .bottom) {
                if let statusMessage {
                    Text(statusMessage)
                        .font(.caption)
                        .padding(8)
                        .background(.ultraThinMaterial)
                        .clipShape(Capsule())
                        .padding()
                        .onAppear {
                            DispatchQueue.main.asyncAfter(deadline: .now() + 2) { self.statusMessage = nil }
                        }
                }
            }
        }
    }

    private func submitPropose(orderId: String) async {
        let reason = reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else { return }
        do {
            let isPreorder = preorders.contains { $0.orderId == orderId }
            if isPreorder {
                _ = try await WarehouseOperationsService.proposePreorderDelivery(
                    orderId: orderId,
                    proposedDeliveryDate: isoDeliveryDate(from: proposeDate),
                    reason: reason
                )
            } else {
                _ = try await WarehouseOperationsService.proposeOrderDelivery(
                    orderId: orderId,
                    proposedDeliveryDate: isoDeliveryDate(from: proposeDate),
                    reason: reason
                )
            }
            statusMessage = "New delivery date proposed · retailer notified"
            proposeTarget = nil
            reasonInput = ""
            load()
        } catch {
            statusMessage = error.localizedDescription
        }
    }

    private func isoDeliveryDate(from date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        var components = Calendar.current.dateComponents(in: TimeZone(secondsFromGMT: 5 * 3600)!, from: date)
        components.hour = 12
        components.minute = 0
        components.second = 0
        let noon = Calendar.current.date(from: components) ?? date
        return formatter.string(from: noon)
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                if hubTab == .active {
                    let state = selectedState == "ALL" ? nil : selectedState
                    let resp = try await WarehouseService.orders(state: state)
                    orders = resp.orders
                } else {
                    let resp = try await WarehouseService.preorders()
                    preorders = resp.items.isEmpty ? resp.preorders : resp.items
                }
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct ProposeSheetOrder: Identifiable {
    let id: String
}


