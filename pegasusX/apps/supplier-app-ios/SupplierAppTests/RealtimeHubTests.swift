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
    
    func testRealtimeHubBumps() {
        let hub = SupplierRealtimeHub()
        
        XCTAssertEqual(hub.refreshEpoch, 0)
        XCTAssertEqual(hub.reconnectEpoch, 0)
        
        hub.bump()
        XCTAssertEqual(hub.refreshEpoch, 1)
        
        hub.bumpReconnect()
        XCTAssertEqual(hub.reconnectEpoch, 1)
    }
}
