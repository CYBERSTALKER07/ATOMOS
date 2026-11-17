import PhotosUI
import SwiftUI
import UIKit

/// Post-delivery claim for COMPLETED orders (concealed damage / missing within 48h window).
struct FileClaimView: View {
    let order: Order
    var preferredSku: String? = nil
    @Environment(\.dismiss) private var dismiss
    @State private var vm: FileClaimViewModel

    init(order: Order, preferredSku: String? = nil) {
        self.order = order
        self.preferredSku = preferredSku
        _vm = State(initialValue: FileClaimViewModel(order: order, preferredSku: preferredSku))
    }

    var body: some View {
        @Bindable var vm = vm
        NavigationStack {
            Form {
                Section {
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(6))"))
                        .font(.headline)
                    if vm.eligLoading {
                        Text("mobile_retailer.ui.checking_claim_window")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else if let eligibility = vm.eligibility, eligibility.eligible {
                        Text(L10n.format("mobile_retailer.ui.eligible_until_formatendsat_hoursremainingh_left_amounts_use_your_order_", "\(vm.formatEndsAt(eligibility.endsAt))", "\(eligibility.hoursRemaining)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else if let eligibility = vm.eligibility, !eligibility.eligible {
                        Text(vm.windowClosedCopy(eligibility))
                            .font(.caption)
                            .foregroundStyle(.red)
                    } else {
                        Text("mobile_retailer.ui.file_within_48_hours_of_delivery_server_enforces_amounts_are_cal")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if let skuWarning = vm.skuWarning {
                        Text(skuWarning)
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                }

                Section("mobile_retailer.claims.claim_type") {
                    Picker("mobile_retailer.claims.type_label", selection: $vm.claimType) {
                        ForEach(vm.claimTypes, id: \.self) { Text($0).tag($0) }
                    }
                }

                Section("mobile_retailer.claims.items_to_claim") {
                    ForEach($vm.lineDrafts) { $line in
                        Stepper(value: $line.claimQuantity, in: 0...max(line.orderedQuantity, 0)) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(line.productName)
                                Text(L10n.format("mobile_retailer.ui.sku_sku_ordered_quantity", "\(line.sku)", "\(line.orderedQuantity)"))
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .onChange(of: line.claimQuantity) { _, _ in
                            Haptics.selection()
                        }
                        .accessibilityLabel("\(line.productName), claiming \(line.claimQuantity) of \(line.orderedQuantity)")
                    }
                }

                Section("mobile_retailer.claims.photo_proof") {
                    let hasPreviewImage = vm.previewImage != nil
                    PhotosPicker(selection: $vm.pickedItem, matching: .images, photoLibrary: .shared()) {
                        Label(
                            hasPreviewImage ? "mobile_retailer.claims.change_photo" : "mobile_retailer.claims.take_or_choose_photo",
                            systemImage: "camera.fill"
                        )
                        .pegasusMinimumHitTarget()
                    }
                    .onChange(of: vm.pickedItem) { _, newItem in
                        Task { await vm.loadAndUpload(item: newItem) }
                    }
                    if vm.isUploadingPhoto {
                        ProgressView("mobile_retailer.claims.uploading")
                    }
                    if let previewImage = vm.previewImage {
                        Image(uiImage: previewImage)
                            .resizable()
                            .scaledToFit()
                            .frame(maxHeight: 180)
                            .clipShape(.rect(cornerRadius: 12))
                            .accessibilityLabel("Uploaded claim evidence photo")
                    }
                    if !vm.photoURL.isEmpty {
                        Text("mobile_retailer.ui.photo_ready")
                            .font(.caption)
                            .foregroundStyle(.green)
                    }
                    Text("mobile_retailer.ui.required_for_damage_concealed_damage_tamper_temperature_claims")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Section("mobile_retailer.claims.details") {
                    TextField("mobile_retailer.ui.what_happened", text: $vm.descriptionText, axis: .vertical)
                        .lineLimit(3...6)
                }

                if let errorMessage = vm.errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }

                if let successClaimId = vm.successClaimId {
                    Section {
                        Text(L10n.format("mobile_retailer.ui.claim_filed_successclaimid_2", "\(successClaimId)"))
                            .foregroundStyle(.green)
                    }
                }

                if !vm.existing.isEmpty {
                    Section("mobile_retailer.claims.previous_claims") {
                        ForEach(vm.existing) { c in
                            let formattedAmount = RetailerMoney.formatWholeCurrency(c.amountMinor ?? 0, currency: c.currency)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(L10n.format("mobile_retailer.ui.claimtype_status", "\(c.claimType)", "\(c.status)"))
                                    .font(.subheadline.weight(.semibold))
                                Text("\(formattedAmount) · \(c.claimId)")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Claim \(c.claimType), status \(c.status), \(formattedAmount)")
                        }
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("mobile_retailer.ui.file_claim")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.close") { dismiss() }
                        .pegasusMinimumHitTarget()
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("mobile_retailer.claims.submit") {
                        Task { await vm.submit() }
                    }
                    .disabled(!vm.canSubmit)
                    .pegasusMinimumHitTarget()
                }
            }
            .task {
                await vm.loadInitial()
            }
        }
    }
}
