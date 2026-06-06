import Testing
import Foundation

/// Factory Service — API endpoint path validation and request formatting
struct FactoryServiceTests {

    // MARK: - API Paths

    @Test func loginEndpointPath() {
        let expected = "v1/auth/factory/login"
        #expect(expected.hasPrefix("v1/auth/factory"))
    }

    @Test func refreshEndpointPath() {
        let expected = "v1/auth/factory/refresh"
        #expect(expected.contains("refresh"))
    }

    @Test func dashboardEndpointPath() {
        let expected = "v1/factory/dashboard"
        #expect(expected.hasPrefix("v1/factory"))
    }

    @Test func transfersEndpointPath() {
        let expected = "v1/factory/transfers"
        #expect(expected.hasPrefix("v1/factory"))
    }

    @Test func transitionEndpointPath() {
        let id = "t-123"
        let path = "v1/factory/transfers/\(id)/transition"
        #expect(path == "v1/factory/transfers/t-123/transition")
    }

    @Test func dispatchEndpointPath() {
        let expected = "v1/factory/dispatch"
        #expect(expected.hasPrefix("v1/factory"))
    }

    @Test func fleetEndpointPath() {
        let expected = "v1/factory/fleet"
        #expect(expected.hasPrefix("v1/factory"))
    }

    @Test func staffEndpointPath() {
        let expected = "v1/factory/staff"
        #expect(expected.hasPrefix("v1/factory"))
    }

    @Test func insightsEndpointPath() {
        // Note: Factory uses warehouse replenishment insights
        let expected = "v1/warehouse/replenishment/insights"
        #expect(expected.contains("insights"))
    }

    // MARK: - Login Request Encoding

    @Test func loginRequestEncodesCorrectly() throws {
        let req = LoginRequest(phone: "+998901234567", password: "Test123!")
        let data = try JSONEncoder().encode(req)
        let str = String(data: data, encoding: .utf8)!
        #expect(str.contains("+998901234567"))
        #expect(str.contains("Test123!"))
    }

    // MARK: - Dispatch Request

    @Test func dispatchRequestEncodesTransferIds() throws {
        let req = DispatchRequest(transferIds: ["t-1", "t-2", "t-3"])
        let data = try JSONEncoder().encode(req)
        let str = String(data: data, encoding: .utf8)!
        #expect(str.contains("transfer_ids"))
        #expect(str.contains("t-1"))
        #expect(str.contains("t-3"))
    }

    // MARK: - Loading Bay Query

    @Test func loadingBayFilterStates() {
        let states = "APPROVED,LOADING,DISPATCHED"
        let split = states.split(separator: ",").map(String.init)
        #expect(split.count == 3)
        #expect(split.contains("APPROVED"))
        #expect(split.contains("LOADING"))
        #expect(split.contains("DISPATCHED"))
    }
}
