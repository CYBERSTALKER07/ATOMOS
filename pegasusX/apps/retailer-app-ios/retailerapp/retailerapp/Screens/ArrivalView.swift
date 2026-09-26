import SwiftUI

private struct SupplierDockGroup: Identifiable {
    var id: String { supplierId }
    let supplierId: String
    let supplierName: String
    let orders: [TrackingOrder]
    let totalAmount: Int
    let hasApproaching: Bool
    let hasArrived: Bool
}

struct DockView: View {
    @State private var orders: [TrackingOrder] = []
    @State private var expandedSupplierIds: Set<String> = []
    @State private var revealedTokenOrderIds: Set<String> = []
    @State private var activeQrOrder: TrackingOrder?
    @State private var autoOpenedOrderIds: Set<String> = []
    @State private var isLoading = false
    @State private var isRefreshing = false
    @State private var loadError: String?

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared
    private let dockStates: Set<String> = [
        "DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED", "AWAITING_PAYMENT",
    ]

    private var activeOrders: [TrackingOrder] {
        orders.filter { dockStates.contains($0.state.uppercased()) }
    }

    private var supplierGroups: [SupplierDockGroup] {
        var map: [String: SupplierDockGroup] = [:]
        for order in activeOrders {
            if var group = map[order.supplierId] {
                group = SupplierDockGroup(
                    supplierId: group.supplierId,
                    supplierName: group.supplierName,
                    orders: group.orders + [order],
                    totalAmount: group.totalAmount + order.totalAmount,
                    hasApproaching: group.hasApproaching || order.isApproaching || order.state == "ARRIVING",
                    hasArrived: group.hasArrived || order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT"
                )
                map[order.supplierId] = group
            } else {
                map[order.supplierId] = SupplierDockGroup(
                    supplierId: order.supplierId,
                    supplierName: order.supplierName.isEmpty ? String(order.supplierId.prefix(8)) : order.supplierName,
                    orders: [order],
                    totalAmount: order.totalAmount,
                    hasApproaching: order.isApproaching || order.state == "ARRIVING",
                    hasArrived: order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT"
                )
            }
        }
        return map.values.sorted { $0.totalAmount > $1.totalAmount }
    }

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                if let loadError, orders.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await loadOrders(refreshing: true) }
                    }
                } else if isLoading && orders.isEmpty {
                    RetailerLoadingView(
                        title: "Loading dock queue",
                        message: "Fetching inbound deliveries grouped by supplier."
                    )
                } else if supplierGroups.isEmpty {
                    RetailerEmptyView(
                        title: "Dock Queue Empty",
                        message: "Inbound deliveries grouped by supplier will appear here.",
                        systemImage: "shippingbox"
                    )
                    .padding(.top, AppTheme.spacingXL)
                } else {
                    summaryRow
                    ForEach(supplierGroups) { group in
                        supplierSection(group)
                    }
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingXXL)
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task {
            await loadOrders()
            await observeWebSocket()
        }
        .refreshable { await loadOrders(refreshing: true) }
        .sheet(item: $activeQrOrder) { order in
            NavigationStack {
                VStack(spacing: AppTheme.spacingLG) {
                    Text("mobile_retailer.ui.driver_approaching")
                        .font(.system(.caption, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textTertiary)
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.orderId.suffix(8))"))
                        .font(.system(.title3, design: .rounded, weight: .bold))
                    if let qrData = order.deliveryQRCodePayload {
                        QRCodeView(data: qrData, size: 220)
                    }
                    Text("mobile_retailer.ui.show_this_qr_to_the_driver_for_delivery_confirmation")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                        .multilineTextAlignment(.center)
                    Spacer()
                }
                .padding(AppTheme.spacingXL)
                .navigationTitle("mobile_retailer.ui.delivery_qr")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("common.action.close") { activeQrOrder = nil }
                    }
                }
            }
            .presentationDetents([.medium, .large])
        }
    }

    private var summaryRow: some View {
        HStack(spacing: AppTheme.spacingSM) {
            dockMetric(title: "Queue", value: "\(activeOrders.count)")
            dockMetric(title: "Arrived", value: "\(activeOrders.filter { $0.state == "ARRIVED" }.count)")
            dockMetric(title: "Approaching", value: "\(activeOrders.filter { $0.isApproaching || $0.state == "ARRIVING" }.count)")
        }
    }

    private func dockMetric(title: String, value: String) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: 4) {
                Text(title.uppercased())
                    .font(.system(.caption2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textTertiary)
                Text(value)
                    .font(.system(.title2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(AppTheme.spacingMD)
        }
    }

    private func supplierSection(_ group: SupplierDockGroup) -> some View {
        let expanded = expandedSupplierIds.contains(group.supplierId)
        return LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                Button {
                    withAnimation(.spring(response: 0.3, dampingFraction: 0.8)) {
                        if expanded {
                            expandedSupplierIds.remove(group.supplierId)
                        } else {
                            expandedSupplierIds.insert(group.supplierId)
                        }
                    }
                } label: {
                    HStack {
                        VStack(alignment: .leading, spacing: 3) {
                            Text(group.supplierName)
                                .font(.system(.subheadline, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)
                            Text(L10n.format("mobile_retailer.ui.count_orders_formatted_uzs", "\(group.orders.count)", "\(group.totalAmount.formatted())"))
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        Spacer()
                        if group.hasApproaching {
                            RetailerStatusBadge(text: "Approaching", tint: AppTheme.warning)
                        }
                        if group.hasArrived {
                            RetailerStatusBadge(text: "Arrived", tint: AppTheme.success)
                        }
                        Image(systemName: expanded ? "chevron.up" : "chevron.down")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }
                .buttonStyle(.plain)

                if expanded {
                    ForEach(group.orders) { order in
                        dockOrderRow(order)
                    }
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func dockOrderRow(_ order: TrackingOrder) -> some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.orderId.suffix(8))"))
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                    Text(L10n.format("mobile_retailer.ui.count_items_displaytotal_uzs", "\(order.items.count)", "\(order.displayTotal)"))
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                Spacer()
                RetailerStatusBadge(text: order.state, tint: AppTheme.statusTint(for: order.state))
            }

            if order.isApproaching || order.state == "ARRIVING" {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "bell.badge.fill")
                        .foregroundStyle(AppTheme.warning)
                    Text("mobile_retailer.ui.driver_approaching_your_store")
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                }
                .padding(AppTheme.spacingSM)
                .background(AppTheme.warningSoft.opacity(0.5))
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }

            let canReveal = !order.deliveryToken.isEmpty &&
                (order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT" || order.isApproaching)

            if canReveal {
                HStack(spacing: AppTheme.spacingSM) {
                    Button(revealedTokenOrderIds.contains(order.orderId) ? "Hide QR" : "Reveal QR") {
                        if revealedTokenOrderIds.contains(order.orderId) {
                            revealedTokenOrderIds.remove(order.orderId)
                        } else {
                            revealedTokenOrderIds.insert(order.orderId)
                        }
                    }
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .buttonStyle(.bordered)

                    if revealedTokenOrderIds.contains(order.orderId) {
                        Button("mobile_retailer.ui.show_fullscreen") {
                            activeQrOrder = order
                        }
                        .font(.system(.caption, design: .rounded, weight: .bold))
                        .buttonStyle(.borderedProminent)
                    }
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.surfaceElevated.opacity(0.5))
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
    }

    private func loadOrders(refreshing: Bool = false) async {
        if refreshing { isRefreshing = true } else { isLoading = true }
        loadError = nil
        do {
            let response: TrackingResponse = try await api.get(path: "/v1/retailer/tracking")
            orders = response.orders.filter { !["COMPLETED", "CANCELLED"].contains($0.state.uppercased()) }
            if expandedSupplierIds.isEmpty {
                expandedSupplierIds = Set(supplierGroups.map(\.supplierId))
            } else {
                let available = Set(supplierGroups.map(\.supplierId))
                expandedSupplierIds = expandedSupplierIds.intersection(available)
            }
            maybeAutoOpenQr()
        } catch {
            if orders.isEmpty {
                loadError = "Could not load dock queue. Check your connection and retry."
            }
        }
        isLoading = false
        isRefreshing = false
    }

    private func maybeAutoOpenQr() {
        guard let candidate = activeOrders.first(where: {
            !$0.deliveryToken.isEmpty &&
            ($0.isApproaching || $0.state == "ARRIVED") &&
            !autoOpenedOrderIds.contains($0.orderId)
        }) else { return }

        autoOpenedOrderIds.insert(candidate.orderId)
        expandedSupplierIds.insert(candidate.supplierId)
        revealedTokenOrderIds.insert(candidate.orderId)
        activeQrOrder = candidate
    }

    private func observeWebSocket() async {
        for await event in ws.eventStream() {
            switch event {
            case .driverApproaching(let orderId, let deliveryToken, _, _, _, _):
                orders = orders.map { order in
                    guard order.orderId == orderId else { return order }
                    return TrackingOrder(
                        orderId: order.orderId,
                        supplierId: order.supplierId,
                        supplierName: order.supplierName,
                        warehouseId: order.warehouseId,
                        warehouseName: order.warehouseName,
                        driverId: order.driverId,
                        state: order.state == "IN_TRANSIT" ? "ARRIVING" : order.state,
                        totalAmount: order.totalAmount,
                        orderSource: order.orderSource,
                        driverLatitude: order.driverLatitude,
                        driverLongitude: order.driverLongitude,
                        liveLocationAvailable: order.liveLocationAvailable,
                        isApproaching: true,
                        deliveryToken: deliveryToken.isEmpty ? order.deliveryToken : deliveryToken,
                        createdAt: order.createdAt,
                        items: order.items
                    )
                }
                maybeAutoOpenQr()
            case .orderStatusChanged(let orderId, let newState):
                orders = orders.map { order in
                    guard order.orderId == orderId else { return order }
                    return TrackingOrder(
                        orderId: order.orderId,
                        supplierId: order.supplierId,
                        supplierName: order.supplierName,
                        warehouseId: order.warehouseId,
                        warehouseName: order.warehouseName,
                        driverId: order.driverId,
                        state: newState,
                        totalAmount: order.totalAmount,
                        orderSource: order.orderSource,
                        driverLatitude: order.driverLatitude,
                        driverLongitude: order.driverLongitude,
                        liveLocationAvailable: order.liveLocationAvailable,
                        isApproaching: order.isApproaching,
                        deliveryToken: order.deliveryToken,
                        createdAt: order.createdAt,
                        items: order.items
                    )
                }
            case .orderCompleted(let event):
                orders.removeAll { $0.orderId == event.orderId }
                if activeQrOrder?.orderId == event.orderId { activeQrOrder = nil }
            default:
                break
            }
        }
    }
}

typealias ArrivalView = DockView

#Preview {
    NavigationStack { DockView() }
}
