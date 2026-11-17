//
//  OffloadReviewViewModel.swift
//  driverappios
//

import Foundation
import Observation
import PhotosUI
import SwiftUI
import UIKit

@Observable
@MainActor
final class OffloadReviewViewModel {
    let response: ValidateQRResponse
    let scannedToken: String

    var rejectedQty: [String: Int] = [:]
    var rejectionReasons: [String: RejectionReason] = [:]
    var customReasons: [String: String] = [:]
    var isSubmitting = false
    var errorMessage: String?
    var pickedPhoto: PhotosPickerItem?
    var previewImage: UIImage?
    var evidencePhotoURL = ""
    var evidencePhotoLocalPath = ""
    var signatureURL = ""
    var signatureLocalPath = ""
    var isUploadingPhoto = false

    private let fleetService: FleetServiceProtocol
    private let reportMissing: @Sendable (String, [MissingItemRequest], String?) async throws -> Void
    private let uploadPhotoJPEG: @Sendable (UIImage, String, String) async throws -> String
    private let uploadSigJPEG: @Sendable (Data, String, String) async throws -> String

    init(
        response: ValidateQRResponse,
        scannedToken: String,
        fleetService: FleetServiceProtocol = FleetServiceLive.shared,
        reportMissing: @escaping @Sendable (String, [MissingItemRequest], String?) async throws -> Void = { orderId, items, photoURL in
            _ = try await APIClient.shared.reportMissingItems(
                orderId: orderId,
                missingItems: items,
                photoURL: photoURL
            )
        },
        uploadPhotoJPEG: @escaping @Sendable (UIImage, String, String) async throws -> String = { image, purpose, orderId in
            try await MediaUploadService.uploadJPEG(image: image, purpose: purpose, orderId: orderId)
        },
        uploadSigJPEG: @escaping @Sendable (Data, String, String) async throws -> String = { data, purpose, orderId in
            try await MediaUploadService.uploadJPEGData(data, purpose: purpose, orderId: orderId)
        }
    ) {
        self.response = response
        self.scannedToken = scannedToken
        self.fleetService = fleetService
        self.reportMissing = reportMissing
        self.uploadPhotoJPEG = uploadPhotoJPEG
        self.uploadSigJPEG = uploadSigJPEG
    }

    var hasRejections: Bool {
        rejectedQty.values.contains { $0 > 0 }
    }

    var needsPhotoProof: Bool {
        response.items.contains { item in
            let rejected = rejectedQty[item.id] ?? 0
            guard rejected > 0 else { return false }
            let reason = selectedReason(for: item)
            return reason == .DAMAGED || reason == .WRONG_ITEM
        }
    }

    func selectedReason(for item: OrderLineItem) -> RejectionReason {
        rejectionReasons[item.id] ?? .DAMAGED
    }

    func reasonLabel(for reason: RejectionReason) -> String {
        reason.rawValue.replacingOccurrences(of: "_", with: " ").capitalized
    }

    func decrementRejection(for item: OrderLineItem) {
        let current = rejectedQty[item.id] ?? 0
        guard current > 0 else { return }
        Haptics.light()
        let nextRejected = current - 1
        rejectedQty[item.id] = nextRejected
        if nextRejected == 0 {
            rejectionReasons.removeValue(forKey: item.id)
            customReasons.removeValue(forKey: item.id)
        }
    }

    func incrementRejection(for item: OrderLineItem) {
        let current = rejectedQty[item.id] ?? 0
        guard current < item.quantity else { return }
        Haptics.light()
        rejectedQty[item.id] = current + 1
        if rejectionReasons[item.id] == nil {
            rejectionReasons[item.id] = .DAMAGED
        }
    }

    func setReason(_ reason: RejectionReason, for itemId: String) {
        Haptics.selection()
        rejectionReasons[itemId] = reason
    }

    func setCustomReason(_ text: String, for itemId: String) {
        if customReasons[itemId] != text {
            customReasons[itemId] = text
        }
    }

    func validateCreditLeave() -> (photoURL: String, signatureURL: String, photoLocal: String?, sigLocal: String?)? {
        let hasPhoto = !evidencePhotoURL.isEmpty || !evidencePhotoLocalPath.isEmpty
        let hasSig = !signatureURL.isEmpty || !signatureLocalPath.isEmpty
        guard hasPhoto else {
            Haptics.warning()
            errorMessage = "PoD photo required for credit leave — take a photo of the handoff first."
            return nil
        }
        guard hasSig else {
            Haptics.warning()
            errorMessage = "Signature required for credit leave — sign the pad and tap Save."
            return nil
        }
        return (
            evidencePhotoURL,
            signatureURL,
            evidencePhotoLocalPath.isEmpty ? nil : evidencePhotoLocalPath,
            signatureLocalPath.isEmpty ? nil : signatureLocalPath
        )
    }

    func uploadPickedPhoto(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        isUploadingPhoto = true
        errorMessage = nil
        defer { isUploadingPhoto = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data) else {
                errorMessage = "Could not read photo."
                return
            }
            await processSelectedImage(image)
        } catch {
            evidencePhotoURL = ""
            errorMessage = "Photo upload failed: \(error.localizedDescription)"
        }
    }

    func processSelectedImage(_ image: UIImage) async {
        previewImage = image
        if let jpeg = image.jpegData(compressionQuality: 0.82) {
            evidencePhotoLocalPath = (try? MediaUploadService.savePodJPEG(jpeg, prefix: "photo")) ?? ""
        }
        do {
            evidencePhotoURL = try await uploadPhotoJPEG(image, "credit_proof", response.orderId)
        } catch {
            evidencePhotoURL = ""
            if evidencePhotoLocalPath.isEmpty {
                errorMessage = "Photo upload failed: \(error.localizedDescription)"
            }
        }
    }

    func saveSignature(_ data: Data) async {
        errorMessage = nil
        signatureLocalPath = (try? MediaUploadService.savePodJPEG(data, prefix: "sig")) ?? ""
        do {
            signatureURL = try await uploadSigJPEG(data, "credit_proof", response.orderId)
        } catch {
            signatureURL = ""
            if signatureLocalPath.isEmpty {
                errorMessage = "Signature upload failed: \(error.localizedDescription)"
            }
        }
    }

    func confirmOffload() async -> ConfirmOffloadResponse? {
        isSubmitting = true
        errorMessage = nil
        defer { isSubmitting = false }

        if hasRejections {
            let missingOther = response.items.first { item in
                let rejected = rejectedQty[item.id] ?? 0
                return rejected > 0
                    && selectedReason(for: item) == .OTHER
                    && (customReasons[item.id] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            }
            if let missingOther {
                Haptics.warning()
                errorMessage = "Describe the issue for \(missingOther.productName)"
                return nil
            }
            if needsPhotoProof && evidencePhotoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                Haptics.warning()
                errorMessage = "Photo required for damaged or wrong-item rejections."
                return nil
            }

            let missingItems: [MissingItemRequest] = response.items.compactMap { item in
                let rejected = rejectedQty[item.id] ?? 0
                guard rejected > 0 else { return nil }
                let reason = selectedReason(for: item)
                let needsLinePhoto = reason == .DAMAGED || reason == .WRONG_ITEM
                return MissingItemRequest(
                    skuId: item.productId,
                    missingQty: rejected,
                    reason: reason.rawValue,
                    photoURL: needsLinePhoto ? evidencePhotoURL : nil
                )
            }
            do {
                try await reportMissing(
                    response.orderId,
                    missingItems,
                    evidencePhotoURL.isEmpty ? nil : evidencePhotoURL
                )
            } catch {
                Haptics.error()
                errorMessage = "Exception report failed: \(error.localizedDescription)"
                return nil
            }
        }

        do {
            let adjustedAmount = response.items.reduce(0) { partial, item in
                let rejected = rejectedQty[item.id] ?? 0
                let accepted = max(0, item.quantity - rejected)
                return partial + (item.unitPrice * accepted)
            }
            let result: ConfirmOffloadResponse
            if !scannedToken.isEmpty {
                let scan = try await fleetService.scanDeliveryQR(
                    orderId: response.orderId,
                    qrToken: scannedToken
                )
                result = ConfirmOffloadResponse(
                    orderId: scan.orderId.isEmpty ? response.orderId : scan.orderId,
                    state: scan.state,
                    paymentMethod: "",
                    amount: adjustedAmount,
                    invoiceId: nil,
                    retailerId: "",
                    message: "Collect payment"
                )
            } else {
                result = try await fleetService.confirmOffload(orderId: response.orderId)
            }
            Haptics.success()
            return result
        } catch {
            Haptics.error()
            errorMessage = error.localizedDescription
            return nil
        }
    }

    func handleReconnect() async {
        let hadInFlight = isSubmitting
        await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
        if hadInFlight {
            isSubmitting = false
            errorMessage = DriverReconnectRecovery.hint
        }
    }
}
