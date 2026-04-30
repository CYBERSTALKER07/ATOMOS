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
                            AnalyticsKpiCard(title: "Avg Delivery", value: "\(data.avgDeliveryMinutes) min", icon: "clock", index: 2)
                            AnalyticsKpiCard(title: "Completion", value: "\(data.completionRate)%", icon: "checkmark.circle", index: 3)
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
                                    .staggeredAppear(index: index + 4)
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
