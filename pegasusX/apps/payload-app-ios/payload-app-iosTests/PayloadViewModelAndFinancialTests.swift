//
//  PayloadViewModelAndFinancialTests.swift
//  payload-app-iosTests
//

import XCTest
@testable import payload_app_ios

@MainActor
final class PayloadViewModelAndFinancialTests: XCTestCase {

    func testInboundReturnsViewModelInitialState() {
        let vm = InboundReturnsViewModel()
        XCTAssertEqual(vm.tab, .queue)
        XCTAssertTrue(vm.rows.isEmpty)
        XCTAssertTrue(vm.history.isEmpty)
        XCTAssertTrue(vm.loading)
        XCTAssertNil(vm.error)
        XCTAssertEqual(vm.barcode, "")
        XCTAssertFalse(vm.scanning)
        XCTAssertTrue(vm.selected.isEmpty)
    }

    func testInboundReturnsViewModelSelectionAndToggle() {
        let vm = InboundReturnsViewModel(offlineQueue: OfflineQueue(inMemory: true))
        vm.toggleSelect("ret-101")
        XCTAssertTrue(vm.selected.contains("ret-101"))

        vm.toggleSelect("ret-101")
        XCTAssertFalse(vm.selected.contains("ret-101"))
    }

    func testInboundReturnsOfflineScanQueuesDeterministically() async {
        let queue = OfflineQueue(inMemory: true)
        let vm = InboundReturnsViewModel(offlineQueue: queue)
        vm.sessionId = "sess-55"
        vm.barcode = " 4780001234567 "

        await vm.handleScan(online: false)

        XCTAssertEqual(vm.barcode, "")
        XCTAssertEqual(vm.statusMessage, "Scan queued (offline)")
        XCTAssertEqual(vm.queuedScans, 1)
        let stored = queue.read()
        XCTAssertEqual(stored.count, 1)
        XCTAssertEqual(stored.first?.endpoint, "v1/returns/inbound/scan")
    }

    func testExceptionsViewModelInitialStateAndEscalation() {
        let vm = ExceptionsViewModel()
        XCTAssertTrue(vm.exceptions.isEmpty)
        XCTAssertFalse(vm.loading)
        XCTAssertNil(vm.error)
        XCTAssertFalse(vm.isEscalatedToDlq)

        vm.overflowAttempts = 2
        XCTAssertFalse(vm.isEscalatedToDlq)

        vm.overflowAttempts = 3
        XCTAssertTrue(vm.isEscalatedToDlq)
    }

    func testInt64MinorUnitsFinancialArithmeticZeroFloat() {
        let item1 = LiveOrderItem(
            lineItemId: "li-1",
            skuId: "sku-water",
            skuName: "Mineral Water 1L",
            quantity: 5,
            unitPrice: 1_500_000,
            status: "CHECKED"
        )
        XCTAssertEqual(item1.totalTiyin, 7_500_000)

        let item2 = LiveOrderItem(
            lineItemId: "li-2",
            skuId: "sku-bread",
            skuName: "Tandir Bread",
            quantity: 12,
            unitPrice: 850_000,
            status: "PENDING"
        )
        XCTAssertEqual(item2.totalTiyin, 10_200_000)

        let order = LiveOrder(
            orderId: "ord-test-99",
            retailerId: "ret-1",
            amount: 17_700_000,
            paymentGateway: "HUMO",
            state: "LOADING",
            routeId: "route-1",
            warehouseId: "wh-1",
            items: [item1, item2]
        )
        XCTAssertEqual(order.calculatedTotalTiyin, 17_700_000)
    }

    func testCurrencyFormattingWithoutFloatingPoint() {
        let formattedWhole = CurrencyFormat.formatTiyin(1_500_000, currency: "UZS")
        XCTAssertTrue(formattedWhole.contains("15 000") || formattedWhole.contains("15,000") || formattedWhole.contains("15000"))
        XCTAssertTrue(formattedWhole.hasSuffix("UZS"))

        let formattedFraction = CurrencyFormat.formatTiyin(1_500_045, currency: "UZS")
        XCTAssertTrue(formattedFraction.contains(".45"))
    }

    func testFormatPackMoneyIntegerArithmeticEdgeCases() {
        let uzPack = MarketPack(
            code: "UZ",
            name: "Uzbekistan",
            timezone: "Asia/Tashkent",
            currencyCode: "uzs",
            fiscalAdapter: "MY_SOLIQ",
            mapsAdapter: "MAPBOX",
            mapCenterLat: 41.3111,
            mapCenterLng: 69.2797,
            checkoutReadsThis: true
        )
        XCTAssertEqual(formatPackMoney(1_500_000, pack: uzPack, decimalPlaces: 2), "15 000 UZS")
        XCTAssertEqual(formatPackMoney(1_500_050, pack: uzPack, decimalPlaces: 2), "15 000.5 UZS")
        XCTAssertEqual(formatPackMoney(1_500_005, pack: uzPack, decimalPlaces: 2), "15 000.05 UZS")
        XCTAssertEqual(formatPackMoney(-250_099, pack: uzPack, decimalPlaces: 2), "-2 500.99 UZS")
        // Empty pack must never invent UZS
        XCTAssertEqual(formatPackMoney(123_45, pack: nil, decimalPlaces: 2), "123.45")
        // Large institutional settlement (> 90 billion UZS in tiyin) without float drift
        XCTAssertEqual(formatPackMoney(9_007_199_254_740_993, pack: uzPack, decimalPlaces: 2), "90 071 992 547 409.93 UZS")
    }
}
