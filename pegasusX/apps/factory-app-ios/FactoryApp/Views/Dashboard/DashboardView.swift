import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    let onOpenManifestExceptions: () -> Void
    let onOpenAnalytics: () -> Void
    let onOpenInsights: () -> Void
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
    private let refreshNanos: UInt64 = 30_000_000_000

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
                    }
                    .labReadableWidth()
                    .padding(.vertical)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Factory")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Notifications", systemImage: "bell") {
                        showNotifications = true
                    }
                    .labelStyle(.iconOnly)
                }
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
                        guard let eventType = event.eventType else { return }
                        switch eventType {
                        case .supplyRequestUpdate, .transferUpdate, .manifestUpdate:
                            Task { await load(silent: true) }
                        case .outboxFailed:
                            break
                        }
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

private struct DashboardMetric {
    let title: String
    let value: String
    let supporting: String
    let icon: String
    let tint: Color
    var chip: (text: String, tint: Color)? = nil
}

private struct WorkflowLaunchCard: View {
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    let onOpenManifestExceptions: () -> Void
    let onOpenManifests: () -> Void
    let onOpenAnalytics: () -> Void
    let onOpenCreateTransfer: () -> Void
    let onOpenInsights: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            Label {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text("Operator workflows")
                        .font(.headline)
                    Text("Warehouse demand and live manifest overrides are available in native mobile flows.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            } icon: {
                Image(systemName: "iphone.gen3")
                    .font(.title3)
                    .foregroundStyle(.secondary)
            }

            WorkflowLaunchRow(
                title: "Supply requests",
                supporting: "Review warehouse demand and advance requests through production states.",
                actionLabel: "Open requests",
                onTap: onOpenSupplyRequests
            )
            WorkflowLaunchRow(
                title: "Payload override",
                supporting: "Move transfers between loading manifests or release them back to approved stock.",
                actionLabel: "Open override",
                onTap: onOpenPayloadOverride
            )
            WorkflowLaunchRow(
                title: "Manifest lifecycle",
                supporting: "Advance manifests through draft, loading, sealed, dispatched, and completed.",
                actionLabel: "Open manifests",
                onTap: onOpenManifests
            )
            WorkflowLaunchRow(
                title: "Gate exceptions",
                supporting: "Review transfers removed from manifests and DLQ escalations.",
                actionLabel: "Open exceptions",
                onTap: onOpenManifestExceptions
            )
            WorkflowLaunchRow(
                title: "Create transfer",
                supporting: "Stage a new factory-to-warehouse movement with volume and optional fleet assignment.",
                actionLabel: "Create transfer",
                onTap: onOpenCreateTransfer
            )
            WorkflowLaunchRow(
                title: "Replenishment insights",
                supporting: "Warehouse stock velocity and reorder pressure linked to this factory.",
                actionLabel: "Open insights",
                onTap: onOpenInsights
            )
            WorkflowLaunchRow(
                title: "Analytics overview",
                supporting: "Factory throughput, active manifests, exception queue, and lead time.",
                actionLabel: "Open analytics",
                onTap: onOpenAnalytics
            )
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
        .padding(.horizontal)
    }
}

private struct WorkflowLaunchRow: View {
    let title: String
    let supporting: String
    let actionLabel: String
    let onTap: () -> Void

    var body: some View {
        HStack(alignment: .center, spacing: LabTheme.spacingMD) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(title)
                    .font(.subheadline.bold())
                Text(supporting)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Button(actionLabel, action: onTap)
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
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
