import SwiftUI

struct ActiveDeliveriesView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = ActiveDeliveriesViewModel()

    private var orders: [Order] { vm.orders }
    private var isLoading: Bool { vm.isLoading }
    private var approachingOrderIds: Set<String> { vm.approachingOrderIds }

    var body: some View {
        @Bindable var vm = vm
        ZStack {
            ScrollView {
                if isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: 200)
                        .tint(AppTheme.accent)
                } else if orders.isEmpty {
                    emptyState
                } else {
                    LazyVStack(spacing: AppTheme.spacingMD) {
                        ForEach(Array(orders.enumerated()), id: \.element.id) { index, order in
                            DeliveryCard(
                                order: order,
                                isApproaching: approachingOrderIds.contains(order.id),
                                onTap: {
                                    Haptics.light()
                                    vm.selectedOrder = order
                                },
                                onShowQR: {
                                    Haptics.selection()
                                    vm.qrOverlayOrder = order
                                }
                            )
                            .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.top, AppTheme.spacingSM)
                    .padding(.bottom, AppTheme.spacingXXL)
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .scrollIndicators(.hidden)
            .background(AppTheme.background)
            .task { await loadOrders() }
            .task(id: refreshCenter.refreshToken) { await loadOrders() }
            .refreshable { await loadOrders() }
            .alert("Failed to Load", isPresented: $vm.loadError) {
                Button("common.action.retry") { Task { await loadOrders() } }
                Button("OK", role: .cancel) {}
            } message: {
                Text("mobile_retailer.ui.could_not_load_deliveries_check_your_connection")
            }
            // 3/4 Detail Sheet
            .sheet(item: $vm.selectedOrder) { order in
                OrderDetailSheet(order: order, onCancelled: {
                    vm.orders.removeAll { $0.id == order.id }
                })
                .presentationDetents([.fraction(0.75)])
                .presentationDragIndicator(.visible)
            }

            // Quick QR Overlay — guard: only if token exists and driver nearby
            if let qrOrder = vm.qrOverlayOrder, qrOrder.status.hasDeliveryToken,
               approachingOrderIds.contains(qrOrder.id) || qrOrder.status == .arrived {
                QROverlay(order: qrOrder) {
                    withAnimation(AnimationConstants.fluid) {
                        vm.qrOverlayOrder = nil
                    }
                }
                .transition(.opacity)
                .zIndex(200)
            }
        }
        .animation(AnimationConstants.fluid, value: vm.qrOverlayOrder?.id)
        .task { await listenForApproaching() }
    }

    // MARK: - WebSocket Listener

    private func listenForApproaching() async {
        await vm.listenForApproaching()
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 100)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: "shippingbox")
                    .font(.title)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            Text("mobile_retailer.ui.no_active_orders")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("mobile_retailer.ui.your_en_route_and_confirmed_deliveries_will_appear_here")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .combine)
    }

    // MARK: - API

    private func loadOrders() async {
        await vm.loadOrders()
    }
}

// MARK: - Delivery Card

struct DeliveryCard: View {
    let order: Order
    var isApproaching: Bool = false
    let onTap: () -> Void
    let onShowQR: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Header
            HStack(alignment: .top) {
                ZStack {
                    Circle()
                        .fill(statusColor.opacity(0.1))
                        .frame(width: 42, height: 42)
                    Image(systemName: statusIcon)
                        .font(.body.weight(.semibold))
                        .foregroundStyle(statusColor)
                }

                VStack(alignment: .leading, spacing: 3) {
                    if let supplierName = order.supplierName, !supplierName.isEmpty {
                        Text(supplierName)
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.accent)
                    }
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text(L10n.format("mobile_retailer.ui.itemcount_items_displaytotal", "\(order.itemCount)", "\(order.displayTotal)"))
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                // Status badge
                HStack(spacing: 4) {
                    Circle().fill(statusColor).frame(width: 6, height: 6)
                    Text(order.status.displayName)
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                }
                .foregroundStyle(statusColor)
                .padding(.horizontal, 10).padding(.vertical, 5)
                .background(statusColor.opacity(0.08))
                .clipShape(.capsule)
            }

            // ETA countdown
            if let eta = order.estimatedDelivery {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "clock")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                    CountdownText(targetISO: eta, font: .system(.caption, design: .monospaced, weight: .bold), color: AppTheme.textPrimary)
                    Text("mobile_retailer.ui.until_arrival")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                }
                .padding(AppTheme.spacingSM)
                .background(AppTheme.surfaceElevated)
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }

            // Quick items preview
            HStack(spacing: 6) {
                ForEach(order.items.prefix(3)) { item in
                    Text(item.productName.split(separator: " ").first.map(String.init) ?? "")
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textSecondary)
                        .padding(.horizontal, 8).padding(.vertical, 4)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.capsule)
                }
                if order.items.count > 3 {
                    Text(L10n.format("mobile_retailer.ui.count_3", "\(order.items.count - 3)"))
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }

            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            // Actions row
            HStack(spacing: AppTheme.spacingMD) {
                // View Details
                Button(action: onTap) {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.text")
                            .font(.caption.weight(.semibold))
                        Text("mobile_retailer.ui.details")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textPrimary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                    .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("View details for order \(order.id.suffix(3))")

                // Quick QR — available when driver is nearby or arrived
                let qrUnlocked = isApproaching || order.status == .arrived
                if order.status.hasDeliveryToken && qrUnlocked {
                    Button(action: onShowQR) {
                        HStack(spacing: 4) {
                            Image(systemName: "qrcode")
                                .font(.caption.weight(.semibold))
                            Text("mobile_retailer.ui.show_qr")
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                        .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel("Show QR code for order \(order.id.suffix(3))")
                } else if order.status.hasDeliveryToken {
                    HStack(spacing: 4) {
                        Image(systemName: "qrcode")
                            .font(.caption.weight(.semibold))
                        Text("mobile_retailer.ui.awaiting_driver")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                } else {
                    HStack(spacing: 4) {
                        Image(systemName: "qrcode")
                            .font(.caption.weight(.semibold))
                        Text("mobile_retailer.ui.awaiting_dispatch")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }

                Spacer()
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Delivery \(order.id.suffix(3)), \(order.status.displayName), \(order.displayTotal)")
    }

    private var statusColor: Color {
        switch order.status {
        case .pending: AppTheme.warning
        case .loaded: AppTheme.info
        case .dispatched: AppTheme.accent
        case .inTransit: AppTheme.success
        case .arrived: AppTheme.success
        case .completed: AppTheme.success
        case .cancelled: AppTheme.destructive
        case .awaitingPayment: AppTheme.warning
        case .pendingCashCollection: AppTheme.warning
        default: AppTheme.textSecondary
        }
    }

    private var statusIcon: String {
        switch order.status {
        case .pending: "clock"
        case .loaded: "shippingbox"
        case .dispatched: "truck"
        case .inTransit: "shippingbox.fill"
        case .arrived: "checkmark.circle"
        case .completed: "checkmark.circle.fill"
        case .cancelled: "xmark.circle"
        case .awaitingPayment: "creditcard"
        case .pendingCashCollection: "banknote"
        default: "circle"
        }
    }
}

#Preview {
    NavigationStack {
        ActiveDeliveriesView()
    }
}
