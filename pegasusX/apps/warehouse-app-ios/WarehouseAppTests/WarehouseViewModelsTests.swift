//
//  WarehouseViewModelsTests.swift
//  WarehouseAppTests
//

import XCTest
@testable import WarehouseAppIOS

@MainActor
final class WarehouseViewModelsTests: XCTestCase {

    func testOperationsViewModelTemplateFormattingAndPriceIntegrity() throws {
        let vm = OperationsViewModel()
        XCTAssertEqual(vm.broadcastRoles, ["DRIVER", "RETAILER", "ALL"])
        XCTAssertEqual(vm.broadcastRole, "DRIVER")
        XCTAssertTrue(vm.loading)
        XCTAssertFalse(vm.broadcasting)

        let templateJson = """
        {
          "id": "tmpl-1",
          "category": "ops",
          "title": "Depot Maintenance",
          "body": "Maintenance on {date} due to {reason}.",
          "default_role": "DRIVER",
          "scope": "WAREHOUSE",
          "source": "system",
          "warehouse_id": "wh-1",
          "placeholder_keys": ["date", "reason"]
        }
        """.data(using: .utf8)!
        let template = try JSONDecoder().decode(BroadcastTemplate.self, from: templateJson)

        vm.templateDate = "2026-11-15"
        vm.customReason = "conveyor upgrade"
        vm.applyTemplate(template)

        XCTAssertEqual(vm.title, "Depot Maintenance")
        XCTAssertEqual(vm.bodyText, "Maintenance on 2026-11-15 due to conveyor upgrade.")
        XCTAssertEqual(vm.broadcastRole, "DRIVER")

        // Price preview scheduling with zero/negative/empty Int64
        vm.productId = "SKU-001"
        vm.proposedPrice = "-500"
        vm.schedulePreview()
        XCTAssertNil(vm.preview)

        vm.proposedPrice = "12500000" // 125,000 UZS in minor tiyin
        XCTAssertEqual(Int64(vm.proposedPrice), 12500000)
    }

    func testDispatchViewModelStateAndCapacityWarnings() throws {
        let vm = DispatchViewModel()
        XCTAssertEqual(vm.selectedSegment, 0)
        XCTAssertEqual(vm.dispatchMode, .smart)
        XCTAssertFalse(vm.hasActiveManualDispatchLock)
        XCTAssertTrue(vm.selectedOrderIds.isEmpty)

        // Capacity exceeded warning message formatting
        let warningJson = """
        {
          "driver_id": "drv-1",
          "loaded_vu": 120.5,
          "max_volume_vu": 100.0,
          "effective_max_vu": 100.0,
          "excess_vu": 20.5,
          "suggested_unselect_order_ids": ["ord-101", "ord-102"],
          "suggested_defer_order_ids": ["ord-103"]
        }
        """.data(using: .utf8)!
        let warning = try JSONDecoder().decode(DispatchCapacityWarning.self, from: warningJson)

        vm.capacityWarnings = [warning]
        XCTAssertEqual(Set(vm.capacitySuggestedUnselect), Set(["ord-101", "ord-102"]))

        let message = vm.capacityExceededMessage
        XCTAssertTrue(message.contains("120.5 VU loaded / 100.0 VU effective max"))
        XCTAssertTrue(message.contains("Suggested unselect: ord-101, ord-102") || message.contains("Suggested unselect: ord-102, ord-101"))

        // ISO delivery date formatting check
        let now = Date()
        let formatted = DispatchViewModel.dispatchIsoDeliveryDate(from: now)
        XCTAssertFalse(formatted.isEmpty)
        XCTAssertTrue(formatted.contains("T"))
    }

    func testInventoryViewModelDefaults() {
        let vm = InventoryViewModel()
        XCTAssertTrue(vm.loading)
        XCTAssertFalse(vm.lowOnly)
        XCTAssertNil(vm.adjustItem)
        XCTAssertNil(vm.error)
        XCTAssertEqual(vm.policies, ["INHERIT", "REJECT", "ACCEPT_BACKORDER"])
    }

    func testLaborCapacityViewModelValidation() async {
        let vm = LaborCapacityViewModel()
        XCTAssertFalse(vm.date.isEmpty)
        XCTAssertEqual(vm.availHours, "8")
        XCTAssertEqual(vm.availStatus, "AVAILABLE")

        // Score loading without driver ID sets validation error
        vm.driverId = ""
        await vm.loadScore()
        XCTAssertEqual(vm.error, "Driver ID required")

        // Save availability without driver ID sets validation error
        await vm.saveAvailability()
        XCTAssertEqual(vm.error, "Driver ID required")
    }

    func testCashierViewModelDenominationMathAndClamping() {
        let vm = CashierViewModel(
            initialCellId: "",
            initialDepotId: "",
            initialSupplierId: ""
        )
        vm.clearCounts()
        XCTAssertEqual(vm.countedCashMinor, 0)
        XCTAssertEqual(vm.countedCashSom, 0)
        XCTAssertEqual(vm.totalBanknotesCount, 0)

        // DenominationRowItem math (5 * 100,000 so'm = 500,000 so'm = 50,000,000 minor)
        let row = DenominationRowItem(noteSom: 100000, count: 5)
        XCTAssertEqual(row.totalSom, 500000)
        XCTAssertEqual(row.totalMinor, 50000000)

        // Clamping negative count to 0
        vm.setDenominationCount(noteSom: 100000, count: -10)
        XCTAssertEqual(vm.denominations[100000], 0)

        // Count adjustments with bounds
        vm.setDenominationCount(noteSom: 100000, count: 10)
        XCTAssertEqual(vm.denominations[100000], 10)
        vm.adjustCount(noteSom: 100000, delta: -3)
        XCTAssertEqual(vm.denominations[100000], 7)
        vm.adjustCount(noteSom: 100000, delta: -20)
        XCTAssertEqual(vm.denominations[100000], 0) // clamped to 0

        // Strap addition (1 strap = 100 notes)
        vm.addStrap(noteSom: 50000, strapCount: 100)
        XCTAssertEqual(vm.denominations[50000], 100)
        // 100 * 50,000 so'm = 5,000,000 so'm = 500,000,000 minor
        XCTAssertEqual(vm.countedCashSom, 5000000)
        XCTAssertEqual(vm.countedCashMinor, 500000000)
        XCTAssertEqual(vm.totalBanknotesCount, 100)
    }

    func testCashierViewModelCleanDefaultsAndFormatters() {
        let vm = CashierViewModel(
            initialCellId: "",
            initialDepotId: "",
            initialSupplierId: ""
        )

        XCTAssertEqual(vm.driverId, "")
        XCTAssertEqual(vm.routeId, "")
        XCTAssertEqual(vm.bagSealBarcode, "")

        vm.setDenominationCount(noteSom: 200000, count: 7)
        vm.setDenominationCount(noteSom: 20000, count: 1)
        vm.setDenominationCount(noteSom: 5000, count: 1)
        XCTAssertEqual(vm.countedCashMinor, 142500000)
        XCTAssertEqual(vm.totalBanknotesCount, 9)

        let formatted = CashierViewModel.formatSom(amount: 1425000)
        XCTAssertEqual(formatted, "1 425 000 so'm")
        XCTAssertEqual(CashierViewModel.formatWholeSom(minor: 142500000), "1 425 000 so'm")
    }

    func testDispatchViewModelRealtimeBannerAndLockState() throws {
        let vm = DispatchViewModel()

        // Realtime banner
        vm.realtimeStatus = .idle
        XCTAssertNil(vm.realtimeBanner)

        vm.realtimeStatus = .live
        XCTAssertNil(vm.realtimeBanner)

        vm.realtimeStatus = .connecting
        XCTAssertEqual(vm.realtimeBanner?.systemImage, "dot.radiowaves.left.and.right")

        vm.realtimeStatus = .reconnecting
        XCTAssertEqual(vm.realtimeBanner?.systemImage, "arrow.triangle.2.circlepath")

        vm.realtimeStatus = .offline
        XCTAssertEqual(vm.realtimeBanner?.systemImage, "wifi.slash")

        // Dispatch locks state checking
        XCTAssertFalse(vm.hasActiveManualDispatchLock)
        let otherLockJson = """
        {
          "lock_id": "lock-1",
          "supplier_id": "sup-1",
          "warehouse_id": "wh-1",
          "factory_id": "fac-1",
          "lock_type": "AUTO_BATCH",
          "locked_at": "2026-11-11T20:00:00Z",
          "locked_by": "system"
        }
        """.data(using: .utf8)!
        let otherLock = try JSONDecoder().decode(WarehouseDispatchLock.self, from: otherLockJson)
        vm.dispatchLocks = [otherLock]
        XCTAssertFalse(vm.hasActiveManualDispatchLock)

        let manualLockJson = """
        {
          "lock_id": "lock-2",
          "supplier_id": "sup-1",
          "warehouse_id": "wh-1",
          "factory_id": "fac-1",
          "lock_type": "MANUAL_DISPATCH",
          "locked_at": "2026-11-11T20:05:00Z",
          "locked_by": "user-42"
        }
        """.data(using: .utf8)!
        let manualLock = try JSONDecoder().decode(WarehouseDispatchLock.self, from: manualLockJson)
        vm.dispatchLocks.append(manualLock)
        XCTAssertTrue(vm.hasActiveManualDispatchLock)
    }

    func testDispatchViewModelAssignmentModesAndCancellation() {
        let vm = DispatchViewModel()

        XCTAssertEqual(DispatchAssignmentMode.smart.id, "smart")
        XCTAssertEqual(DispatchAssignmentMode.smart.label, "Smart fleet")
        XCTAssertEqual(DispatchAssignmentMode.manual.id, "manual")
        XCTAssertEqual(DispatchAssignmentMode.manual.label, "Manual truck")

        // Cancellable supply states
        XCTAssertTrue(vm.cancellableSupplyStates.contains("DRAFT"))
        XCTAssertTrue(vm.cancellableSupplyStates.contains("SUBMITTED"))
        XCTAssertTrue(vm.cancellableSupplyStates.contains("ACKNOWLEDGED"))
        XCTAssertFalse(vm.cancellableSupplyStates.contains("FULFILLED"))
        XCTAssertFalse(vm.cancellableSupplyStates.contains("CANCELLED"))

        // Alert initialization
        let alert = DispatchActionAlert(title: "Warning", message: "Fleet busy")
        XCTAssertEqual(alert.title, "Warning")
        XCTAssertEqual(alert.message, "Fleet busy")

        // Route identity
        let route1 = DispatchOrderDetailRoute(id: "route-1")
        let route2 = DispatchOrderDetailRoute(id: "route-1")
        XCTAssertEqual(route1, route2)
    }

    func testOperationsViewModelTemplateFallbacksAndRoleValidation() throws {
        let vm = OperationsViewModel()

        let templateJson = """
        {
          "id": "tmpl-fallback",
          "category": "ops",
          "title": "Emergency Announcement",
          "body": "Notice for {date} regarding {reason}.",
          "default_role": "INVALID_ROLE",
          "scope": "ALL",
          "source": "manual",
          "warehouse_id": "wh-1",
          "placeholder_keys": ["date", "reason"]
        }
        """.data(using: .utf8)!
        let template = try JSONDecoder().decode(BroadcastTemplate.self, from: templateJson)

        vm.templateDate = ""
        vm.customReason = ""
        vm.broadcastRole = "RETAILER"

        vm.applyTemplate(template)
        XCTAssertEqual(vm.title, "Emergency Announcement")
        XCTAssertEqual(vm.bodyText, "Notice for the selected date regarding operational delay.")
        // Invalid defaultRole must not override existing valid role
        XCTAssertEqual(vm.broadcastRole, "RETAILER")
    }

    func testInventoryViewModelStateTransitions() throws {
        let vm = InventoryViewModel()
        XCTAssertTrue(vm.loading)
        XCTAssertFalse(vm.lowOnly)
        XCTAssertEqual(vm.policies, ["INHERIT", "REJECT", "ACCEPT_BACKORDER"])

        vm.lowOnly = true
        XCTAssertTrue(vm.lowOnly)

        let itemJson = """
        {
          "product_id": "prod-1",
          "product_name": "Mineral Water 1.5L",
          "quantity": 42,
          "reorder_threshold": 50,
          "out_of_stock_policy": "INHERIT"
        }
        """.data(using: .utf8)!
        let item = try JSONDecoder().decode(InventoryItem.self, from: itemJson)
        vm.items = [item]
        XCTAssertEqual(vm.items.count, 1)
        XCTAssertEqual(vm.items.first?.productId, "prod-1")
        XCTAssertEqual(vm.items.first?.quantity, 42)
        XCTAssertEqual(vm.items.first?.reorderThreshold, 50)
        XCTAssertEqual(vm.items.first?.outOfStockPolicy, "INHERIT")
    }

    func testTransferActionsColdLotWarningsAndPreSealBaseline() async throws {
        let vm = TransferActionsViewModel()
        let binsRaw = """
        {
          "lots_enabled": true,
          "bins": [
            {"warehouse_id": "wh-1", "location_id": "recv-default", "zone": "RECV", "location_type": "STAGE"},
            {"warehouse_id": "wh-1", "location_id": "A-01", "zone": "AMBIENT", "location_type": "PICK"},
            {"warehouse_id": "wh-1", "location_id": "CHILL-01", "zone": "CHILL", "location_type": "PICK"}
          ]
        }
        """.data(using: .utf8)!
        let lotsRaw = """
        {
          "lots_enabled": true,
          "cold_lot_warning_count": 1,
          "migration_staging_count": 1,
          "lots": [
            {
              "lot_id": "lot-cold-mig",
              "warehouse_id": "wh-1",
              "product_id": "sku-milk",
              "product_name": "Kefir 1L",
              "location_id": "recv-default",
              "lot_code": "UZ-01",
              "expiry_date": "2026-11-20",
              "quantity_on_hand": 18,
              "quantity_reserved": 0,
              "status": "AVAILABLE",
              "handling_class": "CHILLED",
              "requires_cold_chain": true,
              "required_zone": "CHILL",
              "cold_lot_warning": "cold_lot_migrated_without_live_baseline"
            }
          ]
        }
        """.data(using: .utf8)!
        let binsResp = try JSONDecoder().decode(WarehouseBinListResponse.self, from: binsRaw)
        let lotsResp = try JSONDecoder().decode(StockLotListResponse.self, from: lotsRaw)

        vm.listBinsOp = { binsResp }
        vm.listLotsOp = { lotsResp }
        await vm.refreshBinsAndLots()

        XCTAssertTrue(vm.hasColdLotOrStagingAlert)
        XCTAssertEqual(vm.coldWarningLots.count, 1)
        XCTAssertEqual(vm.stagingPutawayLots.count, 1)

        // Selecting an ambient bin for a CHILL SKU triggers zone mismatch warning
        vm.putawayProduct = "sku-milk"
        vm.putawayLocation = "A-01"
        XCTAssertEqual(vm.putawayZoneMismatchRequiredZone, "CHILL")

        // Stage helper populates form and recommends CHILL-01
        if let lot = vm.coldWarningLots.first {
            vm.stagePutawayLot(lot)
        }
        XCTAssertEqual(vm.putawayLocation, "CHILL-01")
        XCTAssertNil(vm.putawayZoneMismatchRequiredZone)

        // Pick wave with missing live temperature baseline triggers pre-seal alert
        let waveRaw = """
        {
          "wave_id": "wave-cold",
          "warehouse_id": "wh-1",
          "manifest_id": "man-cold-1",
          "status": "READY_TO_SEAL",
          "cold_lot_migrated_without_live_baseline": true,
          "cold_chain_baseline_missing": true,
          "tasks": [
            {
              "task_id": "t-1",
              "order_id": "o-1",
              "product_id": "sku-milk",
              "lot_id": "lot-cold-mig",
              "location_id": "recv-default",
              "quantity_requested": 4,
              "quantity_picked": 4,
              "status": "CONFIRMED",
              "pick_sequence": 1
            }
          ]
        }
        """.data(using: .utf8)!
        let wave = try JSONDecoder().decode(PickWave.self, from: waveRaw)
        vm.activeWave = wave
        XCTAssertTrue(vm.activeWavePreSealWarning)
    }
}
