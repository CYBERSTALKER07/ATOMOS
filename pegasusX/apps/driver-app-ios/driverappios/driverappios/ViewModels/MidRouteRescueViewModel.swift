//
//  MidRouteRescueViewModel.swift
//  driverappios
//

import Foundation
import Observation

@Observable
@MainActor
final class MidRouteRescueViewModel {

    var originalRouteId: String = "route-tashkent-042"
    var brokenVehicleId: String = "vh-isuzu-01"
    var originalDriverId: String = "driver-uz-09"
    var rescueVehicleId: String = "vh-isuzu-04"
    var rescueDriverId: String = "driver-uz-12"
    var lastCompletedStopSeq: Int64 = 2
    var newRescueBoltSeal: String = "SEAL-RESCUE-8812"

    var totes: [RescueToteItem] = [
        RescueToteItem(toteId: "tote-01", stopSequence: 1, isColdChain: false, expectedCashMinor: 50000000),
        RescueToteItem(toteId: "tote-02", stopSequence: 2, isColdChain: false, expectedCashMinor: 75000000),
        RescueToteItem(toteId: "tote-03", stopSequence: 3, isColdChain: true, expectedCashMinor: 120000000, thermalExcursionMinutes: 45, maxAllowedExcursionMins: 30),
        RescueToteItem(toteId: "tote-04", stopSequence: 4, isColdChain: false, expectedCashMinor: 60000000),
        RescueToteItem(toteId: "tote-05", stopSequence: 5, isColdChain: true, expectedCashMinor: 90000000, thermalExcursionMinutes: 10, maxAllowedExcursionMins: 30),
    ]

    var isExecuting: Bool = false
    var rescueResult: MidRouteRescueResult? = nil
    var statusMessage: String? = nil
    var errorMessage: String? = nil

    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    var originalDriverLiability: Int64 {
        totes.filter { $0.stopSequence <= lastCompletedStopSeq }.reduce(0) { $0 + $1.expectedCashMinor }
    }

    var remainingTotes: [RescueToteItem] {
        totes.filter { $0.stopSequence > lastCompletedStopSeq }
    }

    var spoiledTotes: [RescueToteItem] {
        remainingTotes.filter { $0.isColdChain && $0.thermalExcursionMinutes > $0.maxAllowedExcursionMins }
    }

    var transferredTotes: [RescueToteItem] {
        remainingTotes.filter { !($0.isColdChain && $0.thermalExcursionMinutes > $0.maxAllowedExcursionMins) }
    }

    var rescueDriverLiability: Int64 {
        transferredTotes.reduce(0) { $0 + $1.expectedCashMinor }
    }

    func executeRescue() async {
        isExecuting = true
        errorMessage = nil
        statusMessage = nil

        let req = MidRouteRescueRequest(
            cellId: "cell-uz",
            supplierId: "supplier-tashkent-1",
            originalRouteId: originalRouteId,
            brokenVehicleId: brokenVehicleId,
            originalDriverId: originalDriverId,
            rescueVehicleId: rescueVehicleId,
            rescueDriverId: rescueDriverId,
            lastCompletedStopSeq: lastCompletedStopSeq,
            newRescueBoltSeal: newRescueBoltSeal,
            totes: totes
        )

        do {
            let res = try await apiClient.executeMidRouteRescue(request: req)
            isExecuting = false
            rescueResult = res
            statusMessage = "Rescue Route \(res.rescueRouteId) Created! Transferred \(res.transferredToteIds.count) totes. Liability sealed."
        } catch {
            isExecuting = false
            errorMessage = error.localizedDescription
        }
    }
}
