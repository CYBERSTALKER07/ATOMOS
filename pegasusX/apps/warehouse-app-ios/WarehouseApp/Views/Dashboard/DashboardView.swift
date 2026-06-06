import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var stats = DashboardData.empty
    @State private var loading = true
    @State private var error: String?

    private let columns = [
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
                        KpiCard(title: "Active Orders", value: "\(stats.activeOrders)", icon: "cart", index: 0)
                        KpiCard(title: "Completed", value: "\(stats.completedToday)", icon: "checkmark.circle", index: 1)
                        KpiCard(title: "Pending Dispatch", value: "\(stats.pendingDispatch)", icon: "paperplane", index: 2)
                        KpiCard(title: "Revenue Today", value: "\(stats.todayRevenue / 1000)K", icon: "banknote", index: 3)
                        KpiCard(title: "On Route", value: "\(stats.driversOnRoute)", icon: "location", index: 4)
                        KpiCard(title: "Idle Drivers", value: "\(stats.driversIdle)", icon: "person.badge.clock", index: 5)
                        KpiCard(title: "Vehicles", value: "\(stats.totalVehicles)", icon: "truck.box", index: 6)
                        KpiCard(title: "Low Stock", value: "\(stats.lowStockCount)", icon: "exclamationmark.triangle", index: 7)
                        KpiCard(title: "Staff", value: "\(stats.totalStaff)", icon: "person.2", index: 8)
                    }
                    .padding()
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Dashboard")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Sign Out", systemImage: "rectangle.portrait.and.arrow.right") {
                        tokenStore.clear()
                    }
                }
            }
            .task { load() }
            .refreshable { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                stats = try await WarehouseService.dashboard()
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
