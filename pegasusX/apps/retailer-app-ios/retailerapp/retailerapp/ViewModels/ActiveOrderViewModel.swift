import Foundation
import Observation

@Observable
@MainActor
final class ActiveOrderViewModel {
    var activeOrders: [Order] = []
    var isLoading = false
    var loadError = false

    private let api = APIClient.shared

    func loadActiveOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        loadError = false
        do {
            let r: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            activeOrders = r.filter { $0.status == .inTransit || $0.status == .dispatched || $0.status == .arrived }
        } catch {
            activeOrders = []
            loadError = true
        }
        isLoading = false
    }
}
