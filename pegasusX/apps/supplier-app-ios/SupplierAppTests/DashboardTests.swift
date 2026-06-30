import XCTest
@testable import SupplierApp

final class DashboardTests: XCTestCase {
    
    override func setUp() {
        super.setUp()
        APIClient.shared.setTestSession(MockURLProtocol.createMockSession())
    }
    
    override func tearDown() {
        MockURLProtocol.requestHandler = nil
        super.tearDown()
    }
    
    func testDashboardParsing() async throws {
        let json = """
        {
            "pending_orders": 42,
            "inventory_skus": 12,
            "is_configured": true,
            "updated_at": "2026-06-30T10:00:00Z"
        }
        """
        
        MockURLProtocol.requestHandler = { request in
            XCTAssertEqual(request.url?.path, "/v1/supplier/dashboard")
            let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(json.utf8))
        }
        
        let dashboard = try await SupplierService.dashboard()
        
        XCTAssertEqual(dashboard.pendingOrders, 42)
        XCTAssertEqual(dashboard.inventorySKUs, 12)
        XCTAssertTrue(dashboard.isConfigured)
        XCTAssertEqual(dashboard.updatedAt, "2026-06-30T10:00:00Z")
    }
}
