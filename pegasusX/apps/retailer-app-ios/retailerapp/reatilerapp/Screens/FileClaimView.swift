import PhotosUI
import SwiftUI
import UIKit

/// Post-delivery claim for COMPLETED orders (concealed damage / missing within 48h window).
struct FileClaimView: View {
    let order: Order
    var preferredSku: String? = nil
    @Environment(\.dismiss) private var dismiss
    private let api = APIClient.shared

    @State private var claimType = "CONCEALED_DAMAGE"
    @State private var descriptionText = ""
    @State private var selected: [String: Int] = [:] // sku -> qty
    @State private var photoURL = ""
    @State private var pickedItem: PhotosPickerItem?
    @State private var previewImage: UIImage?
    @State private var isUploadingPhoto = false
    @State private var isSubmitting = false
    @State private var errorMessage: String?
    @State private var skuWarning: String?
    @State private var successClaimId: String?
    @State private var existing: [RetailerClaim] = []
    @State private var eligibility: ClaimEligibility?
    @State private var eligLoading = true

    private let claimTypes = ["CONCEALED_DAMAGE", "DAMAGED", "MISSING", "TAMPER", "TEMPERATURE", "OTHER"]

    private var canSubmit: Bool {
        !eligLoading && (eligibility?.eligible ?? true) && selectedQtyTotal > 0 && !isSubmitting && !isUploadingPhoto
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(6))"))
                        .font(.headline)
                    if eligLoading {
                        Text("mobile_retailer.ui.checking_claim_window")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else if let eligibility, eligibility.eligible {
                        Text(L10n.format("mobile_retailer.ui.eligible_until_formatendsat_hoursremainingh_left_amounts_use_your_order_", "\(formatEndsAt(eligibility.endsAt))", "\(eligibility.hoursRemaining)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else if let eligibility, !eligibility.eligible {
                        Text(windowClosedCopy(eligibility))
                            .font(.caption)
                            .foregroundStyle(.red)
                    } else {
                        Text("mobile_retailer.ui.file_within_48_hours_of_delivery_server_enforces_amounts_are_cal")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if let skuWarning {
                        Text(skuWarning)
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                }

                Section("Claim type") {
                    Picker("Type", selection: $claimType) {
                        ForEach(claimTypes, id: \.self) { Text($0).tag($0) }
                    }
                }

                Section("Items to claim") {
                    ForEach(order.items) { item in
                        let sku = item.productId.isEmpty ? item.variantId : item.productId
                        Stepper(value: binding(for: sku), in: 0...max(item.quantity, 0)) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(item.productName)
                                Text(L10n.format("mobile_retailer.ui.sku_sku_ordered_quantity", "\(sku)", "\(item.quantity)"))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }

                Section("Photo proof") {
                    PhotosPicker(selection: $pickedItem, matching: .images, photoLibrary: .shared()) {
                        Label(
                            previewImage == nil ? "Take or choose photo" : "Change photo",
                            systemImage: "camera.fill"
                        )
                    }
                    .onChange(of: pickedItem) { _, newItem in
                        Task { await loadAndUpload(item: newItem) }
                    }
                    if isUploadingPhoto {
                        ProgressView("Uploading…")
                    }
                    if let previewImage {
                        Image(uiImage: previewImage)
                            .resizable()
                            .scaledToFit()
                            .frame(maxHeight: 180)
                            .clipShape(.rect(cornerRadius: 12))
                    }
                    if !photoURL.isEmpty {
                        Text("mobile_retailer.ui.photo_ready")
                            .font(.caption)
                            .foregroundStyle(.green)
                    }
                    Text("mobile_retailer.ui.required_for_damage_concealed_damage_tamper_temperature_claims")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Section("Details") {
                    TextField("mobile_retailer.ui.what_happened", text: $descriptionText, axis: .vertical)
                        .lineLimit(3...6)
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage).foregroundStyle(.red).font(.caption)
                    }
                }

                if let successClaimId {
                    Section {
                        Text(L10n.format("mobile_retailer.ui.claim_filed_successclaimid_2", "\(successClaimId)"))
                            .foregroundStyle(.green)
                    }
                }

                if !existing.isEmpty {
                    Section("Previous claims") {
                        ForEach(existing) { c in
                            VStack(alignment: .leading, spacing: 2) {
                                Text(L10n.format("mobile_retailer.ui.claimtype_status", "\(c.claimType)", "\(c.status)"))
                                    .font(.subheadline.weight(.semibold))
                                Text("\(c.amountMinor ?? 0) \(c.currency ?? "UZS") · \(c.claimId)")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
            .navigationTitle("mobile_retailer.ui.file_claim")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("warehouse_portal.cycle_counts.text.submit") { Task { await submit() } }
                        .disabled(!canSubmit)
                }
            }
            .task {
                await loadExisting()
                await loadEligibility()
                applyPreferredSku()
            }
        }
    }

    private func loadEligibility() async {
        eligLoading = true
        defer { eligLoading = false }
        do {
            eligibility = try await api.getClaimEligibility(orderId: order.id)
        } catch {
            eligibility = nil
        }
    }

    private func formatEndsAt(_ raw: String?) -> String {
        guard let raw, let date = ISO8601DateFormatter().date(from: raw) else {
            return raw ?? "window end"
        }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    private func windowClosedCopy(_ e: ClaimEligibility) -> String {
        switch e.reason {
        case "claim_window_expired":
            return "Window closed — filing deadline passed."
        case "order_not_completed":
            return "Window closed — order not COMPLETED yet."
        default:
            return "Window closed."
        }
    }

    private func applyPreferredSku() {
        let sku = preferredSku?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !sku.isEmpty else {
            skuWarning = nil
            return
        }
        guard let match = order.items.first(where: {
            $0.productId == sku || $0.id == sku || $0.variantId == sku
        }) else {
            skuWarning = "SKU \(sku) is not on this order — pick another line."
            return
        }
        let key = match.productId.isEmpty ? match.variantId : match.productId
        selected[key] = min(1, max(0, match.quantity))
        skuWarning = nil
    }

    private var selectedQtyTotal: Int {
        selected.values.reduce(0, +)
    }

    private func binding(for sku: String) -> Binding<Int> {
        Binding(
            get: { selected[sku] ?? 0 },
            set: { selected[sku] = $0 }
        )
    }

    private func loadExisting() async {
        do {
            existing = try await api.listOrderClaims(orderId: order.id)
        } catch {
            existing = []
        }
    }

    private func loadAndUpload(item: PhotosPickerItem?) async {
        guard let item else { return }
        errorMessage = nil
        isUploadingPhoto = true
        defer { isUploadingPhoto = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data) else {
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
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "Photo upload failed: \(error)"
            photoURL = ""
        }
    }

    private func submit() async {
        errorMessage = nil
        successClaimId = nil
        if let eligibility, !eligibility.eligible {
            errorMessage = "Window closed — claim window has expired."
            return
        }
        isSubmitting = true
        defer { isSubmitting = false }

        let needsPhoto = ["DAMAGED", "CONCEALED_DAMAGE", "TAMPER", "TEMPERATURE"].contains(claimType)
        if needsPhoto && photoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            errorMessage = "Photo required for this claim type."
            return
        }

        let lines: [FileClaimLineBody] = selected.compactMap { sku, qty in
            guard qty > 0 else { return nil }
            return FileClaimLineBody(sku: sku, quantity: Int64(qty), reason: claimType == "MISSING" ? "MISSING" : "DAMAGED")
        }
        guard !lines.isEmpty else {
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
            successClaimId = claim.claimId
            await loadExisting()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "\(error)"
        }
    }
}
