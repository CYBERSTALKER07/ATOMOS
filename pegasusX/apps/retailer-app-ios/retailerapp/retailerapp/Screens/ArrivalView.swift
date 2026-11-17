import SwiftUI

struct DockView: View {
    @State private var vm = ArrivalViewModel()

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                if let loadError = vm.loadError, vm.orders.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await vm.loadOrders(refreshing: true) }
                    }
                } else if vm.isLoading && vm.orders.isEmpty {
                    RetailerLoadingView(
                        title: "Loading Inbound Deliveries",
                        message: "Fetching inbound deliveries grouped by supplier."
                    )
                } else if vm.supplierGroups.isEmpty {
                    RetailerEmptyView(
                        title: "No Inbound Deliveries",
                        message: "Inbound deliveries grouped by supplier will appear here.",
                        systemImage: "shippingbox"
                    )
                    .padding(.top, AppTheme.spacingXL)
                } else {
                    summaryRow
                    ForEach(vm.supplierGroups) { group in
                        supplierSection(group)
                    }
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingXXL)
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task {
            await vm.loadOrders()
            await vm.observeWebSocket()
        }
        .refreshable { await vm.loadOrders(refreshing: true) }
        .sheet(item: $vm.activeQrOrder) { order in
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
                        Button("common.action.close") { vm.activeQrOrder = nil }
                            .pegasusMinimumHitTarget()
                    }
                }
            }
            .presentationDetents([.medium, .large])
        }
    }

    private var summaryRow: some View {
        HStack(spacing: AppTheme.spacingSM) {
            dockMetric(title: "Queue", value: "\(vm.activeOrders.count)")
            dockMetric(title: "Arrived", value: "\(vm.activeOrders.filter { $0.state == "ARRIVED" }.count)")
            dockMetric(title: "Approaching", value: "\(vm.activeOrders.filter { $0.isApproaching || $0.state == "ARRIVING" }.count)")
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
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value)")
    }

    private func supplierSection(_ group: SupplierDockGroup) -> some View {
        let expanded = vm.expandedSupplierIds.contains(group.supplierId)
        return LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                Button {
                    withAnimation(.spring(response: 0.3, dampingFraction: 0.8)) {
                        vm.toggleSupplierExpanded(group.supplierId)
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
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .pegasusMinimumHitTarget()
                }
                .buttonStyle(.plain)
                .accessibilityLabel("\(group.supplierName), \(group.orders.count) orders, \(expanded ? "expanded" : "collapsed")")

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
            .accessibilityElement(children: .combine)
            .accessibilityLabel("Order \(order.orderId.suffix(8)), status \(order.state), total \(order.displayTotal)")

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
                .accessibilityElement(children: .combine)
            }

            let canReveal = !order.deliveryToken.isEmpty &&
                (order.state == "ARRIVED" || order.state == "AWAITING_PAYMENT" || order.isApproaching)

            if canReveal {
                HStack(spacing: AppTheme.spacingSM) {
                    Button(vm.revealedTokenOrderIds.contains(order.orderId) ? "Hide QR" : "Reveal QR") {
                        vm.toggleRevealToken(order.orderId)
                    }
                    .font(.system(.caption, design: .rounded, weight: .bold))
                    .buttonStyle(.bordered)
                    .pegasusMinimumHitTarget()

                    if vm.revealedTokenOrderIds.contains(order.orderId) {
                        Button("mobile_retailer.ui.show_fullscreen") {
                            Haptics.light()
                            vm.activeQrOrder = order
                        }
                        .font(.system(.caption, design: .rounded, weight: .bold))
                        .buttonStyle(.borderedProminent)
                        .pegasusMinimumHitTarget()
                    }
                }
            }
        }
        .padding(AppTheme.spacingMD)
        .background(AppTheme.surfaceElevated.opacity(0.5))
        .clipShape(.rect(cornerRadius: AppTheme.radiusMD))
    }
}

typealias ArrivalView = DockView

#Preview {
    NavigationStack { DockView() }
}
