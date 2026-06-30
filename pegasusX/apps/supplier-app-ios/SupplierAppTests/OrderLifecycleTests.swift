import XCTest
@testable import SupplierApp

final class OrderLifecycleTests: XCTestCase {
    
    override func setUp() {
        super.setUp()
        APIClient.shared.setTestSession(MockURLProtocol.createMockSession())
    }
    
    override func tearDown() {
        MockURLProtocol.requestHandler = nil
        super.tearDown()
    }
    
    func testOrderListParsing() async throws {
        let json = """
        {
            "orders": [
                {
                    "id": "ord_123",
                    "retailer_id": "ret_1",
                    "retailer_name": "Test Retailer",
                    "status": "PENDING",
                    "total_minor": 50000,
                    "currency": "UZS",
                    "created_at": "2026-06-30T10:00:00Z"
                }
            ],
            "total_count": 1
        }
        """
        
        MockURLProtocol.requestHandler = { request in
            XCTAssertEqual(request.url?.path, "/v1/supplier/orders")
            let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(json.utf8))
        }
        
        let response = try await SupplierService.orders()
        
        XCTAssertEqual(response.orders.count, 1)
        XCTAssertEqual(response.orders.first?.id, "ord_123")
        XCTAssertEqual(response.orders.first?.status, "PENDING")
        XCTAssertEqual(response.orders.first?.totalMinor, 50000)
    }
}
