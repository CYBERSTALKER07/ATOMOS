//
//  CashCollectionView.swift
//  driverappios
//
//  Shown after driver confirms offload for CASH orders.
//  Displays the amount to collect, then driver taps "Collected — Complete".
//

import SwiftUI

struct CashCollectionView: View {

    let orderId: String
    let amount: Int
    let onCompleted: () -> Void
    let onCancel: () -> Void
    var onSplitPayment: ((String, Int) -> Void)?

    @State private var isCompleting = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 0) {
            // MARK: - Header
            HStack {
                Spacer()
                Button { onCancel() } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(width: 28, height: 28)
                        .background(LabTheme.fg.opacity(0.06), in: Circle())
                }
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.top, LabTheme.s24)

            Spacer()

            // MARK: - Cash Icon
            Image(systemName: "banknote.fill")
                .font(.system(size: 64))
                .foregroundStyle(LabTheme.success)
                .padding(.bottom, LabTheme.s16)

            Text("Collect Cash")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Amount
            Text(amount.formattedAmount)
                .font(.system(size: 42, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            Text("Collect this amount from the retailer before completing.")
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)

            Spacer()

            // MARK: - Error
            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Edge 35: Split Payment Button
            if let onSplitPayment {
                Button {
                    onSplitPayment(orderId, amount)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "arrow.triangle.branch")
                            .font(.system(size: 14, weight: .semibold))
                        Text("Split Payment (Pay Now + Pay Later)")
                            .font(.system(size: 14, weight: .semibold))
                    }
                    .foregroundStyle(LabTheme.fg)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(LabTheme.fg.opacity(0.15), lineWidth: 1)
                    )
                }
                .disabled(isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Button
            Button {
                completeWithCash()
            } label: {
                HStack(spacing: 8) {
                    if isCompleting {
                        ProgressView().tint(LabTheme.buttonFg)
                    }
                    Text("Cash Collected — Complete")
                        .font(.system(size: 15, weight: .bold))
                }
                .foregroundStyle(LabTheme.buttonFg)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 16)
                .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
            }
            .disabled(isCompleting)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(LabTheme.bg)
    }

    private func completeWithCash() {
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                let resp = try await FleetServiceLive.shared.collectCash(orderId: orderId)
                Haptics.success()
                _ = resp // distanceM available if needed
                onCompleted()
            } catch {
                isCompleting = false
                errorMessage = error.localizedDescription
            }
        }
    }
}
