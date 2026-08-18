import XCTest
@testable import SupplierAppIOS

final class PlanningTests: XCTestCase {
    func testPlanBrainTabAndBlockedForecast() {
        XCTAssertEqual(planBrainTabFromQuery("brain"), .brain)
        XCTAssertEqual(planBrainTabFromQuery(nil), .planning)
        let blocked = ForecastConfidence(
            lowUnits: 10,
            highUnits: 20,
            confidencePct: 80,
            baselineSource: nil,
            blockedReason: "sparsity_blocked",
            label: "insufficient_history"
        )
        XCTAssertTrue(blocked.isBlocked)
        XCTAssertNil(brainForecastLine(confidence: blocked, accuracyPoints: [1, 2, 3]))
        XCTAssertEqual(factoryPlanningDisabledCode(status: 409, body: "{\"error\":\"factory_planning_disabled\"}"), "factory_planning_disabled")
    }
}
