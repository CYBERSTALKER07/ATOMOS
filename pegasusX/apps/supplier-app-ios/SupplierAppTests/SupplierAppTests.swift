import XCTest

final class SupplierAppTests: XCTestCase {
    func testMoneyFormatMinorUnits() {
        let formatted = MoneyFormat.minor(150_000, currency: "UZS")
        XCTAssertTrue(formatted.contains("1500"))
        XCTAssertTrue(formatted.contains("UZS"))
    }
}
