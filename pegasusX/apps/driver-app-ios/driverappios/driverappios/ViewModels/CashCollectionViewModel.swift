//
//  CashCollectionViewModel.swift
//  driverappios
//
//  Cash capture + ADR-009 fiscal wait / retry ViewModel.
//

import Foundation
import Observation

enum CashFiscalPhase {
    case collect
    case fiscalizing
    case fiscalFailed
    case done
}

@Observable
@MainActor
final class CashCollectionViewModel {

    let orderId: String
    let amount: Int

    var isCompleting = false
    var errorMessage: String?
    var statusNotice: String?
    var phase: CashFiscalPhase = .collect
    var amountReceivedText: String = ""
    var shortfallNote: String?

    private let fleetService: FleetServiceProtocol

    init(orderId: String, amount: Int, fleetService: FleetServiceProtocol = FleetServiceLive.shared) {
        self.orderId = orderId
        self.amount = amount
        self.fleetService = fleetService
        self.amountReceivedText = "\(amount)"
        self.refreshShortfallNote()
    }

    var receivedAmountDisplay: Int {
        Int(amountReceivedText.filter(\.isNumber)) ?? amount
    }

    func refreshShortfallNote() {
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

    func handleSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        guard let event, event.orderId == orderId else { return }
        let status = (event.status ?? event.state ?? "").uppercased()
        switch event.type {
        case "ORDER_COMPLETED", "ORDER_FINALIZED", "FISCAL_RECEIPT_SUCCEEDED":
            phase = .done
            Haptics.success()
        case "FISCAL_RECEIPT_FAILED":
            phase = .fiscalFailed
            errorMessage = nil
            statusNotice = L10n.string("mobile_driver.ui.fiscal_retry_queued_body")
            Haptics.warning()
        case "ORDER_STATUS_CHANGED":
            if status == "COMPLETED" {
                phase = .done
                statusNotice = nil
            } else if status == "FISCAL_FAILED" || status == "FISCAL_RETRY_QUEUED" || status == "PENDING_FISCAL_ISSUE" {
                phase = .fiscalFailed
                errorMessage = nil
                statusNotice = L10n.string("mobile_driver.ui.fiscal_retry_queued_body")
            } else if status == "FISCALIZING" {
                phase = .fiscalizing
                statusNotice = nil
            }
        default:
            break
        }
    }

    func completeWithCash() async {
        isCompleting = true
        errorMessage = nil
        let received = Int64(amountReceivedText.filter(\.isNumber)) ?? Int64(amount)
        do {
            let resp = try await fleetService.collectCash(
                orderId: orderId,
                amountReceivedMinor: received
            )
            Haptics.success()
            let st = resp.state.uppercased()
            let fst = (resp.fiscalStatus ?? "").uppercased()
            if st == "COMPLETED" {
                phase = .done
                statusNotice = nil
            } else if st == "FISCAL_FAILED" || fst == "FISCAL_FAILED" || fst == "FISCAL_RETRY_QUEUED" || fst == "PENDING_FISCAL_ISSUE" {
                phase = .fiscalFailed
                errorMessage = nil
                statusNotice = L10n.string("mobile_driver.ui.fiscal_retry_queued_body")
            } else {
                phase = .fiscalizing
                statusNotice = nil
            }
            isCompleting = false
        } catch {
            isCompleting = false
            errorMessage = error.localizedDescription
        }
    }

    /// Non-blocking fiscal: cash and PoD are captured, so the driver can continue immediately.
    func continueNextStop() {
        if phase == .fiscalFailed || phase == .fiscalizing {
            phase = .done
            errorMessage = nil
            Haptics.success()
        }
    }

    func retryFiscal() async {
        isCompleting = true
        errorMessage = nil
        do {
            _ = try await fleetService.retryFiscal(orderId: orderId)
            phase = .fiscalizing
            isCompleting = false
        } catch {
            isCompleting = false
            phase = .fiscalFailed
            errorMessage = error.localizedDescription
        }
    }
}
