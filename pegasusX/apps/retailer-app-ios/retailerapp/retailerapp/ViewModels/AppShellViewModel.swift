import Foundation
import Observation
import SwiftData
import SwiftUI

@Observable
@MainActor
final class AppShellViewModel {
    var selectedTab: AppTab = .home
    var sideMenuSelection: SideMenuTab = .home
    var isSidebarExpanded = true

    var showSidebar = false
    var showFutureDemand = false
    var showAutoOrder = false
    var showActiveOrderDetail = false
    var showProfile = false
    var showCart = false
    var showInsights = false
    var showProcurement = false
    var showCredit = false
    var showHq = false
    var showNotificationInbox = false

    var notificationCount = 0
    var cartBounce = false
    var activeOrders: [Order] = []
    var paymentEvent: PaymentRequiredEvent?
    var shopClosedAlert: ShopClosedAlertEvent?
    var clientPolicyMessage: String?
    var clientPolicyForce = false
    var pendingManifest: AutoUpdater.Manifest?
    var rescueWarningMessage: String?
    var deliveriesHubInitialTab: DeliveriesHubTab = .map

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    func showFloatingBar(horizontalSizeClass: UserInterfaceSizeClass?) -> Bool {
        if horizontalSizeClass == .regular {
            return sideMenuSelection == .home
                || sideMenuSelection == .orders
                || sideMenuSelection == .suppliers
                || sideMenuSelection == .deliveries
        } else {
            return selectedTab == .home
                || selectedTab == .catalog
                || selectedTab == .deliveries
                || selectedTab == .store
        }
    }

    func navigateToSuppliersTab(horizontalSizeClass: UserInterfaceSizeClass?) {
        Haptics.selection()
        if horizontalSizeClass == .regular {
            withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                sideMenuSelection = .suppliers
            }
        } else {
            selectedTab = .more
        }
    }

    func handleTabChange(from oldTab: AppTab, to newTab: AppTab) {
        if newTab == .deliveries && oldTab != .deliveries {
            deliveriesHubInitialTab = .map
        }
    }

    func handleSidebarNavigation(
        _ destination: SidebarDestination,
        horizontalSizeClass: UserInterfaceSizeClass?,
        auth: AuthManager
    ) {
        Haptics.selection()
        switch destination {
        case .dashboard:
            selectedTab = .home
        case .procurement:
            if horizontalSizeClass == .regular {
                withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                    sideMenuSelection = .procurement
                }
            } else {
                showProcurement = true
            }
        case .autoOrder:
            showAutoOrder = true
        case .futureDemand:
            showFutureDemand = true
        case .inbound:
            deliveriesHubInitialTab = .dock
            if horizontalSizeClass == .regular {
                withAnimation(.spring(response: 0.3, dampingFraction: 0.7)) {
                    sideMenuSelection = .deliveries
                }
            } else {
                selectedTab = .deliveries
            }
        case .inbox:
            showNotificationInbox = true
        case .profile:
            selectedTab = .more
        case .insights:
            showInsights = true
        case .credit:
            showCredit = true
        case .hq:
            showHq = true
        case .settings:
            selectedTab = .more
        case .logout:
            auth.logout()
        }
    }

    func bootstrap() async {
        await PushNotificationManager.shared.requestAuthorization()
        await loadActiveOrders()
        await loadPendingPayments()
        await loadNotificationCount()
        await loadClientPolicy()
    }

    func refreshOnActive(refreshCenter: RetailerRefreshCenter) async {
        await loadActiveOrders()
        await loadPendingPayments()
        await loadNotificationCount()
        refreshCenter.trigger()
    }

    func loadActiveOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        do {
            let result: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            activeOrders = result.filter { $0.status.isActive }
        } catch {
            activeOrders = []
        }
    }

    func loadNotificationCount() async {
        do {
            let resp: NotificationsResponse = try await api.get(
                path: "/v1/user/notifications?limit=1&offset=0"
            )
            notificationCount = resp.unreadCount
        } catch {
            notificationCount = max(0, notificationCount)
        }
    }

    func loadPendingPayments(reconcile: Bool = false) async {
        if !reconcile {
            guard paymentEvent == nil else { return }
        }
        do {
            let response = try await api.getPendingPayments()
            guard let session = response.pendingPayments.first else {
                if reconcile {
                    paymentEvent = nil
                }
                return
            }
            let gateway = session.gateway ?? ""
            paymentEvent = PaymentRequiredEvent(
                type: "PAYMENT_REQUIRED",
                orderId: session.orderId,
                invoiceId: session.invoiceId ?? "",
                sessionId: session.sessionId ?? "",
                amountUzs: session.lockedAmount,
                originalAmountUzs: session.lockedAmount,
                availableCardGateways: gateway == "CASH" || gateway.isEmpty ? [] : [gateway],
                message: "Pending payment requires completion.",
                paymentMethod: gateway == "CASH" ? "CASH" : "CARD"
            )
        } catch {
            if reconcile {
                paymentEvent = paymentEvent
            }
        }
    }

    func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool?
                let forceUpdate: Bool?
                let updateDeferred: Bool?
                let minimumVersion: String?
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
            let role = EnterpriseUpdateConfig.policyRole
            let channel = EnterpriseUpdateConfig.channel
            let policy: ClientPolicy = try await api.get(
                path: "/v1/platform/client-policy?role=\(role)&platform=ios&version=\(version)&channel=\(channel)"
            )
            let state = await AutoUpdater.shared.evaluate(
                outdated: policy.outdated == true,
                forceUpdate: policy.forceUpdate == true,
                updateDeferred: policy.updateDeferred == true,
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
            clientPolicyMessage = nil
            clientPolicyForce = false
            pendingManifest = nil
        }
    }

    func connectWebSocket(modelContext: ModelContext, refreshCenter: RetailerRefreshCenter) async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else { return }
        ws.connect(retailerId: rid)
        for await event in ws.eventStream() {
            switch event {
            case .paymentRequired(let payload):
                paymentEvent = payload
            case .driverApproaching:
                await loadActiveOrders()
            case .orderCompleted, .fiscalSucceeded:
                await loadActiveOrders()
            case .fiscalizing:
                await loadActiveOrders()
            case .paymentSettled:
                await loadActiveOrders()
            case .paymentFailed, .paymentExpired:
                await loadActiveOrders()
            case .orderStatusChanged:
                await loadActiveOrders()
            case .orderReassigned(let orderId, let licensePlate):
                withAnimation(AnimationConstants.fluid) {
                    rescueWarningMessage = "Your order #\(orderId.suffix(4)) has been reassigned to rescue truck (\(licensePlate)) due to a breakdown."
                }
                await loadActiveOrders()
            case .preOrderAutoAccepted, .preOrderConfirmed, .preOrderEdited, .preOrderNudge, .preOrderConfirmationPush,
                 .preOrderDateProposed, .preOrderDateAccepted, .preOrderDateRejected, .preOrderCancelled:
                await loadActiveOrders()
            case .shopClosedAlert(let alert):
                shopClosedAlert = alert
                await loadActiveOrders()
            case .cartSyncUpdated:
                break
            case .promotionChanged:
                break
            case .transportReconnected:
                await RetailerSessionReconcile.run(api: api)
                await loadActiveOrders()
                await loadPendingPayments(reconcile: true)
                await loadClientPolicy()
                await PendingOrderReplayer.flush(modelContext: modelContext, api: api)
            }
            refreshCenter.trigger()
        }
    }
}
