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
    var reassignTarget: String?
    var reassignRecommendations: RecommendReassignResponse?
    var isReassigning = false
    var reassignMessage: String?

    let filters: [(id: String, label: String)] = [
        ("ACTIVE", "Active"),
        ("SCHEDULED", "Scheduled"),
        ("COMPLETED", "Completed"),
        ("CANCELLED", "Cancelled"),
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
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let query = resolveSupplierOrdersQuery(commandStatus: commandStatus, coarseFilter: statusFilter)
            let response: SupplierOrdersResponse
            if let status = query.status {
                response = try await SupplierService.orders(status: status, limit: 500)
            } else {
                response = try await SupplierService.orders(filter: query.filter, limit: 500)
            }
            orders = response.orders
            if selection == nil { selection = orders.first }
        } catch {
            if !silent { self.error = error.localizedDescription }
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
                idempotencyKey: "warehouse-order-propose-delivery:\(order.orderId):\(reason.hashValue)"
            )
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
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
                idempotencyKey: "warehouse-order-reject:\(order.orderId):\(reason.hashValue)"
            )
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }

    func openReassignDialog(orderId: String) async {
        reassignTarget = orderId
        reassignRecommendations = nil
        reassignMessage = nil
        do {
            reassignRecommendations = try await SupplierOperationsService.recommendReassign(orderId: orderId)
        } catch {
            reassignMessage = "Failed to load recommendations: \(error.localizedDescription)"
            reassignTarget = nil
        }
    }

func issuePaymentBypass(orderId: String) async {
    mutating = true
    defer { mutating = false }
    do {
        let req = PaymentBypassRequest(orderId: orderId)
        let idempotency = UUID().uuidString
        let res = try await SupplierOperationsService.issuePaymentBypass(req, idempotencyKey: idempotency)
        if let token = res.bypassToken {
            opsError = "Token generated: \(token)"
        } else {
            opsError = "Bypass request successful, but no token returned."
        }
    } catch {
        opsError = error.localizedDescription
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
                idempotencyKey: UUID().uuidString
            )
            reassignMessage = isPartial ? "Reassigned (Partial)" : "Reassigned (Full)"
            reassignTarget = nil
            await load(silent: true)
        } catch {
            reassignMessage = "Failed to reassign: \(error.localizedDescription)"
        }
        isReassigning = false
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
