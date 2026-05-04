import SwiftUI

struct InboxView: View {
    @State private var orders: [Order] = []
    @State private var isLoading = false
    @State private var loadError = false

    private let api = APIClient.shared

    var incomingOrders: [Order] {
        orders.filter { $0.status == .inTransit || $0.status == .loaded || $0.status == .arrived }
    }

    var body: some View {
        ScrollView {
            if isLoading && orders.isEmpty {
                SkeletonOrderList()
            } else if loadError && orders.isEmpty {
                ErrorStateView { await loadOrders() }
            } else if incomingOrders.isEmpty {
                emptyState
            } else {
                LazyVStack(spacing: AppTheme.spacingLG) {
                    ForEach(Array(incomingOrders.enumerated()), id: \.element.id) { index, order in
                        inboxCard(order)
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
        .task { await loadOrders() }
        .refreshable { await loadOrders() }
    }

    // MARK: - Inbox Card

    private func inboxCard(_ order: Order) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                // Header
                HStack(alignment: .top) {
                    ZStack {
                        Circle()
                            .fill(badgeColor(order.status).opacity(0.12))
                            .frame(width: 42, height: 42)
                        Image(systemName: "shippingbox.fill")
                            .font(.system(size: 16, weight: .semibold))
                            .foregroundStyle(badgeColor(order.status))
                    }

                    VStack(alignment: .leading, spacing: 3) {
                        Text("Order #\(order.id.suffix(3))")
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("\(order.itemCount) items · \(order.displayTotal)")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    statusBadge(order.status)
                }

                // ETA
                if let eta = order.estimatedDelivery {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "clock")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(AppTheme.accent)
                        CountdownText(targetISO: eta, font: .system(.caption, design: .monospaced, weight: .bold), color: AppTheme.accent)
                        Spacer()
                    }
                    .padding(AppTheme.spacingSM)
                    .background(AppTheme.accentSoft.opacity(0.2))
                    .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
                }

                // Items
                VStack(alignment: .leading, spacing: 6) {
                    ForEach(order.items) { item in
                        HStack(spacing: AppTheme.spacingSM) {
                            Circle().fill(AppTheme.accentSoft.opacity(0.5)).frame(width: 6, height: 6)
                            Text(item.productName)
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                                .lineLimit(1)
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

                // QR Code
                if let qrData = order.qrCode {
                    HStack {
                        Spacer()
                        VStack(spacing: AppTheme.spacingSM) {
                            QRCodeView(data: qrData, size: 120)
                            Text("Show to driver on arrival")
                                .font(.system(.caption2, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        Spacer()
                    }
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func statusBadge(_ status: OrderStatus) -> some View {
        HStack(spacing: 4) {
            if status.isActive {
                Circle().fill(badgeColor(status)).frame(width: 6, height: 6)
            }
            Text(status.displayName)
                .font(.system(size: 11, weight: .bold, design: .rounded))
        }
        .foregroundStyle(badgeColor(status))
        .padding(.horizontal, 10).padding(.vertical, 5)
        .background(badgeColor(status).opacity(0.1))
        .clipShape(.capsule)
    }

    private func badgeColor(_ status: OrderStatus) -> Color {
        switch status {
        case .inTransit: AppTheme.accent
        case .loaded: AppTheme.info
        case .dispatched: AppTheme.accent
        case .arrived: AppTheme.success
        case .pending: AppTheme.warning
        case .awaitingPayment: AppTheme.warning
        case .pendingCashCollection: AppTheme.warning
        case .completed: AppTheme.success
        case .cancelled: AppTheme.destructive
        default: AppTheme.textSecondary
        }
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "tray").font(.system(size: 32)).foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            Text("No Incoming Deliveries")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("Incoming orders will appear here with QR codes")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        if rid.isEmpty {
            orders = []
            loadError = true
            isLoading = false
            return
        }
        isLoading = true
        loadError = false
        do { let r: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders"); orders = r }
        catch { orders = []; loadError = true }
        isLoading = false
    }
}

#Preview {
    NavigationStack { InboxView() }
}
