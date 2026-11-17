import SwiftUI

struct HistoryView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = HistoryViewModel()

    var body: some View {
        VStack(spacing: 0) {
            statusFilters

            ScrollView {
                if vm.isLoading && vm.orders.isEmpty {
                    SkeletonOrderList()
                } else if vm.loadError && vm.orders.isEmpty {
                    ErrorStateView { await vm.loadOrders() }
                } else if vm.filteredOrders.isEmpty {
                    emptyState
                } else {
                    LazyVStack(spacing: AppTheme.spacingMD) {
                        ForEach(Array(vm.filteredOrders.enumerated()), id: \.element.id) { index, order in
                            OrderCardView(order: order)
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
        }
        .background(AppTheme.background)
        .task { await vm.loadOrders() }
        .task(id: refreshCenter.refreshToken) { await vm.loadOrders() }
        .refreshable { await vm.loadOrders() }
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
        let isSelected = vm.filterStatus == status
        return Button {
            withAnimation(AnimationConstants.express) { vm.filterStatus = status }
            Haptics.light()
        } label: {
            Text(title)
                .font(.system(.caption, design: .rounded, weight: .semibold))
                .foregroundStyle(isSelected ? .white : AppTheme.textSecondary)
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.vertical, AppTheme.spacingSM)
                .background {
                    if isSelected {
                        Capsule().fill(AppTheme.accentGradient)
                    } else {
                        Capsule().fill(AppTheme.cardBackground)
                            .shadow(color: AppTheme.shadowColor, radius: 2, y: 1)
                    }
                }
                .pegasusMinimumHitTarget()
        }
        .accessibilityLabel("Filter by \(title)")
        .accessibilityAddTraits(isSelected ? .isSelected : [])
    }

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 80)
            ZStack {
                Circle().fill(AppTheme.accentSoft.opacity(0.3)).frame(width: 80, height: 80)
                Image(systemName: "clock")
                    .font(.title)
                    .foregroundStyle(AppTheme.accent.opacity(0.4))
            }
            Text("mobile_retailer.ui.no_orders_found")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("mobile_retailer.ui.your_order_history_will_appear_here")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
        .accessibilityElement(children: .combine)
    }
}

#Preview {
    NavigationStack { HistoryView() }
}
