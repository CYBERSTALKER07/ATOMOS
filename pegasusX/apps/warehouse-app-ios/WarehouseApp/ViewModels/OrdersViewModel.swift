import Foundation
import Observation

struct ProposeSheetOrder: Identifiable, Equatable, Sendable {
    let id: String
}

@Observable
@MainActor
final class OrdersViewModel {
    typealias FetchOrders = @MainActor (String?) async throws -> OrderListResponse
    typealias FetchPreorders = @MainActor () async throws -> WarehousePreordersResponse
    typealias ProposeOrderDelivery = @MainActor (String, String, String) async throws -> WarehouseOrderMutationResponse
    typealias ProposePreorderDelivery = @MainActor (String, String, String) async throws -> WarehouseOrderMutationResponse
    typealias RejectOrder = @MainActor (String, String) async throws -> WarehouseOrderMutationResponse
    typealias RejectPreorder = @MainActor (String, String) async throws -> WarehouseOrderMutationResponse

    let states = ["ALL", "PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "COMPLETED", "CANCELLED"]

    var hubTab: OrdersHubTab = .active
    var orders: [Order] = []
    var preorders: [WarehousePreorderRow] = []
    var loading = true
    var error: String?
    var selectedState = "ALL"
    var proposeTarget: String?
    var proposeDate = Date()
    var rejectTarget: String?
    var reasonInput = ""
    var statusMessage: String?

    private let fetchOrders: FetchOrders
    private let fetchPreorders: FetchPreorders
    private let proposeOrderDeliveryOp: ProposeOrderDelivery
    private let proposePreorderDeliveryOp: ProposePreorderDelivery
    private let rejectOrderOp: RejectOrder
    private let rejectPreorderOp: RejectPreorder

    init(
        initialState: String? = nil,
        fetchOrders: FetchOrders? = nil,
        fetchPreorders: FetchPreorders? = nil,
        proposeOrderDelivery: ProposeOrderDelivery? = nil,
        proposePreorderDelivery: ProposePreorderDelivery? = nil,
        rejectOrder: RejectOrder? = nil,
        rejectPreorder: RejectPreorder? = nil
    ) {
        if let initialState, !initialState.isEmpty {
            self.selectedState = initialState
        }
        self.fetchOrders = fetchOrders ?? { state in
            try await WarehouseService.orders(state: state)
        }
        self.fetchPreorders = fetchPreorders ?? {
            try await WarehouseService.preorders()
        }
        self.proposeOrderDeliveryOp = proposeOrderDelivery ?? { orderId, proposedDate, reason in
            try await WarehouseOperationsService.proposeOrderDelivery(
                orderId: orderId,
                proposedDeliveryDate: proposedDate,
                reason: reason
            )
        }
        self.proposePreorderDeliveryOp = proposePreorderDelivery ?? { orderId, proposedDate, reason in
            try await WarehouseOperationsService.proposePreorderDelivery(
                orderId: orderId,
                proposedDeliveryDate: proposedDate,
                reason: reason
            )
        }
        self.rejectOrderOp = rejectOrder ?? { orderId, reason in
            try await WarehouseOperationsService.rejectOrder(orderId: orderId, reason: reason)
        }
        self.rejectPreorderOp = rejectPreorder ?? { orderId, reason in
            try await WarehouseOperationsService.rejectPreorder(orderId: orderId, reason: reason)
        }
    }

    var filterStates: [String] {
        states.contains(selectedState) ? states : states + [selectedState]
    }

    var trimmedReason: String {
        reasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    var proposeSheetOrder: ProposeSheetOrder? {
        get { proposeTarget.map { ProposeSheetOrder(id: $0) } }
        set { proposeTarget = newValue?.id }
    }

    var showRejectAlert: Bool {
        get { rejectTarget != nil }
        set {
            if !newValue {
                rejectTarget = nil
                reasonInput = ""
            }
        }
    }

    func applyInitialState(_ state: String?) {
        if let state, !state.isEmpty, selectedState != state {
            selectedState = state
        }
    }

    func beginPropose(orderId: String) {
        PegasusHaptics.light()
        proposeTarget = orderId
        proposeDate = Date()
        reasonInput = ""
    }

    func beginReject(orderId: String) {
        PegasusHaptics.warning()
        rejectTarget = orderId
        reasonInput = ""
    }

    func cancelPropose() {
        proposeTarget = nil
        reasonInput = ""
    }

    func cancelReject() {
        rejectTarget = nil
        reasonInput = ""
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            if hubTab == .active {
                let state = selectedState == "ALL" ? nil : selectedState
                let resp = try await fetchOrders(state)
                orders = resp.orders
            } else {
                let resp = try await fetchPreorders()
                preorders = resp.items.isEmpty ? resp.preorders : resp.items
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func submitPropose(orderId: String) async {
        let reason = trimmedReason
        guard !reason.isEmpty else { return }
        let iso = Self.isoDeliveryDate(from: proposeDate)
        do {
            let isPreorder = preorders.contains { $0.orderId == orderId }
            if isPreorder {
                _ = try await proposePreorderDeliveryOp(orderId, iso, reason)
            } else {
                _ = try await proposeOrderDeliveryOp(orderId, iso, reason)
            }
            statusMessage = "New delivery date proposed · retailer notified"
            proposeTarget = nil
            reasonInput = ""
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func submitReject() async {
        let reason = trimmedReason
        guard let orderId = rejectTarget, !reason.isEmpty else { return }
        do {
            let isPreorder = preorders.contains { $0.orderId == orderId }
            if isPreorder {
                _ = try await rejectPreorderOp(orderId, reason)
            } else {
                _ = try await rejectOrderOp(orderId, reason)
            }
            statusMessage = "Order cancelled · retailer notified"
            rejectTarget = nil
            reasonInput = ""
            PegasusHaptics.success()
            await load()
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func autoClearStatusMessage() async {
        guard statusMessage != nil else { return }
        try? await Task.sleep(nanoseconds: 2_000_000_000)
        guard !Task.isCancelled else { return }
        statusMessage = nil
    }

    static let tashkentIsoFractionalFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        return formatter
    }()

    static func isoDeliveryDate(from date: Date) -> String {
        let tz = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
        var components = Calendar.current.dateComponents(in: tz, from: date)
        components.hour = 12
        components.minute = 0
        components.second = 0
        let noon = Calendar.current.date(from: components) ?? date
        return tashkentIsoFractionalFormatter.string(from: noon)
    }
}
