import SwiftUI

struct ArrivalView: View {
    @State private var trackingOrders: [TrackingOrder] = []
    @State private var isLoading = false
    @State private var loadError: String?

    private let api = APIClient.shared
    private let arrivalStates: Set<String> = ["IN_TRANSIT", "DISPATCHED", "ARRIVING", "ARRIVED"]

    var body: some View {
        ScrollView {
            if isLoading && trackingOrders.isEmpty {
                RetailerLoadingView(
                    title: "Loading arrivals",
                    message: "Fetching live delivery tracking for incoming orders."
                )
            } else if let loadError, trackingOrders.isEmpty {
                RetailerErrorView(message: loadError) {
                    Task { await loadOrders() }
                }
            } else if trackingOrders.isEmpty {
                emptyState
            } else {
                LazyVStack(spacing: AppTheme.spacingLG) {
                    RetailerSectionHeader(
                        title: "Live arrivals",
                        subtitle: "\(trackingOrders.count) en route",
                        icon: "location.fill"
                    )
                    .padding(.top, AppTheme.spacingSM)

                    ForEach(Array(trackingOrders.enumerated()), id: \.element.id) { index, order in
                        arrivalCard(order)
                            .staggeredSlideIn(index: index)
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingXXL)
            }
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task { await loadOrders() }
        .refreshable { await loadOrders() }
    }

    private func arrivalCard(_ order: TrackingOrder) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
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
                        Text("Order #\(order.orderId.suffix(8))")
                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text(order.supplierName)
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    RetailerStatusBadge(
                        text: order.state == "IN_TRANSIT" ? "LIVE" : "WAITING",
                        tint: order.state == "IN_TRANSIT" ? AppTheme.success : AppTheme.warning,
                        showsLiveDot: order.state == "IN_TRANSIT"
                    )
                }

                if order.isApproaching {
                    HStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "bell.badge.fill")
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(AppTheme.warning)
                        Text("Driver approaching your store")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Spacer()
                    }
                    .padding(AppTheme.spacingSM)
                    .background(AppTheme.warningSoft.opacity(0.5))
                    .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
                }

                Text("\(order.items.count) items · \(order.displayTotal) UZS")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                RetailerStatusBadge(
                    text: order.state,
                    tint: AppTheme.statusTint(for: order.state)
                )

                Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

                HStack(spacing: AppTheme.spacingMD) {
                    NavigationLink {
                        DeliveryMapView()
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "map").font(.system(size: 12, weight: .semibold))
                            Text("Track").font(.system(.caption, design: .rounded, weight: .semibold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                    }

                    Spacer()
                }

                Text("Completion is handled through delivery handoff and payment — retailer status patches are not available here.")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private var emptyState: some View {
        RetailerEmptyView(
            title: "No Active Arrivals",
            message: "Incoming deliveries will appear here from live tracking.",
            systemImage: "shippingbox"
        )
        .padding(AppTheme.spacingXL)
    }

    private func loadOrders() async {
        isLoading = true
        loadError = nil
        do {
            let response: TrackingResponse = try await api.get(path: "/v1/retailer/tracking")
            trackingOrders = response.orders.filter { arrivalStates.contains($0.state.uppercased()) }
        } catch {
            trackingOrders = []
            loadError = "Could not load live arrivals. Check your connection and retry."
        }
        isLoading = false
    }
}

#Preview {
    NavigationStack { ArrivalView() }
}
