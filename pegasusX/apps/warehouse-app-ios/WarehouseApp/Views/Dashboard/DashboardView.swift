import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var stats = DashboardData.empty
    @State private var loading = true
    @State private var error: String?
    @State private var clientPolicyMessage: String?

    private let columns = [
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
        GridItem(.flexible(), spacing: LabTheme.spacingMD),
    ]

    var body: some View {
        NavigationStack {
            ScrollView {
                ClientPolicyBanner(message: clientPolicyMessage)
                    .padding(.top, LabTheme.spacingSM)
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
                    FleetLiveMapSection(mapHeight: 300, showsExpand: false)
                        .padding(.horizontal)
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
