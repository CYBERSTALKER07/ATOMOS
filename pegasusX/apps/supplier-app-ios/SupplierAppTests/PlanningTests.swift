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

    func testFailedRouteLoadDoesNotBecomeAnEmptySuccess() {
        let kept = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: ["r1"], failure: "timeout")
        XCTAssertEqual(kept.events, ["r1"])
        XCTAssertEqual(kept.error, "timeout")

        let first = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: [], failure: "timeout")
        XCTAssertTrue(first.events.isEmpty)
        XCTAssertEqual(first.error, "timeout")

        let empty = PulseHonesty.apply(ok: true, incoming: [String](), previous: ["r1"], failure: "timeout")
        XCTAssertTrue(empty.events.isEmpty)
        XCTAssertNil(empty.error)
    }

    func testFailedNetworkModeDoesNotLookLikeEnginesOff() {
        let previous = NetworkModeSnapshot(mode: "SPEED", planningEnabled: true)
        let kept = PulseHonesty.applyObject(ok: false, incoming: nil as NetworkModeSnapshot?, previous: previous, failure: "timeout")
        XCTAssertEqual(kept.value?.mode, "SPEED")
        XCTAssertEqual(kept.value?.planningEnabled, true)
        XCTAssertEqual(kept.error, "timeout")

        let unknown = PulseHonesty.applyObject(ok: false, incoming: nil as NetworkModeSnapshot?, previous: nil, failure: "timeout")
        XCTAssertNil(unknown.value)
        XCTAssertEqual(unknown.error, "timeout")
    }

    func testPlanningSettingsKillSwitchReasonValidation() {
        let emptyValidation = PlanningSettingsView.killSwitchReasonValidation(reason: "")
        XCTAssertEqual(emptyValidation, "Typed reason required")
        XCTAssertNotEqual(emptyValidation, "mobile_supplier.chrome.typed_reason_required")

        let whitespaceValidation = PlanningSettingsView.killSwitchReasonValidation(reason: "   \n\t  ")
        XCTAssertEqual(whitespaceValidation, "Typed reason required")

        let validValidation = PlanningSettingsView.killSwitchReasonValidation(reason: "Emergency line maintenance")
        XCTAssertNil(validValidation)
    }
}

private struct NetworkModeSnapshot {
    let mode: String
    let planningEnabled: Bool
}
