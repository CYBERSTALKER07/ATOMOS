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
                Image(systemName: "hourglass")
                    .font(.system(size: 64))
                    .foregroundStyle(LabTheme.warning)
                    .padding(.bottom, LabTheme.s16)
                Text("Fiscalizing")
                    .font(.system(size: 24, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.bottom, LabTheme.s8)
                Text(amount.formattedAmount)
                    .font(.system(size: 36, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.bottom, LabTheme.s16)
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.bottom, LabTheme.s8)
                Text("Cash captured. Waiting for fiscal receipt…")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)

            case .fiscalFailed:
                Image(systemName: "exclamationmark.triangle.fill")
                    .font(.system(size: 64))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.bottom, LabTheme.s16)
                Text("Fiscal Failed")
                    .font(.system(size: 24, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.bottom, LabTheme.s8)
                Text("Retry fiscal receipt or call supervisor for force-complete.")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)

            default:
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
                Text(amount.formattedAmount)
                    .font(.system(size: 42, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.bottom, LabTheme.s8)
                Text("Collect this amount from the retailer before completing.")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)
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

            if phase == .collect {
                Button { completeWithCash() } label: {
                    HStack(spacing: 8) {
                        if isCompleting { ProgressView().tint(LabTheme.buttonFg) }
                        Text("Cash Collected — Capture")
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
                        Text("Retry Fiscal")
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

    private func completeWithCash() {
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                let resp = try await FleetServiceLive.shared.collectCash(
                    orderId: orderId,
                    amountReceivedMinor: Int64(amount)
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
