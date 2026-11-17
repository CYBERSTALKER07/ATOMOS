import SwiftUI

struct AnalyticsView: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = AnalyticsViewModel()

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 200 : 150
    }

    var body: some View {
        ScrollView {
            Group {
                if viewModel.loading && !viewModel.hasSnapshot {
                    SupplierLoadingView(
                        title: "mobile_supplier.loading.analytics",
                        message: "mobile_supplier.loading.analytics_message"
                    )
                } else if let error = viewModel.error, !viewModel.hasSnapshot {
                    SupplierErrorView(message: error) { Task { await viewModel.load() } }
                } else {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                        SupplierSectionHeader(
                            title: "mobile_supplier.chrome.intelligence",
                            subtitle: "mobile_supplier.chrome.revenue_and_demand"
                        )

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: gridMin), spacing: SupplierTheme.spacingMD)],
                            spacing: SupplierTheme.spacingMD
                        ) {
                            KpiTile(
                                title: "mobile_supplier.chrome.revenue_30_day",
                                value: viewModel.revenueLabel,
                                systemImage: "banknote",
                                tint: SupplierTheme.success
                            )
                            KpiTile(
                                title: "mobile_supplier.chrome.demand_predictions",
                                value: "\(viewModel.predictionCount)",
                                systemImage: "sparkles",
                                tint: .accentColor
                            )
                            KpiTile(
                                title: "mobile_supplier.chrome.forecast_units_24h",
                                value: "\(viewModel.forecastUnits)",
                                systemImage: "chart.line.uptrend.xyaxis",
                                tint: SupplierTheme.warning
                            )
                            KpiTile(
                                title: "mobile_supplier.chrome.orders_created_velocity",
                                value: "\(viewModel.velocityCreated)",
                                systemImage: "clock.arrow.circlepath",
                                tint: SupplierTheme.secondaryLabel
                            )
                        }

                        SupplierSectionHeader(
                            title: "mobile_supplier.chrome.operational_snapshot",
                            subtitle: "mobile_supplier.chrome.current_queue_depth"
                        )

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: gridMin), spacing: SupplierTheme.spacingMD)],
                            spacing: SupplierTheme.spacingMD
                        ) {
                            KpiTile(
                                title: "mobile_supplier.chrome.pending_orders",
                                value: MissingNumber.label(viewModel.pendingOrders),
                                systemImage: "shippingbox",
                                tint: SupplierTheme.warning
                            )
                            KpiTile(
                                title: "mobile_supplier.chrome.inventory_skus",
                                value: MissingNumber.label(viewModel.inventorySKUs),
                                systemImage: "archivebox",
                                tint: .accentColor
                            )
                        }

                        PlanningBrainSection()

                        if let demandConfidence = viewModel.demandConfidence {
                            ForecastConfidenceView(
                                confidence: demandConfidence,
                                updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: viewModel.demandGeneratedAt),
                                stale: ForecastConfidenceSupport.isForecastStale(generatedAt: viewModel.demandGeneratedAt)
                            )
                        }

                        SupplierSectionHeader(
                            title: "mobile_supplier.chrome.planning_tools",
                            subtitle: "mobile_supplier.chrome.sandbox_graph_seasonal"
                        )

                        VStack(spacing: SupplierTheme.spacingSM) {
                            NavigationLink { PlanningBrainView() } label: {
                                Label("mobile_supplier.ui.planning_sandbox", systemImage: "brain.head.profile")
                            }
                            NavigationLink { KnowledgeGraphView() } label: {
                                Label("mobile_supplier.ui.knowledge_graph", systemImage: "point.3.connected.trianglepath.dotted")
                            }
                            NavigationLink { PlanningSettingsView() } label: {
                                Label("supplier_portal.settings.planning.text.planning_settings", systemImage: "calendar")
                            }
                            NavigationLink { ReturnPolicySettingsView() } label: {
                                Label("portal.nav.return_policy", systemImage: "arrow.uturn.backward.circle")
                            }
                        }
                    }
                    .supplierReadableWidth()
                    .padding()
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.analytics")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await viewModel.load(silent: true) }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("portal.page.orders.action.refresh"))
            }
        }
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
        .silentRealtimeRefresh(
            refreshEpoch: realtimeHub.refreshEpoch,
            reconnectEpoch: realtimeHub.reconnectEpoch
        ) { silent in
            Task { await viewModel.load(silent: silent) }
        }
    }
}

private struct PlanningBrainSection: View {
    @State private var viewModel = PlanningBrainSectionViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        VStack(alignment: .leading, spacing: SupplierTheme.spacingMD) {
            SupplierSectionHeader(
                title: "mobile_supplier.chrome.planning_sandbox",
                subtitle: "mobile_supplier.chrome.readonly_what_if"
            )

            if let error = viewModel.error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.destructive)
            }

            if viewModel.loading && viewModel.sandop == nil {
                ProgressView("mobile_supplier.loading.sandop")
            } else if let sandop = viewModel.sandop {
                LazyVGrid(
                    columns: [GridItem(.adaptive(minimum: 140), spacing: SupplierTheme.spacingMD)],
                    spacing: SupplierTheme.spacingMD
                ) {
                    KpiTile(title: "mobile_supplier.chrome.factory_cap_7d", value: "\(sandop.factoryCapacityUnits)", systemImage: "building.2", tint: .accentColor)
                    KpiTile(title: "mobile_supplier.chrome.wh_inbound", value: "\(sandop.warehouseInboundCapUnits)", systemImage: "arrow.down.to.line", tint: SupplierTheme.secondaryLabel)
                    KpiTile(title: "mobile_supplier.chrome.utilization", value: "\(Int(sandop.utilizationPct))%", systemImage: "gauge", tint: SupplierTheme.warning)
                    KpiTile(
                        title: "mobile_supplier.chrome.capacity",
                        value: sandop.capacityAlert ? L10n.format("mobile_supplier.chrome.breach") : L10n.format("mobile_supplier.chrome.ok"),
                        systemImage: sandop.capacityAlert ? "exclamationmark.triangle" : "checkmark.circle",
                        tint: sandop.capacityAlert ? SupplierTheme.destructive : SupplierTheme.success
                    )
                }
            }

            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                Text("supplier_portal.planning_brain_panel.text.scenario_run")
                    .font(.subheadline.bold())
                HStack {
                    Stepper(LocalizedStringKey(L10n.format("mobile_supplier.chrome.downtime_hours", "\(Int(viewModel.downtimeHours))")), value: $vm.downtimeHours, in: 0...168, step: 1)
                }
                HStack {
                    Stepper(LocalizedStringKey(L10n.format("mobile_supplier.chrome.demand_delta", "\(Int(viewModel.demandDeltaPct))")), value: $vm.demandDeltaPct, in: -50...200, step: 5)
                }
                Button(LocalizedStringKey(viewModel.running ? "mobile_supplier.chrome.running" : "mobile_supplier.chrome.run_scenario")) {
                    Task { await viewModel.runScenario() }
                }
                .buttonStyle(.borderedProminent)
                .frame(minHeight: 44)
                .disabled(viewModel.running)

                if let scenario = viewModel.scenario {
                    Text(L10n.format("mobile_supplier.ui.sla_risk_slariskpct_fleet_fleetvolumeorders_stockouts_count", "\(Int(scenario.slaRiskPct))", "\(scenario.fleetVolumeOrders)", "\(scenario.stockoutSkus.count)"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .padding()
            .background(SupplierTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
        }
        .task { await viewModel.loadSandop() }
    }
}
