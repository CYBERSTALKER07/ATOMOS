import SwiftUI

struct ArrivalView: View {
    @State private var orders: [Order] = []
    @State private var isLoading = false
    @State private var updatingIds: Set<String> = []
    @State private var updateError = false

    private let api = APIClient.shared

    var body: some View {
        ScrollView {
            if orders.isEmpty {
                emptyState
            } else {
                LazyVStack(spacing: AppTheme.spacingLG) {
                    ForEach(Array(orders.enumerated()), id: \.element.id) { index, order in
                        arrivalCard(order)
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
        .alert("Update Failed", isPresented: $updateError) {
            Button("OK", role: .cancel) {}
        } message: {
            Text("Could not update order status. Please try again.")
        }
    }

    // MARK: - Arrival Card

    private func arrivalCard(_ order: Order) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                // Header
                HStack(alignment: .top) {
                    ZStack {
                        Circle()
                            .fill(AppTheme.accent.opacity(0.12))
                            .frame(width: 42, height: 42)
                        Image(systemName: "arrow.down.circle.fill")
                            .font(.system(size: 18, weight: .semibold))
                            .foregroundStyle(AppTheme.accent)
                    }

                    VStack(alignment: .leading, spacing: 3) {
                        Text("Order #\(order.id.suffix(3))")
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text(order.status.displayName)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    // Live indicator
                    HStack(spacing: 6) {
                        Circle()
                            .fill(order.status == .inTransit ? AppTheme.success : AppTheme.warning)
                            .frame(width: 7, height: 7)
                            .shadow(color: (order.status == .inTransit ? AppTheme.success : AppTheme.warning).opacity(0.5), radius: 4)
                        Text(order.status == .inTransit ? "LIVE" : "WAITING")
                            .font(.system(size: 10, weight: .bold, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    .padding(.horizontal, 10).padding(.vertical, 5)
                    .background(.ultraThinMaterial)
                    .clipShape(.capsule)
                }

                // ETA
                if let eta = order.estimatedDelivery {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "clock")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(AppTheme.accent)
                        CountdownText(targetISO: eta, font: .system(.subheadline, design: .monospaced, weight: .bold), color: AppTheme.accent)
                        Text("until arrival")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                        Spacer()
                    }
                    .padding(AppTheme.spacingSM)
                    .background(AppTheme.accentSoft.opacity(0.2))
                    .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
                }

                // Items summary
                Text("\(order.itemCount) items · \(order.displayTotal)")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

                // Actions
                HStack(spacing: AppTheme.spacingMD) {
                    LabButton("Confirm", variant: .primary, icon: "checkmark.circle") {
                        Task { await updateStatus(orderId: order.id, status: "COMPLETED") }
                    }
                    LabButton("Reject", variant: .destructive, icon: "xmark.circle") {
                        Task { await updateStatus(orderId: order.id, status: "CANCELLED") }
                    }
                }
                .disabled(updatingIds.contains(order.id))
                .opacity(updatingIds.contains(order.id) ? 0.5 : 1)
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
            Text("No Active Arrivals")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("Incoming deliveries will appear here")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do { let r: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders"); orders = r.filter { $0.status.isActive } }
        catch { orders = [] }
        isLoading = false
    }

    private func updateStatus(orderId: String, status: String) async {
        guard !updatingIds.contains(orderId) else { return }
        updatingIds.insert(orderId)
        defer { updatingIds.remove(orderId) }
        do {
            let _: Order = try await api.patch(path: "/v1/orders/\(orderId)/status", body: ["status": status])
            Haptics.success()
            withAnimation(AnimationConstants.fluid) { orders.removeAll { $0.id == orderId } }
        } catch {
            Haptics.error()
            updateError = true
        }
    }
}

#Preview {
    NavigationStack { ArrivalView() }
}
