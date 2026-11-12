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

    func testDeviceTokenKeyStaysWithTheToken() {
        let first = SupplierIdempotencyKeys.deviceToken(platform: "ios", token: "fcm-1")
        XCTAssertEqual(first, SupplierIdempotencyKeys.deviceToken(platform: "ios", token: "fcm-1"))
        XCTAssertNotEqual(first, SupplierIdempotencyKeys.deviceToken(platform: "ios", token: "fcm-2"))
        XCTAssertFalse(first.contains("fcm-1"))
    }
}
