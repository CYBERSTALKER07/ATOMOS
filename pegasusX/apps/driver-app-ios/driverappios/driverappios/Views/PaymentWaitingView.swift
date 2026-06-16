//
//  PaymentWaitingView.swift
//  driverappios
//
//  Shown after driver confirms offload. Waits for retailer payment, then
//  auto-finalizes delivery when payment settles or the order completes.
//

import SwiftUI

struct PaymentWaitingView: View {

    let orderId: String
    let amount: Int
    let driverId: String
    let onCompleted: () -> Void
    var onCashCollectionRequired: () -> Void = {}

    @State private var isSettled = false
    @State private var isCompleting = false
    @State private var isDone = false
    @State private var errorMessage: String?
    @State private var driverSocketState = DriverSocketState.shared

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            Image(systemName: isSettled ? "checkmark.seal.fill" : "clock.fill")
                .font(.system(size: 64))
                .foregroundStyle(isSettled ? LabTheme.success : LabTheme.warning)
                .padding(.bottom, LabTheme.s16)

            Text(isSettled ? "Payment Received" : "Awaiting Payment")
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

            if !isSettled {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.bottom, LabTheme.s8)
                Text("Retailer is completing payment...")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .multilineTextAlignment(.center)
            } else if isCompleting {
                ProgressView()
                    .padding(.top, LabTheme.s8)
            }

            Spacer()

            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            if errorMessage != nil && isSettled {
                Button {
                    finalizeDelivery()
                } label: {
                    Text("Retry Complete Delivery")
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
        .task {
            if let notice = DriverSocketState.shared.outdatedNotice {
                errorMessage = notice.message
                return
            }
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

    private func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        if let notice = DriverSocketState.shared.outdatedNotice {
            errorMessage = notice.message
            Haptics.warning()
            return
        }

        guard let event, event.orderId == orderId else { return }

        switch event.type {
        case "ORDER_COMPLETED", "ORDER_FINALIZED":
            isSettled = true
            isDone = true
            Haptics.success()
        case "PAYMENT_SETTLED", "PAYMENT_CLEARED":
            isSettled = true
            Haptics.success()
            finalizeDelivery()
        case "ORDER_STATUS_CHANGED", "PAYMENT_REQUIRED":
            let status = (event.status ?? event.state ?? "").uppercased()
            if status == "PENDING_CASH_COLLECTION" {
                onCashCollectionRequired()
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
                isDone = true
            } catch {
                let message = error.localizedDescription
                if message.localizedCaseInsensitiveContains("COMPLETED") ||
                    message.localizedCaseInsensitiveContains("invalid_status") {
                    isDone = true
                } else {
                    isCompleting = false
                    errorMessage = message
                }
            }
        }
    }
}
