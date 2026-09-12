import Foundation
import Testing
@testable import retailerapp

struct DeviceTokenPayloadTests {
    @Test func encodesTokenAndPlatformOnly() throws {
        let payload = DeviceTokenPayload(token: "fcm-reg-token", platform: "ios")
        let data = try JSONEncoder().encode(payload)
        let json = try JSONSerialization.jsonObject(with: data) as? [String: String]
        #expect(json?["token"] == "fcm-reg-token")
        #expect(json?["platform"] == "ios")
        #expect(json?["retailer_id"] == nil)
    }
}
