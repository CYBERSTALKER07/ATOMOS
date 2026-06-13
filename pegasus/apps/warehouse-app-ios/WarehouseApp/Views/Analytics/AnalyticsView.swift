import SwiftUI

struct AnalyticsView: View {
    @State private var data = AnalyticsData.empty
    @State private var loading = true
    @State private var error: String?
    @State private var period = "7d"

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
                            AnalyticsKpiCard(title: "Anomaly Rows (30d)", value: "\(data.importAnomalyQueue.openRows30d)", icon: "exclamationmark.triangle", index: 5)
                        }

                        ImportFreshnessDetailCard(freshness: data.importFreshness)
                            .staggeredAppear(index: 6)

                        ImportAnomalyDetailCard(queue: data.importAnomalyQueue)
                            .staggeredAppear(index: 7)

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
                                    .staggeredAppear(index: index + 8)
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

private struct ImportFreshnessDetailCard: View {
    let freshness: ImportFreshness

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Import Freshness")
                .font(.headline)
            HStack {
                VStack(alignment: .leading) {
                    Text("SKUs Updated (30d)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(freshness.appliedSkus30d)")
                        .font(.title3.bold())
                }
                Spacer()
                VStack(alignment: .leading) {
                    Text("Quantity Delta (30d)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(freshness.quantityDelta30d)")
                        .font(.title3.bold())
                }
            }
            let session = freshness.lastSessionId.isEmpty ? "N/A" : freshness.lastSessionId
            let applied = freshness.lastAppliedAt.isEmpty ? "No imports applied yet" : freshness.lastAppliedAt
            Text("Last session: \(session) • \(applied)")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct ImportAnomalyDetailCard: View {
    let queue: ImportAnomalyQueue

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Import Anomaly Queue")
                .font(.headline)
            HStack {
                VStack(alignment: .leading) {
                    Text("Open Rows (30d)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(queue.openRows30d)")
                        .font(.title3.bold())
                }
                Spacer()
                VStack(alignment: .leading) {
                    Text("Affected Sessions (30d)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(queue.affectedSessions30d)")
                        .font(.title3.bold())
                }
            }
            let session = queue.lastSessionId.isEmpty ? "N/A" : queue.lastSessionId
            let detected = queue.lastDetectedAt.isEmpty ? "No anomalies detected" : queue.lastDetectedAt
            Text("Last session: \(session) • \(detected)")
                .font(.caption)
                .foregroundStyle(.secondary)
            if !queue.lastDetail.isEmpty {
                Text(queue.lastDetail)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
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
