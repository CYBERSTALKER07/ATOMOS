import Foundation

@Observable
@MainActor
final class OrdersViewModel {
    var orders: [SupplierOrder] = []
    var loading = true
    var error: String?
    var statusFilter = "ACTIVE"
    var commandStatus: String?
    var selection: SupplierOrder?
    var vettingOrderId: String?

    // Reassignment
    var reassignTarget: String? {
        didSet { showReassignSheet = reassignTarget != nil }
    }
    var reassignRecommendations: RecommendReassignResponse?
    var isReassigning = false
    var reassignMessage: String? {
        didSet { showReassignMessageAlert = reassignMessage != nil }
    }
    var showReassignSheet = false {
        didSet {
            if !showReassignSheet && reassignTarget != nil {
                reassignTarget = nil
            }
        }
    }
    var showReassignMessageAlert = false {
        didSet {
            if !showReassignMessageAlert && reassignMessage != nil {
                reassignMessage = nil
            }
        }
    }
    var notice: String?

    private var loadGeneration = 0

    let filters: [(id: String, label: String)] = [
        ("ACTIVE", "portal.page.orders.filter.active_tab"),
        ("SCHEDULED", "portal.page.orders.filter.scheduled"),
        ("COMPLETED", "portal.page.orders.filter.completed"),
        ("CANCELLED", "portal.page.orders.filter.cancelled"),
    ]

    var loadIdentity: String { "\(commandStatus ?? "")|\(statusFilter)" }

    func applyCommandStatus(_ status: String) {
        let key = canonicalizeOrderStatus(status)
        guard orderStatusFunnel.contains(key) else { return }
        commandStatus = key
    }

    func setCoarseFilter(_ id: String) {
        commandStatus = nil
        statusFilter = id
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        if !silent {
            loading = true
            error = nil
        }
        let offline = !NetworkLifecycleEngine.shared.isOnline
        do {
            let query = resolveSupplierOrdersQuery(commandStatus: commandStatus, coarseFilter: statusFilter)
            let response: SupplierOrdersResponse
            if let status = query.status {
                response = try await SupplierService.orders(status: status, limit: 500)
            } else {
                response = try await SupplierService.orders(filter: query.filter, limit: 500)
            }
            guard generation == loadGeneration else { return }
            orders = response.orders
            error = nil
            notice = nil
            if let selection, !orders.contains(where: { $0.orderId == selection.orderId }) {
                self.selection = orders.first
            } else if selection == nil {
                selection = orders.first
            }
        } catch {
            guard generation == loadGeneration else { return }
            let outcome = OrdersLoadFailure.message(
                offline: offline,
                silent: silent,
                hasOrders: !orders.isEmpty,
                underlying: error.localizedDescription
            )
            self.error = outcome.error
            if let outcomeNotice = outcome.notice {
                notice = outcomeNotice
            }
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func canWarehouseOps(for order: SupplierOrder) -> Bool {
        guard let warehouseId = order.warehouseId, !warehouseId.isEmpty else { return false }
        return statusFilter == "ACTIVE" || statusFilter == "SCHEDULED"
    }

    func proposeWarehouseOrder(_ order: SupplierOrder, proposedDeliveryDate: String, reason: String) async {
        guard let warehouseId = order.warehouseId, !warehouseId.isEmpty else { return }
        vettingOrderId = order.orderId
        defer { vettingOrderId = nil }
        do {
            _ = try await SupplierOperationsService.proposeWarehouseOrder(
                orderId: order.orderId,
                warehouseId: warehouseId,
                proposedDeliveryDate: proposedDeliveryDate,
                reason: reason,
                idempotencyKey: "warehouse-order-propose-delivery:\(order.orderId):\(proposedDeliveryDate):\(reason)"
            )
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !orders.isEmpty, error: &self.error)
        }
    }

    func rejectWarehouseOrder(_ order: SupplierOrder, reason: String) async {
        guard let warehouseId = order.warehouseId, !warehouseId.isEmpty else { return }
        vettingOrderId = order.orderId
        defer { vettingOrderId = nil }
        do {
            _ = try await SupplierOperationsService.rejectWarehouseOrder(
                orderId: order.orderId,
                warehouseId: warehouseId,
                reason: reason,
                idempotencyKey: "warehouse-order-reject:\(order.orderId):\(reason)"
            )
            await load(silent: true)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !orders.isEmpty, error: &self.error)
        }
    }

    func openReassignDialog(orderId: String) async {
        reassignTarget = orderId
        reassignRecommendations = nil
        reassignMessage = nil
        do {
            reassignRecommendations = try await SupplierOperationsService.recommendReassign(orderId: orderId)
        } catch {
            reassignMessage = L10n.format("mobile_supplier.chrome.reassign_recommendations_failed", error.localizedDescription)
            reassignTarget = nil
        }
    }

    func issuePaymentBypass(orderId: String, reason: String = "OPERATIONS_BYPASS") async {
        do {
            let req = PaymentBypassRequest(orderId: orderId, reason: reason)
            let idempotency = "payment-bypass:\(orderId):\(reason)"
            let res = try await SupplierOperationsService.issuePaymentBypass(req, idempotencyKey: idempotency)
            if !res.bypassToken.isEmpty {
                notice = L10n.format("mobile_supplier.chrome.payment_bypass_token_generated", res.bypassToken)
            } else {
                notice = L10n.format("mobile_supplier.chrome.payment_bypass_no_token")
            }
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !orders.isEmpty, error: &self.error)
        }
    }

    func closeReassignDialog() {
        if !isReassigning {
            reassignTarget = nil
            reassignRecommendations = nil
        }
    }

    func applyReassign(orderId: String, driverId: String, isPartial: Bool) async {
        isReassigning = true
        reassignMessage = nil
        do {
            try await SupplierOperationsService.applyReassign(
                orderId: orderId,
                driverId: driverId,
                isPartial: isPartial,
                idempotencyKey: "order-reassign:\(orderId):\(driverId):\(isPartial)"
            )
            reassignMessage = isPartial
                ? L10n.format("mobile_supplier.chrome.reassigned_partial")
                : L10n.format("mobile_supplier.chrome.reassigned_full")
            reassignTarget = nil
            await load(silent: true)
        } catch {
            reassignMessage = L10n.format("mobile_supplier.chrome.reassign_failed_with_error", error.localizedDescription)
        }
        isReassigning = false
    }
}

enum OrdersLoadFailure {
    static func message(
        offline: Bool,
        silent: Bool,
        hasOrders: Bool,
        underlying: String,
        noun: String = "orders"
    ) -> (error: String?, notice: String?) {
        if hasOrders {
            let notice = offline
                ? L10n.format("mobile_supplier.chrome.offline_showing_last", noun)
                : underlying
            return (nil, notice)
        }
        if offline {
            return (L10n.format("mobile_supplier.chrome.offline_connect", noun), nil)
        }
        return (underlying, nil)
    }
}

func resolveSupplierOrdersQuery(commandStatus: String?, coarseFilter: String) -> (status: String?, filter: String?) {
    if let commandStatus, !commandStatus.isEmpty {
        let key = canonicalizeOrderStatus(commandStatus)
        if orderStatusFunnel.contains(key) {
            return (key, nil)
        }
    }
    if coarseFilter == "SCHEDULED" {
        return ("SCHEDULED", nil)
    }
    return (nil, coarseFilter)
}
