import XCTest
@testable import SupplierAppIOS

final class OrderLifecycleTests: XCTestCase {
    
    override func setUp() {
        super.setUp()
    }
    
    override func tearDown() {
        super.tearDown()
    }
    
    func testResolveSupplierOrdersQueryPrefersCanonicalStatus() {
        let query = resolveSupplierOrdersQuery(commandStatus: "delivered", coarseFilter: "ACTIVE")
        XCTAssertEqual(query.status, "COMPLETED")
        XCTAssertNil(query.filter)
    }

    func testResolveSupplierOrdersQueryScheduledFilter() {
        let query = resolveSupplierOrdersQuery(commandStatus: nil, coarseFilter: "SCHEDULED")
        XCTAssertEqual(query.status, "SCHEDULED")
        XCTAssertNil(query.filter)
    }

    func testResolveSupplierOrdersQueryCoarseFilter() {
        let query = resolveSupplierOrdersQuery(commandStatus: nil, coarseFilter: "ACTIVE")
        XCTAssertNil(query.status)
        XCTAssertEqual(query.filter, "ACTIVE")
    }

    func testSilentFailureKeepsVisibleOrders() {
        let outcome = OrdersLoadFailure.message(
            offline: false,
            silent: true,
            hasOrders: true,
            underlying: "timeout"
        )
        XCTAssertNil(outcome.error)
        XCTAssertEqual(outcome.notice, "timeout")
    }

    func testForegroundReloadFailureKeepsVisibleOrders() {
        let outcome = OrdersLoadFailure.message(
            offline: false,
            silent: false,
            hasOrders: true,
            underlying: "timeout"
        )
        XCTAssertNil(outcome.error)
        XCTAssertEqual(outcome.notice, "timeout")
    }

    func testOfflineFirstLoadIsAnError() {
        let outcome = OrdersLoadFailure.message(
            offline: true,
            silent: false,
            hasOrders: false,
            underlying: "network"
        )
        XCTAssertEqual(outcome.error, "You are offline. Connect to load orders.")
        XCTAssertNil(outcome.notice)
    }

    func testOfflineRefreshKeepsLastOrders() {
        let outcome = OrdersLoadFailure.message(
            offline: true,
            silent: true,
            hasOrders: true,
            underlying: "network"
        )
        XCTAssertNil(outcome.error)
        XCTAssertEqual(outcome.notice, "You are offline. Showing the last orders.")
    }

    func testOfflineRefreshNamesTheScreen() {
        let outcome = OrdersLoadFailure.message(
            offline: true,
            silent: true,
            hasOrders: true,
            underlying: "network",
            noun: "data"
        )
        XCTAssertNil(outcome.error)
        XCTAssertEqual(outcome.notice, "You are offline. Showing the last data.")
    }

    func testSilentFailureWithoutContentSurfacesTheError() {
        let outcome = OrdersLoadFailure.message(
            offline: false,
            silent: true,
            hasOrders: false,
            underlying: "timeout"
        )
        XCTAssertEqual(outcome.error, "timeout")
        XCTAssertNil(outcome.notice)
    }
}
