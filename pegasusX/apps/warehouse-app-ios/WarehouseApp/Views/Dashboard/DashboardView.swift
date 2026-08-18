import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var stats = DashboardData.empty
    @State private var loading = true
    @State private var hasData = false
    @State private var error: String?
    @State private var commandJump: CommandStatusJump?

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 180 : 140
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                Group {
                    if loading && !hasData {
                        WarehouseLoadingView(
                            title: "Loading dashboard",
                            message: "Fetching orders, fleet status, and warehouse KPIs."
                        )
                    } else if let error, !hasData {
                        WarehouseErrorView(message: error) { load() }
                    } else {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXL) {
                            FleetLiveMapSection(mapHeight: 300, showsExpand: false)

                            WarehouseSectionHeader(
                                title: "Operations at a glance",
                                subtitle: "Live warehouse KPIs"
                            )

                            LazyVGrid(
                                columns: [GridItem(.adaptive(minimum: gridMin), spacing: LabTheme.spacingMD)],
                                spacing: LabTheme.spacingMD
                            ) {
                                KpiTile(
                                    title: "Pending Dispatch",
                                    value: "\(stats.pendingDispatch)",
                                    systemImage: "paperplane",
                                    tint: LabTheme.warning,
                                    chip: stats.pendingDispatch > 5 ? ("ALERT", LabTheme.destructive) : nil
                                )
                                KpiTile(title: "Active Orders", value: "\(stats.activeOrders)", systemImage: "cart", tint: .accentColor)
                                KpiTile(title: "Vehicles", value: "\(stats.totalVehicles)", systemImage: "truck.box", tint: .accentColor)
                                KpiTile(
                                    title: "Low Stock",
                                    value: "\(stats.lowStockCount)",
                                    systemImage: "exclamationmark.triangle",
                                    tint: LabTheme.warning,
                                    chip: stats.lowStockCount > 0 ? ("ALERT", LabTheme.destructive) : nil
                                )
                                KpiTile(title: "Drivers", value: "\(stats.totalDrivers)", systemImage: "person.2", tint: .accentColor)
                                KpiTile(
                                    title: "Completed",
                                    value: stats.completedTodayAvailable ? "\(stats.completedToday)" : "unavailable",
                                    systemImage: "checkmark.circle",
                                    tint: LabTheme.success
                                )
                                KpiTile(title: "Revenue Today", value: stats.todayRevenueAvailable ? "\(stats.todayRevenue / 1000)K" : "unavailable", systemImage: "banknote", tint: .accentColor)
                            }

                            StatusStackView(
                                dictionary: orderStatusFunnel,
                                counts: stats.ordersByStatus,
                                source: "live",
                                onSelect: { commandJump = CommandStatusJump(status: $0) }
                            )
                            StatusStackView(
                                dictionary: truckDutyStatuses,
                                counts: stats.truckDuty,
                                source: "live"
                            )
                            if !stats.holdReasons.isEmpty {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("Hold reasons")
                                        .font(.subheadline.bold())
                                    ForEach(stats.holdReasons, id: \.code) { row in
                                        Text("\(row.code) · \(row.count)")
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                            HStack {
                                SourceChip(source: stats.demandSource)
                                Text(stats.demandSource == "empty" ? "Planner empty" : "Demand \(stats.demandSource)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityIdentifier("gs-u-demand-source")
                            if !stats.historyAvailable {
                                HStack {
                                    SourceChip(source: "unavailable")
                                    Text("History unavailable")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        .labReadableWidth()
                        .padding()
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.dashboard")
            .navigationDestination(item: $commandJump) { jump in
                OrdersView(initialState: jump.status)
            }
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("common.action.sign_out", systemImage: "rectangle.portrait.and.arrow.right") {
                        tokenStore.clear()
                    }
                }
            }
            .task {
                load()
            }
            .refreshable {
                load(silent: false)
            }
            .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
                load(silent: silent)
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                stats = try await WarehouseService.dashboard()
                hasData = true
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }
}
