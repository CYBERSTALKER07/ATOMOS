import Foundation

@Observable
@MainActor
final class OrdersViewModel {
    var orders: [SupplierOrder] = []
    var loading = true
    var error: String?
    var statusFilter = "ACTIVE"
    var selection: SupplierOrder?
    var vettingOrderId: String?

    let filters: [(id: String, label: String)] = [
        ("ACTIVE", "Active"),
        ("AWAITING_REVIEW", "Review"),
        ("SCHEDULED", "Scheduled"),
        ("COMPLETED", "Completed"),
        ("CANCELLED", "Cancelled"),
    ]

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let response: SupplierOrdersResponse
            if statusFilter == "AWAITING_REVIEW" {
                response = try await SupplierService.orders(status: statusFilter, limit: 500)
            } else if statusFilter == "SCHEDULED" {
                response = try await SupplierService.orders(status: "SCHEDULED", limit: 500)
            } else {
                response = try await SupplierService.orders(filter: statusFilter, limit: 500)
            }
            orders = response.orders
            if selection == nil { selection = orders.first }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func vet(order: SupplierOrder, decision: String, note: String = "") async {
        vettingOrderId = order.orderId
        defer { vettingOrderId = nil }
        do {
            let body: [String: String] = [
                "order_id": order.orderId,
                "retailer_id": order.retailerId,
                "decision": decision,
                "note": note,
            ]
            try await SupplierOperationsService.vetOrder(
                body: body,
                idempotencyKey: "supplier-vet:\(order.orderId):\(decision)"
            )
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
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
}
