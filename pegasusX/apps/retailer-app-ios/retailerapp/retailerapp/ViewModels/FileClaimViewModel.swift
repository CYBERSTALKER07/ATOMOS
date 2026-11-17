import Foundation
import Observation
import PhotosUI
import SwiftUI
import UIKit

struct ClaimLineDraft: Identifiable, Hashable {
    let id: String
    let sku: String
    let productName: String
    let orderedQuantity: Int
    var claimQuantity: Int
}

@Observable
@MainActor
final class FileClaimViewModel {
    let order: Order
    let preferredSku: String?

    var claimType = "CONCEALED_DAMAGE"
    var descriptionText = ""
    var lineDrafts: [ClaimLineDraft] = []
    var photoURL = ""
    var pickedItem: PhotosPickerItem?
    var previewImage: UIImage?
    var isUploadingPhoto = false
    var isSubmitting = false
    var errorMessage: String?
    var skuWarning: String?
    var successClaimId: String?
    var existing: [RetailerClaim] = []
    var eligibility: ClaimEligibility?
    var eligLoading = true

    let claimTypes = ["CONCEALED_DAMAGE", "DAMAGED", "MISSING", "TAMPER", "TEMPERATURE", "OTHER"]

    private let api = APIClient.shared
    private static let isoFormatter = ISO8601DateFormatter()

    init(order: Order, preferredSku: String? = nil) {
        self.order = order
        self.preferredSku = preferredSku
        self.lineDrafts = order.items.map { item in
            let sku = item.productId.isEmpty ? item.variantId : item.productId
            return ClaimLineDraft(
                id: item.id,
                sku: sku,
                productName: item.productName,
                orderedQuantity: item.quantity,
                claimQuantity: 0
            )
        }
        applyPreferredSku()
    }

    var selectedQtyTotal: Int {
        lineDrafts.reduce(0) { $0 + $1.claimQuantity }
    }

    var canSubmit: Bool {
        !eligLoading && (eligibility?.eligible ?? true) && selectedQtyTotal > 0 && !isSubmitting && !isUploadingPhoto
    }

    func loadInitial() async {
        await loadExisting()
        await loadEligibility()
    }

    func loadEligibility() async {
        eligLoading = true
        defer { eligLoading = false }
        do {
            eligibility = try await api.getClaimEligibility(orderId: order.id)
        } catch {
            eligibility = nil
        }
    }

    func loadExisting() async {
        do {
            existing = try await api.listOrderClaims(orderId: order.id)
        } catch {
            existing = []
        }
    }

    func applyPreferredSku() {
        let sku = preferredSku?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !sku.isEmpty else {
            skuWarning = nil
            return
        }
        guard let idx = lineDrafts.firstIndex(where: { draft in
            draft.sku == sku || draft.id == sku
        }) ?? order.items.firstIndex(where: {
            $0.productId == sku || $0.id == sku || $0.variantId == sku
        }) else {
            skuWarning = "SKU \(sku) is not on this order — pick another line."
            return
        }
        if lineDrafts.indices.contains(idx) {
            lineDrafts[idx].claimQuantity = min(1, max(0, lineDrafts[idx].orderedQuantity))
        }
        skuWarning = nil
    }

    func loadAndUpload(item: PhotosPickerItem?) async {
        guard let item else { return }
        errorMessage = nil
        isUploadingPhoto = true
        defer { isUploadingPhoto = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data) else {
                Haptics.error()
                errorMessage = "Could not read photo."
                return
            }
            previewImage = image
            photoURL = try await MediaUploadService.uploadJPEG(
                image: image,
                purpose: "claim_evidence",
                orderId: order.id,
                api: api
            )
            Haptics.success()
        } catch {
            Haptics.error()
            errorMessage = RetailerW2ErrorParser.userFriendlyClaimError(for: error)
            photoURL = ""
        }
    }

    func submit() async {
        errorMessage = nil
        successClaimId = nil
        if let eligibility, !eligibility.eligible {
            Haptics.error()
            errorMessage = String(localized: "mobile_retailer.claims.error.window_expired")
            return
        }
        isSubmitting = true
        defer { isSubmitting = false }

        let needsPhoto = ["DAMAGED", "CONCEALED_DAMAGE", "TAMPER", "TEMPERATURE"].contains(claimType)
        if needsPhoto && photoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            Haptics.warning()
            errorMessage = String(localized: "mobile_retailer.claims.error.photo_required")
            return
        }

        var totalsBySku: [String: Int] = [:]
        var orderedSkus: [String] = []
        for draft in lineDrafts where draft.claimQuantity > 0 {
            if totalsBySku[draft.sku] == nil {
                orderedSkus.append(draft.sku)
            }
            totalsBySku[draft.sku, default: 0] += draft.claimQuantity
        }

        let lines: [FileClaimLineBody] = orderedSkus.compactMap { sku in
            guard let qty = totalsBySku[sku], qty > 0 else { return nil }
            return FileClaimLineBody(
                sku: sku,
                quantity: Int64(qty),
                reason: claimType == "MISSING" ? "MISSING" : "DAMAGED"
            )
        }
        guard !lines.isEmpty else {
            Haptics.warning()
            errorMessage = "Select at least one item quantity."
            return
        }

        do {
            let photo = photoURL.trimmingCharacters(in: .whitespacesAndNewlines)
            let fingerprint = ([claimType, descriptionText] + lines.map { "\($0.sku):\($0.quantity)" } + [photo])
                .joined(separator: "|")
            let claim = try await api.fileOrderClaim(
                orderId: order.id,
                claimType: claimType,
                description: descriptionText,
                lines: lines,
                photoURL: photo,
                idempotencyKey: RetailerIdempotency.claimFile(orderId: order.id, bodyFingerprint: fingerprint)
            )
            Haptics.success()
            successClaimId = claim.claimId
            await loadExisting()
        } catch {
            Haptics.error()
            errorMessage = RetailerW2ErrorParser.userFriendlyClaimError(for: error)
        }
    }

    func formatEndsAt(_ raw: String?) -> String {
        guard let raw, let date = Self.isoFormatter.date(from: raw) else {
            return raw ?? "window end"
        }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    func windowClosedCopy(_ e: ClaimEligibility) -> String {
        switch e.reason {
        case "claim_window_expired":
            return "Window closed — filing deadline passed."
        case "order_not_completed":
            return "Window closed — order not COMPLETED yet."
        default:
            return "Window closed."
        }
    }
}
