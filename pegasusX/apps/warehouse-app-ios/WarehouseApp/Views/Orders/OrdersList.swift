import SwiftUI

enum OrdersHubTab: String, CaseIterable, Identifiable {
    case active = "Active orders"
    case preorders = "Pre-orders"
    var id: String { rawValue }
}

struct OrdersList: View {
    let hubTab: OrdersHubTab
    let loading: Bool
    let error: String?
    let orders: [Order]
    let preorders: [WarehousePreorderRow]
    let onRetry: () -> Void
    let onProposeActive: (String) -> Void
    let onRejectActive: (String) -> Void
    let onProposePreorder: (String) -> Void
    let onRejectPreorder: (String) -> Void

    var body: some View {
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
                    Button("Retry") { onRetry() }
                }
            } else if hubTab == .active && orders.isEmpty {
                ContentUnavailableView("No Orders", systemImage: "cart", description: Text("No orders found for this filter"))
            } else if hubTab == .preorders && preorders.isEmpty {
                ContentUnavailableView("No pre-orders", systemImage: "calendar")
            } else {
                ResponsiveGridContentWrapper {
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
                                    onDelay: { onProposeActive(order.orderId) },
                                    onReject: { onRejectActive(order.orderId) }
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
                                    onDelay: { onProposePreorder(row.orderId) },
                                    onReject: { onRejectPreorder(row.orderId) }
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

private struct OrderActionFlags {
    let canDelay: Bool
    let canReject: Bool
}

private func orderActionFlags(_ state: String) -> OrderActionFlags {
    let s = state.uppercased()
    let terminal = s == "COMPLETED" || s == "CANCELLED"
    let inFlight = s == "LOADED" || s == "IN_TRANSIT"
    return OrderActionFlags(
        canDelay: !terminal && !inFlight,
        canReject: ["PENDING", "LOADED", "IN_TRANSIT", "SCHEDULED", "AUTO_ACCEPTED", "DELAYED", "ARRIVED"].contains(s)
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
    var delayLabel: String = "Propose date"
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
        .labCard()
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
