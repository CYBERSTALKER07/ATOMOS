import SwiftUI

/// Post-delivery claim for COMPLETED orders (concealed damage / missing within 48h window).
struct FileClaimView: View {
    let order: Order
    @Environment(\.dismiss) private var dismiss
    private let api = APIClient.shared

    @State private var claimType = "CONCEALED_DAMAGE"
    @State private var descriptionText = ""
    @State private var selected: [String: Int] = [:] // sku -> qty
    @State private var photoURL = ""
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

                Section("Details") {
                    TextField("What happened?", text: $descriptionText, axis: .vertical)
                        .lineLimit(3...6)
                    TextField("Photo URL (required for damage)", text: $photoURL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                    Text("Until in-app camera upload ships, paste a public HTTPS image URL.")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
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
                        .disabled(isSubmitting || selectedQtyTotal == 0)
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
            // Non-fatal: list may 404 if no claims table on old backend.
            existing = []
        }
    }

    private func submit() async {
        errorMessage = nil
        successClaimId = nil
        isSubmitting = true
        defer { isSubmitting = false }

        let needsPhoto = ["DAMAGED", "CONCEALED_DAMAGE", "TAMPER", "TEMPERATURE"].contains(claimType)
        if needsPhoto && photoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            errorMessage = "Photo URL required for this claim type."
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
