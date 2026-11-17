//
//  HardenedViewModelsTests.swift
//  driverappiosTests
//

import Testing
import Foundation
import SwiftData
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

        // Fiscal failed event -> non-blocking statusNotice, no blocking red error
        let failEvent = DriverSocketState.DriverEvent(
            type: "FISCAL_RECEIPT_FAILED",
            orderId: "ORD-123",
            status: "FISCAL_FAILED"
        )
        vm.handleSocketEvent(failEvent)
        #expect(vm.phase == .fiscalFailed)
        #expect(vm.errorMessage == nil)
        #expect(vm.statusNotice != nil)
        #expect(vm.statusNotice?.localizedCaseInsensitiveContains("call supervisor") == false)

        // Driver can immediately continue to next stop
        vm.continueNextStop()
        #expect(vm.phase == .done)
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

    // MARK: - W5 Non-Blocking Fiscal, Safe SyncQueue Escalation & Locale Tests

    @Test func driverLocaleManager_defaultsToUzbekAndSwitchesLanguages() {
        #expect(DriverLocaleManager.defaultLanguage == "uz")
        #expect(DriverLocaleManager.normalizeCode(nil) == "uz")
        #expect(DriverLocaleManager.normalizeCode("ru-RU") == "ru")
        #expect(DriverLocaleManager.normalizeCode(" EN ") == "en")
        #expect(DriverLocaleManager.normalizeCode("fr") == "uz")
        #expect(DriverLocaleManager.supportedLanguages.map(\.code) == ["uz", "ru", "en"])

        let suiteName = "w5.locale.\(UUID().uuidString)"
        let isolatedDefaults = UserDefaults(suiteName: suiteName)!
        defer { isolatedDefaults.removePersistentDomain(forName: suiteName) }

        let manager = DriverLocaleManager(defaults: isolatedDefaults)
        #expect(manager.currentCode == "uz")
        manager.setLanguage("ru")
        #expect(manager.currentCode == "ru")
        #expect(isolatedDefaults.string(forKey: DriverLocaleManager.storageKey) == "ru")

        for code in ["uz", "ru", "en"] {
            let body = L10n.string("mobile_driver.ui.fiscal_retry_queued_body", localeCode: code)
            #expect(!body.isEmpty)
            #expect(!body.localizedCaseInsensitiveContains("call supervisor"))
        }
    }

    @Test func openFiscalResponse_retryQueuedDoesNotFreezeCashBag() throws {
        let json = """
        {
          "open_fiscal_count": 3,
          "retry_queued_count": 3,
          "fiscal_issue_count": 0,
          "order_ids": ["ord-1", "ord-2", "ord-3"],
          "retry_order_ids": ["ord-1", "ord-2", "ord-3"],
          "cash_bag_frozen": false
        }
        """
        let snap = try JSONDecoder().decode(APIClient.OpenFiscalResponse.self, from: Data(json.utf8))
        #expect(snap.openFiscalCount == 3)
        #expect(snap.retryQueuedCount == 3)
        #expect(snap.fiscalIssueCount == 0)
        #expect(snap.retryOrderIds?.count == 3)
        #expect(snap.cashBagFrozen == false)
    }

    @Test func paymentWaitingViewModel_nonBlockingFiscalRetryQueuedAllowsContinueNextStop() {
        let vm = PaymentWaitingViewModel(orderId: "ORD-W5", amount: 120_000_00, driverId: "DRV-5")

        vm.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(
                type: "ORDER_STATUS_CHANGED",
                orderId: "ORD-W5",
                status: "FISCAL_RETRY_QUEUED"
            )
        )
        #expect(vm.isSettled == true)
        #expect(vm.isFiscalizing == false)
        #expect(vm.isFiscalFailed == true)
        #expect(vm.errorMessage == nil)
        #expect(!vm.subtitle.localizedCaseInsensitiveContains("call supervisor"))

        vm.continueNextStop()
        #expect(vm.isDone == true)

        let issueVM = PaymentWaitingViewModel(orderId: "ORD-W6", amount: 90_000_00, driverId: "DRV-5")
        issueVM.handleDriverSocketEvent(
            DriverSocketState.DriverEvent(
                type: "ORDER_STATUS_CHANGED",
                orderId: "ORD-W6",
                status: "PENDING_FISCAL_ISSUE"
            )
        )
        #expect(issueVM.isFiscalFailed == true)
        #expect(issueVM.errorMessage == nil)
    }

    @Test func syncQueueViewModel_safeSupportEscalationAndPerItemRetry() async throws {
        let config = ModelConfiguration(isStoredInMemoryOnly: true)
        let container = try ModelContainer(for: OfflineDelivery.self, QueuedDriverAction.self, configurations: config)
        let queue = DriverOfflineQueue.shared
        queue.attach(container: container)

        let leftover = queue.dead() + queue.pending()
        if !leftover.isEmpty {
            queue.deleteActions(leftover)
        }

        let context = ModelContext(container)
        let dead1 = QueuedDriverAction(
            id: "dead-w5-1",
            endpoint: "v1/order/collect-cash",
            method: "POST",
            bodyJSON: "{\"amount_received_minor\":5000000}",
            priority: 20,
            clientTimestampIso: "2026-11-13T01:00:00Z",
            orderId: "ORD-901",
            attemptCount: 5,
            lastError: "HTTP 503",
            status: "DEAD"
        )
        let dead2 = QueuedDriverAction(
            id: "dead-w5-2",
            endpoint: "v1/order/deliver",
            method: "POST",
            bodyJSON: "{\"order_id\":\"ORD-902\"}",
            priority: 30,
            clientTimestampIso: "2026-11-13T01:05:00Z",
            orderId: "ORD-902",
            attemptCount: 5,
            lastError: "timeout",
            status: "DEAD"
        )
        context.insert(dead1)
        context.insert(dead2)
        try context.save()

        var flushCalls = 0
        let failingVM = SyncQueueViewModel(
            queue: queue,
            escalateDeadLetters: { _ in
                throw URLError(.notConnectedToInternet)
            },
            flushHandler: {
                flushCalls += 1
                return (0, queue.pending().count)
            }
        )
        failingVM.reload()
        #expect(failingVM.dead.count == 2)

        await failingVM.escalateDeadToSupport()
        #expect(failingVM.dead.count == 2)
        #expect(failingVM.status?.localizedCaseInsensitiveContains("kept safely") == true)

        await failingVM.retryDeadItem(dead1)
        #expect(flushCalls == 1)
        #expect(failingVM.pending.count == 1)
        #expect(failingVM.dead.count == 1)

        var capturedItems: [APIClient.SyncEscalationItemRequest] = []
        let successVM = SyncQueueViewModel(
            queue: queue,
            escalateDeadLetters: { items in
                #expect(queue.dead().count == 1)
                capturedItems = items
                return APIClient.SyncEscalationResponse(ok: true, escalated: items.count, escalationId: "esc-ios-1")
            },
            flushHandler: { (0, 0) }
        )
        await successVM.escalateDeadToSupport()
        #expect(capturedItems.map(\.id) == ["dead-w5-2"])
        #expect(capturedItems.first?.orderId == "ORD-902")
        #expect(successVM.dead.isEmpty)
        #expect(successVM.status?.contains("esc-ios-1") == true)

        queue.deleteActions(queue.pending() + queue.dead())
    }
}
