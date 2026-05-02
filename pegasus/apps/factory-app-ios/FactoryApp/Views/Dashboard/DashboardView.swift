import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var stats = DashboardStats.empty
    @State private var loading = true
    @State private var error: String?

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
                        Button("Retry") {
                            Task { await load() }
                        }
                    }
                } else {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        DashboardHeroCard(stats: stats)
                        Text("Operations at a glance")
                            .font(.headline)
                            .padding(.horizontal)

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: 160), spacing: LabTheme.spacingMD)],
                            spacing: LabTheme.spacingMD
                        ) {
                            ForEach(Array(dashboardMetrics.enumerated()), id: \.element.title) { index, metric in
                                KpiCard(metric: metric, index: index)
                            }
                        }
                        .padding(.horizontal)
                    }
                    .padding(.vertical)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Factory")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Sign Out", systemImage: "rectangle.portrait.and.arrow.right") {
                        tokenStore.clear()
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task { await load() }
        }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil

        do {
            stats = try await FactoryService.dashboard()
        } catch {
            self.error = error.localizedDescription
        }

        loading = false
    }
}

private struct DashboardMetric {
    let title: String
    let value: String
    let supporting: String
    let icon: String
}

private struct KpiCard: View {
    let metric: DashboardMetric
    let index: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            Image(systemName: metric.icon)
                .font(.title3)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(metric.value)
                    .font(.title2.bold())
                Text(metric.title)
                    .font(.subheadline.bold())
                Text(metric.supporting)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .staggeredAppear(index: index)
    }
}

private struct DashboardHeroCard: View {
    let stats: DashboardStats

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text("Outbound floor status")
                    .font(.title2.bold())
                Text("\(stats.pendingTransfers + stats.loadingTransfers) transfers are active across release and bay lanes.")
                    .font(.body)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: LabTheme.spacingSM) {
                OverviewMetric(label: "Queued", value: "\(stats.pendingTransfers)")
                OverviewMetric(label: "Loading", value: "\(stats.loadingTransfers)")
                OverviewMetric(label: "Critical", value: "\(stats.criticalInsights)")
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .padding(.horizontal)
    }
}

private struct OverviewMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private extension DashboardView {
    var dashboardMetrics: [DashboardMetric] {
        [
            DashboardMetric(title: "Pending transfers", value: "\(stats.pendingTransfers)", supporting: "Awaiting release to loading", icon: "tray.full"),
            DashboardMetric(title: "Now loading", value: "\(stats.loadingTransfers)", supporting: "Transfers staged at the bay", icon: "shippingbox"),
            DashboardMetric(title: "Active manifests", value: "\(stats.activeManifests)", supporting: "Live outbound manifest groups", icon: "list.clipboard"),
            DashboardMetric(title: "Dispatched today", value: "\(stats.dispatchedToday)", supporting: "Completed releases this shift", icon: "checkmark.circle"),
            DashboardMetric(title: "Vehicles total", value: "\(stats.vehiclesTotal)", supporting: "Fleet capacity on record", icon: "truck.box"),
            DashboardMetric(title: "Vehicles available", value: "\(stats.vehiclesAvailable)", supporting: "Ready for assignment", icon: "truck.box.badge.clock"),
            DashboardMetric(title: "Staff on shift", value: "\(stats.staffOnShift)", supporting: "Operators currently active", icon: "person.2"),
            DashboardMetric(title: "Critical insights", value: "\(stats.criticalInsights)", supporting: "Restock and exception pressure", icon: "exclamationmark.triangle")
        ]
    }
}
