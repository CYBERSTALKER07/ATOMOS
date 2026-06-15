//
//  PaymentWaitingView.swift
//  driverappios
//
//  Shown after driver confirms offload for card-payment orders.
//  Subscribes to shared /v1/ws/driver state and waits for PAYMENT_SETTLED push.
//  Once settled, driver can tap "Complete" to finalize delivery.
//

import SwiftUI

struct PaymentWaitingView: View {

    let orderId: String
    let amount: Int
    let driverId: String
    let onCompleted: () -> Void

    @State private var isSettled = false
    @State private var isCompleting = false
    @State private var errorMessage: String?
    @State private var driverSocketState = DriverSocketState.shared

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // MARK: - Status Icon
            Image(systemName: isSettled ? "checkmark.seal.fill" : "clock.fill")
                .font(.system(size: 64))
                .foregroundStyle(isSettled ? LabTheme.success : LabTheme.warning)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Title
            Text(isSettled ? "Payment Received" : "Awaiting Payment")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            // MARK: - Order ID
            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Amount
            Text(amount.formattedAmount)
                .font(.system(size: 36, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s24)

            if !isSettled {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.bottom, LabTheme.s8)
                Text("Retailer is completing card payment...")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .multilineTextAlignment(.center)
            }

            Spacer()

            // MARK: - Error
            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Complete Button
            Button {
                completeDelivery()
            } label: {
                Text("Complete Delivery")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(isSettled ? LabTheme.buttonFg : LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(
                        isSettled ? LabTheme.fg : LabTheme.fg.opacity(0.08),
                        in: .rect(cornerRadius: LabTheme.buttonRadius)
                    )
            }
            .disabled(!isSettled || isCompleting)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(LabTheme.bg)
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

        guard let event else { return }
        guard event.type == "PAYMENT_SETTLED", event.orderId == orderId else { return }

        isSettled = true
        Haptics.success()
    }

    // MARK: - Complete

    private func completeDelivery() {
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                try await FleetServiceLive.shared.completeOrder(orderId: orderId)
                Haptics.success()
                onCompleted()
            } catch {
                isCompleting = false
                errorMessage = error.localizedDescription
            }
        }
    }
}
