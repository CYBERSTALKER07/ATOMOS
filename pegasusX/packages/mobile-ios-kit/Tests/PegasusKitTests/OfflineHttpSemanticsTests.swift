import XCTest
@testable import PegasusKit

final class OfflineHttpSemanticsTests: XCTestCase {
    func testSuccessIncludesConflict() {
        XCTAssertTrue(OfflineHttpSemantics.isSuccessHTTP(200))
        XCTAssertTrue(OfflineHttpSemantics.isSuccessHTTP(409))
        XCTAssertEqual(OfflineHttpSemantics.outcome(forHTTP: 409), .ack)
    }

    func testRetryable() {
        XCTAssertTrue(OfflineHttpSemantics.isRetryableHTTP(503))
        XCTAssertEqual(OfflineHttpSemantics.outcome(forHTTP: 429), .retry)
    }

    func testNormalize() {
        XCTAssertEqual(OfflineHttpSemantics.normalizeEndpoint("/api/v1/order/deliver"), "v1/order/deliver")
    }

    func testPayloadCoordsMerge() {
        let record = QueuedMutationRecord(
            id: "1",
            endpoint: "v1/order/deliver",
            payloadJSON: #"{"order_id":"o1"}"#,
            idempotencyKey: "1",
            capturedLat: 41.3,
            capturedLng: 69.2
        )
        let merged = record.payloadJSONWithCapturedCoords()
        guard let data = merged.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return XCTFail("merged payload is not JSON object")
        }
        XCTAssertEqual(obj["order_id"] as? String, "o1")
        XCTAssertEqual((obj["latitude"] as? NSNumber)?.doubleValue ?? -1, 41.3, accuracy: 0.0001)
        XCTAssertEqual((obj["longitude"] as? NSNumber)?.doubleValue ?? -1, 69.2, accuracy: 0.0001)
    }
}
