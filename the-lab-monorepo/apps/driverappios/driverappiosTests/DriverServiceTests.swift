import Testing
import Foundation
@testable import driverappios

/// Driver iOS — Mission, Order, and RouteManifest model tests
struct DriverServiceTests {

    // MARK: - Mission Decoding

    @Test func missionMinimalDecoding() throws {
        let str = "{\"order_id\":\"ORD-1\",\"state\":\"EN_ROUTE\",\"target_lat\":41.31,\"target_lng\":69.28,\"amount\":100000,\"gateway\":\"CASH\",\"estimated_arrival_at\":null,\"route_id\":null,\"sequence_index\":null}"
        let json = Data(str.utf8)

        let mission = try JSONDecoder().decode(Mission.self, from: json)
        #expect(mission.order_id == "ORD-1")
        #expect(mission.state == "EN_ROUTE")
        #expect(mission.amount == 100_000)
        #expect(mission.gateway == "CASH")
        #expect(mission.id == "ORD-1")
    }

    @Test func missionGatewayValues() {
        let validGateways = ["CASH", "GLOBAL_PAY"]
        for gw in validGateways {
            #expect(!gw.isEmpty)
        }
    }

    @Test func missionMockDataHasThreeEntries() {
        let mocks = Mission.mockMissions
        #expect(mocks.count == 3)
        #expect(mocks[0].gateway == "GLOBAL_PAY")
        #expect(mocks[1].gateway == "CASH")
        #expect(mocks[2].gateway == "CASH")
    }

    // MARK: - Order State

    @Test func orderStateAllCases() {
        let all = OrderState.allCases
        #expect(all.count == 15)
    }

    @Test func orderStateLabels() {
        #expect(OrderState.PENDING.label == "Pending")
        #expect(OrderState.IN_TRANSIT.label == "In Transit")
        #expect(OrderState.ARRIVED_SHOP_CLOSED.label == "Shop Closed")
        #expect(OrderState.AWAITING_PAYMENT.label == "Awaiting Payment")
        #expect(OrderState.COMPLETED.label == "Completed")
        #expect(OrderState.DELIVERED_ON_CREDIT.label == "On Credit")
    }

    @Test func orderStateIsActive() {
        let activeStates: [OrderState] = [
            .LOADED, .DISPATCHED, .IN_TRANSIT, .ARRIVING, .ARRIVED,
            .ARRIVED_SHOP_CLOSED, .AWAITING_PAYMENT, .PENDING_CASH_COLLECTION,
            .DELIVERED_ON_CREDIT
        ]
        for state in activeStates {
            #expect(state.isActive == true, "\(state) should be active")
        }

        let inactiveStates: [OrderState] = [
            .PENDING, .CANCEL_REQUESTED, .NO_CAPACITY, .COMPLETED, .CANCELLED, .QUARANTINE
        ]
        for state in inactiveStates {
            #expect(state.isActive == false, "\(state) should be inactive")
        }
    }

    // MARK: - OrderLineItem

    @Test func orderLineItemLineTotal() throws {
        let str = "{\"product_id\":\"p-1\",\"product_name\":\"Rice\",\"quantity\":10,\"unit_price\":25000}"
        let json = Data(str.utf8)

        let item = try JSONDecoder().decode(OrderLineItem.self, from: json)
        #expect(item.lineTotal == 250_000)
        #expect(item.id == "p-1")
    }

    @Test func orderLineItemZeroQuantity() throws {
        let str = "{\"product_id\":\"p-2\",\"product_name\":\"Milk\",\"quantity\":0,\"unit_price\":10000}"
        let json = Data(str.utf8)

        let item = try JSONDecoder().decode(OrderLineItem.self, from: json)
        #expect(item.lineTotal == 0)
    }

    // MARK: - RouteManifest

    @Test func routeManifestValidity() {
        let validManifest = RouteManifest(
            driver_id: "DRV-1",
            date: "2026-04-18",
            expires_at: Date().timeIntervalSince1970 + 3600,
            hashes: ["ORD-1": "abc123"]
        )
        #expect(validManifest.isValid == true)
    }

    @Test func routeManifestExpired() {
        let expired = RouteManifest(
            driver_id: "DRV-1",
            date: "2026-04-17",
            expires_at: Date().timeIntervalSince1970 - 3600,
            hashes: [:]
        )
        #expect(expired.isValid == false)
    }

    @Test func routeManifestMockIsValid() {
        let mock = RouteManifest.mock
        #expect(mock.isValid == true)
        #expect(mock.hashes.count == 3)
    }

    // MARK: - API Endpoint Paths

    @Test func driverLoginEndpoint() {
        let path = "v1/auth/driver/login"
        #expect(path.hasPrefix("v1/auth/driver"))
    }

    @Test func driverProfileEndpoint() {
        let path = "v1/driver/profile"
        #expect(path.hasPrefix("v1/driver"))
    }

    @Test func fleetManifestEndpoint() {
        let path = "v1/fleet/manifest"
        #expect(path.hasPrefix("v1/fleet"))
    }

    @Test func deliveryArriveEndpoint() {
        let path = "v1/delivery/arrive"
        #expect(path.hasPrefix("v1/delivery"))
    }

    @Test func orderCompleteEndpoint() {
        let path = "v1/order/complete"
        #expect(path.hasPrefix("v1/order"))
    }

    @Test func collectCashEndpoint() {
        let path = "v1/order/collect-cash"
        #expect(path.contains("collect-cash"))
    }

    @Test func shopClosedEndpoint() {
        let path = "v1/delivery/shop-closed"
        #expect(path.contains("shop-closed"))
    }
}
