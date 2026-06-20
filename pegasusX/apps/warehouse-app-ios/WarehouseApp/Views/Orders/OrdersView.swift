import SwiftUI

private enum OrdersHubTab: String, CaseIterable, Identifiable {
    case active = "Active orders"
    case preorders = "Pre-orders"

    var id: String { rawValue }
}

struct OrdersView: View {
    @State private var hubTab: OrdersHubTab = .active
    @State private var orders: [Order] = []
    @State private var preorders: [WarehousePreorderRow] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedState = "ALL"
    @State private var delayTarget: String?
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

                Group {
                    if loading {
                        ProgressView()
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else if let error {
                        ContentUnavailableView {
                            Label("Error", systemImage: "exclamationmark.triangle")
                        } description: {
                            Text(error)
                        } actions: {
                            Button("Retry") { load() }
                        }
                    } else if hubTab == .active && orders.isEmpty {
                        ContentUnavailableView("No Orders", systemImage: "cart", description: Text("No orders found for this filter"))
                    } else if hubTab == .preorders && preorders.isEmpty {
                        ContentUnavailableView("No pre-orders", systemImage: "calendar")
                    } else {
                        List {
                            if hubTab == .active {
                                ForEach(orders) { order in
                                    NavigationLink(value: order.orderId) {
                                        OrderOpsCardView(
                                            title: order.retailerName.isEmpty ? String(order.orderId.prefix(8)) : order.retailerName,
                                            orderId: order.orderId,
                                            state: order.state,
                                            amountLabel: "\(order.totalUzs.formatted()) UZS",
                                            canDelay: orderActionFlags(order.state).canDelay,
                                            canReject: orderActionFlags(order.state).canReject,
                                            onDelay: { delayTarget = order.orderId; reasonInput = "" },
                                            onReject: { rejectTarget = order.orderId; reasonInput = "" },
                                        )
                                    }
                                }
                            } else {
                                ForEach(preorders) { row in
                                    NavigationLink(value: row.orderId) {
                                        OrderOpsCardView(
                                            title: String(row.orderId.prefix(12)),
                                            orderId: row.orderId,
                                            state: row.status,
                                            amountLabel: row.requestedDeliveryDate ?? "Pre-order",
                                            badge: "Pre-order",
                                            canDelay: true,
                                            canReject: true,
                                            delayLabel: "Propose delivery",
                                            onDelay: nil,
                                            onReject: nil,
                                        )
                                    }
                                }
                            }
                        }
                        .listStyle(.insetGrouped)
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Orders")
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
                            Label("Filter", systemImage: "line.3.horizontal.decrease.circle")
                        }
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task(id: "\(hubTab)-\(selectedState)") { load() }
            .refreshable { load() }
            .onChange(of: hubTab) { load() }
            .alert("Delay delivery", isPresented: Binding(
                get: { delayTarget != nil },
                set: { if !$0 { delayTarget = nil; reasonInput = "" } },
            )) {
                TextField("Reason (optional)", text: $reasonInput)
                Button("Delay") {
                    guard let orderId = delayTarget else { return }
                    Task {
                        do {
                            _ = try await WarehouseService.delayOrder(
                                orderId: orderId,
                                body: WarehouseOrderMutationRequest(reason: reasonInput.isEmpty ? nil : reasonInput),
                            )
                            statusMessage = "Order delayed"
                            delayTarget = nil
                            reasonInput = ""
                            load()
                        } catch {
                            statusMessage = error.localizedDescription
                        }
                    }
                }
                Button("Cancel", role: .cancel) { delayTarget = nil; reasonInput = "" }
            }
            .alert("Reject order", isPresented: Binding(
                get: { rejectTarget != nil },
                set: { if !$0 { rejectTarget = nil; reasonInput = "" } },
            )) {
                TextField("Reason (required)", text: $reasonInput)
                Button("Reject", role: .destructive) {
                    guard let orderId = rejectTarget, !reasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return }
                    Task {
                        do {
                            _ = try await WarehouseService.rejectOrder(
                                orderId: orderId,
                                body: WarehouseOrderMutationRequest(reason: reasonInput),
                            )
                            statusMessage = "Order rejected"
                            rejectTarget = nil
                            reasonInput = ""
                            load()
                        } catch {
                            statusMessage = error.localizedDescription
                        }
                    }
                }
                Button("Cancel", role: .cancel) { rejectTarget = nil; reasonInput = "" }
            } message: {
                Text("The retailer will be notified.")
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

private struct OrderActionFlags {
    let canDelay: Bool
    let canReject: Bool
}

private func orderActionFlags(_ state: String) -> OrderActionFlags {
    let s = state.uppercased()
    return OrderActionFlags(
        canDelay: s == "PENDING" || s == "LOADED",
        canReject: ["PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED"].contains(s),
    )
}

private struct OrderOpsCardView: View {
    let title: String
    let orderId: String
    let state: String
    let amountLabel: String
    var badge: String?
    var canDelay: Bool
    var canReject: Bool
    var delayLabel: String = "Delay delivery"
    var onDelay: (() -> Void)?
    var onReject: (() -> Void)?

    var body: some View {
        HStack(alignment: .top) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                HStack {
                    Text(title).font(.headline)
                    Text(state)
                        .font(.caption.bold())
                        .padding(.horizontal, LabTheme.spacingSM)
                        .padding(.vertical, LabTheme.spacingXS)
                        .background(.quaternary, in: Capsule())
                    if let badge {
                        Text(badge)
                            .font(.caption2)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(.orange.opacity(0.15))
                            .clipShape(Capsule())
                    }
                }
                Text(orderId).font(.caption.monospaced()).foregroundStyle(.secondary)
            }
            Spacer()
            Text(amountLabel)
                .font(.subheadline.monospacedDigit())
                .foregroundStyle(.secondary)
        }
        .contextMenu {
            if let onDelay, canDelay {
                Button(delayLabel) { onDelay() }
            }
            if let onReject, canReject {
                Button("Reject", role: .destructive) { onReject() }
            }
        }
    }
}
