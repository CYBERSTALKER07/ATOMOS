import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var dashboard: SupplierDashboard?
    @State private var meiSummary: SupplierMEIONetworkSummary?
    @State private var pulseEvents: [SupplierPulseEvent] = []
    @State private var pulseLoading = true
    @State private var demandConfidence: ForecastConfidence?
    @State private var demandGeneratedAt: String?
    @State private var loading = true
    @State private var error: String?

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 200 : 150
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                Group {
                    if loading {
                        SupplierLoadingView(
                            title: "Loading dashboard",
                            message: "Fetching pending orders, inventory, and billing status."
                        )
                    } else if let error {
                        SupplierErrorView(message: error) {
                            Task { await load() }
                        }
                    } else if let dashboard {
                        VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                            if !tokenStore.isConfigured {
                                billingBanner
                            }

                            SupplierSectionHeader(
                                title: "Operations at a glance",
                                subtitle: "Live supplier KPIs"
                            )

                            if let meiSummary {
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                    Text("MEIO network")
                                        .font(.headline)
                                    Text("\(meiSummary.warehousesScanned) warehouses · \(meiSummary.transferRecommendations) transfer recs · \(meiSummary.insightsGenerated) insights")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding()
                                .background(SupplierTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
                            }

                            if let demandConfidence {
                                ForecastConfidenceView(
                                    confidence: demandConfidence,
                                    updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: demandGeneratedAt),
                                    stale: ForecastConfidenceSupport.isForecastStale(generatedAt: demandGeneratedAt)
                                )
                            }

                            NetworkPulseStrip(events: pulseEvents, loading: pulseLoading)

                            LazyVGrid(
                                columns: [GridItem(.adaptive(minimum: gridMin), spacing: SupplierTheme.spacingMD)],
                                spacing: SupplierTheme.spacingMD
                            ) {
                                KpiTile(
                                    title: "Pending orders",
                                    value: "\(dashboard.pendingOrders)",
                                    systemImage: "shippingbox",
                                    tint: SupplierTheme.warning
                                )
                                KpiTile(
                                    title: "Inventory SKUs",
                                    value: "\(dashboard.inventorySKUs)",
                                    systemImage: "archivebox",
                                    tint: .accentColor
                                )
                                KpiTile(
                                    title: "Configured",
                                    value: dashboard.isConfigured ? "Yes" : "No",
                                    systemImage: "checkmark.seal",
                                    tint: dashboard.isConfigured ? SupplierTheme.success : SupplierTheme.destructive
                                )
                            }

                            Text("Updated \(dashboard.updatedAt)")
                                .font(.caption2)
                                .foregroundStyle(.tertiary)
                        }
                        .supplierReadableWidth()
                        .padding()
                    }
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Dashboard")
            .toolbar {
                signOutToolbar
                ToolbarItem(placement: .topBarTrailing) {
                    NavigationLink {
                        NotificationInboxView()
                    } label: {
                        Image(systemName: "bell")
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load(silent: true) }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .refreshable { await load(silent: true) }
            .task {
                await load()
                while !Task.isCancelled {
                    try? await Task.sleep(nanoseconds: 30_000_000_000)
                    await load(silent: true)
                }
            }
            .onChange(of: realtimeHub.refreshEpoch) { _, _ in
                Task { await load(silent: true) }
            }
            .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
                Task { await load(silent: true) }
            }
        }
    }

    private var billingBanner: some View {
        HStack(spacing: SupplierTheme.spacingMD) {
            Image(systemName: "creditcard")
            VStack(alignment: .leading, spacing: 4) {
                Text("Billing incomplete")
                    .font(.subheadline.bold())
                Text("Finish setup to enable treasury and payouts.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Button("Setup") {
                tokenStore.showBillingGate()
            }
            .font(.caption.bold())
        }
        .supplierCard()
    }

    @ToolbarContentBuilder
    private var signOutToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Button("Sign Out", systemImage: "rectangle.portrait.and.arrow.right") {
                tokenStore.clear()
            }
            .labelStyle(.iconOnly)
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            dashboard = try await SupplierService.dashboard()
            _ = try? await SupplierOperationsService.activity()
            _ = try? await SupplierOperationsService.exceptions()
            meiSummary = try? await SupplierOperationsService.meiNetworkSummary()
            if let demand = try? await SupplierOperationsService.demandToday() {
                demandGeneratedAt = demand.generatedAt
                demandConfidence = ForecastConfidenceSupport.fromDemand(demand)
            } else {
                demandGeneratedAt = nil
                demandConfidence = nil
            }
            pulseLoading = true
            pulseEvents = (try? await SupplierOperationsService.pulse())?.events ?? []
            pulseLoading = false
            if let configured = dashboard?.isConfigured {
                tokenStore.markConfigured(configured)
            }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}
