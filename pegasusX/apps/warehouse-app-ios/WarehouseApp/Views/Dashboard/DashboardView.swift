import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var vm = DashboardViewModel()

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 180 : 140
    }

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            Group {
                if vm.loading && !vm.hasData {
                    WarehouseLoadingView(
                        title: "Loading dashboard",
                        message: "Fetching orders, fleet status, and warehouse KPIs."
                    )
                } else if let error = vm.error, !vm.hasData {
                    WarehouseErrorView(message: error) {
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
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
                            NavigationLink {
                                DispatchView()
                            } label: {
                                KpiTile(
                                    title: "Pending Dispatch",
                                    value: "\(vm.stats.pendingDispatch)",
                                    systemImage: "paperplane",
                                    tint: LabTheme.warning,
                                    chip: vm.stats.pendingDispatch > 5 ? ("ALERT", LabTheme.destructive) : nil
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                OrdersView()
                            } label: {
                                KpiTile(
                                    title: "Active Orders",
                                    value: "\(vm.stats.activeOrders)",
                                    systemImage: "cart",
                                    tint: LabTheme.accentWarm
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                VehiclesView()
                            } label: {
                                KpiTile(
                                    title: "Vehicles",
                                    value: "\(vm.stats.totalVehicles)",
                                    systemImage: "truck.box",
                                    tint: LabTheme.accentWarm
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                InventoryView()
                            } label: {
                                KpiTile(
                                    title: "Low Stock",
                                    value: "\(vm.stats.lowStockCount)",
                                    systemImage: "exclamationmark.triangle",
                                    tint: LabTheme.warning,
                                    chip: vm.stats.lowStockCount > 0 ? ("ALERT", LabTheme.destructive) : nil
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                DriversView()
                            } label: {
                                KpiTile(
                                    title: "Drivers",
                                    value: "\(vm.stats.totalDrivers)",
                                    systemImage: "person.2",
                                    tint: LabTheme.accentWarm
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                OrdersView(initialState: "COMPLETED")
                            } label: {
                                KpiTile(
                                    title: "Completed",
                                    value: vm.stats.completedTodayAvailable ? "\(vm.stats.completedToday)" : "unavailable",
                                    systemImage: "checkmark.circle",
                                    tint: LabTheme.success
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                TreasuryView()
                            } label: {
                                KpiTile(
                                    title: "Revenue Today",
                                    value: vm.stats.todayRevenueAvailable ? "\(vm.stats.todayRevenue / 1000)K" : "unavailable",
                                    systemImage: "banknote",
                                    tint: LabTheme.accentWarm
                                )
                            }
                            .buttonStyle(.pressable)
                        }

                        StatusStackView(
                            dictionary: orderStatusFunnel,
                            counts: vm.stats.ordersByStatus,
                            source: "live",
                            onSelect: { commandJump($0) }
                        )
                        StatusStackView(
                            dictionary: truckDutyStatuses,
                            counts: vm.stats.truckDuty,
                            source: "live"
                        )
                        if !vm.stats.holdReasons.isEmpty {
                            VStack(alignment: .leading, spacing: 4) {
                                Text("Hold reasons")
                                    .font(.subheadline.bold())
                                ForEach(vm.stats.holdReasons, id: \.code) { row in
                                    Text("\(row.code) · \(row.count)")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        HStack {
                            SourceChip(source: vm.stats.demandSource)
                            Text(vm.stats.demandSource == "empty" ? "Planner empty" : "Demand \(vm.stats.demandSource)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .accessibilityElement(children: .combine)
                        .accessibilityIdentifier("gs-u-demand-source")
                        if !vm.stats.historyAvailable {
                            HStack {
                                SourceChip(source: "unavailable")
                                Text("History unavailable")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityElement(children: .combine)
                        }
                    }
                    .labReadableWidth()
                    .padding()
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.background)
        .navigationTitle("portal.nav.dashboard")
        .navigationDestination(item: $vm.commandJump) { jump in
            OrdersView(initialState: jump.status)
        }
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
                .accessibilityLabel("Refresh dashboard")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.warning()
                    tokenStore.clear()
                } label: {
                    Label("common.action.sign_out", systemImage: "rectangle.portrait.and.arrow.right")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Sign out")
            }
        }
        .task {
            await vm.load()
        }
        .refreshable {
            await vm.load(silent: false)
        }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
    }

    private func commandJump(_ status: String) {
        PegasusHaptics.selection()
        vm.commandJump = CommandStatusJump(status: status)
    }
}
