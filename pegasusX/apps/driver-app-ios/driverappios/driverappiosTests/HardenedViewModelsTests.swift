//
//  HardenedViewModelsTests.swift
//  driverappiosTests
//

import Testing
import Foundation
@testable import driverappios

@MainActor
struct HardenedViewModelsTests {

    // MARK: - YardGateViewModel Tests

    @Test func yardGateViewModel_initialStateAndDirectionSlotUpdate() {
        let vm = YardGateViewModel()
        #expect(vm.direction == "GATE_OUT")
        #expect(vm.sealIntact == true)
        #expect(vm.assignedBaySlot == "OUTBOUND_LOADING")

        vm.direction = "GATE_IN"
        vm.updateBaySlot()
        #expect(vm.assignedBaySlot == "BAY_3_INSPECTION")

        vm.sealIntact = false
        vm.updateBaySlot()
        #expect(vm.assignedBaySlot == "SECURITY_QUARANTINE_BAY")

        vm.direction = "GATE_OUT"
        vm.updateBaySlot()
        #expect(vm.assignedBaySlot == "SECURITY_QUARANTINE_BAY", "Broken seal always quarantines regardless of gate direction")
    }

    // MARK: - MidRouteRescueViewModel Tests

    @Test func midRouteRescueViewModel_liabilityAndThermalExcursionMath() {
        let vm = MidRouteRescueViewModel()
        vm.lastCompletedStopSeq = 2

        // Stops <= 2 are original driver liability: 50_000_000 + 75_000_000 = 125_000_000
        #expect(vm.originalDriverLiability == 125_000_000)

        // Stops > 2 are remaining: tote-03 (120M, cold, 45m > 30m max -> spoiled), tote-04 (60M, ok), tote-05 (90M, cold, 10m <= 30m -> transferred)
        #expect(vm.remainingTotes.count == 3)
        #expect(vm.spoiledTotes.count == 1)
        #expect(vm.spoiledTotes.first?.toteId == "tote-03")
        #expect(vm.transferredTotes.count == 2)

        // Rescue driver liability is sum of transferred: 60_000_000 + 90_000_000 = 150_000_000
        #expect(vm.rescueDriverLiability == 150_000_000)
    }

    // MARK: - CashCollectionViewModel Tests

    @Test func cashCollectionViewModel_financialMathAndShortfall() {
        let vm = CashCollectionViewModel(orderId: "ORD-999", amount: 100_000_00) // 100,000 tiyin
        #expect(vm.shortfallNote == nil)

        // Exact payment
        vm.amountReceivedText = "10000000"
        vm.refreshShortfallNote()
        #expect(vm.shortfallNote == nil)

        // Shortfall (underpayment)
        vm.amountReceivedText = "8000000"
        vm.refreshShortfallNote()
        #expect(vm.shortfallNote?.contains("Shortfall") == true)

        // Overage (overpayment)
        vm.amountReceivedText = "12000000"
        vm.refreshShortfallNote()
        #expect(vm.shortfallNote?.contains("Overage") == true)
    }

    @Test func cashCollectionViewModel_socketEvents() {
        let vm = CashCollectionViewModel(orderId: "ORD-123", amount: 50_000)
        #expect(vm.phase == .collect)

        // Order completed event
        let completeEvent = DriverSocketState.DriverEvent(
            type: "ORDER_COMPLETED",
            orderId: "ORD-123",
            status: "COMPLETED"
        )
        vm.handleSocketEvent(completeEvent)
        #expect(vm.phase == .done)

        // Fiscal failed event
        let failEvent = DriverSocketState.DriverEvent(
            type: "FISCAL_RECEIPT_FAILED",
            orderId: "ORD-123",
            status: "FISCAL_FAILED"
        )
        vm.handleSocketEvent(failEvent)
        #expect(vm.phase == .fiscalFailed)
        #expect(vm.errorMessage != nil)
    }

    // MARK: - SupplyTransfersViewModel Tests

    @Test func supplyTransfersViewModel_statePredicates() {
        let vm = SupplyTransfersViewModel()
        let inTransit = SupplyTransferRow(
            transferId: "tr-001",
            warehouseId: "wh-001",
            supplyRequestId: "req-01",
            state: "IN_TRANSIT",
            totalVolumeVu: 12.5
        )
        let arrived = SupplyTransferRow(
            transferId: "tr-002",
            warehouseId: "wh-001",
            supplyRequestId: "req-02",
            state: "ARRIVED",
            totalVolumeVu: 12.5
        )

        #expect(vm.canArrive(inTransit) == true)
        #expect(vm.canArrive(arrived) == false)

        vm.transfers = [inTransit, arrived]
        #expect(vm.activeCount == 1)
    }

    // MARK: - FleetDVIRViewModel Tests

    @Test func fleetDVIRViewModel_checklistToggles() {
        let vm = FleetDVIRViewModel()
        #expect(vm.checklist.steeringAndHornOk == true)
        #expect(vm.hasCriticalDefect == false)

        vm.toggleSteeringAndHorn()
        #expect(vm.checklist.steeringAndHornOk == false)
        #expect(vm.hasCriticalDefect == true)
        #expect(vm.failedPoints.contains("steering_and_horn") == true)
    }

    // MARK: - Observation Migration Tests

    @Test func networkLifecycleEngine_observationStateTransitions() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()
        #expect(engine.isOnline == true)
        #expect(engine.state == .connected)

        engine.setOnlineForTesting(false)
        #expect(engine.isOnline == false)
        #expect(engine.state == .offline)

        let initialEpoch = engine.syncEpoch
        await engine.triggerSyncEpoch()
        #expect(engine.syncEpoch == initialEpoch + 1)
    }
}
