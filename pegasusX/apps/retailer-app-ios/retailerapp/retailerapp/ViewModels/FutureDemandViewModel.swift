import Foundation
import Observation

@Observable
@MainActor
final class FutureDemandViewModel {
    var items: [RetailerAIPrediction] = []
    var isLoading = false
    var actingId: String?

    private let api = APIClient.shared

    var totalUnits: Int64 {
        items.reduce(0) { $0 + $1.quantity }
    }

    func loadPredictions() async {
        isLoading = true
        do {
            items = try await api.getRetailerAIPredictions()
        } catch {
            items = []
        }
        isLoading = false
    }

    func confirm(_ item: RetailerAIPrediction) async {
        actingId = item.orderId
        do {
            try await api.confirmAiOrder(orderId: item.orderId)
            Haptics.success()
            await loadPredictions()
        } catch {
            Haptics.error()
        }
        actingId = nil
    }

    func reject(_ item: RetailerAIPrediction) async {
        actingId = item.orderId
        do {
            try await api.rejectAiOrder(orderId: item.orderId, reason: "Retailer rejected")
            Haptics.success()
            await loadPredictions()
        } catch {
            Haptics.error()
        }
        actingId = nil
    }
}
