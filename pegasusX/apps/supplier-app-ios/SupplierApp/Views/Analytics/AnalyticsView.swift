import SwiftUI

struct AnalyticsView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var loading = true
    @State private var error: String?
    @State private var hasSnapshot = false
    @State private var pendingOrders = 0
    @State private var inventorySKUs = 0
    @State private var revenueLabel = "—"
    @State private var predictionCount = 0
    @State private var forecastUnits = 0
    @State private var velocityCreated = 0
    @State private var demandGeneratedAt: String?
    @State private var demandConfidence: ForecastConfidence?

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 200 : 150
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                Group {
                    if loading && !hasSnapshot {
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

                            PlanningBrainSection()

                            if let demandConfidence {
                                ForecastConfidenceView(
                                    confidence: demandConfidence,
                                    updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: demandGeneratedAt),
                                    stale: ForecastConfidenceSupport.isForecastStale(generatedAt: demandGeneratedAt)
                                )
                            }

                            SupplierSectionHeader(
                                title: "Planning tools",
                                subtitle: "Sandbox, graph, and seasonal settings"
                            )

                            VStack(spacing: SupplierTheme.spacingSM) {
                                NavigationLink { PlanningBrainView() } label: {
                                    Label("Planning sandbox", systemImage: "brain.head.profile")
                                }
                                NavigationLink { KnowledgeGraphView() } label: {
                                    Label("Knowledge graph", systemImage: "point.3.connected.trianglepath.dotted")
                                }
                                NavigationLink { PlanningSettingsView() } label: {
                                    Label("Planning settings", systemImage: "calendar")
                                }
                                NavigationLink { ReturnPolicySettingsView() } label: {
                                    Label("Return policy", systemImage: "arrow.uturn.backward.circle")
                                }
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
            .silentRealtimeRefresh(
                refreshEpoch: realtimeHub.refreshEpoch,
                reconnectEpoch: realtimeHub.reconnectEpoch
            ) { silent in
                Task { await load(silent: silent) }
            }
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
            demandGeneratedAt = demandValue.generatedAt
            demandConfidence = ForecastConfidenceSupport.fromDemand(demandValue)
            velocityCreated = velocityValue.points.reduce(0) { $0 + $1.ordersCreated }
            hasSnapshot = true
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}

private struct PlanningBrainSection: View {
    @State private var sandop: PlanningSAndOPSnapshot?
    @State private var scenario: PlanningScenarioResult?
    @State private var downtimeHours = 8.0
    @State private var demandDeltaPct = 10.0
    @State private var loading = true
    @State private var running = false
    @State private var error: String?

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
            SupplierSectionHeader(
                title: "Planning sandbox",
                subtitle: "Read-only what-if and lightweight S&OP"
            )

            if let error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.destructive)
            }

            if loading {
                ProgressView("Loading S&OP…")
            } else if let sandop {
                LazyVGrid(
                    columns: [GridItem(.adaptive(minimum: 140), spacing: SupplierTheme.spacingMD)],
                    spacing: SupplierTheme.spacingMD
                ) {
                    KpiTile(title: "Factory cap (7d)", value: "\(sandop.factoryCapacityUnits)", systemImage: "building.2", tint: .accentColor)
                    KpiTile(title: "WH inbound", value: "\(sandop.warehouseInboundCapUnits)", systemImage: "arrow.down.to.line", tint: SupplierTheme.secondaryLabel)
                    KpiTile(title: "Utilization", value: "\(Int(sandop.utilizationPct))%", systemImage: "gauge", tint: SupplierTheme.warning)
                    KpiTile(
                        title: "Capacity",
                        value: sandop.capacityAlert ? "Breach" : "OK",
                        systemImage: sandop.capacityAlert ? "exclamationmark.triangle" : "checkmark.circle",
                        tint: sandop.capacityAlert ? SupplierTheme.destructive : SupplierTheme.success
                    )
                }
            }

            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                Text("Scenario run")
                    .font(.subheadline.bold())
                HStack {
                    Stepper("Downtime \(Int(downtimeHours))h", value: $downtimeHours, in: 0...168, step: 1)
                }
                HStack {
                    Stepper("Demand \(Int(demandDeltaPct))%", value: $demandDeltaPct, in: -50...200, step: 5)
                }
                Button(running ? "Running…" : "Run scenario") {
                    Task { await runScenario() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(running)

                if let scenario {
                    Text("SLA risk \(Int(scenario.slaRiskPct))% · fleet \(scenario.fleetVolumeOrders) · stockouts \(scenario.stockoutSkus.count)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .padding()
            .background(SupplierTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG))
        }
        .task { await loadSandop() }
    }

    @MainActor
    private func loadSandop() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            sandop = try await SupplierOperationsService.planningSAndOP()
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func runScenario() async {
        running = true
        error = nil
        defer { running = false }
        do {
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            let key = SupplierIdempotencyKeys.planningScenario(
                scopeId: scope,
                factoryDowntimeHours: Int(downtimeHours),
                demandDeltaPct: demandDeltaPct
            )
            scenario = try await SupplierOperationsService.runPlanningScenario(
                factoryDowntimeHours: Int(downtimeHours),
                demandDeltaPct: demandDeltaPct,
                idempotencyKey: key
            )
        } catch {
            self.error = error.localizedDescription
        }
    }
}
