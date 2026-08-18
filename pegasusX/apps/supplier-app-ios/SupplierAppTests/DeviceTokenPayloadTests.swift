import XCTest
@testable import SupplierAppIOS

final class DeviceTokenPayloadTests: XCTestCase {
    func testEncodesTokenAndPlatformOnly() throws {
        let payload = DeviceTokenRequest(token: "fcm-reg-token", platform: "ios")
        let data = try JSONEncoder().encode(payload)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: String]
        XCTAssertEqual(json?["token"], "fcm-reg-token")
        XCTAssertEqual(json?["platform"], "ios")
        XCTAssertNil(json?["supplier_id"])
    }
}
