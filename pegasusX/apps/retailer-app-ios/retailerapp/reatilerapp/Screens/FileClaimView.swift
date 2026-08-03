import PhotosUI
import SwiftUI
import UIKit

/// Post-delivery claim for COMPLETED orders (concealed damage / missing within 48h window).
struct FileClaimView: View {
    let order: Order
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
    @State private var successClaimId: String?
    @State private var existing: [RetailerClaim] = []

    private let claimTypes = ["CONCEALED_DAMAGE", "DAMAGED", "MISSING", "TAMPER", "TEMPERATURE", "OTHER"]

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Text("Order #\(order.id.suffix(6))")
                        .font(.headline)
                    Text("File within 48 hours of delivery. Amounts are calculated from your order prices.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
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
                                Text("SKU \(sku) · ordered \(item.quantity)")
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
                        Text("Photo ready")
                            .font(.caption)
                            .foregroundStyle(.green)
                    }
                    Text("Required for damage / concealed damage / tamper / temperature claims.")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Section("Details") {
                    TextField("What happened?", text: $descriptionText, axis: .vertical)
                        .lineLimit(3...6)
                }

                if let errorMessage {
                    Section {
                        Text(errorMessage).foregroundStyle(.red).font(.caption)
                    }
                }

                if let successClaimId {
                    Section {
                        Text("Claim filed: \(successClaimId)")
                            .foregroundStyle(.green)
                    }
                }

                if !existing.isEmpty {
                    Section("Previous claims") {
                        ForEach(existing) { c in
                            VStack(alignment: .leading, spacing: 2) {
                                Text("\(c.claimType) · \(c.status)")
                                    .font(.subheadline.weight(.semibold))
                                Text("\(c.amountMinor ?? 0) \(c.currency ?? "UZS") · \(c.claimId)")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
            .navigationTitle("File claim")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") { Task { await submit() } }
                        .disabled(isSubmitting || isUploadingPhoto || selectedQtyTotal == 0)
                }
            }
            .task { await loadExisting() }
        }
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
            let claim = try await api.fileOrderClaim(
                orderId: order.id,
                claimType: claimType,
                description: descriptionText,
                lines: lines,
                photoURL: photoURL.trimmingCharacters(in: .whitespacesAndNewlines)
            )
            successClaimId = claim.claimId
            await loadExisting()
        } catch {
            errorMessage = (error as? LocalizedError)?.errorDescription ?? "\(error)"
        }
    }
}
