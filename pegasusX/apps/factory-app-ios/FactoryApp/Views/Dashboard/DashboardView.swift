import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    @State private var showManifestExceptions = false
    @State private var showManifests = false
    @State private var showAnalytics = false
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var stats = DashboardStats.empty
    @State private var loading = true
    @State private var error: String?
    @State private var clientPolicyMessage: String?
    @State private var showNotifications = false
    private let refreshNanos: UInt64 = 30_000_000_000

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
                        ClientPolicyBanner(message: clientPolicyMessage)
                        DashboardHeroCard(stats: stats)
                        WorkflowLaunchCard(
                            onOpenSupplyRequests: onOpenSupplyRequests,
                            onOpenPayloadOverride: onOpenPayloadOverride,
                            onOpenManifestExceptions: { showManifestExceptions = true },
                            onOpenManifests: { showManifests = true },
                            onOpenAnalytics: { showAnalytics = true }
                        )
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
            .sheet(isPresented: $showManifestExceptions) {
                NavigationStack {
                    ManifestExceptionsView()
                }
            }
            .sheet(isPresented: $showManifests) {
                ManifestsView()
            }
            .sheet(isPresented: $showAnalytics) {
                NavigationStack {
                    AnalyticsView()
                }
            }
            .sheet(isPresented: $showNotifications) {
                NotificationInboxView()
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
                    "role": "FACTORY",
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
}

private struct WorkflowLaunchCard: View {
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    let onOpenManifestExceptions: () -> Void
    let onOpenManifests: () -> Void
    let onOpenAnalytics: () -> Void

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
            DashboardMetric(title: "Pending transfers", value: "\(stats.pendingTransfers)", supporting: "Awaiting release to loading", icon: "tray.full"),
            DashboardMetric(title: "Now loading", value: "\(stats.loadingTransfers)", supporting: "Transfers staged at the bay", icon: "shippingbox"),
            DashboardMetric(title: "Active manifests", value: "\(stats.activeManifests)", supporting: "Live outbound manifest groups", icon: "list.clipboard"),
            DashboardMetric(title: "Dispatched today", value: "\(stats.dispatchedToday)", supporting: "Completed releases this shift", icon: "checkmark.circle"),
            DashboardMetric(title: "Vehicles total", value: "\(stats.vehiclesTotal)", supporting: "Fleet capacity on record", icon: "truck.box"),
            DashboardMetric(title: "Vehicles available", value: "\(stats.vehiclesAvailable)", supporting: "Ready for assignment", icon: "truck.box.badge.clock"),
            DashboardMetric(title: "Staff on shift", value: "\(stats.staffOnShift)", supporting: "Operators currently active", icon: "person.2"),
            DashboardMetric(title: "Gate exceptions", value: "\(stats.criticalInsights)", supporting: "Transfers removed during loading", icon: "exclamationmark.triangle")
        ]
    }
}
