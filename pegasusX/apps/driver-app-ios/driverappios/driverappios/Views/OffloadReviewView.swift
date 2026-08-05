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
    /// orderId + PoD photo public URL (required for credit leave).
    var onCreditDelivery: ((String, String) -> Void)?
    var onReportMissing: ((String) -> Void)?

    @State private var rejectedQty: [String: Int] = [:]
    @State private var rejectionReasons: [String: RejectionReason] = [:]
    @State private var customReasons: [String: String] = [:]
    @State private var isSubmitting = false
    @State private var errorMessage: String?
    @State private var driverSocketState = DriverSocketState.shared
    @State private var pickedPhoto: PhotosPickerItem?
    @State private var previewImage: UIImage?
    @State private var evidencePhotoURL = ""
    @State private var isUploadingPhoto = false

    private let fleetService: FleetServiceProtocol = FleetServiceLive.shared

    private var hasRejections: Bool {
        rejectedQty.values.contains { $0 > 0 }
    }

    private var needsPhotoProof: Bool {
        response.items.contains { item in
            let rejected = rejectedQty[item.id] ?? 0
            guard rejected > 0 else { return false }
            let reason = selectedReason(for: item)
            return reason == .DAMAGED || reason == .WRONG_ITEM
        }
    }

    private func selectedReason(for item: OrderLineItem) -> RejectionReason {
        rejectionReasons[item.id] ?? .DAMAGED
    }

    private func reasonLabel(for reason: RejectionReason) -> String {
        reason.rawValue.replacingOccurrences(of: "_", with: " ").capitalized
    }

    var body: some View {
        VStack(spacing: 0) {
            // MARK: - Header
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text("OFFLOAD REVIEW")
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Text(response.orderId)
                        .font(.system(size: 22, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                }
                Spacer()
                Button { onCancel() } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(width: 28, height: 28)
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
                        let rejected = rejectedQty[item.id] ?? 0
                        let fullyRejected = rejected == item.quantity
                        let partiallyRejected = rejected > 0 && rejected < item.quantity
                        VStack(alignment: .leading, spacing: 10) {
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(item.productName)
                                        .font(.system(size: 14, weight: .medium))
                                        .foregroundStyle(fullyRejected ? LabTheme.fgTertiary : LabTheme.fg)
                                        .strikethrough(fullyRejected)
                                    Text("\(item.quantity) × \(item.unitPrice.formattedAmount)")
                                        .font(.system(size: 12, design: .monospaced))
                                        .foregroundStyle(LabTheme.fgTertiary)
                                }
                                Spacer()

                                HStack(spacing: 6) {
                                    Button {
                                        if (rejectedQty[item.id] ?? 0) > 0 {
                                            let nextRejected = (rejectedQty[item.id] ?? 0) - 1
                                            rejectedQty[item.id] = nextRejected
                                            if nextRejected == 0 {
                                                rejectionReasons.removeValue(forKey: item.id)
                                                customReasons.removeValue(forKey: item.id)
                                            }
                                        }
                                    } label: {
                                        Image(systemName: "minus.circle.fill")
                                            .font(.system(size: 20))
                                            .foregroundStyle((rejectedQty[item.id] ?? 0) > 0 ? LabTheme.destructive : LabTheme.fgTertiary)
                                    }
                                    Text("\(rejected)")
                                        .font(.system(size: 14, weight: .bold, design: .monospaced))
                                        .foregroundStyle(
                                            fullyRejected ? LabTheme.destructive :
                                            partiallyRejected ? Color.orange :
                                            LabTheme.success
                                        )
                                        .frame(minWidth: 22, alignment: .center)
                                    Button {
                                        if (rejectedQty[item.id] ?? 0) < item.quantity {
                                            rejectedQty[item.id] = (rejectedQty[item.id] ?? 0) + 1
                                            if rejectionReasons[item.id] == nil {
                                                rejectionReasons[item.id] = .DAMAGED
                                            }
                                        }
                                    } label: {
                                        Image(systemName: "plus.circle.fill")
                                            .font(.system(size: 20))
                                            .foregroundStyle((rejectedQty[item.id] ?? 0) < item.quantity ? LabTheme.destructive : LabTheme.fgTertiary)
                                    }
                                }
                            }

                            if rejected > 0 {
                                HStack {
                                    Spacer()
                                    Menu {
                                        ForEach(RejectionReason.allCases, id: \.self) { reason in
                                            Button {
                                                rejectionReasons[item.id] = reason
                                            } label: {
                                                if selectedReason(for: item) == reason {
                                                    Label(reasonLabel(for: reason), systemImage: "checkmark")
                                                } else {
                                                    Text(reasonLabel(for: reason))
                                                }
                                            }
                                        }
                                    } label: {
                                        HStack(spacing: 6) {
                                            Image(systemName: "exclamationmark.circle")
                                            Text(reasonLabel(for: selectedReason(for: item)))
                                            Image(systemName: "chevron.down")
                                                .font(.system(size: 11, weight: .bold))
                                        }
                                        .font(.system(size: 12, weight: .semibold))
                                        .foregroundStyle(LabTheme.fg)
                                        .padding(.horizontal, 10)
                                        .padding(.vertical, 8)
                                        .background(LabTheme.fg.opacity(0.06), in: Capsule())
                                    }
                                }
                                if selectedReason(for: item) == .OTHER {
                                    TextField("Describe the issue", text: Binding(
                                        get: { customReasons[item.id] ?? "" },
                                        set: { customReasons[item.id] = $0 }
                                    ), axis: .vertical)
                                    .lineLimit(2...4)
                                    .textFieldStyle(.roundedBorder)
                                    .padding(.leading, 0)
                                }
                            }
                        }
                        .padding(.horizontal, LabTheme.s24)
                        .padding(.vertical, LabTheme.s8)
                    }
                }
            }

            // MARK: - PoD / damage photo proof
            VStack(alignment: .leading, spacing: 8) {
                Text("PROOF OF DELIVERY PHOTO")
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                PhotosPicker(selection: $pickedPhoto, matching: .images) {
                    Label(
                        evidencePhotoURL.isEmpty ? "Take or choose photo" : "Photo ready — change",
                        systemImage: "camera.fill"
                    )
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                    .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .onChange(of: pickedPhoto) { _, item in
                    Task { await uploadPickedPhoto(item) }
                }
                if isUploadingPhoto {
                    ProgressView("Uploading proof…")
                        .tint(LabTheme.fg)
                }
                if let previewImage {
                    Image(uiImage: previewImage)
                        .resizable()
                        .scaledToFit()
                        .frame(maxHeight: 140)
                        .clipShape(.rect(cornerRadius: 10))
                }
                Text("Required for credit leave and for damaged or wrong-item rejections.")
                    .font(.caption2)
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s12)

            // MARK: - Error
            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            OffloadActionFooter(
                orderId: response.orderId,
                hasRejections: hasRejections,
                isSubmitting: isSubmitting || isUploadingPhoto,
                onShopClosed: onShopClosed,
                onCreditDelivery: { orderId in
                    guard !evidencePhotoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
                        errorMessage = "PoD photo required for credit leave — take a photo of the handoff first."
                        return
                    }
                    onCreditDelivery?(orderId, evidencePhotoURL)
                },
                onReportMissing: onReportMissing,
                onConfirm: { confirmOffload() }
            )
        }
        .background(LabTheme.bg)
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task {
                let hadInFlight = isSubmitting
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
                if hadInFlight {
                    isSubmitting = false
                    errorMessage = DriverReconnectRecovery.hint
                }
            }
        }
    }

    private func uploadPickedPhoto(_ item: PhotosPickerItem?) async {
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
            previewImage = image
            evidencePhotoURL = try await MediaUploadService.uploadJPEG(
                image: image,
                purpose: "driver_exception",
                orderId: response.orderId
            )
        } catch {
            evidencePhotoURL = ""
            errorMessage = "Photo upload failed: \(error.localizedDescription)"
        }
    }

    private func confirmOffload() {
        isSubmitting = true
        errorMessage = nil

        Task {
            let hasRejections = rejectedQty.values.contains { $0 > 0 }
            if hasRejections {
                let missingOther = response.items.first { item in
                    let rejected = rejectedQty[item.id] ?? 0
                    return rejected > 0 && selectedReason(for: item) == .OTHER && (customReasons[item.id] ?? "").trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                }
                if let missingOther {
                    isSubmitting = false
                    errorMessage = "Describe the issue for \(missingOther.productName)"
                    return
                }
                if needsPhotoProof && evidencePhotoURL.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    isSubmitting = false
                    errorMessage = "Photo required for damaged or wrong-item rejections."
                    return
                }

                // Route OS&D through exception-report so claims + photo_url are enforced.
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
                    _ = try await APIClient.shared.reportMissingItems(
                        orderId: response.orderId,
                        missingItems: missingItems,
                        photoURL: evidencePhotoURL.isEmpty ? nil : evidencePhotoURL
                    )
                } catch {
                    isSubmitting = false
                    errorMessage = "Exception report failed: \(error.localizedDescription)"
                    return
                }
            }

            // Canonical ARRIVED → AWAITING_PAYMENT via scan-qr (validate-qr already done).
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
                isSubmitting = false
                onConfirm(result)
            } catch {
                isSubmitting = false
                errorMessage = error.localizedDescription
            }
        }
    }
}
