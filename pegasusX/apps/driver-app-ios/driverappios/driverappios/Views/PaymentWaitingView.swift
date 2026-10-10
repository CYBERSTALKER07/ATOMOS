//
//  PaymentWaitingView.swift
//  driverappios
//
//  Card path + ADR-009 fiscal hard-gate wait / retry.
//

import SwiftUI

struct PaymentWaitingView: View {

    let orderId: String
    let amount: Int
    let driverId: String
    let onCompleted: () -> Void
    var onCashCollectionRequired: () -> Void = {}

    @State private var isSettled = false
    @State private var isFiscalizing = false
    @State private var isFiscalFailed = false
    @State private var isCompleting = false
    @State private var isDone = false
    @State private var errorMessage: String?
    @State private var driverSocketState = DriverSocketState.shared

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            Image(systemName: headerIcon)
                .font(.system(size: 64))
                .foregroundStyle(headerTint)
                .padding(.bottom, LabTheme.s16)

            Text(headerTitle)
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            Text(amount.formattedAmount)
                .font(.system(size: 36, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s24)

            Text(subtitle)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)

            if isFiscalizing || isCompleting || (!isSettled && !isFiscalFailed) {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.top, LabTheme.s16)
            }

            Spacer()

            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            if isFiscalFailed {
                Button { retryFiscal() } label: {
                    Text("mobile_driver.ui.retry_fiscal")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .disabled(isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            } else if errorMessage != nil && isSettled && !isFiscalizing {
                Button { finalizeDelivery() } label: {
                    Text("mobile_driver.ui.retry_capture")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .disabled(isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            }
        }
        .background(LabTheme.bg)
        .onChange(of: isDone) { _, done in
            if done { onCompleted() }
        }
        .task(id: driverSocketState.eventSequence) {
            handleDriverSocketEvent(driverSocketState.lastEvent)
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task {
                let hadInFlight = isCompleting
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
                if hadInFlight {
                    isCompleting = false
                    errorMessage = DriverReconnectRecovery.hint
                }
            }
        }
    }

    private var headerIcon: String {
        if isFiscalFailed { return "exclamationmark.triangle.fill" }
        if isFiscalizing { return "hourglass" }
        if isSettled { return "checkmark.seal.fill" }
        return "clock.fill"
    }

    private var headerTint: Color {
        if isFiscalFailed { return LabTheme.destructive }
        if isSettled && !isFiscalizing { return LabTheme.success }
        return LabTheme.warning
    }

    private var headerTitle: String {
        if isFiscalFailed { return "Fiscal Failed" }
        if isFiscalizing { return "Fiscalizing" }
        if isSettled { return "Payment Received" }
        return "Awaiting Payment"
    }

    private var subtitle: String {
        if isFiscalFailed { return "Retry fiscal receipt or call supervisor for force-complete." }
        if isFiscalizing { return "Payment captured. Waiting for fiscal receipt…" }
        if isSettled { return "Finalizing delivery…" }
        return "Retailer is completing payment..."
    }

    private func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        if let notice = DriverSocketState.shared.outdatedNotice {
            errorMessage = notice.message
            Haptics.warning()
            return
        }

        guard let event, event.orderId == orderId else { return }
        let status = (event.status ?? event.state ?? "").uppercased()

        switch event.type {
        case "ORDER_COMPLETED", "ORDER_FINALIZED", "FISCAL_RECEIPT_SUCCEEDED":
            isSettled = true
            isFiscalizing = false
            isFiscalFailed = false
            isDone = true
            Haptics.success()
        case "PAYMENT_SETTLED", "PAYMENT_CLEARED":
            isSettled = true
            Haptics.success()
            finalizeDelivery()
        case "FISCAL_RECEIPT_REQUESTED":
            isSettled = true
            isFiscalizing = true
            isFiscalFailed = false
        case "FISCAL_RECEIPT_FAILED":
            isSettled = true
            isFiscalizing = false
            isFiscalFailed = true
            errorMessage = "Fiscal receipt failed. Retry or call supervisor."
            Haptics.warning()
        case "ORDER_STATUS_CHANGED", "PAYMENT_REQUIRED":
            if status == "PENDING_CASH_COLLECTION" {
                onCashCollectionRequired()
            } else if status == "FISCALIZING" {
                isSettled = true
                isFiscalizing = true
                isFiscalFailed = false
            } else if status == "FISCAL_FAILED" {
                isSettled = true
                isFiscalizing = false
                isFiscalFailed = true
                errorMessage = "Fiscal receipt failed. Retry or call supervisor."
            } else if status == "COMPLETED" {
                isDone = true
            }
        default:
            break
        }
    }

    private func finalizeDelivery() {
        if isDone || isCompleting { return }
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                try await FleetServiceLive.shared.completeOrder(orderId: orderId)
                Haptics.success()
                // ADR-009: complete enters FISCALIZING; wait for fiscal WS.
                isSettled = true
                isFiscalizing = true
                isFiscalFailed = false
                isCompleting = false
            } catch {
                let message = error.localizedDescription
                if message.localizedCaseInsensitiveContains("COMPLETED") {
                    isDone = true
                } else if message.localizedCaseInsensitiveContains("FISCALIZING") {
                    isSettled = true
                    isFiscalizing = true
                    isCompleting = false
                } else {
                    isCompleting = false
                    errorMessage = message
                }
            }
        }
    }

    private func retryFiscal() {
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                _ = try await FleetServiceLive.shared.retryFiscal(orderId: orderId)
                isFiscalizing = true
                isFiscalFailed = false
                isCompleting = false
            } catch {
                isCompleting = false
                isFiscalFailed = true
                errorMessage = error.localizedDescription
            }
        }
    }
}
