//
//  OffloadReviewView.swift
//  driverappios
//
//  Post-QR scan: shows order info + line items.
//  Driver can exclude/mark damaged items, then confirm offload.
//

import PhotosUI
import SwiftUI
import UIKit

struct OffloadReviewView: View {
    let response: ValidateQRResponse
    let scannedToken: String
    let driverId: String
    let onConfirm: (ConfirmOffloadResponse) -> Void
    let onCancel: () -> Void
    var onShopClosed: ((String) -> Void)?
    /// orderId + photo URL + signature URL + optional local paths for offline flush.
    var onCreditDelivery: ((String, String, String, String?, String?) -> Void)?
    var onReportMissing: ((String) -> Void)?

    @State private var vm: OffloadReviewViewModel
    @State private var driverSocketState = DriverSocketState.shared

    init(
        response: ValidateQRResponse,
        scannedToken: String,
        driverId: String,
        onConfirm: @escaping (ConfirmOffloadResponse) -> Void,
        onCancel: @escaping () -> Void,
        onShopClosed: ((String) -> Void)? = nil,
        onCreditDelivery: ((String, String, String, String?, String?) -> Void)? = nil,
        onReportMissing: ((String) -> Void)? = nil
    ) {
        self.response = response
        self.scannedToken = scannedToken
        self.driverId = driverId
        self.onConfirm = onConfirm
        self.onCancel = onCancel
        self.onShopClosed = onShopClosed
        self.onCreditDelivery = onCreditDelivery
        self.onReportMissing = onReportMissing
        self._vm = State(initialValue: OffloadReviewViewModel(response: response, scannedToken: scannedToken))
    }

    var body: some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            // MARK: - Header
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("mobile_driver.ui.offload_review")
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Text(response.orderId)
                        .font(.system(.title2, design: .monospaced, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                }
                Spacer()
                Button {
                    Haptics.light()
                    onCancel()
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(.caption, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(width: 44, height: 44)
                        .background(LabTheme.fg.opacity(0.06), in: Circle())
                }
                .accessibilityLabel("Close")
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.top, LabTheme.s24)
            .padding(.bottom, LabTheme.s16)

            // MARK: - Retailer + Total
            OffloadSummaryCard(
                retailerName: response.retailerName,
                totalAmount: response.totalAmount
            )

            // MARK: - Line Items
            ScrollView {
                VStack(spacing: 8) {
                    ForEach(response.items) { item in
                        OffloadLineItemRow(item: item, vm: vm)
                    }
                }
            }
            .scrollDismissesKeyboard(.interactively)

            // MARK: - PoD / damage photo proof
            VStack(alignment: .leading, spacing: 8) {
                Text("mobile_driver.ui.proof_of_delivery_photo")
                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
                PhotosPicker(selection: $vm.pickedPhoto, matching: .images) {
                    Label(
                        vm.evidencePhotoURL.isEmpty ? "Take or choose photo" : "Photo ready — change",
                        systemImage: "camera.fill"
                    )
                    .font(.system(.subheadline, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .padding(.vertical, 6)
                    .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .onChange(of: vm.pickedPhoto) { _, item in
                    Task { await vm.uploadPickedPhoto(item) }
                }
                if vm.isUploadingPhoto {
                    ProgressView("Uploading proof…")
                        .tint(LabTheme.fg)
                }
                if let previewImage = vm.previewImage {
                    Image(uiImage: previewImage)
                        .resizable()
                        .scaledToFit()
                        .frame(maxHeight: 140)
                        .clipShape(.rect(cornerRadius: 10))
                }
                Text("mobile_driver.ui.required_for_credit_leave_and_for_damaged_or_wrong_item_rejectio")
                    .font(.caption2)
                    .foregroundStyle(LabTheme.fgTertiary)
                SignaturePadView { data in
                    Task { await vm.saveSignature(data) }
                }
                if !vm.signatureURL.isEmpty || !vm.signatureLocalPath.isEmpty {
                    Text("Signature ready")
                        .font(.caption2)
                        .foregroundStyle(LabTheme.success)
                }
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s12)

            // MARK: - Error
            if let error = vm.errorMessage {
                Text(error)
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            OffloadActionFooter(
                orderId: response.orderId,
                hasRejections: vm.hasRejections,
                isSubmitting: vm.isSubmitting || vm.isUploadingPhoto,
                onShopClosed: onShopClosed,
                onCreditDelivery: { orderId in
                    guard let proof = vm.validateCreditLeave() else { return }
                    onCreditDelivery?(
                        orderId,
                        proof.photoURL,
                        proof.signatureURL,
                        proof.photoLocal,
                        proof.sigLocal
                    )
                },
                onReportMissing: onReportMissing,
                onConfirm: {
                    Task {
                        if let result = await vm.confirmOffload() {
                            onConfirm(result)
                        }
                    }
                }
            )
        }
        .background(LabTheme.bg)
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task { await vm.handleReconnect() }
        }
    }
}

private struct OffloadLineItemRow: View {
    let item: OrderLineItem
    @Bindable var vm: OffloadReviewViewModel
    @State private var customReasonText: String = ""

    var body: some View {
        let rejected = vm.rejectedQty[item.id] ?? 0
        let fullyRejected = rejected == item.quantity
        let partiallyRejected = rejected > 0 && rejected < item.quantity

        VStack(alignment: .leading, spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.productName)
                        .font(.system(.subheadline, weight: .medium))
                        .foregroundStyle(fullyRejected ? LabTheme.fgTertiary : LabTheme.fg)
                        .strikethrough(fullyRejected)
                    Text(L10n.format("mobile_driver.ui.quantity_formattedamount", "\(item.quantity)", "\(item.unitPrice.formattedAmount)"))
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }
                Spacer()

                HStack(spacing: 4) {
                    Button {
                        vm.decrementRejection(for: item)
                    } label: {
                        Image(systemName: "minus.circle.fill")
                            .font(.system(.title3))
                            .foregroundStyle(rejected > 0 ? LabTheme.destructive : LabTheme.fgTertiary)
                            .frame(width: 44, height: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Decrease rejected quantity for \(item.productName)")

                    Text("\(rejected)")
                        .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        .foregroundStyle(
                            fullyRejected ? LabTheme.destructive :
                            partiallyRejected ? Color.orange :
                            LabTheme.success
                        )
                        .frame(minWidth: 24, alignment: .center)

                    Button {
                        vm.incrementRejection(for: item)
                    } label: {
                        Image(systemName: "plus.circle.fill")
                            .font(.system(.title3))
                            .foregroundStyle(rejected < item.quantity ? LabTheme.destructive : LabTheme.fgTertiary)
                            .frame(width: 44, height: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Increase rejected quantity for \(item.productName)")
                }
            }

            if rejected > 0 {
                HStack {
                    Spacer()
                    Menu {
                        ForEach(RejectionReason.allCases, id: \.self) { reason in
                            Button {
                                vm.setReason(reason, for: item.id)
                            } label: {
                                if vm.selectedReason(for: item) == reason {
                                    Label(vm.reasonLabel(for: reason), systemImage: "checkmark")
                                } else {
                                    Text(vm.reasonLabel(for: reason))
                                }
                            }
                        }
                    } label: {
                        HStack(spacing: 6) {
                            Image(systemName: "exclamationmark.circle")
                            Text(vm.reasonLabel(for: vm.selectedReason(for: item)))
                            Image(systemName: "chevron.down")
                                .font(.system(.caption2, weight: .bold))
                        }
                        .font(.system(.caption, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                        .padding(.horizontal, 12)
                        .frame(minHeight: 44)
                        .background(LabTheme.fg.opacity(0.06), in: Capsule())
                    }
                    .accessibilityLabel("Rejection reason for \(item.productName)")
                }

                if vm.selectedReason(for: item) == .OTHER {
                    TextField("mobile_driver.ui.describe_the_issue", text: $customReasonText, axis: .vertical)
                        .lineLimit(2...4)
                        .textFieldStyle(.roundedBorder)
                        .onAppear {
                            customReasonText = vm.customReasons[item.id] ?? ""
                        }
                        .onChange(of: customReasonText) { _, newValue in
                            vm.setCustomReason(newValue, for: item.id)
                        }
                }
            }
        }
        .padding(.horizontal, LabTheme.s24)
        .padding(.vertical, LabTheme.s8)
    }
}
