import Foundation
import Observation

@Observable
@MainActor
final class HistoryViewModel {
    var orders: [Order] = []
    var isLoading = false
    var loadError = false
    var filterStatus: OrderStatus?

    private let api = APIClient.shared

    var filteredOrders: [Order] {
        guard let filter = filterStatus else { return orders }
        return orders.filter { $0.status == filter }
    }

    func loadOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        loadError = false
        do {
            let r: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            orders = r
        } catch {
            orders = []
            loadError = true
        }
        isLoading = false
    }
}
