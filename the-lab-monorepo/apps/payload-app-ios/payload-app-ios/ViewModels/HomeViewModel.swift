//
//  HomeViewModel.swift
//  payload-app-ios
//
//  Phase 3 + 4 vertical:
//   - sidebar: vehicle list
//   - detail: open manifest summary, per-order checklist, per-order seal,
//             60s post-seal double-check countdown, manifest seal, All Sealed.
//  Mirrors the Android HomeViewModel state machine — both apps surface the
//  identical state per backend payload.
//

import Foundation
import Observation

@MainActor
@Observable
final class HomeViewModel {
    // Sidebar
    private(set) var trucks: [Truck] = []
    private(set) var selectedTruckId: String?
    private(set) var loadingTrucks = false

    // Detail
    private(set) var manifest: Manifest?
    private(set) var orders: [LiveOrder] = []
    private(set) var selectedOrderId: String?
    /// Local-only line-item check state.
    private(set) var checkedItems: Set<String> = []
    private(set) var sealedOrderIds: Set<String> = []
    private(set) var dispatchCodes: [String: String] = [:]
    private(set) var postSealOrderId: String?
    private(set) var postSealCountdown: Int = 0
    private(set) var manifestSealed = false

    // Loading flags
    private(set) var loadingManifest = false
    private(set) var loadingOrders = false
    private(set) var startingLoading = false
    private(set) var sealingOrderId: String?
    private(set) var sealingManifest = false

    private(set) var error: String?

    // MARK: - Phase 6 state
    private(set) var notifications: [NotificationItem] = []
    private(set) var unreadCount: Int = 0
    private(set) var online: Bool = false
    private(set) var queuedActions: Int = 0
    var showNotificationsPanel: Bool = false
    var queuedNoticeMessage: String?
    var syncCompleteMessage: String?

    // MARK: - Phase 5 state
    /// Order currently being removed via manifest-exception.
    private(set) var exceptionLoadingOrderId: String?
    /// One-shot DLQ-escalated banner message.
    private(set) var escalatedMessage: String?
    /// True while POST inject-order is in flight.
    private(set) var injectingOrder = false
    /// Order id currently being re-dispatched (drives Re-Dispatch sheet).
    private(set) var reDispatchOrderId: String?
    private(set) var loadingRecommendations = false
    private(set) var recommendations: RecommendReassignResponse?
    private(set) var reassigning = false

    private let api: APIClient
    private let ws: WebSocketClient
    private let queue: OfflineQueue
    private var countdownTask: Task<Void, Never>?

    init() {
        self.api = APIClient.shared
        self.ws = WebSocketClient()
        self.queue = OfflineQueue.shared
        self.queuedActions = queue.read().count
        self.ws.onFrame = { [weak self] frame in self?.handleFrame(frame) }
        self.ws.onReconnect = { [weak self] in
            Task { @MainActor [weak self] in
                guard let self else { return }
                self.online = self.ws.online
                await self.loadNotifications()
                await self.flushQueue()
            }
        }
    }

    deinit {
        // ws is @MainActor; HomeView triggers disconnectPhase6() on disappear.
    }

    // MARK: - Truck list
    func refreshTrucks() async {
        loadingTrucks = true
        error = nil
        defer { loadingTrucks = false }
        do {
            let result = try await api.trucks()
            trucks = result
            if selectedTruckId == nil, let first = result.first?.id {
                await selectTruck(first)
            }
        } catch {
            self.error = describe(error)
        }
    }

    func selectTruck(_ truckId: String) async {
        if selectedTruckId == truckId, manifest != nil { return }
        cancelCountdown()
        selectedTruckId = truckId
        manifest = nil
        orders = []
        selectedOrderId = nil
        checkedItems = []
        sealedOrderIds = []
        dispatchCodes = [:]
        postSealOrderId = nil
        postSealCountdown = 0
        manifestSealed = false
        error = nil

        await loadManifest(for: truckId)
        await loadOrders(for: truckId)
    }

    func refreshManifest() async {
        guard let id = selectedTruckId else { return }
        await selectTruck(id)
    }

    private func loadManifest(for truckId: String) async {
        loadingManifest = true
        defer { loadingManifest = false }
        do {
            // Backend has no "draft OR loading" filter — try DRAFT first then LOADING.
            let draft = try await api.draftManifests(truckId: truckId)
            if let m = draft.manifests.first {
                manifest = m
                return
            }
            let loading = try await loadingManifests(truckId: truckId)
            manifest = loading.manifests.first
        } catch {
            self.error = describe(error)
        }
    }

    /// Backend supports `?state=LOADING&truck_id=X` against the same endpoint.
    private func loadingManifests(truckId: String) async throws -> ManifestsResponse {
        // Reuse APIClient.draftManifests by inlining a typed call:
        try await api.manifests(state: "LOADING", truckId: truckId)
    }

    private func loadOrders(for truckId: String) async {
        loadingOrders = true
        defer { loadingOrders = false }
        do {
            let result = try await api.orders(vehicleId: truckId, state: "LOADED")
            orders = result
            if selectedOrderId == nil { selectedOrderId = result.first?.orderId }
        } catch {
            self.error = describe(error)
        }
    }

    // MARK: - Per-order checklist + seal
    func selectOrder(_ orderId: String) { selectedOrderId = orderId }

    func toggleItem(_ lineItemId: String) {
        if checkedItems.contains(lineItemId) { checkedItems.remove(lineItemId) }
        else { checkedItems.insert(lineItemId) }
    }

    /// True when every line item of [orderId] is checked AND the order is not yet sealed.
    func canSealOrder(_ orderId: String) -> Bool {
        if sealedOrderIds.contains(orderId) { return false }
        guard let order = orders.first(where: { $0.orderId == orderId }) else { return false }
        let items = order.items ?? []
        guard !items.isEmpty else { return false }
        return items.allSatisfy { checkedItems.contains($0.lineItemId) }
    }

    func sealSelectedOrder() async {
        guard let orderId = selectedOrderId,
              let truckId = selectedTruckId,
              canSealOrder(orderId) else { return }
        sealingOrderId = orderId
        error = nil
        defer { sealingOrderId = nil }
        do {
            let resp = try await api.sealOrder(orderId: orderId, terminalId: truckId)
            sealedOrderIds.insert(orderId)
            dispatchCodes[orderId] = resp.dispatchCode
            postSealOrderId = orderId
            postSealCountdown = 60
            startCountdown()
        } catch {
            self.error = describe(error)
        }
    }

    private func startCountdown() {
        cancelCountdown()
        countdownTask = Task { @MainActor [weak self] in
            while let self, self.postSealCountdown > 0 {
                try? await Task.sleep(nanoseconds: 1_000_000_000)
                if Task.isCancelled { return }
                self.postSealCountdown = max(0, self.postSealCountdown - 1)
            }
            self?.advanceAfterCountdown()
        }
    }

    private func advanceAfterCountdown() {
        postSealOrderId = nil
        if let next = orders.first(where: { !sealedOrderIds.contains($0.orderId) }) {
            selectedOrderId = next.orderId
        }
    }

    func dismissCountdown() {
        cancelCountdown()
        postSealCountdown = 0
        advanceAfterCountdown()
    }

    private func cancelCountdown() {
        countdownTask?.cancel()
        countdownTask = nil
    }

    // MARK: - Manifest-level transitions
    func startLoading() async {
        guard let id = manifest?.manifestId, !startingLoading else { return }
        startingLoading = true
        error = nil
        defer { startingLoading = false }
        do {
            _ = try await api.startLoading(manifestId: id)
            manifest = manifest.map { mutateState($0, to: "LOADING") }
        } catch {
            self.error = describe(error)
        }
    }

    var allOrdersSealed: Bool {
        !orders.isEmpty && orders.allSatisfy { sealedOrderIds.contains($0.orderId) }
    }

    func sealManifest() async {
        guard let id = manifest?.manifestId, !sealingManifest else { return }
        sealingManifest = true
        error = nil
        defer { sealingManifest = false }
        do {
            _ = try await api.sealManifest(manifestId: id)
            manifestSealed = true
            manifest = manifest.map { mutateState($0, to: "SEALED") }
        } catch {
            self.error = describe(error)
        }
    }

    /// Reset to a fresh state and reload trucks (after All Sealed).
    func startNewManifest() async {
        cancelCountdown()
        trucks = []
        selectedTruckId = nil
        manifest = nil
        orders = []
        selectedOrderId = nil
        checkedItems = []
        sealedOrderIds = []
        dispatchCodes = [:]
        postSealOrderId = nil
        postSealCountdown = 0
        manifestSealed = false
        error = nil
        await refreshTrucks()
    }

    func clearError() { error = nil }
    func clearEscalatedMessage() { escalatedMessage = nil }

    // MARK: - Phase 5: Exception (remove order from manifest)

    /// Reasons: OVERFLOW | DAMAGED | MANUAL. 3+ OVERFLOW → DLQ escalation.
    func reportException(orderId: String, reason: String) async {
        guard let manifestId = manifest?.manifestId else { return }
        guard exceptionLoadingOrderId == nil else { return }
        exceptionLoadingOrderId = orderId
        error = nil
        defer { exceptionLoadingOrderId = nil }
        do {
            let resp = try await api.manifestException(manifestId: manifestId, orderId: orderId, reason: reason)
            orders.removeAll { $0.orderId == orderId }
            if selectedOrderId == orderId { selectedOrderId = orders.first?.orderId }
            if resp.escalated == true {
                let count = resp.overflowCount ?? 0
                escalatedMessage = "DLQ ESCALATION: order \(orderId.prefix(8)) escalated after \(count) overflow attempts."
            }
        } catch {
            self.error = describe(error)
        }
    }

    // MARK: - Phase 5: Mid-load order injection

    func injectOrder(_ orderId: String) async {
        let trimmed = orderId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty,
              let manifestId = manifest?.manifestId,
              let truckId = selectedTruckId,
              !injectingOrder else { return }
        // Phase 6: when offline, persist to the queue and notify the user.
        if !online {
            let body = (try? JSONEncoder().encode(InjectOrderRequest(orderId: trimmed))).flatMap { String(data: $0, encoding: .utf8) } ?? ""
            let action = QueuedAction(
                id: UUID().uuidString,
                endpoint: "/v1/supplier/manifests/\(manifestId)/inject-order",
                method: "POST",
                body: body,
                createdAt: Date().timeIntervalSince1970
            )
            queue.enqueue(action)
            queuedActions = queue.read().count
            queuedNoticeMessage = "Queued offline. Will sync when connection restores."
            return
        }
        injectingOrder = true
        error = nil
        defer { injectingOrder = false }
        do {
            _ = try await api.injectOrder(manifestId: manifestId, orderId: trimmed)
            await loadManifest(for: truckId)
            await loadOrders(for: truckId)
        } catch {
            self.error = describe(error)
        }
    }

    // MARK: - Phase 5: Re-dispatch (recommend + reassign)

    func openReDispatch(orderId: String) async {
        reDispatchOrderId = orderId
        loadingRecommendations = true
        recommendations = nil
        error = nil
        defer { loadingRecommendations = false }
        do {
            recommendations = try await api.recommendReassign(orderId: orderId)
        } catch {
            self.error = describe(error)
        }
    }

    func closeReDispatch() {
        reDispatchOrderId = nil
        recommendations = nil
        loadingRecommendations = false
    }

    /// `newDriverId` is the chosen recommendation's driver_id (RouteId == DriverId in this codebase).
    func reassignTo(_ newDriverId: String) async {
        guard let orderId = reDispatchOrderId, !reassigning else { return }
        reassigning = true
        error = nil
        defer { reassigning = false }
        do {
            let resp = try await api.fleetReassign(orderIds: [orderId], newRouteId: newDriverId)
            if let conflict = resp.conflicts?.first(where: { $0.orderId == orderId }) {
                error = "Reassign conflict: \(conflict.reason ?? "unknown")"
                return
            }
            orders.removeAll { $0.orderId == orderId }
            if selectedOrderId == orderId { selectedOrderId = orders.first?.orderId }
            reDispatchOrderId = nil
            recommendations = nil
        } catch {
            self.error = describe(error)
        }
    }

    // MARK: - Phase 6: WebSocket / notifications / FCM / queue

    /// Call once after a successful login (HomeView triggers this in `.task`).
    func bootstrapPhase6(token: String) async {
        ws.connect(token: token)
        online = ws.online
        queuedActions = queue.read().count
        await loadNotifications()
    }

    func disconnectPhase6() {
        ws.disconnect()
        online = false
    }

    func loadNotifications() async {
        do {
            let resp = try await api.notifications(limit: 50)
            notifications = resp.notifications
            unreadCount = resp.unreadCount
        } catch {
            // soft-fail; surface via error only on user-initiated retry
        }
    }

    func toggleNotificationsPanel() {
        showNotificationsPanel.toggle()
    }

    func markNotificationRead(_ id: String) {
        if let idx = notifications.firstIndex(where: { $0.notificationId == id }), notifications[idx].isUnread {
            unreadCount = max(0, unreadCount - 1)
        }
        Task { _ = try? await api.markRead(ids: [id], all: nil) }
    }

    func markAllNotificationsRead() {
        unreadCount = 0
        Task { _ = try? await api.markRead(ids: nil, all: true) }
    }

    func clearSyncCompleteMessage() { syncCompleteMessage = nil }
    func clearQueuedNoticeMessage() { queuedNoticeMessage = nil }

    func registerFcmToken(_ token: String) {
        Task { _ = try? await api.registerDeviceToken(token) }
    }

    private func handleFrame(_ frame: WsMessage) {
        let item = NotificationItem(
            notificationId: "live-\(Int(Date().timeIntervalSince1970 * 1000))",
            type: frame.type ?? "",
            title: frame.title ?? "",
            body: frame.body ?? "",
            payload: nil,
            channel: frame.channel ?? "",
            readAt: nil,
            createdAt: ""
        )
        notifications.insert(item, at: 0)
        unreadCount += 1
    }

    private func flushQueue() async {
        let (sent, kept) = await queue.flush(api: api)
        queuedActions = kept
        if sent > 0 {
            syncCompleteMessage = "Synced \(sent) queued action\(sent == 1 ? "" : "s")."
        }
    }

    // MARK: - Helpers

    /// Manifest is a struct of `let` constants; rebuild it with the new state.
    private func mutateState(_ m: Manifest, to state: String) -> Manifest {
        Manifest(
            manifestId: m.manifestId,
            truckId: m.truckId,
            driverId: m.driverId,
            state: state,
            totalVolumeVu: m.totalVolumeVu,
            maxVolumeVu: m.maxVolumeVu,
            stopCount: m.stopCount,
            regionCode: m.regionCode,
            sealedAt: m.sealedAt,
            dispatchedAt: m.dispatchedAt,
            createdAt: m.createdAt,
            orders: m.orders,
            overflowCount: m.overflowCount
        )
    }

    private func describe(_ e: Error) -> String {
        if let api = e as? APIError {
            switch api {
            case .unauthorized: return "Session expired. Sign in again."
            case .forbidden: return "Not authorized for this scope."
            case .problemDetail(let p): return p.detail ?? p.title ?? "Request failed."
            case .httpError(let s): return "Server error (\(s))."
            case .networkError: return "Network unavailable."
            case .decodingError: return "Unexpected response from server."
            case .invalidURL: return "Invalid request URL."
            }
        }
        return e.localizedDescription
    }
}
