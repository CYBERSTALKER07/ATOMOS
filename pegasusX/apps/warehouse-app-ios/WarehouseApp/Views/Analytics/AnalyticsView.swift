import SwiftUI

struct AnalyticsView: View {
    @State private var data = AnalyticsData.empty
    @State private var loading = true
    @State private var error: String?
    @State private var period = "30d"

    private let columns = [
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
    ]

    var body: some View {
        NavigationStack {
            ScrollView {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: 200)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else {
                    VStack(spacing: LabTheme.spacingLG) {
                        // Period picker
                        Picker("Period", selection: $period) {
                            Text("7 Days").tag("7d")
                            Text("30 Days").tag("30d")
                        }
                        .pickerStyle(.segmented)

                        // KPI grid
                        LazyVGrid(columns: columns, spacing: LabTheme.spacingMD) {
                            AnalyticsKpiCard(title: "Total Orders", value: "\(data.totalOrders)", icon: "cart", index: 0)
                            AnalyticsKpiCard(title: "Revenue", value: "\(data.totalRevenue.formatted()) UZS", icon: "banknote", index: 1)
                            AnalyticsKpiCard(title: "Avg Order", value: "\(Int(data.avgOrderValue.rounded()).formatted()) UZS", icon: "clock", index: 2)
                            AnalyticsKpiCard(title: "Fleet Utilization", value: "\(Int(data.fleetUtilizationPct.rounded()))%", icon: "checkmark.circle", index: 3)
                        }

                        // Import health
                        LazyVGrid(columns: columns, spacing: LabTheme.spacingMD) {
                            AnalyticsKpiCard(title: "Imported Rows (30d)", value: "\(data.importFreshness.appliedRows30d)", icon: "tray.and.arrow.down", index: 4)
                            AnalyticsKpiCard(title: "Imported SKUs (30d)", value: "\(data.importFreshness.appliedSkus30d)", icon: "cube.box", index: 5)
                            AnalyticsKpiCard(title: "Qty Delta (30d)", value: "\(data.importFreshness.quantityDelta30d)", icon: "plusminus", index: 6)
                            AnalyticsKpiCard(title: "Anomaly Rows (30d)", value: "\(data.importAnomalyQueue.openRows30d)", icon: "exclamationmark.triangle", index: 7)
                            AnalyticsKpiCard(title: "Anomaly Sessions", value: "\(data.importAnomalyQueue.affectedSessions30d)", icon: "tray.full", index: 8)
                        }

                        if !data.importFreshness.lastSessionId.isEmpty || !data.importFreshness.lastAppliedAt.isEmpty {
                            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                                Text("Last import")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !data.importFreshness.lastSessionId.isEmpty {
                                    Text("Session: \(data.importFreshness.lastSessionId)")
                                        .font(.footnote)
                                }
                                if !data.importFreshness.lastAppliedAt.isEmpty {
                                    Text(data.importFreshness.lastAppliedAt)
                                        .font(.footnote)
                                        .foregroundStyle(.secondary)
                                }
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .labCard()
                            .staggeredAppear(index: 9)
                        }

                        if !data.importAnomalyQueue.lastDetail.isEmpty || !data.importAnomalyQueue.lastSessionId.isEmpty {
                            VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                                Text("Latest anomaly")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if !data.importAnomalyQueue.lastSessionId.isEmpty {
                                    Text("Session: \(data.importAnomalyQueue.lastSessionId)")
                                        .font(.footnote)
                                }
                                if !data.importAnomalyQueue.lastDetectedAt.isEmpty {
                                    Text(data.importAnomalyQueue.lastDetectedAt)
                                        .font(.footnote)
                                        .foregroundStyle(.secondary)
                                }
                                if !data.importAnomalyQueue.lastDetail.isEmpty {
                                    Text(data.importAnomalyQueue.lastDetail)
                                        .font(.footnote)
                                }
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .labCard()
                            .staggeredAppear(index: 10)
                        }

                        if !data.chartDaily.isEmpty {
                            AnalyticsChartGrid(daily: data.chartDaily)
                                .staggeredAppear(index: 11)
                        }

                        // Top products
                        if !data.topProducts.isEmpty {
                            VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
                                Text("Top Products")
                                    .font(.title3.bold())
                                ForEach(Array(data.topProducts.enumerated()), id: \.element.id) { index, product in
                                    HStack {
                                        Text(product.productName)
                                            .font(.body)
                                        Spacer()
                                        Text("\(product.unitsSold) units · \(product.revenue.formatted()) UZS")
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    .labCard()
                                    .staggeredAppear(index: index + 7)
                                }
                            }
                        }
                    }
                    .padding()
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Analytics")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task { load() }
            .refreshable { load() }
            .onChange(of: period) { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                data = try await WarehouseService.analytics(period: period)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
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
        .staggeredAppear(index: index)
    }
}
