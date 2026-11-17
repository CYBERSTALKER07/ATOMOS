import SwiftUI

struct ActiveOrderView: View {
    @State private var vm = ActiveOrderViewModel()

    var body: some View {
        ScrollView {
            if vm.isLoading && vm.activeOrders.isEmpty {
                SkeletonOrderList()
            } else if vm.loadError && vm.activeOrders.isEmpty {
                ErrorStateView { await vm.loadActiveOrders() }
            } else if vm.activeOrders.isEmpty {
                emptyState
            } else {
                LazyVStack(spacing: AppTheme.spacingLG) {
                    ForEach(Array(vm.activeOrders.enumerated()), id: \.element.id) { index, order in
                        activeOrderCard(order)
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
        .task { await vm.loadActiveOrders() }
        .refreshable { await vm.loadActiveOrders() }
    }

    // MARK: - Active Order Card

    private func activeOrderCard(_ order: Order) -> some View {
        LabCard {
            VStack(spacing: AppTheme.spacingLG) {
                // Header
                HStack(alignment: .top) {
                    ZStack {
                        Circle()
                            .fill(AppTheme.success.opacity(0.12))
                            .frame(width: 42, height: 42)
                        Image(systemName: "bolt.fill")
                            .font(.body.weight(.semibold))
                            .foregroundStyle(AppTheme.success)
                    }

                    VStack(alignment: .leading, spacing: 3) {
                        Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text(order.status.displayName)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.accent)
                    }

                    Spacer()

                    HStack(spacing: 6) {
                        Circle()
                            .fill(AppTheme.success)
                            .frame(width: 6, height: 6)
                        Text("TACTICAL")
                            .font(.system(.caption2, design: .rounded, weight: .black))
                            .foregroundStyle(AppTheme.success)
                    }
                    .padding(.horizontal, 10).padding(.vertical, 5)
                    .background(AppTheme.successSoft.opacity(0.4))
                    .clipShape(.capsule)
                }

                // ETA
                if let eta = order.estimatedDelivery {
                    VStack(spacing: AppTheme.spacingSM) {
                        Text("mobile_retailer.ui.impact_time")
                            .font(.system(.caption2, design: .rounded, weight: .black))
                            .foregroundStyle(AppTheme.textTertiary)
                            .tracking(1)
                        CountdownText(targetISO: eta, font: .system(.title2, design: .monospaced, weight: .bold), color: AppTheme.accent)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(AppTheme.spacingMD)
                    .background {
                        RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous)
                            .fill(AppTheme.accentSoft.opacity(0.12))
                    }
                }

                // Items
                VStack(alignment: .leading, spacing: 6) {
                    ForEach(order.items) { item in
                        HStack(spacing: AppTheme.spacingSM) {
                            Circle().fill(AppTheme.accentSoft.opacity(0.5)).frame(width: 6, height: 6)
                            Text(item.productName)
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                            Spacer()
                            Text(L10n.format("mobile_retailer.ui.quantity", "\(item.quantity)"))
                                .font(.system(.caption2, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textTertiary)
                                .padding(.horizontal, 6).padding(.vertical, 2)
                                .background(AppTheme.surfaceElevated)
                                .clipShape(.capsule)
                        }
                    }
                }

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

                // QR Code — JIT: only after dispatch
                if order.status.hasDeliveryToken, let qrData = order.deliveryQRCodePayload {
                    VStack(spacing: AppTheme.spacingSM) {
                        Text("mobile_retailer.ui.qr_code_for_driver")
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textSecondary)
                        QRCodeView(data: qrData, size: 180)
                        Text("mobile_retailer.ui.show_this_code_for_delivery_confirmation")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                } else if !order.status.hasDeliveryToken {
                    VStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "lock.fill")
                            .font(.title2)
                            .foregroundStyle(AppTheme.textTertiary)
                        Text("mobile_retailer.ui.awaiting_dispatch")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textSecondary)
                        Text("mobile_retailer.ui.qr_code_will_appear_when_dispatched")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .frame(maxWidth: .infinity)
                }

                // Total
                HStack {
                    Text("retailer_desktop.pos.text.total")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                    Text(order.displayTotal)
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.accent)
                }
            }
            .padding(AppTheme.spacingLG)
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Active order \(order.id.suffix(3)), \(order.status.displayName), total \(order.displayTotal)")
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "shippingbox")
                    .font(.title)
                    .foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            Text("mobile_retailer.ui.no_active_orders")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("mobile_retailer.ui.orders_en_route_will_appear_here_with_qr_codes")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
        .accessibilityElement(children: .combine)
    }
}

#Preview {
    NavigationStack { ActiveOrderView() }
}
