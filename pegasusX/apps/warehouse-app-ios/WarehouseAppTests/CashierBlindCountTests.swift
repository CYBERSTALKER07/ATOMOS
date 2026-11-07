import XCTest
@testable import WarehouseAppIOS

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
}
