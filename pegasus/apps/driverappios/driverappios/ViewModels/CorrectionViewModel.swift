//
//  CorrectionViewModel.swift
//  driverappios
//

import SwiftUI

@Observable
@MainActor
final class CorrectionViewModel {

    // MARK: - State

    var lineItems: [LineItem] = []
    var rejectionReasons: [String: RejectionReason] = [:]
    var isLoading = false
    var showConfirmation = false
    var isSubmitting = false
    var submitError: String?

    private let fleetService: FleetServiceProtocol

    // MARK: - Init

    convenience init() {
        self.init(fleetService: FleetServiceLive.shared)
    }

    init(fleetService: FleetServiceProtocol) {
        self.fleetService = fleetService
    }

    // MARK: - Computed

    var rejectedCount: Int {
        lineItems.filter { $0.status == .REJECTED_DAMAGED }.count
    }

    var originalTotal: Int {
        lineItems.reduce(0) { $0 + $1.lineTotal }
    }

    var adjustedTotal: Int {
        lineItems.filter { $0.status == .DELIVERED }.reduce(0) { $0 + $1.lineTotal }
    }

    var refundDelta: Int {
        originalTotal - adjustedTotal
    }

    var hasRejections: Bool { rejectedCount > 0 }

    // MARK: - Actions

    func loadLineItems(orderId: String) async {
        isLoading = true
        defer { isLoading = false }
        do {
            lineItems = try await fleetService.fetchOrderLineItems(orderId: orderId)
        } catch {
            lineItems = []
        }
    }

    func toggleStatus(for itemId: String) {
        guard let index = lineItems.firstIndex(where: { $0.id == itemId }) else { return }
        Haptics.selectionChanged()
        let isRejecting = lineItems[index].status == .DELIVERED
        lineItems[index].status = isRejecting ? .REJECTED_DAMAGED : .DELIVERED
        if isRejecting {
            rejectionReasons[itemId] = rejectionReasons[itemId] ?? .DAMAGED
        } else {
            rejectionReasons.removeValue(forKey: itemId)
        }
    }

    func reason(for itemId: String) -> RejectionReason {
        rejectionReasons[itemId] ?? .DAMAGED
    }

    func setReason(_ reason: RejectionReason, for itemId: String) {
        rejectionReasons[itemId] = reason
    }

    func submitAmendment(orderId: String, driverId: String) async -> Bool {
        isSubmitting = true
        defer { isSubmitting = false }
        do {
            let items = lineItems.map {
                (
                    lineItemId: $0.id,
                    rejectedQty: $0.status == .REJECTED_DAMAGED ? $0.quantity : 0,
                    status: $0.status,
                    reason: $0.status == .REJECTED_DAMAGED ? reason(for: $0.id).rawValue : ""
                )
            }
            try await fleetService.amendOrder(
                orderId: orderId,
                driverId: driverId,
                items: items
            )
            Haptics.success()
            return true
        } catch {
            submitError = error.localizedDescription
            Haptics.error()
            return false
        }
    }
}
