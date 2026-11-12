import Foundation
import Observation

enum OrderMutationAction: String, Identifiable {
    case reject
    case overflow
    case paymentBypass

    var id: String { rawValue }
}

@Observable
@MainActor
final class OrderDetailViewModel {
    typealias FetchOrder = @MainActor (String) async throws -> Order
    typealias ProposeDelivery = @MainActor (String, String, String) async throws -> WarehouseOrderMutationResponse
    typealias FetchReceiptMeta = @MainActor (String) async throws -> OrderReceiptMeta
    typealias IssuePaymentBypass = @MainActor (String) async throws -> PaymentBypassResponse
    typealias RejectOrder = @MainActor (String, String) async throws -> WarehouseOrderMutationResponse
    typealias OverflowOrder = @MainActor (String, String?) async throws -> WarehouseOrderMutationResponse
    typealias RecommendReassign = @MainActor (String) async throws -> RecommendReassignResponse
    typealias ReassignOrder = @MainActor (ReassignOrderRequest, String) async throws -> Void

    private(set) var orderId: String
    var order: Order?
    var loading = true
    var error: String?
    var mutating = false
    var pendingAction: OrderMutationAction?
    var reasonInput = ""
    var proposeDate = Date()
    var showProposeSheet = false
    var statusMessage: String?
    var recommendations: [TruckRecommendation] = []
    var showReassignSheet = false
    var selectedDriverId: String?
    var selectedManifestId: String?
    var isPartialReassign = false

    private let fetchOrder: FetchOrder
    private let proposeDeliveryOp: ProposeDelivery
    private let fetchReceiptMeta: FetchReceiptMeta
    private let issuePaymentBypassOp: IssuePaymentBypass
    private let rejectOrderOp: RejectOrder
    private let overflowOrderOp: OverflowOrder
    private let recommendReassignOp: RecommendReassign
    private let reassignOrderOp: ReassignOrder

    init(
        orderId: String,
        fetchOrder: FetchOrder? = nil,
        proposeDelivery: ProposeDelivery? = nil,
        fetchReceiptMeta: FetchReceiptMeta? = nil,
        issuePaymentBypass: IssuePaymentBypass? = nil,
        rejectOrder: RejectOrder? = nil,
        overflowOrder: OverflowOrder? = nil,
        recommendReassign: RecommendReassign? = nil,
        reassignOrder: ReassignOrder? = nil
    ) {
        self.orderId = orderId
        self.fetchOrder = fetchOrder ?? { id in
            try await WarehouseService.order(id: id)
        }
        self.proposeDeliveryOp = proposeDelivery ?? { id, proposedDate, reason in
            try await WarehouseOperationsService.proposeOrderDelivery(
                orderId: id,
                proposedDeliveryDate: proposedDate,
                reason: reason
            )
        }
        self.fetchReceiptMeta = fetchReceiptMeta ?? { id in
            try await APIClient.shared.get(
                "v1/warehouse/orders/\(id)/receipt",
                query: ["format": "json"]
            )
        }
        self.issuePaymentBypassOp = issuePaymentBypass ?? { id in
            try await WarehouseOperationsService.issuePaymentBypass(orderId: id)
        }
        self.rejectOrderOp = rejectOrder ?? { id, reason in
            try await WarehouseOperationsService.rejectOrder(orderId: id, reason: reason)
        }
        self.overflowOrderOp = overflowOrder ?? { id, reason in
            try await WarehouseOperationsService.overflowOrder(orderId: id, reason: reason)
        }
        self.recommendReassignOp = recommendReassign ?? { id in
            try await WarehouseOperationsService.recommendReassign(orderId: id)
        }
        self.reassignOrderOp = reassignOrder ?? { request, idempotencyKey in
            try await WarehouseOperationsService.reassignOrder(request, idempotencyKey: idempotencyKey)
        }
    }

    var trimmedReason: String {
        reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    func updateOrderIdIfNeeded(_ newOrderId: String) {
        if orderId != newOrderId {
            orderId = newOrderId
            order = nil
        }
    }

    func selectRecommendation(_ rec: TruckRecommendation) {
        PegasusHaptics.selection()
        selectedDriverId = rec.driverId
        selectedManifestId = rec.toRoute.isEmpty ? nil : rec.toRoute
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            order = try await fetchOrder(orderId)
        } catch {
            self.error = error.localizedDescription
        }
    }

    func submitPropose() async {
        let reason = trimmedReason
        guard !reason.isEmpty else {
            statusMessage = "Reason is required"
            PegasusHaptics.warning()
            return
        }
        mutating = true
        statusMessage = nil
        defer { mutating = false }
        do {
            let response = try await proposeDeliveryOp(
                orderId,
                Self.isoDeliveryDate(from: proposeDate),
                reason
            )
            statusMessage = "New delivery date proposed · retailer notified · \(response.status)"
            showProposeSheet = false
            reasonInput = ""
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func resolveReceiptURL() async -> URL? {
        do {
            let meta = try await fetchReceiptMeta(orderId)
            let raw = [meta.htmlUrl, meta.qrUrl, meta.pdfUrl]
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .first { !$0.isEmpty }
            if let raw, let url = URL(string: raw) {
                return url
            }
            statusMessage = "Receipt unavailable"
            PegasusHaptics.warning()
            return nil
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
            return nil
        }
    }

    func runMutation(_ action: OrderMutationAction) async {
        mutating = true
        statusMessage = nil
        defer { mutating = false }
        do {
            switch action {
            case .paymentBypass:
                let response = try await issuePaymentBypassOp(orderId)
                statusMessage = "Token: \(response.bypassToken ?? "Unknown")"
                PegasusHaptics.success()
                await load()
                return
            case .reject:
                let reason = trimmedReason
                guard !reason.isEmpty else {
                    statusMessage = "Reason is required to cancel"
                    PegasusHaptics.warning()
                    return
                }
                let response = try await rejectOrderOp(orderId, reason)
                statusMessage = "Order updated · \(response.status)"
            case .overflow:
                let response = try await overflowOrderOp(orderId, reasonInput.isEmpty ? nil : reasonInput)
                statusMessage = "Order updated · \(response.status)"
            }
            reasonInput = ""
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func loadRecommendations() async {
        mutating = true
        statusMessage = "Loading recommendations..."
        defer { mutating = false }
        do {
            let response = try await recommendReassignOp(orderId)
            recommendations = response.recommendations
            if recommendations.isEmpty {
                statusMessage = "No available vehicles found for reassignment"
                PegasusHaptics.warning()
            } else {
                statusMessage = nil
                showReassignSheet = true
            }
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func submitReassign() async {
        guard let driverId = selectedDriverId else { return }
        mutating = true
        statusMessage = "Reassigning..."
        defer { mutating = false }
        do {
            let request = ReassignOrderRequest(
                orderId: orderId,
                toDriverId: driverId,
                reason: trimmedReason.isEmpty ? "warehouse-reassign" : reasonInput,
                toManifestId: selectedManifestId,
                isPartial: isPartialReassign
            )
            try await reassignOrderOp(request, UUID().uuidString)
            statusMessage = "Order reassigned successfully"
            showReassignSheet = false
            reasonInput = ""
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func alertTitle(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Cancel order?"
        case .overflow: return "Return to dispatch pool?"
        case .paymentBypass: return "Issue Payment Bypass?"
        }
    }

    func reasonPrompt(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Enter a cancellation reason in the field above before confirming."
        case .overflow: return "Optional reason can be entered above."
        case .paymentBypass: return "Generate a token for the driver if the POS terminal failed."
        }
    }

    func confirmLabel(for action: OrderMutationAction) -> String {
        switch action {
        case .reject: return "Cancel order"
        case .overflow: return "Return to pool"
        case .paymentBypass: return "Issue Bypass Token"
        }
    }

    static let isoFractionalFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static func isoDeliveryDate(from date: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
        let components = calendar.dateComponents([.year, .month, .day], from: date)
        var merged = DateComponents()
        merged.year = components.year
        merged.month = components.month
        merged.day = components.day
        merged.hour = 12
        merged.minute = 0
        merged.second = 0
        merged.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        let normalized = calendar.date(from: merged) ?? date
        return isoFractionalFormatter.string(from: normalized)
    }
}
