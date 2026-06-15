import SwiftUI

struct AnalyticsView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var loading = true
    @State private var error: String?
    @State private var pendingOrders = 0
    @State private var inventorySKUs = 0
    @State private var revenueLabel = "—"
    @State private var predictionCount = 0
    @State private var forecastUnits = 0
    @State private var velocityCreated = 0

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 200 : 150
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                Group {
                    if loading {
                        SupplierLoadingView(
                            title: "Loading analytics",
                            message: "Fetching revenue, demand, and velocity metrics."
                        )
                    } else if let error {
                        SupplierErrorView(message: error) { Task { await load() } }
                    } else {
                        VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                            SupplierSectionHeader(
                                title: "Intelligence",
                                subtitle: "Revenue and demand signals"
                            )

                            LazyVGrid(
                                columns: [GridItem(.adaptive(minimum: gridMin), spacing: SupplierTheme.spacingMD)],
                                spacing: SupplierTheme.spacingMD
                            ) {
                                KpiTile(
                                    title: "30-day revenue",
                                    value: revenueLabel,
                                    systemImage: "banknote",
                                    tint: SupplierTheme.success
                                )
                                KpiTile(
                                    title: "Demand predictions",
                                    value: "\(predictionCount)",
                                    systemImage: "sparkles",
                                    tint: .accentColor
                                )
                                KpiTile(
                                    title: "Forecast units (24h)",
                                    value: "\(forecastUnits)",
                                    systemImage: "chart.line.uptrend.xyaxis",
                                    tint: SupplierTheme.warning
                                )
                                KpiTile(
                                    title: "Orders created (velocity)",
                                    value: "\(velocityCreated)",
                                    systemImage: "clock.arrow.circlepath",
                                    tint: SupplierTheme.secondaryLabel
                                )
                            }

                            SupplierSectionHeader(
                                title: "Operational snapshot",
                                subtitle: "Current queue and catalog depth"
                            )

                            LazyVGrid(
                                columns: [GridItem(.adaptive(minimum: gridMin), spacing: SupplierTheme.spacingMD)],
                                spacing: SupplierTheme.spacingMD
                            ) {
                                KpiTile(
                                    title: "Pending orders",
                                    value: "\(pendingOrders)",
                                    systemImage: "shippingbox",
                                    tint: SupplierTheme.warning
                                )
                                KpiTile(
                                    title: "Inventory SKUs",
                                    value: "\(inventorySKUs)",
                                    systemImage: "archivebox",
                                    tint: .accentColor
                                )
                            }
                        }
                        .supplierReadableWidth()
                        .padding()
                    }
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Analytics")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load(silent: true) }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await load() }
            .refreshable { await load(silent: true) }
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            async let dash = SupplierService.dashboard()
            async let revenue = SupplierOperationsService.analyticsRevenue()
            async let demand = SupplierOperationsService.demandToday()
            async let velocity = SupplierOperationsService.analyticsVelocity()

            let dashValue = try await dash
            let revenueValue = try await revenue
            let demandValue = try await demand
            let velocityValue = try await velocity

            pendingOrders = dashValue.pendingOrders
            inventorySKUs = dashValue.inventorySKUs
            revenueLabel = MoneyFormat.minor(revenueValue.totalMinor, currency: revenueValue.currency)
            predictionCount = demandValue.predictionCount
            forecastUnits = demandValue.totalPallets
            velocityCreated = velocityValue.points.reduce(0) { $0 + $1.ordersCreated }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}
