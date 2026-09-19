//
//  CashCollectionView.swift
//  driverappios
//
//  Cash capture + ADR-009 fiscal wait / retry.
//

import SwiftUI

private enum CashFiscalPhase {
    case collect
    case fiscalizing
    case fiscalFailed
    case done
}

struct CashCollectionView: View {

    let orderId: String
    let amount: Int
    let onCompleted: () -> Void
    let onCancel: () -> Void
    var onSplitPayment: ((String, Int) -> Void)?

    @State private var isCompleting = false
    @State private var errorMessage: String?
    @State private var phase: CashFiscalPhase = .collect
    @State private var driverSocketState = DriverSocketState.shared
    /// Cash actually taken (tiyin string). Fiscal uses this amount.
    @State private var amountReceivedText: String = ""
    @State private var shortfallNote: String?

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Spacer()
                Button { onCancel() } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(width: 28, height: 28)
                        .background(LabTheme.fg.opacity(0.06), in: Circle())
                }
                .disabled(phase == .fiscalizing || isCompleting)
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.top, LabTheme.s24)

            Spacer()

            switch phase {
            case .fiscalizing:
                FiscalizingView(amount: amount)
            case .fiscalFailed:
                FiscalFailedView()
            default:
                CollectCashView(
                    orderId: orderId,
                    amount: amount,
                    amountReceivedText: $amountReceivedText,
                    shortfallNote: shortfallNote
                )
                .onChange(of: amountReceivedText) { _, _ in
                    refreshShortfallNote()
                }
            }

            Spacer()

            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            if phase == .collect, let onSplitPayment {
                Button {
                    onSplitPayment(orderId, amount)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "arrow.triangle.branch")
                            .font(.system(size: 14, weight: .semibold))
                        Text("mobile_driver.ui.split_payment_pay_now_pay_later")
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

            if phase == .collect {
                Button { completeWithCash() } label: {
                    HStack(spacing: 8) {
                        if isCompleting { ProgressView().tint(LabTheme.buttonFg) }
                        Text("mobile_driver.ui.cash_collected_capture")
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
            } else if phase == .fiscalFailed {
                Button { retryFiscal() } label: {
                    HStack(spacing: 8) {
                        if isCompleting { ProgressView().tint(LabTheme.buttonFg) }
                        Text("mobile_driver.ui.retry_fiscal")
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
        }
        .background(LabTheme.bg)
        .onAppear {
            if amountReceivedText.isEmpty {
                amountReceivedText = "\(amount)"
            }
            refreshShortfallNote()
        }
        .onChange(of: phase) { _, newPhase in
            if newPhase == .done { onCompleted() }
        }
        .task(id: driverSocketState.eventSequence) {
            handleSocketEvent(driverSocketState.lastEvent)
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

    private func handleSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        guard let event, event.orderId == orderId else { return }
        let status = (event.status ?? event.state ?? "").uppercased()
        switch event.type {
        case "ORDER_COMPLETED", "ORDER_FINALIZED", "FISCAL_RECEIPT_SUCCEEDED":
            phase = .done
            Haptics.success()
        case "FISCAL_RECEIPT_FAILED":
            phase = .fiscalFailed
            errorMessage = "Fiscal receipt failed. Retry or call supervisor."
            Haptics.warning()
        case "ORDER_STATUS_CHANGED":
            if status == "COMPLETED" {
                phase = .done
            } else if status == "FISCAL_FAILED" {
                phase = .fiscalFailed
                errorMessage = "Fiscal receipt failed. Retry or call supervisor."
            } else if status == "FISCALIZING" {
                phase = .fiscalizing
            }
        default:
            break
        }
    }

    private var receivedAmountDisplay: Int {
        Int(amountReceivedText.filter(\.isNumber)) ?? amount
    }

    private func refreshShortfallNote() {
        let received = Int64(amountReceivedText.filter(\.isNumber)) ?? Int64(amount)
        let expected = Int64(amount)
        if received < expected {
            shortfallNote = "Shortfall \(Int(expected - received).formattedAmount) — fiscal uses received amount"
        } else if received > expected {
            shortfallNote = "Overage \(Int(received - expected).formattedAmount) recorded"
        } else {
            shortfallNote = nil
        }
    }

    private func completeWithCash() {
        isCompleting = true
        errorMessage = nil
        let received = Int64(amountReceivedText.filter(\.isNumber)) ?? Int64(amount)
        Task {
            do {
                let resp = try await FleetServiceLive.shared.collectCash(
                    orderId: orderId,
                    amountReceivedMinor: received
                )
                Haptics.success()
                let st = resp.state.uppercased()
                if st == "COMPLETED" {
                    phase = .done
                } else if st == "FISCAL_FAILED" {
                    phase = .fiscalFailed
                    errorMessage = "Fiscal receipt failed. Retry or call supervisor."
                } else {
                    phase = .fiscalizing
                }
                isCompleting = false
            } catch {
                isCompleting = false
                errorMessage = error.localizedDescription
            }
        }
    }

    private func retryFiscal() {
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                _ = try await FleetServiceLive.shared.retryFiscal(orderId: orderId)
                phase = .fiscalizing
                isCompleting = false
            } catch {
                isCompleting = false
                phase = .fiscalFailed
                errorMessage = error.localizedDescription
            }
        }
    }
}
