import SwiftUI

struct AnalyticsView: View {
    @State private var overview = FactoryAnalyticsOverview(
        dailyActivity: [],
        transfersTotal: 0,
        manifestsActive: 0,
        exceptionQueue: 0,
        avgLeadTimeMins: 0
    )
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        Text("Factory throughput, manifest pressure, and exception queue.")
                            .font(.caption)
                            .foregroundStyle(.secondary)

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: 150), spacing: LabTheme.spacingSM)],
                            spacing: LabTheme.spacingSM
                        ) {
                            AnalyticsKpiCard(title: "Transfers Total", value: "\(overview.transfersTotal)")
                            AnalyticsKpiCard(title: "Active Manifests", value: "\(overview.manifestsActive)")
                            AnalyticsKpiCard(
                                title: "Exception Queue",
                                value: "\(overview.exceptionQueue)",
                                alert: overview.exceptionQueue > 0
                            )
                            AnalyticsKpiCard(
                                title: "Avg Lead Time (min)",
                                value: String(format: "%.1f", overview.avgLeadTimeMins)
                            )
                        }

                        if !overview.dailyActivity.isEmpty {
                            Text("7-day transfer activity")
                                .font(.headline)
                            ForEach(overview.dailyActivity, id: \.date) { day in
                                HStack {
                                    Text(day.date)
                                    Spacer()
                                    Text("\(day.transfers) transfers")
                                        .font(.subheadline.monospacedDigit())
                                }
                                .labCard()
                            }
                        }
                    }
                    .padding()
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("Analytics Overview")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                overview = try await FactoryService.analyticsOverview()
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
    var alert: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            HStack {
                Text(title)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Spacer()
                if alert {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .font(.caption)
                        .foregroundStyle(LabTheme.destructive)
                }
            }
            Text(value)
                .font(.title2.bold().monospacedDigit())
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}
