import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var stats = DashboardStats.empty
    @State private var loading = true
    @State private var error: String?

    private let columns = [
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
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
                    LazyVGrid(columns: columns, spacing: LabTheme.spacingMD) {
                        KpiCard(title: "Pending", value: "\(stats.pendingTransfers)", icon: "tray.full", index: 0)
                        KpiCard(title: "Loading", value: "\(stats.loadingTransfers)", icon: "shippingbox", index: 1)
                        KpiCard(title: "Manifests", value: "\(stats.activeManifests)", icon: "list.clipboard", index: 2)
                        KpiCard(title: "Dispatched", value: "\(stats.dispatchedToday)", icon: "checkmark.circle", index: 3)
                        KpiCard(title: "Vehicles", value: "\(stats.vehiclesTotal)", icon: "truck.box", index: 4)
                        KpiCard(title: "Available", value: "\(stats.vehiclesAvailable)", icon: "truck.box.badge.clock", index: 5)
                        KpiCard(title: "Staff", value: "\(stats.staffOnShift)", icon: "person.2", index: 6)
                        KpiCard(title: "Critical", value: "\(stats.criticalInsights)", icon: "exclamationmark.triangle", index: 7)
                    }
                    .padding()
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Dashboard")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        tokenStore.clear()
                    } label: {
                        Image(systemName: "rectangle.portrait.and.arrow.right")
                    }
                }
            }
            .task { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                stats = try await FactoryService.dashboard()
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

// MARK: - KPI Card
private struct KpiCard: View {
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
                .font(.title.bold())
            Text(title)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .staggeredAppear(index: index)
    }
}
