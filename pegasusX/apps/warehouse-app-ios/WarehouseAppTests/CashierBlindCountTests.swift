import XCTest
@testable import WarehouseAppIOS

@MainActor
final class CashierBlindCountTests: XCTestCase {

    func testDenominationTiyinArithmeticAndBlindMasking() {
        let vm = CashierViewModel()
        XCTAssertTrue(vm.isBlindCountMasked)
        XCTAssertEqual(vm.countedCashMinor, 0)
        XCTAssertEqual(vm.amlCeilingMinor, 0) // Default uncapped

        // 5 notes of 200,000 UZS = 1,000,000 UZS = 100,000,000 Tiyin
        // 3 notes of 100,000 UZS = 300,000 UZS = 30,000,000 Tiyin
        vm.setDenominationCount(noteUzs: 200000, count: 5)
        vm.setDenominationCount(noteUzs: 100000, count: 3)
        XCTAssertEqual(vm.countedCashMinor, 130_000_000)

        // Negative counts clamp to zero
        vm.setDenominationCount(noteUzs: 50000, count: -4)
        XCTAssertEqual(vm.denominations[50000], 0)
    }

    func testCashierSessionViewMasksExpectedCashUntilBalanced() throws {
        let json = """
        {
          "session_id": "sess-01",
          "supplier_id": "sup-01",
          "depot_id": "depot-01",
          "driver_id": "drv-01",
          "route_id": "route-01",
          "bag_seal_barcode": "BAG-01",
          "attempt_number": 1,
          "counted_cash_minor": 0,
          "status": "BLIND_COUNT_PENDING",
          "created_at": "2026-10-30T12:00:00Z"
        }
        """
        guard let data = json.data(using: .utf8) else {
            XCTFail("Failed to encode UTF-8 test payload")
            return
        }
        let session = try JSONDecoder().decode(CashierSessionView.self, from: data)
        XCTAssertNil(session.expectedCashMinor)
        XCTAssertNil(session.varianceMinor)
        XCTAssertEqual(session.status, "BLIND_COUNT_PENDING")
    }

    func testStrapBundlesAndTotalBanknoteCount() {
        let vm = CashierViewModel()
        XCTAssertEqual(vm.totalBanknotesCount, 0)

        // 10-strap bundle of 200k + 50-strap bundle of 100k
        vm.addStrap(noteUzs: 200000, strapCount: 10)
        vm.addStrap(noteUzs: 100000, strapCount: 50)

        XCTAssertEqual(vm.totalBanknotesCount, 60)
        // 10 * 200,000 = 2,000,000 UZS + 50 * 100,000 = 5,000,000 UZS = 7,000,000 UZS = 700,000,000 Tiyin
        XCTAssertEqual(vm.countedCashMinor, 700_000_000)
    }

    func testFormatUzsThousandsGrouping() {
        let formatted = CashierViewModel.formatUzs(amount: 1425000)
        XCTAssertTrue(formatted.contains("1 425 000") || formatted.contains("1,425,000"), "Currency must format with thousands grouping")
        XCTAssertTrue(formatted.hasSuffix("UZS"))
    }

    func testQuickSamplePresets() {
        let vm = CashierViewModel()
        vm.quickFillSampleBalanced()
        XCTAssertEqual(vm.totalBanknotesCount, 9)
        XCTAssertEqual(vm.countedCashMinor, 142_500_000) // 1,425,000 UZS

        vm.quickFillSampleDiscrepant()
        XCTAssertEqual(vm.totalBanknotesCount, 6)
        XCTAssertEqual(vm.countedCashMinor, 120_000_000) // 1,200,000 UZS
    }
}
