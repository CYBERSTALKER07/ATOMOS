import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var stats = DashboardData.empty
    @State private var loading = true
    @State private var error: String?
    @State private var clientPolicyMessage: String?

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 180 : 140
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                Group {
                    if loading {
                        WarehouseLoadingView(
                            title: "Loading dashboard",
                            message: "Fetching orders, fleet status, and warehouse KPIs."
                        )
                    } else if let error {
                        WarehouseErrorView(message: error) { load() }
                    } else {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXL) {
                            ClientPolicyBanner(message: clientPolicyMessage)

                            FleetLiveMapSection(mapHeight: 300, showsExpand: false)

                            WarehouseSectionHeader(
                                title: "Operations at a glance",
                                subtitle: "Live warehouse KPIs"
                            )

                            LazyVGrid(
                                columns: [GridItem(.adaptive(minimum: gridMin), spacing: LabTheme.spacingMD)],
                                spacing: LabTheme.spacingMD
                            ) {
                                KpiTile(title: "Active Orders", value: "\(stats.activeOrders)", systemImage: "cart", tint: .accentColor)
                                KpiTile(
                                    title: "Completed",
                                    value: "\(stats.completedToday)",
                                    systemImage: "checkmark.circle",
                                    tint: LabTheme.success,
                                    chip: stats.completedToday > 0 ? ("DONE", LabTheme.success) : nil
                                )
                                KpiTile(
                                    title: "Pending Dispatch",
                                    value: "\(stats.pendingDispatch)",
                                    systemImage: "paperplane",
                                    tint: LabTheme.warning,
                                    chip: stats.pendingDispatch > 5 ? ("ALERT", LabTheme.destructive) : nil
                                )
                                KpiTile(title: "Revenue Today", value: "\(stats.todayRevenue / 1000)K", systemImage: "banknote", tint: .accentColor)
                                KpiTile(title: "On Route", value: "\(stats.driversOnRoute)", systemImage: "location", tint: LabTheme.live)
                                KpiTile(title: "Idle Drivers", value: "\(stats.driversIdle)", systemImage: "person.badge.clock", tint: LabTheme.secondaryLabel)
                                KpiTile(title: "Vehicles", value: "\(stats.totalVehicles)", systemImage: "truck.box", tint: .accentColor)
                                KpiTile(
                                    title: "Low Stock",
                                    value: "\(stats.lowStockCount)",
                                    systemImage: "exclamationmark.triangle",
                                    tint: LabTheme.warning,
                                    chip: stats.lowStockCount > 0 ? ("ALERT", LabTheme.destructive) : nil
                                )
                                KpiTile(title: "Staff", value: "\(stats.totalStaff)", systemImage: "person.2", tint: .accentColor)
                            }

                            if !stats.fleetStatus.isEmpty {
                                FleetStatusBreakdown(entries: stats.fleetStatus)
                            }
                        }
                        .labReadableWidth()
                        .padding()
                    }
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
            .task {
                load()
                await loadClientPolicy()
            }
            .refreshable {
                load()
                await loadClientPolicy()
            }
        }
    }

    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool
                let forceUpdate: Bool
                let minimumVersion: String
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case minimumVersion = "minimum_version"
                    case deferReason = "defer_reason"
                }
            }
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": "WAREHOUSE",
                    "platform": "ios",
                    "version": version,
                    "channel": "production",
                ],
            )
            if policy.outdated || policy.forceUpdate {
                var message = policy.forceUpdate ? "Update required" : "Update available"
                if !policy.minimumVersion.isEmpty {
                    message += " — minimum version \(policy.minimumVersion)"
                }
                if let deferReason = policy.deferReason, !deferReason.isEmpty {
                    message += ". \(deferReason)"
                }
                clientPolicyMessage = message
            } else {
                clientPolicyMessage = nil
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
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

private struct FleetStatusBreakdown: View {
    let entries: [FleetStatusEntry]

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            WarehouseSectionHeader(
                title: "Fleet status tracking",
                subtitle: "Manifest and driver states"
            )
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(spacing: LabTheme.spacingSM) {
                    ForEach(entries, id: \.self) { entry in
                        WarehouseStatusBadge(text: "\(entry.status): \(entry.count)")
                    }
                }
            }
        }
        .labCard()
    }
}
