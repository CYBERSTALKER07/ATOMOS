import XCTest
@testable import SupplierAppIOS

final class AuthTests: XCTestCase {
    override func tearDown() {
        APIClient.shared.setTestSession(URLSession(configuration: .ephemeral))
        super.tearDown()
    }

    func testLoginSuccessReturnsTheSessionToken() async throws {
        SupplierAuthStub.statusCode = 200
        SupplierAuthStub.body = Data(
            #"{"supplier_id":"sup_1","is_registered":true,"is_configured":true,"next_step":"DASHBOARD","token":"jwt-1","refresh_token":"ref-1"}"#.utf8
        )
        installAuthStub()

        let auth = try await SupplierService.login(phone: "998901112233", password: "secret")

        XCTAssertEqual(auth.supplierId, "sup_1")
        XCTAssertEqual(auth.token, "jwt-1")
        XCTAssertEqual(auth.refreshToken, "ref-1")
        XCTAssertTrue(auth.isConfigured)
    }

    func testLoginWithoutTokenDoesNotInventOne() throws {
        let data = Data(#"{"supplier_id":"sup_1","is_registered":true}"#.utf8)
        let auth = try JSONDecoder().decode(LoginResponse.self, from: data)
        XCTAssertNil(auth.token)
        XCTAssertNil(auth.refreshToken)
        XCTAssertFalse(auth.isConfigured)
    }

    func testLoginUnauthorizedClearsTheSession() async {
        SupplierAuthStub.statusCode = 401
        SupplierAuthStub.body = Data(#"{"error":"unauthorized"}"#.utf8)
        installAuthStub()

        do {
            _ = try await SupplierService.login(phone: "998901112233", password: "wrong")
            XCTFail("A 401 login must not return a session")
        } catch let error as APIError {
            guard case .unauthorized = error else {
                XCTFail("Expected unauthorized, got \(error)")
                return
            }
        } catch {
            XCTFail("Expected APIError.unauthorized, got \(error)")
        }
    }

    func testResolvePlaceDoesNotInventACoordinate() async {
        SupplierAuthStub.statusCode = 404
        SupplierAuthStub.body = Data(#"{"error":"not_found"}"#.utf8)
        installAuthStub()

        do {
            let loc = try await GeocodeService.resolvePlace("unknown-place")
            XCTFail("Missing place must not become a pin, got \(loc.address) \(loc.lat),\(loc.lng)")
        } catch {
            XCTAssertFalse(error.localizedDescription.isEmpty)
        }
    }

    func testAutocompleteDoesNotInventDistricts() async throws {
        SupplierAuthStub.statusCode = 200
        SupplierAuthStub.body = Data(#"{"predictions":[]}"#.utf8)
        installAuthStub()

        let rows = try await GeocodeService.autocomplete("zzzz-not-a-district")
        XCTAssertTrue(rows.isEmpty)
    }

    @MainActor
    func testFailedDeviceTokenUploadStaysVisible() async {
        SupplierAuthStub.statusCode = 500
        SupplierAuthStub.body = Data(#"{"error":"push_unavailable"}"#.utf8)
        installAuthStub()
        let manager = PushNotificationManager.shared
        let tokenBefore = manager.deviceToken
        await manager.uploadDeviceToken("fcm-test-token")
        XCTAssertEqual(manager.deviceToken, tokenBefore)
        XCTAssertEqual(manager.errorMessage, "push unavailable")

        SupplierAuthStub.statusCode = 200
        SupplierAuthStub.body = Data(#"{}"#.utf8)
        await manager.uploadDeviceToken("fcm-test-token")
        XCTAssertNil(manager.errorMessage)
    }

    func testMalformedBaseURLDoesNotParse() {
        XCTAssertNil(SupplierURL.parse("%%%"))
        XCTAssertNil(SupplierURL.parse("   "))
        XCTAssertNotNil(SupplierURL.parse("https://api.pegasusx.app/"))
    }

    private func installAuthStub() {
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [SupplierAuthStub.self]
        APIClient.shared.setTestSession(URLSession(configuration: config))
    }
}

private final class SupplierAuthStub: URLProtocol {
    nonisolated(unsafe) static var statusCode = 200
    nonisolated(unsafe) static var body = Data()

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }

    override func startLoading() {
        let url = request.url ?? SupplierURL.devBootstrap
        guard let response = HTTPURLResponse(
            url: url,
            statusCode: Self.statusCode,
            httpVersion: nil,
            headerFields: ["Content-Type": "application/json"]
        ) else {
            client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
            return
        }
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: Self.body)
        client?.urlProtocolDidFinishLoading(self)
    }

    override func stopLoading() {}
}
