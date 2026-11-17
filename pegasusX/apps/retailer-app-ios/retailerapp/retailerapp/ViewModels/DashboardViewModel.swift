import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class DashboardViewModel {
    var activeOrders: [Order] = []
    var predictions: [RetailerAIPrediction] = []
    var reorderProducts: [Product] = []
    var isLoading = false
    var actingId: String?
    var orderActionPending = false
    var loadError: String?
    var pulseEvents: [RetailerPulseEvent] = []
    var pulseLoading = false
    var pulseError: String?
    var commandPulse: ControlTowerPulseWire?
    var commandPulseError: String?
    var commandJump: CommandStatusJump?

    // W5 Fast Mode state
    var fastHome: RetailerFastHomeResponse?
    var fastHomeUsingFallback = false
    var serverClockOffset: TimeInterval = 0
    var cutoffCountdown: CutoffCountdownState?
    var retailerRole = ""
    var isOwner = false
    var workspaceMode: RetailerWorkspaceMode = .fast
    var fastActionMessage: String?
    var fastActionError: String?
    var showEditProposalSheet = false

    private let api = APIClient.shared
    private let preferencesStore: RetailerPreferencesStore
    private let retailerIdProvider: () -> String
    private let dateProvider: () -> Date
    private let fastHomeLoader: () async throws -> RetailerFastHomeResponse
    private let meLoader: () async throws -> RetailerMeResponse
    private let ordersLoader: (String) async throws -> [Order]
    private let predictionsLoader: () async throws -> [RetailerAIPrediction]
    private let creditProfileLoader: () async throws -> CreditProfile
    private let creditRelationshipsLoader: () async throws -> [CreditRelationshipWire]
    private let confirmAiAction: (String) async throws -> Void
    private let rejectAiAction: (String, String) async throws -> Void
    private let confirmPreorderAction: (String) async throws -> Void
    private let rejectPreorderAction: (String, String) async throws -> Void
    private let editPreorderAction: (String, String, [APIClient.EditPreorderItem]) async throws -> Void
    private let shouldLoadSecondaryModules: Bool

    init(
        preferencesStore: RetailerPreferencesStore = .shared,
        retailerIdProvider: @escaping () -> String = { AuthManager.shared.currentUser?.id ?? "" },
        dateProvider: @escaping () -> Date = Date.init,
        fastHomeLoader: (() async throws -> RetailerFastHomeResponse)? = nil,
        meLoader: (() async throws -> RetailerMeResponse)? = nil,
        ordersLoader: ((String) async throws -> [Order])? = nil,
        predictionsLoader: (() async throws -> [RetailerAIPrediction])? = nil,
        creditProfileLoader: (() async throws -> CreditProfile)? = nil,
        creditRelationshipsLoader: (() async throws -> [CreditRelationshipWire])? = nil,
        confirmAiAction: ((String) async throws -> Void)? = nil,
        rejectAiAction: ((String, String) async throws -> Void)? = nil,
        confirmPreorderAction: ((String) async throws -> Void)? = nil,
        rejectPreorderAction: ((String, String) async throws -> Void)? = nil,
        editPreorderAction: ((String, String, [APIClient.EditPreorderItem]) async throws -> Void)? = nil,
        shouldLoadSecondaryModules: Bool = true
    ) {
        self.preferencesStore = preferencesStore
        self.retailerIdProvider = retailerIdProvider
        self.dateProvider = dateProvider
        let client = APIClient.shared
        self.fastHomeLoader = fastHomeLoader ?? { try await client.getRetailerFastHome() }
        self.meLoader = meLoader ?? { try await client.getRetailerMe() }
        self.ordersLoader = ordersLoader ?? { rid in try await client.get(path: "/v1/retailers/\(rid)/orders") }
        self.predictionsLoader = predictionsLoader ?? { try await client.getRetailerAIPredictions() }
        self.creditProfileLoader = creditProfileLoader ?? { try await client.getCreditProfile() }
        self.creditRelationshipsLoader = creditRelationshipsLoader ?? { try await client.getCreditRelationships() }
        self.confirmAiAction = confirmAiAction ?? { orderId in try await client.confirmAiOrder(orderId: orderId) }
        self.rejectAiAction = rejectAiAction ?? { orderId, reason in try await client.rejectAiOrder(orderId: orderId, reason: reason) }
        self.confirmPreorderAction = confirmPreorderAction ?? { orderId in try await client.confirmPreorder(orderId: orderId) }
        self.rejectPreorderAction = rejectPreorderAction ?? { orderId, reason in try await client.rejectPreorder(orderId: orderId, reason: reason) }
        self.editPreorderAction = editPreorderAction ?? { orderId, date, items in
            try await client.editPreorder(orderId: orderId, deliveryDate: date, items: items)
        }
        self.shouldLoadSecondaryModules = shouldLoadSecondaryModules
        self.workspaceMode = FastHomeContractHelper.resolveEffectiveWorkspaceMode(
            persistedOverride: preferencesStore.workspaceModeOverride,
            retailerRole: "",
            isOwner: false
        )
    }

    func switchWorkspaceMode(_ mode: RetailerWorkspaceMode) {
        preferencesStore.setWorkspaceMode(mode)
        workspaceMode = mode
    }

    func syncWorkspaceModeFromPreferences() {
        workspaceMode = FastHomeContractHelper.resolveEffectiveWorkspaceMode(
            persistedOverride: preferencesStore.workspaceModeOverride,
            retailerRole: retailerRole,
            isOwner: isOwner
        )
    }

    func tickCountdown(now: Date? = nil) {
        cutoffCountdown = FastHomeContractHelper.computeCutoffCountdown(
            cutoffAtISO: fastHome?.cutoffAt,
            serverClockOffset: serverClockOffset,
            now: now ?? dateProvider()
        )
    }

    func loadData(silent: Bool = false) async {
        let rid = retailerIdProvider()
        let now = dateProvider()
        if !silent { isLoading = true }
        if !silent { loadError = nil }
        pulseLoading = true
        pulseError = nil

        if let me = try? await meLoader() {
            retailerRole = me.retailerRole
            isOwner = me.resolvedIsOwner
        }
        syncWorkspaceModeFromPreferences()

        var loadedAllOrders: [Order] = []
        do {
            let orders = try await ordersLoader(rid)
            loadedAllOrders = orders
            activeOrders = orders.filter { $0.status.isActive }
        } catch {
            activeOrders = []
            if !silent {
                loadError = error.localizedDescription
            }
        }

        do {
            let p = try await predictionsLoader()
            predictions = p
        } catch {
            predictions = []
        }

        do {
            let bffFastHome = try await fastHomeLoader()
            fastHome = bffFastHome
            fastHomeUsingFallback = false
            serverClockOffset = FastHomeContractHelper.computeServerClockOffset(
                serverNowISO: bffFastHome.serverNow,
                deviceNow: now
            )
            cutoffCountdown = FastHomeContractHelper.computeCutoffCountdown(
                cutoffAtISO: bffFastHome.cutoffAt,
                serverClockOffset: serverClockOffset,
                now: now
            )
            loadError = nil
        } catch {
            let cp = try? await creditProfileLoader()
            let rels = (try? await creditRelationshipsLoader()) ?? []
            let fallback = FastHomeContractHelper.buildFastHomeFallback(
                orders: loadedAllOrders,
                predictions: predictions,
                creditProfile: cp,
                creditRelationships: rels,
                locale: preferencesStore.currentLanguage.rawValue,
                now: now
            )
            fastHome = fallback
            fastHomeUsingFallback = true
            serverClockOffset = 0
            cutoffCountdown = FastHomeContractHelper.computeCutoffCountdown(
                cutoffAtISO: fallback.cutoffAt,
                serverClockOffset: 0,
                now: now
            )
        }

        if shouldLoadSecondaryModules {
            do {
                let products: [Product] = try await api.get(path: "/v1/catalog/products")
                reorderProducts = Array(products.prefix(6))
            } catch {
                reorderProducts = []
            }

            do {
                let pulse = try await api.getRetailerPulse()
                let result = PulseHonesty.apply(ok: true, incoming: pulse.events, previous: pulseEvents)
                pulseEvents = result.events
                pulseError = result.error
            } catch {
                pulseEvents = []
                pulseError = error.localizedDescription
            }
            await loadCommandPulse()
        }
        pulseLoading = false
        if !silent { isLoading = false }
    }

    func pollCommandPulseLoop() async {
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: 60_000_000_000)
            guard !Task.isCancelled else { break }
            if NetworkLifecycleEngine.shared.isOnline {
                await loadCommandPulse()
            }
        }
    }

    func loadCommandPulse() async {
        do {
            let incoming = try await api.getControlTowerPulse()
            let result = PulseHonesty.applyObject(ok: true, incoming: incoming, previous: commandPulse)
            commandPulse = result.value
            commandPulseError = result.error
        } catch {
            commandPulse = nil
            commandPulseError = error.localizedDescription
        }
    }

    func cancelOrder(_ orderId: String) async {
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        do {
            let _: [String: String] = try await api.post(
                path: "/v1/order/cancel",
                body: [
                    "order_id": orderId,
                    "retailer_id": retailerId,
                ],
                headers: ["Idempotency-Key": RetailerIdempotency.cancel(orderId: orderId)]
            )
            withAnimation(AnimationConstants.fluid) {
                activeOrders.removeAll { $0.id == orderId }
            }
        } catch {
            loadError = error.localizedDescription
        }
    }

    func confirmAiOrder(_ orderId: String) async {
        actingId = orderId
        orderActionPending = true
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            try await APIClient.shared.confirmAiOrder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm AI order"
        }
    }

    func rejectAiOrder(_ orderId: String) async {
        actingId = orderId
        orderActionPending = true
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            try await APIClient.shared.rejectAiOrder(orderId: orderId, reason: "Retailer rejected")
            await loadData()
        } catch {
            loadError = "Failed to reject AI order"
        }
    }

    func confirmPreorder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        do {
            try await APIClient.shared.confirmPreorder(orderId: orderId)
            await loadData()
        } catch {
            loadError = "Failed to confirm preorder"
        }
    }

    func editPreorder(_ orderId: String) async {
        orderActionPending = true
        defer { orderActionPending = false }
        guard let order = await findOrder(orderId) else { return }
        let deliveryDate = order.deliverBefore ?? order.autoConfirmAt ?? order.estimatedDelivery ?? ""
        guard !deliveryDate.isEmpty else { return }
        let lineItems = order.items.map { item in
            APIClient.EditPreorderItem(
                sku: item.productId.isEmpty ? item.id : item.productId,
                name: item.productName,
                quantity: Int64(item.quantity),
                unitPriceMinor: item.unitPrice
            )
        }
        guard !lineItems.isEmpty else { return }
        do {
            try await APIClient.shared.editPreorder(orderId: orderId, deliveryDate: deliveryDate, items: lineItems)
            await loadData()
        } catch {
            print("Failed to edit preorder")
        }
    }

    func findOrder(_ orderId: String) async -> Order? {
        if let order = activeOrders.first(where: { $0.id == orderId }) {
            return order
        }
        let retailerId = retailerIdProvider()
        guard !retailerId.isEmpty else { return nil }
        let orders: [Order] = (try? await ordersLoader(retailerId)) ?? []
        return orders.first { $0.id == orderId }
    }

    func confirmFastProposal() async {
        guard let proposal = fastHome?.pendingProposal, !proposal.orderId.isEmpty else { return }
        actingId = proposal.orderId
        orderActionPending = true
        fastActionMessage = nil
        fastActionError = nil
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            let source = (proposal.orderSource ?? "").uppercased()
            if source == "AI" {
                try await confirmAiAction(proposal.orderId)
            } else {
                do {
                    try await confirmPreorderAction(proposal.orderId)
                } catch {
                    try await confirmAiAction(proposal.orderId)
                }
            }
            fastActionMessage = L10n.string("mobile_retailer.fast_home.action_done_confirm")
            Haptics.success()
            await loadData(silent: true)
        } catch {
            fastActionError = error.localizedDescription
            Haptics.error()
        }
    }

    func holdFastProposal() async {
        guard let proposal = fastHome?.pendingProposal, !proposal.orderId.isEmpty else { return }
        actingId = proposal.orderId
        orderActionPending = true
        fastActionMessage = nil
        fastActionError = nil
        defer {
            actingId = nil
            orderActionPending = false
        }
        do {
            let source = (proposal.orderSource ?? "").uppercased()
            if source == "PREORDER" {
                try await rejectPreorderAction(proposal.orderId, "Retailer placed on hold")
            } else {
                try await rejectAiAction(proposal.orderId, "Retailer placed on hold")
            }
            fastActionMessage = L10n.string("mobile_retailer.fast_home.action_done_hold")
            Haptics.selection()
            await loadData(silent: true)
        } catch {
            fastActionError = error.localizedDescription
            Haptics.error()
        }
    }

    func editFastProposal(updatedItems: [FastHomeProposalItem]) async {
        guard let proposal = fastHome?.pendingProposal, !proposal.orderId.isEmpty else { return }
        let normalizedItems: [APIClient.EditPreorderItem] = updatedItems.compactMap { item in
            let qty = max(0, item.quantity)
            guard qty > 0 else { return nil }
            let snapped: Int64
            if let packSize = item.packSize, packSize > 1 {
                snapped = Int64(RetailerPackHelper.snapToPackMultiple(Int(qty), packSize: packSize))
            } else {
                snapped = qty
            }
            return APIClient.EditPreorderItem(
                sku: item.sku,
                name: item.name,
                quantity: snapped,
                unitPriceMinor: item.unitPriceMinor
            )
        }
        guard !normalizedItems.isEmpty else { return }

        actingId = proposal.orderId
        orderActionPending = true
        fastActionMessage = nil
        fastActionError = nil
        defer {
            actingId = nil
            orderActionPending = false
        }

        let fallbackTomorrow = String(
            FastHomeContractHelper.formatISO8601Date(dateProvider().addingTimeInterval(86_400)).prefix(10)
        )
        let deliveryDate = [
            proposal.requestedDeliveryDate,
            fastHome?.nextDeliveryDate,
            fallbackTomorrow,
        ]
        .compactMap { $0?.trimmingCharacters(in: .whitespacesAndNewlines) }
        .first { !$0.isEmpty } ?? fallbackTomorrow

        do {
            try await editPreorderAction(proposal.orderId, deliveryDate, normalizedItems)
            showEditProposalSheet = false
            fastActionMessage = L10n.string("mobile_retailer.fast_home.action_done_edit")
            Haptics.success()
            await loadData(silent: true)
        } catch {
            fastActionError = error.localizedDescription
            Haptics.error()
        }
    }
}
