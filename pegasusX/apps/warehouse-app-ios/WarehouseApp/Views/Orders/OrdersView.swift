import SwiftUI

struct OrdersView: View {
    @State private var orders: [Order] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedState = "ALL"

    private let states = ["ALL", "PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "COMPLETED", "CANCELLED"]

    var body: some View {
        NavigationStack {
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
                } else if orders.isEmpty {
                    ContentUnavailableView("No Orders", systemImage: "cart", description: Text("No orders found for this filter"))
                } else {
                    List(orders) { order in
                        NavigationLink(value: order.orderId) {
                            OrderRow(order: order)
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Orders")
            .navigationDestination(for: String.self) { orderId in
                OrderDetailView(orderId: orderId)
            }
            .toolbar {
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
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
            .onChange(of: selectedState) { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let state = selectedState == "ALL" ? nil : selectedState
                let resp = try await WarehouseService.orders(state: state)
                orders = resp.orders
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

private struct OrderRow: View {
    let order: Order

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(order.retailerName.isEmpty ? String(order.orderId.prefix(8)) : order.retailerName)
                    .font(.headline)
                Text("\(order.totalUzs.formatted()) UZS")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Text(order.state)
                .font(.caption.bold())
                .padding(.horizontal, LabTheme.spacingSM)
                .padding(.vertical, LabTheme.spacingXS)
                .background(.quaternary, in: Capsule())
        }
    }
}
