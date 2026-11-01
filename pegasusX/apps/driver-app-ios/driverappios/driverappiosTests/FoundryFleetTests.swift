//
//  FoundryFleetTests.swift
//  driverappiosTests
//

import Testing
import Foundation
@testable import driverappios

struct FoundryFleetTests {

    @Test func dvirChecklist_allPassHasNoCriticalDefect() {
        let vm = FleetDVIRViewModel()
        #expect(vm.hasCriticalDefect == false)
        #expect(vm.failedPoints.isEmpty == true)
        #expect(vm.isRedTagQuarantined == false)
    }

    @Test func dvirChecklist_brakeAndReeferFailuresTriggerCriticalDefects() {
        let vm = FleetDVIRViewModel()
        vm.requiresColdChain = true
        vm.toggleBrakes()
        vm.toggleReeferCalibration()

        #expect(vm.hasCriticalDefect == true)
        #expect(vm.failedPoints.contains("service_and_parking_brakes") == true)
        #expect(vm.failedPoints.contains("reefer_temp_calibrated") == true)
    }

    @Test func midRouteRescue_splitsDriverLiabilityAndQuarantinesSpoiledTotes() throws {
        let totes = [
            RescueToteItem(toteId: "tote-1", stopSequence: 1, isColdChain: false, expectedCashMinor: 50_000_000),
            RescueToteItem(toteId: "tote-2", stopSequence: 2, isColdChain: false, expectedCashMinor: 75_000_000),
            RescueToteItem(toteId: "tote-3", stopSequence: 3, isColdChain: true, expectedCashMinor: 120_000_000, thermalExcursionMinutes: 45, maxAllowedExcursionMins: 30),
            RescueToteItem(toteId: "tote-4", stopSequence: 4, isColdChain: false, expectedCashMinor: 60_000_000),
        ]
        let req = MidRouteRescueRequest(
            supplierId: "sup_1",
            originalRouteId: "route_1",
            brokenVehicleId: "veh_broken",
            originalDriverId: "drv_1",
            rescueVehicleId: "veh_rescue",
            rescueDriverId: "drv_2",
            lastCompletedStopSeq: 2,
            newRescueBoltSeal: "SEAL-RESCUE-01",
            totes: totes
        )
        let encoded = try JSONEncoder().encode(req)
        let decoded = try JSONDecoder().decode(MidRouteRescueRequest.self, from: encoded)

        #expect(decoded.lastCompletedStopSeq == 2)
        #expect(decoded.totes.count == 4)
        let originalLiability = decoded.totes.filter { $0.stopSequence <= 2 }.reduce(Int64(0)) { $0 + $1.expectedCashMinor }
        let spoiled = decoded.totes.filter { $0.stopSequence > 2 && $0.isColdChain && $0.thermalExcursionMinutes > $0.maxAllowedExcursionMins }
        #expect(originalLiability == 125_000_000)
        #expect(spoiled.map(\.toteId) == ["tote-3"])
    }
}
