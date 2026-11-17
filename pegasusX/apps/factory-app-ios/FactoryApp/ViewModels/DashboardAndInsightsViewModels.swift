import Foundation
import Observation
import SwiftUI

struct FactoryNotificationsPage: Decodable {
    let notifications: [FactoryNotificationItem]
    let unreadCount: Int
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case notifications
        case unreadCount = "unread_count"
        case hasMore = "has_more"
    }
}

struct FactoryNotificationItem: Decodable, Identifiable {
    let id: String
    let type: String
    let title: String
    let body: String
    let payload: String
    let channel: String
    let readAt: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, type, title, body, payload, channel
        case readAt = "read_at"
        case createdAt = "created_at"
    }

    init(
        id: String,
        type: String = "",
        title: String,
        body: String = "",
        payload: String = "",
        channel: String = "in_app",
        readAt: String? = nil,
        createdAt: String = ""
    ) {
        self.id = id
        self.type = type
        self.title = title
        self.body = body
        self.payload = payload
        self.channel = channel
        self.readAt = readAt
        self.createdAt = createdAt
    }
}

private struct MarkNotificationsReadRequest: Encodable {
    let markAll: Bool

    enum CodingKeys: String, CodingKey {
        case markAll = "mark_all"
    }
}

@Observable
@MainActor
final class DashboardViewModel {
    var stats: DashboardStats = .empty
    var loading: Bool = true
    var error: String?
    var clientPolicyMessage: String?
    var clientPolicyForce: Bool = false
    var pendingManifest: AutoUpdater.Manifest?
    var showManifests: Bool = false
    var showNotifications: Bool = false
    var showCreateTransfer: Bool = false

    private let realtimeClient = FactoryRealtimeClient()
    private let refreshNanos: UInt64 = 60_000_000_000
    private let fetchDashboard: @MainActor () async throws -> DashboardStats

    init(
        fetchDashboard: @escaping @MainActor () async throws -> DashboardStats = { try await FactoryService.dashboard() }
    ) {
        self.fetchDashboard = fetchDashboard
    }

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
            ),
        ]
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil

        do {
            stats = try await fetchDashboard()
        } catch {
            stats = .empty
            self.error = error.localizedDescription
        }

        if !silent {
            loading = false
        }
    }

    func runLifecycleLoop() async {
        startRealtime()
        await load()
        await loadClientPolicy()
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: refreshNanos)
            guard !Task.isCancelled else { break }
            await load(silent: true)
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                if event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_")
                    || event.type.hasPrefix("FACTORY_SUPPLY_") {
                    Task { await self.load(silent: true) }
                }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }

    func dismissPolicyBanner() {
        clientPolicyMessage = nil
    }

    func promptInstallUpdate() {
        AutoUpdater.shared.promptInstall(manifest: pendingManifest, force: clientPolicyForce)
    }

    func loadClientPolicy() async {
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
                ]
            )
            let state = await AutoUpdater.shared.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred ?? false,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true)
            }
        } catch {
            clientPolicyForce = false
        }
    }
}

@Observable
@MainActor
final class AnalyticsViewModel {
    var overview = FactoryAnalyticsOverview(
        dailyActivity: [],
        transfersTotal: 0,
        manifestsActive: 0,
        exceptionQueue: 0,
        avgLeadTimeMins: 0
    )
    var loading: Bool = true
    var error: String?

    private let fetchAnalytics: @MainActor () async throws -> FactoryAnalyticsOverview

    init(
        fetchAnalytics: @escaping @MainActor () async throws -> FactoryAnalyticsOverview = {
            try await FactoryService.analyticsOverview()
        }
    ) {
        self.fetchAnalytics = fetchAnalytics
    }

    func load() async {
        loading = true
        error = nil
        do {
            overview = try await fetchAnalytics()
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }
}

@Observable
@MainActor
final class InsightsViewModel {
    var insights: [Insight] = []
    var loading: Bool = true
    var error: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchInsights: @MainActor () async throws -> [Insight]

    init(
        fetchInsights: @escaping @MainActor () async throws -> [Insight] = {
            try await FactoryService.insights().insights
        }
    ) {
        self.fetchInsights = fetchInsights
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil
        do {
            insights = try await fetchInsights()
        } catch {
            self.error = error.localizedDescription
        }
        if !silent {
            loading = false
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_")
                    || event.type.hasPrefix("FACTORY_SUPPLY_") else { return }
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class NotificationsViewModel {
    var items: [FactoryNotificationItem] = []
    var unreadCount: Int = 0
    var loading: Bool = true
    var error: String?

    private let fetchPage: @MainActor () async throws -> FactoryNotificationsPage
    private let markAllReadAction: @MainActor () async throws -> Void

    init(
        fetchPage: @escaping @MainActor () async throws -> FactoryNotificationsPage = {
            try await APIClient.shared.get(
                "v1/user/notifications",
                query: ["limit": "50", "offset": "0"]
            )
        },
        markAllReadAction: @escaping @MainActor () async throws -> Void = {
            try await APIClient.shared.postVoid(
                "v1/user/notifications/read",
                body: MarkNotificationsReadRequest(markAll: true)
            )
        }
    ) {
        self.fetchPage = fetchPage
        self.markAllReadAction = markAllReadAction
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let page = try await fetchPage()
            items = page.notifications
            unreadCount = page.unreadCount
        } catch let apiError as APIError {
            switch apiError {
            case .httpError(503):
                self.error = "Notifications unavailable on this stack"
            default:
                self.error = "Failed to load notifications"
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func markAllRead() async {
        do {
            try await markAllReadAction()
            PegasusHaptics.confirm()
            await load()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
