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

    @State private var vm: PaymentWaitingViewModel
    @State private var driverSocketState = DriverSocketState.shared

    init(
        orderId: String,
        amount: Int,
        driverId: String,
        onCompleted: @escaping () -> Void,
        onCashCollectionRequired: @escaping () -> Void = {}
    ) {
        self.orderId = orderId
        self.amount = amount
        self.driverId = driverId
        self.onCompleted = onCompleted
        self.onCashCollectionRequired = onCashCollectionRequired
        self._vm = State(initialValue: PaymentWaitingViewModel(orderId: orderId, amount: amount, driverId: driverId))
    }

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            Image(systemName: vm.headerIcon)
                .font(.system(.largeTitle, weight: .bold))
                .foregroundStyle(vm.headerTint)
                .padding(.bottom, LabTheme.s16)
                .accessibilityHidden(true)

            Text(vm.headerTitle)
                .font(.system(.title2, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            Text(orderId)
                .font(.system(.subheadline, design: .monospaced, weight: .semibold))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            Text(amount.formattedAmount)
                .font(.system(.largeTitle, design: .monospaced, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s24)

            Text(vm.subtitle)
                .font(.system(.footnote, weight: .medium))
                .foregroundStyle(LabTheme.fgTertiary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)

            if vm.isFiscalizing || vm.isCompleting || (!vm.isSettled && !vm.isFiscalFailed) {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.top, LabTheme.s16)
            }

            Spacer()

            if let error = vm.errorMessage {
                Text(error)
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            if vm.isFiscalFailed {
                Button {
                    Haptics.light()
                    Task { await vm.retryFiscal() }
                } label: {
                    Text("mobile_driver.ui.retry_fiscal")
                        .font(.system(.subheadline, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 10)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .disabled(vm.isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            } else if vm.errorMessage != nil && vm.isSettled && !vm.isFiscalizing {
                Button {
                    Haptics.light()
                    Task { await vm.finalizeDelivery() }
                } label: {
                    Text("mobile_driver.ui.retry_capture")
                        .font(.system(.subheadline, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 10)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .disabled(vm.isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            }
        }
        .background(LabTheme.bg)
        .onChange(of: vm.isDone) { _, done in
            if done { onCompleted() }
        }
        .onChange(of: vm.requiresCashCollection) { _, required in
            if required {
                vm.requiresCashCollection = false
                onCashCollectionRequired()
            }
        }
        .task(id: driverSocketState.eventSequence) {
            vm.handleDriverSocketEvent(driverSocketState.lastEvent)
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task { await vm.handleReconnect() }
        }
    }
}
