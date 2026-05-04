import SwiftUI

struct HistoryView: View {
    @State private var orders: [Order] = []
    @State private var isLoading = false
    @State private var loadError = false
    @State private var filterStatus: OrderStatus?

    private let api = APIClient.shared

    var filteredOrders: [Order] {
        guard let filter = filterStatus else { return orders }
        return orders.filter { $0.status == filter }
    }

    var body: some View {
        VStack(spacing: 0) {
            statusFilters

            ScrollView {
                if isLoading && orders.isEmpty {
                    SkeletonOrderList()
                } else if loadError && orders.isEmpty {
                    ErrorStateView { await loadOrders() }
                } else if filteredOrders.isEmpty {
                    emptyState
                } else {
                    LazyVStack(spacing: AppTheme.spacingMD) {
                        ForEach(Array(filteredOrders.enumerated()), id: \.element.id) { index, order in
                            OrderCardView(order: order)
                                .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.top, AppTheme.spacingSM)
                    .padding(.bottom, AppTheme.spacingXXL)
                }
            }
            .scrollIndicators(.hidden)
        }
        .background(AppTheme.background)
        .task { await loadOrders() }
        .refreshable { await loadOrders() }
    }

    // MARK: - Filters

    private var statusFilters: some View {
        ScrollView(.horizontal) {
            HStack(spacing: AppTheme.spacingSM) {
                filterChip("All", status: nil)
                ForEach([OrderStatus.completed, .cancelled, .inTransit, .pending], id: \.self) { status in
                    filterChip(status.displayName, status: status)
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingSM)
        }
        .scrollIndicators(.hidden)
    }

    private func filterChip(_ title: String, status: OrderStatus?) -> some View {
        Button {
            withAnimation(AnimationConstants.express) { filterStatus = status }
            Haptics.light()
        } label: {
            Text(title)
                .font(.system(.caption, design: .rounded, weight: .semibold))
                .foregroundStyle(filterStatus == status ? .white : AppTheme.textSecondary)
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.vertical, AppTheme.spacingSM)
                .background {
                    if filterStatus == status {
                        Capsule().fill(AppTheme.accentGradient)
                    } else {
                        Capsule().fill(AppTheme.cardBackground)
                            .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
                    }
                }
        }
    }

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "clock").font(.system(size: 32)).foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            Text("No Orders Found")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("Your order history will appear here")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
    }

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
    NavigationStack { HistoryView() }
}
