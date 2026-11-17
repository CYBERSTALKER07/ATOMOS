import SwiftUI

struct DemandForecastView: View {
    @State private var vm = DemandForecastViewModel()

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            Picker("View", selection: $vm.selectedSegment) {
                Text("portal.nav.products").tag(0)
                Text("mobile_warehouse.ui.series").tag(1)
            }
            .pickerStyle(.segmented)
            .padding()
            .onChange(of: vm.selectedSegment) { _, _ in
                PegasusHaptics.selection()
            }

            forecastBody
        }
        .navigationTitle("portal.nav.demand_forecast")
        .toolbar { forecastToolbar }
        .onChange(of: vm.horizon) { _, _ in
            PegasusHaptics.selection()
            Task { await vm.load() }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
    }

    @ToolbarContentBuilder
    private var forecastToolbar: some ToolbarContent {
        @Bindable var vm = vm
        ToolbarItem(placement: .topBarTrailing) {
            Picker("Days", selection: $vm.horizon) {
                Text("mobile_warehouse.ui.7d").tag(7)
                Text("mobile_warehouse.ui.14d").tag(14)
                Text("mobile_warehouse.ui.30d").tag(30)
            }
            .pickerStyle(.menu)
            .frame(minHeight: 44)
        }
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                PegasusHaptics.light()
                Task { await vm.load() }
            } label: {
                Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
            }
            .accessibilityLabel("Refresh demand forecast")
        }
    }

    @ViewBuilder
    private var forecastBody: some View {
        if vm.loading {
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if let error = vm.error {
            ContentUnavailableView {
                Label("Forecast unavailable", systemImage: "exclamationmark.triangle")
            } description: {
                Text(error)
            } actions: {
                Button("common.action.retry") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
            }
        } else if vm.selectedSegment == 0 {
            productsSegment
        } else {
            seriesSegment
        }
    }

    @ViewBuilder
    private var productsSegment: some View {
        if vm.forecast.products.isEmpty {
            ContentUnavailableView(
                "No product recommendations",
                systemImage: "cube.box",
                description: Text("mobile_warehouse.ui.try_another_horizon_or_switch_to_series_for_daily_projection")
            )
        } else {
            ScrollView {
                VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                    Text("mobile_warehouse.ui.ai_powered_stock_recommendations_from_4_data_sources")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    ForecastChartPanel(
                        criticalCount: vm.criticalCount,
                        urgentCount: vm.urgentCount,
                        normalCount: vm.normalCount
                    )

                    ForecastSkuTable(
                        products: vm.forecast.products,
                        generatedAt: vm.forecast.generatedAt,
                        forecastDays: vm.forecast.forecastDays
                    )
                }
                .padding()
            }
            .scrollDismissesKeyboard(.interactively)
        }
    }

    @ViewBuilder
    private var seriesSegment: some View {
        if vm.forecast.series.isEmpty {
            ContentUnavailableView(
                "No series data",
                systemImage: "chart.line.uptrend.xyaxis",
                description: Text("mobile_warehouse.ui.no_daily_demand_projection_for_this_window")
            )
        } else {
            ResponsiveGridContentWrapper {
                ForEach(vm.forecast.series) { day in
                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                        Text(day.date)
                            .font(.headline)
                        Text(L10n.format("mobile_warehouse.ui.projected_units_projectedunits", "\(day.projectedUnits)"))
                            .font(.subheadline)
                        Text(L10n.format("mobile_warehouse.ui.committed_committedunits_pending_pendingconfirmationunits", "\(day.committedUnits)", "\(day.pendingConfirmationUnits)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.vertical, LabTheme.spacingXS)
                    .frame(minHeight: 44)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Date \(day.date), projected \(day.projectedUnits) units, committed \(day.committedUnits) units, pending \(day.pendingConfirmationUnits) units")
                }
            }
            .scrollDismissesKeyboard(.interactively)
        }
    }
}
