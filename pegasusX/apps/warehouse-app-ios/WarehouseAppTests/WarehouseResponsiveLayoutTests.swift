import XCTest
import SwiftUI
@testable import WarehouseAppIOS

final class WarehouseResponsiveLayoutTests: XCTestCase {

    func testTacticalColorTokens() {
        XCTAssertNotNil(TacticalTheme.canvas)
        XCTAssertNotNil(TacticalTheme.surface)
        XCTAssertNotNil(TacticalTheme.border)
        XCTAssertNotNil(TacticalTheme.accentCobalt)
        XCTAssertNotNil(TacticalTheme.accentOrange)
        XCTAssertNotNil(TacticalTheme.accentLime)
        XCTAssertNotNil(TacticalTheme.accentCyan)
        XCTAssertNotNil(TacticalTheme.statusSuccess)
        XCTAssertNotNil(TacticalTheme.statusWarning)
        XCTAssertNotNil(TacticalTheme.statusDanger)
    }

    func testWarehouseCompactTabs() {
        let tabs = WarehouseSection.compactTabs
        XCTAssertEqual(tabs.count, 4)
        XCTAssertTrue(tabs.contains(.dashboard))
        XCTAssertTrue(tabs.contains(.manifests))
        XCTAssertTrue(tabs.contains(.inventory))
        XCTAssertTrue(tabs.contains(.dispatch))
    }

    func testWarehouseSectionPartitioning() {
        XCTAssertFalse(WarehouseSection.primarySections.isEmpty)
        XCTAssertFalse(WarehouseSection.fulfillmentSections.isEmpty)
        XCTAssertFalse(WarehouseSection.inventorySections.isEmpty)
        XCTAssertFalse(WarehouseSection.operationsSections.isEmpty)
        XCTAssertFalse(WarehouseSection.portalSections.isEmpty)

        // Ensure total sections coverage
        let allSections = WarehouseSection.sidebarSections
        XCTAssertGreaterThan(allSections.count, 20)
    }

    func testTacticalStatusBadgeTints() {
        let successBadge = TacticalStatusBadge(status: "COMPLETED")
        XCTAssertNotNil(successBadge)

        let alertBadge = TacticalStatusBadge(status: "EXCEPTION")
        XCTAssertNotNil(alertBadge)

        let inTransitBadge = TacticalStatusBadge(status: "IN_TRANSIT")
        XCTAssertNotNil(inTransitBadge)
    }

    func testTacticalGaugeCardClamping() {
        let normalGauge = TacticalGaugeCard(title: "DOCK UTILIZATION", percentage: 0.75)
        XCTAssertEqual(normalGauge.percentage, 0.75)

        let overGauge = TacticalGaugeCard(title: "OVERLOAD", percentage: 1.5)
        XCTAssertEqual(overGauge.percentage, 1.5)
    }
}
