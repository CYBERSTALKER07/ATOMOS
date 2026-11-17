import SwiftUI

struct AnalyticsView: View {
    @State private var vm = AnalyticsViewModel()

    private let columns = [
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
    ]

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, minHeight: 200)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
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
            } else {
                VStack(spacing: LabTheme.spacingLG) {
                    Picker("Period", selection: $vm.period) {
                        Text("mobile_warehouse.ui.7_days").tag("7d")
                        Text("mobile_warehouse.ui.30_days").tag("30d")
                    }
                    .pickerStyle(.segmented)

                    LazyVGrid(columns: columns, spacing: LabTheme.spacingMD) {
                        AnalyticsKpiCard(title: "Total Orders", value: "\(vm.data.totalOrders)", icon: "cart", index: 0)
                        AnalyticsKpiCard(title: "Revenue", value: "\(vm.data.totalRevenue.formatted()) \(packCurrency(MarketPackStore.pack))", icon: "banknote", index: 1)
                        AnalyticsKpiCard(title: "Avg Order", value: "\(Int(vm.data.avgOrderValue.rounded()).formatted()) \(packCurrency(MarketPackStore.pack))", icon: "clock", index: 2)
                        AnalyticsKpiCard(title: "Fleet Utilization", value: "\(Int(vm.data.fleetUtilizationPct.rounded()))%", icon: "checkmark.circle", index: 3)
                    }

                    LazyVGrid(columns: columns, spacing: LabTheme.spacingMD) {
                        AnalyticsKpiCard(title: "Imported Rows (30d)", value: "\(vm.data.importFreshness.appliedRows30d)", icon: "tray.and.arrow.down", index: 4)
                        AnalyticsKpiCard(title: "Imported SKUs (30d)", value: "\(vm.data.importFreshness.appliedSkus30d)", icon: "cube.box", index: 5)
                        AnalyticsKpiCard(title: "Qty Delta (30d)", value: "\(vm.data.importFreshness.quantityDelta30d)", icon: "plusminus", index: 6)
                        AnalyticsKpiCard(title: "Anomaly Rows (30d)", value: "\(vm.data.importAnomalyQueue.openRows30d)", icon: "exclamationmark.triangle", index: 7)
                        AnalyticsKpiCard(title: "Anomaly Sessions", value: "\(vm.data.importAnomalyQueue.affectedSessions30d)", icon: "tray.full", index: 8)
                    }

                    if !vm.data.importFreshness.lastSessionId.isEmpty || !vm.data.importFreshness.lastAppliedAt.isEmpty {
                        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                            Text("mobile_warehouse.ui.last_import")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if !vm.data.importFreshness.lastSessionId.isEmpty {
                                Text(L10n.format("mobile_warehouse.ui.session_lastsessionid", "\(vm.data.importFreshness.lastSessionId)"))
                                    .font(.footnote)
                            }
                            if !vm.data.importFreshness.lastAppliedAt.isEmpty {
                                Text(vm.data.importFreshness.lastAppliedAt)
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .labCard()
                        .accessibilityElement(children: .combine)
                        .staggeredAppear(index: 9)
                    }

                    if !vm.data.importAnomalyQueue.lastDetail.isEmpty || !vm.data.importAnomalyQueue.lastSessionId.isEmpty {
                        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                            Text("mobile_warehouse.ui.latest_anomaly")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if !vm.data.importAnomalyQueue.lastSessionId.isEmpty {
                                Text(L10n.format("mobile_warehouse.ui.session_lastsessionid", "\(vm.data.importAnomalyQueue.lastSessionId)"))
                                    .font(.footnote)
                            }
                            if !vm.data.importAnomalyQueue.lastDetectedAt.isEmpty {
                                Text(vm.data.importAnomalyQueue.lastDetectedAt)
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                            }
                            if !vm.data.importAnomalyQueue.lastDetail.isEmpty {
                                Text(vm.data.importAnomalyQueue.lastDetail)
                                    .font(.footnote)
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .labCard()
                        .accessibilityElement(children: .combine)
                        .staggeredAppear(index: 10)
                    }

                    if !vm.data.chartDaily.isEmpty {
                        AnalyticsChartGrid(daily: vm.data.chartDaily)
                            .staggeredAppear(index: 11)
                    }

                    if !vm.data.topProducts.isEmpty {
                        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
                            Text("warehouse_portal.analytics.text.top_products")
                                .font(.title3.bold())
                            ForEach(Array(vm.data.topProducts.enumerated()), id: \.element.id) { index, product in
                                HStack {
                                    Text(product.productName)
                                        .font(.body)
                                    Spacer()
                                    Text(L10n.format("mobile_warehouse.ui.unitssold_units_formatted_uzs", "\(product.unitsSold)", "\(product.revenue.formatted())"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .frame(minHeight: 44)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(product.productName), \(product.unitsSold) units, \(product.revenue.formatted()) UZS")
                                .labCard()
                                .staggeredAppear(index: index + 7)
                            }
                        }
                    }
                }
                .padding()
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.background)
        .navigationTitle("portal.nav.analytics")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh analytics")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load() }
        .onChange(of: vm.period) { _, _ in
            PegasusHaptics.selection()
            Task { await vm.load() }
        }
    }
}

private struct AnalyticsKpiCard: View {
    let title: String
    let value: String
    let icon: String
    let index: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Image(systemName: icon)
                .font(.title3)
                .foregroundStyle(.secondary)
            Spacer(minLength: 0)
            Text(value)
                .font(.title2.bold())
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title), \(value)")
        .staggeredAppear(index: index)
    }
}
