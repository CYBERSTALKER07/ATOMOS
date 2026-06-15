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
                ProgressView().padding(.top, 80)
            } else if let loadError, trackingOrders.isEmpty {
                Text(loadError)
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                    .padding(AppTheme.spacingXL)
            } else if trackingOrders.isEmpty {
                emptyState
            } else {
                LazyVStack(spacing: AppTheme.spacingLG) {
                    ForEach(Array(trackingOrders.enumerated()), id: \.element.id) { index, order in
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
                        Text(order.state.replacingOccurrences(of: "_", with: " "))
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    NavigationLink {
                        DeliveryMapView()
                    } label: {
                        Label("Track", systemImage: "map")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                }

                Text("\(order.items.count) items · \(order.totalAmount) UZS")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                Text("Completion is handled through delivery handoff and payment — retailer status patches are not available here.")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(AppTheme.spacingLG)
        }
    }

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
            Text("Incoming deliveries will appear here from live tracking.")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
        }
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
