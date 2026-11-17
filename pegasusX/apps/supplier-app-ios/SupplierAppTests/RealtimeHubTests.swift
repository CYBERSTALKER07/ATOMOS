import XCTest
@testable import SupplierAppIOS

final class RealtimeHubTests: XCTestCase {
    
    func testSupplierLiveEventDecoding() throws {
        let json = """
        {
            "type": "ORDER_UPDATED",
            "minimum_version": "1.0.0"
        }
        """
        
        let data = Data(json.utf8)
        let event = try JSONDecoder().decode(SupplierLiveEvent.self, from: data)
        
        XCTAssertEqual(event.type, "ORDER_UPDATED")
        XCTAssertEqual(event.minimum_version, "1.0.0")
    }
    
    @MainActor
    func testRealtimeHubBumps() {
        let hub = SupplierRealtimeHub()
        
        XCTAssertEqual(hub.refreshEpoch, 0)
        XCTAssertEqual(hub.reconnectEpoch, 0)
        
        hub.bump()
        XCTAssertEqual(hub.refreshEpoch, 1)
        
        hub.bumpReconnect()
        XCTAssertEqual(hub.reconnectEpoch, 1)
    }

    func testWebSocketURLSkipsAMalformedHost() {
        XCTAssertNil(supplierWebSocketURL(base: "%%%", sessionToken: "tok", version: "1", lastEventId: nil))
        XCTAssertNil(supplierWebSocketURL(base: "not a host", sessionToken: "tok", version: "1", lastEventId: nil))
    }

    func testWebSocketURLUsesWSSAndKeepsTheResumeCursor() throws {
        let url = try XCTUnwrap(
            supplierWebSocketURL(
                base: "https://api.pegasusx.app",
                sessionToken: "tok",
                version: "1.2.3",
                lastEventId: "evt_9"
            )
        )
        XCTAssertEqual(url.scheme, "wss")
        XCTAssertEqual(url.host, "api.pegasusx.app")
        XCTAssertEqual(url.path, "/v1/ws")
        let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems ?? []
        XCTAssertTrue(items.contains(URLQueryItem(name: "token", value: "tok")))
        XCTAssertTrue(items.contains(URLQueryItem(name: "platform", value: "ios")))
        XCTAssertTrue(items.contains(URLQueryItem(name: "version", value: "1.2.3")))
        XCTAssertTrue(items.contains(URLQueryItem(name: "last_event_id", value: "evt_9")))
    }
}
