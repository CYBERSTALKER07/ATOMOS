import XCTest
@testable import SupplierApp

final class AuthTests: XCTestCase {
    
    override func setUp() {
        super.setUp()
        APIClient.shared.setTestSession(MockURLProtocol.createMockSession())
    }
    
    override func tearDown() {
        MockURLProtocol.requestHandler = nil
        super.tearDown()
    }
    
    func testLoginSuccess() async throws {
        let json = """
        {
            "token": "fake-jwt-token",
            "is_registered": true,
            "is_configured": true
        }
        """
        
        MockURLProtocol.requestHandler = { request in
            XCTAssertEqual(request.url?.path, "/v1/auth/supplier/login")
            XCTAssertEqual(request.httpMethod, "POST")
            
            let response = HTTPURLResponse(url: request.url!, statusCode: 200, httpVersion: nil, headerFields: nil)!
            return (response, Data(json.utf8))
        }
        
        let response = try await SupplierService.login(phone: "998901234567", password: "password")
        
        XCTAssertEqual(response.token, "fake-jwt-token")
        XCTAssertTrue(response.isRegistered)
        XCTAssertTrue(response.isConfigured)
    }
    
    func testLoginUnauthorized() async {
        let json = """
        {
            "error": "Invalid credentials"
        }
        """
        
        MockURLProtocol.requestHandler = { request in
            let response = HTTPURLResponse(url: request.url!, statusCode: 401, httpVersion: nil, headerFields: nil)!
            return (response, Data(json.utf8))
        }
        
        do {
            _ = try await SupplierService.login(phone: "998901234567", password: "wrong")
            XCTFail("Expected unauthorized error")
        } catch let error as APIError {
            if case .unauthorized = error {
                // Success
            } else {
                XCTFail("Expected APIError.unauthorized, got \(error)")
            }
        } catch {
            XCTFail("Expected APIError.unauthorized, got \(error)")
        }
    }
}
