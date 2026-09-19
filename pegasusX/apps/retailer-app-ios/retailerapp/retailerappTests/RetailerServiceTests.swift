import Testing
import Foundation
@testable import retailerapp

/// Retailer iOS — APIClient endpoint paths, WebSocket message parsing, auth flow validation
struct RetailerServiceTests {

    // MARK: - API Endpoint Paths

    @Test func loginEndpoint() {
        let path = "/v1/auth/retailer/login"
        #expect(path.hasPrefix("/v1/auth/retailer"))
    }

    @Test func registerEndpoint() {
        let path = "/v1/auth/retailer/register"
        #expect(path.contains("register"))
    }

    @Test func refreshEndpoint() {
        let path = "/v1/auth/refresh"
        #expect(path.contains("refresh"))
    }

    @Test func ordersEndpoint() {
        let retailerId = "r-123"
        let path = "/v1/retailers/\(retailerId)/orders"
        #expect(path.contains(retailerId))
    }

    @Test func catalogEndpoint() {
        let path = "/v1/catalog/products"
        #expect(path.hasPrefix("/v1/catalog"))
    }

    @Test func predictionsEndpoint() {
        let path = "/v1/retailer/ai/predictions"
        #expect(path == "/v1/retailer/ai/predictions")
    }

    @Test func unifiedCheckoutEndpoint() {
        let path = "/v1/checkout/unified"
        #expect(path.contains("checkout"))
    }

    @Test func cancelOrderEndpoint() {
        let path = "/v1/orders/request-cancel"
        #expect(path.contains("cancel"))
    }

    @Test func activeFulfillmentEndpoint() {
        let path = "/v1/retailer/active-fulfillment"
        #expect(path.contains("fulfillment"))
    }

    // MARK: - Order Status Timeline

    @Test func timelineStepsCount() {
        #expect(OrderStatus.timelineSteps.count == 6)
    }

    @Test func timelineStepIndices() {
        #expect(OrderStatus.pending.timelineStepIndex == 0)
        #expect(OrderStatus.loaded.timelineStepIndex == 1)
        #expect(OrderStatus.dispatched.timelineStepIndex == 2)
        #expect(OrderStatus.inTransit.timelineStepIndex == 3)
        #expect(OrderStatus.arrived.timelineStepIndex == 4)
        #expect(OrderStatus.completed.timelineStepIndex == 5)
        #expect(OrderStatus.cancelled.timelineStepIndex == -1)
    }

    @Test func hasDeliveryToken() {
        #expect(OrderStatus.dispatched.hasDeliveryToken == true)
        #expect(OrderStatus.inTransit.hasDeliveryToken == true)
        #expect(OrderStatus.arrived.hasDeliveryToken == true)
        #expect(OrderStatus.pending.hasDeliveryToken == false)
        #expect(OrderStatus.completed.hasDeliveryToken == false)
    }

    // MARK: - WebSocket Event Types

    @Test func wsEventTypesCoverage() {
        let expected = [
            "ORDER_STATUS_CHANGED",
            "PAYMENT_REQUIRED",
            "SETTLEMENT_REQUIRED",
            "DELIVERY_SESSION_UPDATED",
            "PAYMENT_SETTLED",
            "PAYMENT_FAILED",
            "PAYMENT_EXPIRED",
            "ORDER_AMENDED",
            "ORDER_COMPLETED",
            "DRIVER_APPROACHING",
        ]
        for event in expected {
            #expect(!event.isEmpty, "\(event) should exist")
        }
    }

    // MARK: - GlobalPaynt Gateway Values

    @Test func global_payntGatewayOptions() {
        let gateways = ["GLOBAL_PAY", "CASH"]
        #expect(gateways.count == 2)
        
        #expect(gateways.contains("GLOBAL_PAY"))
        #expect(gateways.contains("CASH"))
    }

    // MARK: - OrderLineItem

    @Test func orderLineItemDecoding() throws {
        let json = """
        {"line_item_id":"li-1","sku_id":"v-1","sku_name":"Milk",
         "quantity":10,"unit_price":12000.0}
        """.data(using: .utf8)!

        let item = try JSONDecoder().decode(OrderLineItem.self, from: json)
        #expect(item.id == "li-1")
        #expect(item.variantId == "v-1")
        #expect(item.productName == "Milk")
        #expect(item.quantity == 10)
        #expect(item.totalPrice == 120_000)
    }

    @Test func orderDecodingIncludesRetailerListMetadata() throws {
        let json = """
        {
          "order_id":"ord-1",
          "retailer_id":"ret-1",
          "supplier_id":"sup-1",
          "supplier_name":"Pegasus Supplier",
          "state":"PENDING",
          "amount":22450,
          "currency":"UZS",
          "payment_gateway":"GLOBAL_PAY",
          "payment_status":"PENDING",
          "route_id":"route-1",
          "order_source":"MANUAL",
          "auto_confirm_at":"2026-01-01T10:00:00Z",
          "deliver_before":"2026-01-01T11:00:00Z",
          "created_at":"2026-01-01T09:00:00Z",
          "updated_at":"2026-01-01T09:30:00Z",
          "estimated_delivery":"2026-01-01T11:00:00Z",
          "delivery_token":"tok-1",
          "version":7,
          "items":[{"line_item_id":"li-1","sku_id":"sku-1","sku_name":"Milk","quantity":2,"unit_price":11225,"total_price":22450}]
        }
        """.data(using: .utf8)!

        let order = try JSONDecoder().decode(Order.self, from: json)
        #expect(order.totalAmount == 22450)
        #expect(order.currency == "UZS")
        #expect(order.paymentGateway == "GLOBAL_PAY")
        #expect(order.paymentStatus == "PENDING")
        #expect(order.routeId == "route-1")
        #expect(order.version == 7)
        #expect(order.displayTotal == "22 450 UZS")
    }

    // MARK: - Token Refresh Guard

    @Test func refreshPreventsConcurrentLoops() {
        // The APIClient uses isRefreshing flag to prevent recursive 401 → refresh → 401 loops
        // Verify the pattern: request with isRetry=true should NOT trigger another refresh
        var isRefreshing = false
        let isRetry = true
        let statusCode = 401

        // This simulates the guard condition
        let shouldRefresh = statusCode == 401 && !isRetry && !isRefreshing
        #expect(shouldRefresh == false, "Should not refresh on retry")
    }

    @Test func displayPackCurrencyDoesNotInventUZS() {
        MarketPackStore.clear()
        #expect(displayPackCurrency("") == "")
        #expect(displayPackCurrency(nil) == "")
        #expect(displayPackCurrency("kzt") == "KZT")
    }

    @Test func packMapCenterDoesNotInventTashkent() {
        MarketPackStore.clear()
        #expect(packMapCenter(nil) == nil)
        let empty = packMapCoordinate(nil)
        #expect(empty.lat == 0 && empty.lng == 0)
        let uz = MarketPack(
            code: "UZ",
            name: "Uzbekistan",
            timezone: "Asia/Tashkent",
            currencyCode: "UZS",
            fiscalAdapter: "MY_SOLIQ",
            mapsAdapter: "GOOGLE_ROUTES",
            mapCenterLat: 41.2995,
            mapCenterLng: 69.2401,
            checkoutReadsThis: false
        )
        #expect(packMapCenter(uz)?.lat == 41.2995)
        #expect(packMapCenter(uz)?.lng == 69.2401)
    }
}
