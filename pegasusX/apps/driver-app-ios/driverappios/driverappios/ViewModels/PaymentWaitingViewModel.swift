//
//  PaymentWaitingViewModel.swift
//  driverappios
//

import Foundation
import Observation
import SwiftUI

@Observable
@MainActor
final class PaymentWaitingViewModel {
    let orderId: String
    let amount: Int
    let driverId: String

    var isSettled = false
    var isFiscalizing = false
    var isFiscalFailed = false
    var isCompleting = false
    var isDone = false
    var requiresCashCollection = false
    var errorMessage: String?

    private let fleetService: FleetServiceProtocol

    init(
        orderId: String,
        amount: Int,
        driverId: String,
        fleetService: FleetServiceProtocol = FleetServiceLive.shared
    ) {
        self.orderId = orderId
        self.amount = amount
        self.driverId = driverId
        self.fleetService = fleetService
    }

    var headerIcon: String {
        if isFiscalFailed { return "exclamationmark.triangle.fill" }
        if isFiscalizing { return "hourglass" }
        if isSettled { return "checkmark.seal.fill" }
        return "clock.fill"
    }

    var headerTint: Color {
        if isFiscalFailed { return LabTheme.destructive }
        if isSettled && !isFiscalizing { return LabTheme.success }
        return LabTheme.warning
    }

    var headerTitle: String {
        if isFiscalFailed { return "Fiscal Failed" }
        if isFiscalizing { return "Fiscalizing" }
        if isSettled { return "Payment Received" }
        return "Awaiting Payment"
    }

    var subtitle: String {
        if isFiscalFailed { return "Retry fiscal receipt or call supervisor for force-complete." }
        if isFiscalizing { return "Payment captured. Waiting for fiscal receipt…" }
        if isSettled { return "Finalizing delivery…" }
        return "Retailer is completing payment..."
    }

    func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
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
            Task { await finalizeDelivery() }
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
                requiresCashCollection = true
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

    func finalizeDelivery() async {
        guard !isDone, !isCompleting else { return }
        isCompleting = true
        errorMessage = nil
        do {
            try await fleetService.completeOrder(orderId: orderId)
            Haptics.success()
            isSettled = true
            isFiscalizing = true
            isFiscalFailed = false
            isCompleting = false
        } catch {
            let message = error.localizedDescription
            if message.localizedCaseInsensitiveContains("COMPLETED") {
                isCompleting = false
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

    func retryFiscal() async {
        isCompleting = true
        errorMessage = nil
        do {
            _ = try await fleetService.retryFiscal(orderId: orderId)
            isFiscalizing = true
            isFiscalFailed = false
            isCompleting = false
        } catch {
            isCompleting = false
            isFiscalFailed = true
            errorMessage = error.localizedDescription
        }
    }

    func handleReconnect() async {
        let hadInFlight = isCompleting
        await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
        if hadInFlight {
            isCompleting = false
            errorMessage = DriverReconnectRecovery.hint
        }
    }
}
