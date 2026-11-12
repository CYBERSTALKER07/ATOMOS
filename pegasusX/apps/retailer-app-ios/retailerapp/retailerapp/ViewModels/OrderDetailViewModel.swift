import Foundation
import Observation

@Observable
@MainActor
final class OrderDetailViewModel {
    var showCancelConfirm = false
    var isCancelling = false
    var cancelError = false
    var showFileClaim = false
    var claimElig: ClaimEligibility?

    var timelineItems: [OrderTimelineEntry] = []
    var isTimelineLoading = true

    private let api = APIClient.shared

    func isClaimCandidate(_ order: Order) -> Bool {
        order.status == .completed || order.status == .deliveredOnCredit
    }

    func loadClaimEligibility(for order: Order) async {
        guard isClaimCandidate(order) else {
            claimElig = nil
            return
        }
        claimElig = try? await api.getClaimEligibility(orderId: order.id)
    }

    func cancelOrder(_ order: Order) async -> Bool {
        isCancelling = true
        defer { isCancelling = false }
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        let requestCancelStates: Set<OrderStatus> = [.dispatched, .inTransit, .arrived]
        let useRequestCancel = requestCancelStates.contains(order.status)
        do {
            if useRequestCancel {
                try await api.requestCancelOrder(orderId: order.id, retailerId: retailerId)
            } else {
                let _: [String: String] = try await api.post(
                    path: "/v1/order/cancel",
                    body: [
                        "order_id": order.id,
                        "retailer_id": retailerId,
                    ],
                    headers: ["Idempotency-Key": RetailerIdempotency.cancel(orderId: order.id)]
                )
            }
            Haptics.success()
            return true
        } catch {
            Haptics.error()
            cancelError = true
            return false
        }
    }

    func loadTimeline(orderId: String) async {
        isTimelineLoading = true
        defer { isTimelineLoading = false }
        do {
            let response = try await api.getOrderTimeline(orderId: orderId)
            timelineItems = response.items
        } catch {
            timelineItems = []
        }
    }
}
