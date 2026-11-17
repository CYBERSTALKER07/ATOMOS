import Foundation
import Observation
import SwiftData
import UserNotifications

struct PendingOrderStatusNotice: Equatable, Sendable {
    let endpoint: String
    let idempotencyKey: String
    let status: String
    let message: String
    let disposition: PendingOrderQueueDisposition
}

struct PendingOrderFlushResult: Sendable {
    let completedCount: Int
    let parkedCount: Int
    let retriedCount: Int
    let purgedRejectedCount: Int
    let notices: [PendingOrderStatusNotice]
}

@Observable
@MainActor
final class PendingOrderStatusCenter {
    static let shared = PendingOrderStatusCenter()
    var latestNotice: PendingOrderStatusNotice?
    var recentNotices: [PendingOrderStatusNotice] = []

    func publish(_ notice: PendingOrderStatusNotice) {
        latestNotice = notice
        recentNotices.append(notice)
        if recentNotices.count > 20 {
            recentNotices.removeFirst(recentNotices.count - 20)
        }
    }
}

/// Drains SwiftData pending checkout/procurement mutations after reconnect.
enum PendingOrderReplayer {
    typealias Dispatcher = @Sendable (_ endpoint: String, _ payloadData: Data, _ idempotencyKey: String) async throws -> Void
    typealias StatusNotifier = @MainActor (_ title: String, _ body: String, _ identifier: String) -> Void

    @MainActor
    @discardableResult
    static func flush(
        modelContext: ModelContext,
        api: APIClient? = nil,
        dispatcher: Dispatcher? = nil,
        notifier: StatusNotifier? = nil
    ) async -> PendingOrderFlushResult {
        let resolvedApi = api ?? .shared
        let descriptor = FetchDescriptor<PendingOrder>(sortBy: [SortDescriptor(\.createdAt)])
        guard let pending = try? modelContext.fetch(descriptor), !pending.isEmpty else {
            return PendingOrderFlushResult(
                completedCount: 0,
                parkedCount: 0,
                retriedCount: 0,
                purgedRejectedCount: 0,
                notices: []
            )
        }

        let effectiveDispatcher: Dispatcher = dispatcher ?? { endpoint, data, idempotencyKey in
            try await dispatchDefault(endpoint: endpoint, data: data, idempotencyKey: idempotencyKey, api: resolvedApi)
        }
        let effectiveNotifier = notifier ?? defaultNotifier

        var completedCount = 0
        var parkedCount = 0
        var retriedCount = 0
        var purgedRejectedCount = 0
        var notices: [PendingOrderStatusNotice] = []

        for order in pending {
            guard order.method.uppercased() == "POST" else {
                order.status = PendingOrder.statusFailed
                order.lastError = "Unsupported pending mutation \(order.method) \(order.endpoint)"
                order.retryCount += 1
                retriedCount += 1
                continue
            }
            guard let data = order.payloadJson.data(using: .utf8), !order.payloadJson.isEmpty else {
                modelContext.delete(order)
                purgedRejectedCount += 1
                continue
            }
            let key = pendingOrderIdempotencyKey(order)
            do {
                try await effectiveDispatcher(order.endpoint, data, key)
                modelContext.delete(order)
                completedCount += 1
            } catch {
                let assessment = RetailerOrderContractHelper.assessPendingOrderError(error)
                switch assessment.disposition {
                case .purgeIdempotentDuplicate:
                    modelContext.delete(order)
                    completedCount += 1
                case .purgeRejectedValidation:
                    modelContext.delete(order)
                    purgedRejectedCount += 1
                    let notice = PendingOrderStatusNotice(
                        endpoint: order.endpoint,
                        idempotencyKey: key,
                        status: assessment.queueStatus,
                        message: assessment.userMessage,
                        disposition: assessment.disposition
                    )
                    notices.append(notice)
                    PendingOrderStatusCenter.shared.publish(notice)
                    if assessment.shouldNotify {
                        effectiveNotifier("Order sync failed", assessment.userMessage, "pending-order-\(key)")
                    }
                case .parkAwaitingOwnerApproval, .parkRequestInProgress, .parkRateLimited, .parkAuthRequired:
                    order.status = assessment.queueStatus
                    order.lastError = assessment.userMessage
                    order.retryCount += 1
                    parkedCount += 1
                    let notice = PendingOrderStatusNotice(
                        endpoint: order.endpoint,
                        idempotencyKey: key,
                        status: assessment.queueStatus,
                        message: assessment.userMessage,
                        disposition: assessment.disposition
                    )
                    notices.append(notice)
                    PendingOrderStatusCenter.shared.publish(notice)
                    if assessment.shouldNotify {
                        let title: String
                        switch assessment.disposition {
                        case .parkAwaitingOwnerApproval:
                            title = "Owner approval required"
                        case .parkRateLimited:
                            title = "Sync delayed (429)"
                        case .parkAuthRequired:
                            title = "Sign-in required (401)"
                        default:
                            title = "Order submission in progress"
                        }
                        effectiveNotifier(title, assessment.userMessage, "pending-order-\(key)")
                    }
                case .retryTransient:
                    order.status = PendingOrder.statusPending
                    order.lastError = assessment.userMessage
                    order.retryCount += 1
                    retriedCount += 1
                }
            }
        }
        try? modelContext.save()
        return PendingOrderFlushResult(
            completedCount: completedCount,
            parkedCount: parkedCount,
            retriedCount: retriedCount,
            purgedRejectedCount: purgedRejectedCount,
            notices: notices
        )
    }

    private static func dispatchDefault(
        endpoint: String,
        data: Data,
        idempotencyKey: String,
        api: APIClient
    ) async throws {
        switch endpoint {
        case "/v1/checkout/unified":
            if let payload = try? JSONDecoder().decode(UnifiedCheckoutPayload.self, from: data) {
                _ = try await RetailerCheckoutService.completeCheckout(
                    api: api,
                    payload: payload,
                    gateway: payload.paymentGateway,
                    idempotencyKey: idempotencyKey
                )
            } else {
                _ = try await api.postRaw(
                    path: endpoint,
                    rawBody: data,
                    headers: ["Idempotency-Key": idempotencyKey]
                )
            }
        case "/v1/order/create", "/v1/ai/preorder":
            if let payload = try? JSONDecoder().decode(ProcurementOrderRequest.self, from: data) {
                let _: ProcurementOrderResponse = try await api.post(
                    path: endpoint,
                    body: payload,
                    headers: ["Idempotency-Key": idempotencyKey]
                )
            } else {
                _ = try await api.postRaw(
                    path: endpoint,
                    rawBody: data,
                    headers: ["Idempotency-Key": idempotencyKey]
                )
            }
        default:
            let supportedSecondaryEndpoints: Set<String> = [
                "/v1/orders/confirm-preorder",
                "/v1/orders/edit-preorder",
                "/v1/retailer/orders/confirm-ai",
            ]
            guard supportedSecondaryEndpoints.contains(endpoint) else {
                throw APIError.serverError(statusCode: 400, message: "Unsupported pending mutation POST \(endpoint)")
            }
            _ = try await api.postRaw(
                path: endpoint,
                rawBody: data,
                headers: ["Idempotency-Key": idempotencyKey]
            )
        }
    }

    @MainActor
    private static func defaultNotifier(title: String, body: String, identifier: String) {
        guard ProcessInfo.processInfo.environment["XCTestConfigurationFilePath"] == nil else { return }
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        content.sound = .default
        let request = UNNotificationRequest(identifier: identifier, content: content, trigger: nil)
        UNUserNotificationCenter.current().add(request)
    }

    static func pendingOrderIdempotencyKey(_ order: PendingOrder) -> String {
        if !order.idempotencyKey.isEmpty {
            return order.idempotencyKey
        }
        return "retailer-checkout-pending:\(Int64(order.createdAt.timeIntervalSince1970 * 1000))"
    }
}
