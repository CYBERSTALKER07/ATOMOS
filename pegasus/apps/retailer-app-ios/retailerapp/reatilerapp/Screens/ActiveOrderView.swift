import SwiftUI

struct ActiveOrderView: View {
    @State private var activeOrders: [Order] = []
    @State private var isLoading = false
    @State private var loadError = false

    private let api = APIClient.shared

    var body: some View {
        ScrollView {
            if isLoading && activeOrders.isEmpty {
                SkeletonOrderList()
            } else if loadError && activeOrders.isEmpty {
                ErrorStateView { await loadActiveOrders() }
            } else if activeOrders.isEmpty {
                emptyState
            } else {
                LazyVStack(spacing: AppTheme.spacingLG) {
                    ForEach(Array(activeOrders.enumerated()), id: \.element.id) { index, order in
                        activeOrderCard(order)
                            .staggeredSlideIn(index: index)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .padding(.bottom, AppTheme.spacingXXL)
            }
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task { await loadActiveOrders() }
        .refreshable { await loadActiveOrders() }
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
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(AppTheme.success)
                    }

                    VStack(alignment: .leading, spacing: 3) {
                        Text("Order #\(order.id.suffix(3))")
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
                            .frame(width: 6, height: 6) // Slightly smaller
                        Text("TACTICAL") // Strategic status
                            .font(.system(size: 10, weight: .black, design: .rounded))
                            .foregroundStyle(AppTheme.success)
                    }
                    .padding(.horizontal, 10).padding(.vertical, 5)
                    .background(AppTheme.successSoft.opacity(0.4))
                    .clipShape(.capsule)
                }

                // ETA
                if let eta = order.estimatedDelivery {
                    VStack(spacing: AppTheme.spacingSM) {
                        Text("IMPACT TIME") // Tactical label
                            .font(.system(size: 10, weight: .black, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .tracking(1)
                        CountdownText(targetISO: eta, font: .system(.title2, design: .monospaced, weight: .bold), color: AppTheme.accent)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(AppTheme.spacingMD)
                    .background {
                        RoundedRectangle(cornerRadius: AppTheme.radiusMD, style: .continuous)
                            .fill(AppTheme.accentSoft.opacity(0.12)) // Softer background
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
                            Text("×\(item.quantity)")
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
                if order.status.hasDeliveryToken, let qrData = order.qrCode {
                    VStack(spacing: AppTheme.spacingSM) {
                        Text("QR Code for Driver")
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textSecondary)
                        QRCodeView(data: qrData, size: 180)
                        Text("Show this code for delivery confirmation")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                } else if !order.status.hasDeliveryToken {
                    VStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "lock.fill")
                            .font(.system(size: 28))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text("Awaiting Dispatch")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textSecondary)
                        Text("QR code will appear when dispatched")
                            .font(.system(.caption2, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .frame(maxWidth: .infinity)
                }

                // Total
                HStack {
                    Text("Total")
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
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "shippingbox").font(.system(size: 32)).foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            Text("No Active Orders")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("Orders en route will appear here with QR codes")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadActiveOrders() async {
        isLoading = true
        loadError = false
        do { let r: [Order] = try await api.get(path: "/v1/orders?state=IN_TRANSIT"); activeOrders = r }
        catch { activeOrders = []; loadError = true }
        isLoading = false
    }
}

#Preview {
    NavigationStack { ActiveOrderView() }
}
