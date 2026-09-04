import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    let onOpenManifestExceptions: () -> Void
    let onOpenAnalytics: () -> Void
    let onOpenInsights: () -> Void
    var onOpenTransfers: (String) -> Void = { _ in }
    var onOpenLoadingBay: () -> Void = {}
    @State private var showManifests = false
    @State private var showCreateTransfer = false
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var stats = DashboardStats.empty
    @State private var loading = true
    @State private var error: String?
    @State private var clientPolicyMessage: String?
    @State private var clientPolicyForce = false
    @State private var pendingManifest: AutoUpdater.Manifest?
    @State private var showNotifications = false
    private let refreshNanos: UInt64 = 60_000_000_000

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 180 : 160
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                if loading {
                    FactoryLoadingState(
                        title: "Loading dashboard",
                        message: "Fetching live factory metrics for loading, fleet, and staffing."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        ClientPolicyBanner(
                            message: clientPolicyMessage,
                            force: clientPolicyForce,
                            onUpdate: clientPolicyMessage == nil ? nil : {
                                AutoUpdater.shared.promptInstall(manifest: pendingManifest, force: clientPolicyForce)
                            },
                            onDismiss: clientPolicyForce ? nil : { clientPolicyMessage = nil },
                        )
                        DashboardHeroCard(stats: stats)
                        WorkflowLaunchCard(
                            onOpenSupplyRequests: onOpenSupplyRequests,
                            onOpenPayloadOverride: onOpenPayloadOverride,
                            onOpenManifestExceptions: onOpenManifestExceptions,
                            onOpenManifests: { showManifests = true },
                            onOpenAnalytics: onOpenAnalytics,
                            onOpenCreateTransfer: { showCreateTransfer = true },
                            onOpenInsights: onOpenInsights
                        )
                        FactorySectionHeader(
                            title: "Operations at a glance",
                            subtitle: "Live factory KPIs across transfers, fleet, and staffing"
                        )
                        .padding(.horizontal)

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: gridMin), spacing: LabTheme.spacingMD)],
                            spacing: LabTheme.spacingMD
                        ) {
                            ForEach(Array(dashboardMetrics.enumerated()), id: \.element.title) { index, metric in
                                KpiTile(
                                    title: metric.title,
                                    value: metric.value,
                                    systemImage: metric.icon,
                                    tint: metric.tint,
                                    supporting: metric.supporting,
                                    chip: metric.chip,
                                    staggerIndex: index
                                )
                            }
                        }
                        .padding(.horizontal)

                        HStack {
                            SourceChip(source: stats.source)
                            Text(stats.source == "empty" ? "No factory rows yet" : "Dashboard \(stats.source)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .padding(.horizontal)
                        .accessibilityIdentifier("gs-u-factory-source")

                        FactorySectionHeader(
                            title: "Transfers",
                            subtitle: "Factory plane only"
                        )
                        .padding(.horizontal)
                        StatusStackView(
                            dictionary: factoryTransferStates,
                            counts: stats.transfersByState,
                            source: stats.source,
                            onSelect: { onOpenTransfers($0) }
                        )
                        .padding(.horizontal)

                        FactorySectionHeader(
                            title: "Factory trucks",
                            subtitle: "FactoryTruckManifests. Last-mile retailer IN_TRANSIT is not a factory truck."
                        )
                        .padding(.horizontal)
                        StatusStackView(
                            dictionary: manifestStates,
                            counts: stats.manifestsByState,
                            source: stats.source,
                            onSelect: { _ in onOpenLoadingBay() }
                        )
                        .padding(.horizontal)
                        .accessibilityIdentifier("gs-u-factory-trucks")

                        StatusStackView(
                            dictionary: factoryVehicleStates,
                            counts: stats.vehiclesByState,
                            source: stats.source
                        )
                        .padding(.horizontal)
                        StatusStackView(
                            dictionary: factoryDriverDuty,
                            counts: stats.driverDuty,
                            source: stats.source
                        )
                        .padding(.horizontal)
                    }
                    .labReadableWidth()
                    .padding(.vertical)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("factory_portal.setup.factory.text.factory")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.nav.notifications", systemImage: "bell") {
                        showNotifications = true
                    }
                    .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("common.action.sign_out", systemImage: "rectangle.portrait.and.arrow.right") {
                        tokenStore.clear()
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .task {
                await load()
                await loadClientPolicy()
                while !Task.isCancelled {
                    try? await Task.sleep(nanoseconds: refreshNanos)
                    await load(silent: true)
                }
            }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        if event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") || event.type.hasPrefix("FACTORY_SUPPLY_") { Task { await load(silent: true) } }
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
            .sheet(isPresented: $showManifests) {
                ManifestsView()
            }
            .sheet(isPresented: $showNotifications) {
                NotificationInboxView()
            }
            .sheet(isPresented: $showCreateTransfer) {
                CreateTransferView { _ in
                    showCreateTransfer = false
                }
            }
        }
    }

    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool
                let forceUpdate: Bool
                let updateDeferred: Bool?
                let minimumVersion: String
                let recommendedVersion: String?
                let updateURL: String?
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case updateDeferred = "update_deferred"
                    case minimumVersion = "minimum_version"
                    case recommendedVersion = "recommended_version"
                    case updateURL = "update_url"
                    case deferReason = "defer_reason"
                }
            }
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": EnterpriseUpdateConfig.policyRole,
                    "platform": "ios",
                    "version": version,
                    "channel": EnterpriseUpdateConfig.channel,
                ],
            )
            let state = await AutoUpdater.shared.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred ?? false,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL,
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true)
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil

        do {
            stats = try await FactoryService.dashboard()
        } catch {
            self.error = error.localizedDescription
        }

        if !silent {
            loading = false
        }
    }
}



private extension DashboardView {
    var dashboardMetrics: [DashboardMetric] {
        [
            DashboardMetric(title: "Pending transfers", value: "\(stats.pendingTransfers)", supporting: "Awaiting release to loading", icon: "tray.full", tint: LabTheme.warning),
            DashboardMetric(title: "Now loading", value: "\(stats.loadingTransfers)", supporting: "Transfers staged at the bay", icon: "shippingbox", tint: LabTheme.warning),
            DashboardMetric(title: "Active manifests", value: "\(stats.activeManifests)", supporting: "Live outbound manifest groups", icon: "list.clipboard", tint: .accentColor),
            DashboardMetric(
                title: "Dispatched today",
                value: "\(stats.dispatchedToday)",
                supporting: "Completed releases this shift",
                icon: "checkmark.circle",
                tint: LabTheme.success,
                chip: stats.dispatchedToday > 0 ? ("DONE", LabTheme.success) : nil
            ),
            DashboardMetric(title: "Vehicles total", value: "\(stats.vehiclesTotal)", supporting: "Fleet capacity on record", icon: "truck.box", tint: .accentColor),
            DashboardMetric(title: "Vehicles available", value: "\(stats.vehiclesAvailable)", supporting: "Ready for assignment", icon: "truck.box.badge.clock", tint: LabTheme.success),
            DashboardMetric(title: "Staff on shift", value: "\(stats.staffOnShift)", supporting: "Operators currently active", icon: "person.2", tint: .accentColor),
            DashboardMetric(
                title: "Gate exceptions",
                value: "\(stats.criticalInsights)",
                supporting: "Transfers removed during loading",
                icon: "exclamationmark.triangle",
                tint: LabTheme.destructive,
                chip: stats.criticalInsights > 0 ? ("ALERT", LabTheme.destructive) : nil
            )
        ]
    }
}
