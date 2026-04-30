//
//  OffloadReviewView.swift
//  driverappios
//
//  Post-QR scan: shows order info + line items.
//  Driver can exclude/mark damaged items, then confirm offload.
//

import SwiftUI

struct OffloadReviewView: View {

    let response: ValidateQRResponse
    let driverId: String
    let onConfirm: (ConfirmOffloadResponse) -> Void
    let onCancel: () -> Void
    var onShopClosed: ((String) -> Void)?
    var onCreditDelivery: ((String) -> Void)?
    var onReportMissing: ((String) -> Void)?

    @State private var rejectedQty: [String: Int] = [:]
    @State private var isSubmitting = false
    @State private var errorMessage: String?

    private let fleetService: FleetServiceProtocol = FleetServiceLive.shared

    private var hasRejections: Bool {
        rejectedQty.values.contains { $0 > 0 }
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
            HStack {
                Text(response.retailerName)
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(LabTheme.fg)
                Spacer()
                Text(response.totalAmount.formattedAmount)
                    .font(.system(size: 15, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s16)

            // MARK: - Line Items
            ScrollView {
                VStack(spacing: 8) {
                    ForEach(response.items) { item in
                        let rejected = rejectedQty[item.id] ?? 0
                        let fullyRejected = rejected == item.quantity
                        let partiallyRejected = rejected > 0 && rejected < item.quantity
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

                            // Stepper: how many units to reject (0 … item.quantity)
                            HStack(spacing: 6) {
                                Button {
                                    if (rejectedQty[item.id] ?? 0) > 0 {
                                        rejectedQty[item.id] = (rejectedQty[item.id] ?? 0) - 1
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
                                    }
                                } label: {
                                    Image(systemName: "plus.circle.fill")
                                        .font(.system(size: 20))
                                        .foregroundStyle((rejectedQty[item.id] ?? 0) < item.quantity ? LabTheme.destructive : LabTheme.fgTertiary)
                                }
                            }
                        }
                        .padding(.horizontal, LabTheme.s24)
                        .padding(.vertical, LabTheme.s8)
                    }
                }
            }

            // MARK: - Error
            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Shop Closed Button
            if let onShopClosed {
                Button {
                    onShopClosed(response.orderId)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "door.left.hand.closed")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Shop Closed / No Answer")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .foregroundStyle(Color.orange)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Color.orange.opacity(0.12), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(Color.orange.opacity(0.3), lineWidth: 1)
                    )
                }
                .disabled(isSubmitting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Edge 32: Credit Delivery Button
            if let onCreditDelivery {
                Button {
                    onCreditDelivery(response.orderId)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "creditcard.fill")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Deliver on Credit")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .foregroundStyle(.blue)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(Color.blue.opacity(0.12), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(Color.blue.opacity(0.3), lineWidth: 1)
                    )
                }
                .disabled(isSubmitting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Edge 33: Missing Items Button
            if let onReportMissing, hasRejections {
                Button {
                    onReportMissing(response.orderId)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Report Missing Items")
                            .font(.system(size: 15, weight: .bold))
                    }
                    .foregroundStyle(LabTheme.destructive)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LabTheme.destructive.opacity(0.12), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(LabTheme.destructive.opacity(0.3), lineWidth: 1)
                    )
                }
                .disabled(isSubmitting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Confirm Offload Button
            Button {
                confirmOffload()
            } label: {
                HStack(spacing: 8) {
                    if isSubmitting {
                        ProgressView().tint(LabTheme.buttonFg)
                    }
                    Text("Confirm Offload")
                        .font(.system(size: 15, weight: .bold))
                }
                .foregroundStyle(LabTheme.buttonFg)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 16)
                .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
            }
            .disabled(isSubmitting)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(LabTheme.bg)
    }

    private func confirmOffload() {
        isSubmitting = true
        errorMessage = nil

        Task {
            // Build amendment items from stepper state
            let hasRejections = rejectedQty.values.contains { $0 > 0 }
            if hasRejections {
                let amendItems: [(lineItemId: String, rejectedQty: Int, status: LineItemStatus)] = response.items.map { item in
                    let rejected = rejectedQty[item.id] ?? 0
                    let status: LineItemStatus = rejected == item.quantity ? .REJECTED_DAMAGED : .DELIVERED
                    return (lineItemId: item.productId, rejectedQty: rejected, status: status)
                }
                do {
                    try await fleetService.amendOrder(
                        orderId: response.orderId,
                        driverId: driverId,
                        items: amendItems
                    )
                } catch {
                    isSubmitting = false
                    errorMessage = "Amendment failed: \(error.localizedDescription)"
                    return
                }
            }

            // Confirm offload → ARRIVED → AWAITING_PAYMENT
            do {
                let result = try await fleetService.confirmOffload(orderId: response.orderId)
                isSubmitting = false
                onConfirm(result)
            } catch {
                isSubmitting = false
                errorMessage = error.localizedDescription
            }
        }
    }
}
