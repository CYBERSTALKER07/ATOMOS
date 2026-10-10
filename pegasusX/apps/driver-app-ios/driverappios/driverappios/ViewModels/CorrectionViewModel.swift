//
//  CorrectionViewModel.swift
//  driverappios
//

import CoreLocation
import SwiftUI
import UIKit

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
    var evidencePhotoURL = ""
    var isUploadingPhoto = false
    var previewImage: UIImage?

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

    var needsPhotoProof: Bool {
        lineItems.contains { item in
            guard item.status == .REJECTED_DAMAGED else { return false }
            let r = reason(for: item.id)
            return r == .DAMAGED || r == .WRONG_ITEM
        }
    }

    // MARK: - Actions

    func uploadEvidence(image: UIImage, orderId: String) async {
        isUploadingPhoto = true
        submitError = nil
        defer { isUploadingPhoto = false }
        do {
            previewImage = image
            evidencePhotoURL = try await MediaUploadService.uploadJPEG(
                image: image,
                purpose: "driver_exception",
                orderId: orderId
            )
        } catch {
            evidencePhotoURL = ""
            submitError = "Photo upload failed: \(error.localizedDescription)"
        }
    }

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

    func startTransitForPartialOrder(orderId: String) async -> Bool {
        isSubmitting = true
        defer { isSubmitting = false }
        do {
            try await fleetService.reassignHandshake(orderId: orderId)
            Haptics.success()
            return true
        } catch {
            submitError = error.localizedDescription
            Haptics.error()
            return false
        }
    }

    func submitAmendment(orderId: String, driverId: String) async -> Bool {
        isSubmitting = true
        defer { isSubmitting = false }
        do {
            if needsPhotoProof && evidencePhotoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                submitError = "Photo required for damaged or wrong-item rejections."
                Haptics.error()
                return false
            }

            // G1.C: skip mid-delivery update (no durable writer). Missing-items / amend is SoT.

            let missingItems: [MissingItemRequest] = lineItems.compactMap { item in
                guard item.status == .REJECTED_DAMAGED else { return nil }
                let r = reason(for: item.id)
                let needsLinePhoto = r == .DAMAGED || r == .WRONG_ITEM
                return MissingItemRequest(
                    skuId: item.sku_id,
                    missingQty: item.quantity,
                    reason: r.rawValue,
                    photoURL: needsLinePhoto ? evidencePhotoURL : nil
                )
            }
            if missingItems.isEmpty {
                submitError = "No rejected items."
                Haptics.error()
                return false
            }
            _ = try await APIClient.shared.reportMissingItems(
                orderId: orderId,
                missingItems: missingItems,
                photoURL: evidencePhotoURL.isEmpty ? nil : evidencePhotoURL
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
