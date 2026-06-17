import Foundation

@Observable
@MainActor
final class OrdersViewModel {
    var orders: [SupplierOrder] = []
    var loading = true
    var error: String?
    var statusFilter = "PENDING"
    var selection: SupplierOrder?
    var vettingOrderId: String?

    let filters: [(id: String, label: String)] = [
        ("", "All"),
        ("PENDING", "Pending"),
        ("AWAITING_REVIEW", "Review"),
        ("COMPLETED", "Completed"),
    ]

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let response = try await SupplierService.orders(
                status: statusFilter.isEmpty ? nil : statusFilter,
                limit: 500
            )
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
}
