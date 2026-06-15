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
                FactoryLoadingView(
                    title: "Loading analytics",
                    message: "Fetching factory throughput, manifest pressure, and exception queue."
                )
            } else if let error {
                FactoryErrorView(message: error, retry: load)
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        FactorySectionHeader(
                            title: "Analytics overview",
                            subtitle: "Factory throughput, manifest pressure, and exception queue"
                        )

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: 150), spacing: LabTheme.spacingSM)],
                            spacing: LabTheme.spacingSM
                        ) {
                            KpiTile(
                                title: "Transfers Total",
                                value: "\(overview.transfersTotal)",
                                systemImage: "arrow.left.arrow.right",
                                tint: .accentColor
                            )
                            KpiTile(
                                title: "Active Manifests",
                                value: "\(overview.manifestsActive)",
                                systemImage: "list.clipboard",
                                tint: LabTheme.warning
                            )
                            KpiTile(
                                title: "Exception Queue",
                                value: "\(overview.exceptionQueue)",
                                systemImage: "exclamationmark.triangle",
                                tint: LabTheme.destructive,
                                chip: overview.exceptionQueue > 0 ? ("ALERT", LabTheme.destructive) : nil
                            )
                            KpiTile(
                                title: "Avg Lead Time (min)",
                                value: String(format: "%.1f", overview.avgLeadTimeMins),
                                systemImage: "clock",
                                tint: LabTheme.secondaryLabel
                            )
                        }

                        if !overview.dailyActivity.isEmpty {
                            FactorySectionHeader(
                                title: "7-day transfer activity",
                                subtitle: "Daily transfer volume"
                            )
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
                    .labReadableWidth()
                    .padding()
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("Analytics Overview")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise", action: load)
                    .labelStyle(.iconOnly)
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
