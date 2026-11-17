//
//  CashCollectionView.swift
//  driverappios
//
//  Cash capture + ADR-009 fiscal wait / retry.
//

import SwiftUI

struct CashCollectionView: View {

    let orderId: String
    let amount: Int
    let onCompleted: () -> Void
    let onCancel: () -> Void
    var onSplitPayment: ((String, Int) -> Void)?

    @State private var vm: CashCollectionViewModel
    @State private var driverSocketState = DriverSocketState.shared

    init(
        orderId: String,
        amount: Int,
        onCompleted: @escaping () -> Void,
        onCancel: @escaping () -> Void,
        onSplitPayment: ((String, Int) -> Void)? = nil
    ) {
        self.orderId = orderId
        self.amount = amount
        self.onCompleted = onCompleted
        self.onCancel = onCancel
        self.onSplitPayment = onSplitPayment
        _vm = State(initialValue: CashCollectionViewModel(orderId: orderId, amount: amount))
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Spacer()
                Button { onCancel() } label: {
                    Image(systemName: "xmark")
                        .font(.system(.caption2, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(width: 44, height: 44)
                        .background(LabTheme.fg.opacity(0.06), in: Circle())
                }
                .accessibilityLabel("Close")
                .disabled(vm.phase == .fiscalizing || vm.isCompleting)
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.top, LabTheme.s24)

            Spacer()

            switch vm.phase {
            case .fiscalizing:
                FiscalizingView(amount: amount)
            case .fiscalFailed:
                FiscalFailedView(subtitle: vm.statusNotice)
            default:
                CollectCashView(
                    orderId: orderId,
                    amount: amount,
                    amountReceivedText: $vm.amountReceivedText,
                    shortfallNote: vm.shortfallNote
                )
                .onChange(of: vm.amountReceivedText) { _, _ in
                    vm.refreshShortfallNote()
                }
            }

            Spacer()

            if let error = vm.errorMessage {
                Text(error)
                    .font(.system(.footnote, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            if vm.phase == .collect, let onSplitPayment {
                Button {
                    onSplitPayment(orderId, amount)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "arrow.triangle.branch")
                            .font(.system(.subheadline, weight: .semibold))
                        Text("mobile_driver.ui.split_payment_pay_now_pay_later")
                            .font(.system(.subheadline, weight: .semibold))
                    }
                    .foregroundStyle(LabTheme.fg)
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .padding(.vertical, 14)
                    .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    .overlay(
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                            .stroke(LabTheme.fg.opacity(0.15), lineWidth: 1)
                    )
                }
                .disabled(vm.isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s8)
            }

            if vm.phase == .collect {
                Button {
                    Task { await vm.completeWithCash() }
                } label: {
                    HStack(spacing: 8) {
                        if vm.isCompleting { ProgressView().tint(LabTheme.buttonFg) }
                        Text("mobile_driver.ui.cash_collected_capture")
                            .font(.system(.subheadline, weight: .bold))
                    }
                    .foregroundStyle(LabTheme.buttonFg)
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .padding(.vertical, 16)
                    .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .disabled(vm.isCompleting)
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            } else if vm.phase == .fiscalFailed {
                VStack(spacing: 10) {
                    Button {
                        Haptics.success()
                        vm.continueNextStop()
                    } label: {
                        Text(L10n.string("mobile_driver.ui.continue_next_stop"))
                            .font(.system(.subheadline, weight: .bold))
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .padding(.vertical, 14)
                            .background(LabTheme.success, in: .rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .disabled(vm.isCompleting)

                    Button {
                        Task { await vm.retryFiscal() }
                    } label: {
                        HStack(spacing: 8) {
                            if vm.isCompleting { ProgressView().tint(LabTheme.fg) }
                            Text(L10n.string("mobile_driver.ui.retry_soliq_now"))
                                .font(.system(.subheadline, weight: .semibold))
                        }
                        .foregroundStyle(LabTheme.fg)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 12)
                        .overlay(
                            RoundedRectangle(cornerRadius: LabTheme.buttonRadius)
                                .stroke(LabTheme.fgTertiary.opacity(0.4), lineWidth: 1)
                        )
                    }
                    .disabled(vm.isCompleting)
                }
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            }
        }
        .background(LabTheme.bg)
        .scrollDismissesKeyboard(.interactively)
        .onChange(of: vm.phase) { _, newPhase in
            if newPhase == .done { onCompleted() }
        }
        .task(id: driverSocketState.eventSequence) {
            vm.handleSocketEvent(driverSocketState.lastEvent)
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task {
                let hadInFlight = vm.isCompleting
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
                if hadInFlight {
                    vm.isCompleting = false
                    vm.errorMessage = DriverReconnectRecovery.hint
                }
            }
        }
    }
}
