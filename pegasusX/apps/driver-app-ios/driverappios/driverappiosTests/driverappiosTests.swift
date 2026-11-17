//
//  driverappiosTests.swift
//  driverappiosTests
//
//  Created by Shakhzod on 3/18/26.
//

import Testing
import CoreLocation
@testable import driverappios

// MARK: - Order Model Tests

struct OrderModelTests {

    @Test func orderStateEnum_allCases() async throws {
        let count = OrderState.allCases.count
        #expect(count == 24, "OrderState should have 24 cases, got \(count)")
    }

    @Test func orderStateEnum_labels() async throws {
        #expect(OrderState.IN_TRANSIT.label == "In Transit")
        #expect(OrderState.ARRIVED.label == "Arrived")
        #expect(OrderState.PENDING.label == "Pending")
    }

    @Test func orderState_activeStates() async throws {
        #expect(OrderState.IN_TRANSIT.isActive == true)
        #expect(OrderState.ARRIVED.isActive == true)
        #expect(OrderState.ARRIVED_SHOP_CLOSED.isActive == true)
        #expect(OrderState.FISCAL_FAILED.isActive == true)
        #expect(OrderState.COMPLETED.isActive == false)
        #expect(OrderState.CANCELLED.isActive == false)
        #expect(OrderState.PENDING.isActive == false)
    }

    @Test func orderDisplayTotal_containsAmount() async throws {
        let order = Order(
            id: "ORD-001",
            retailerId: "RET-001",
            retailerName: "Test Shop",
            state: .PENDING,
            totalAmount: 150000,
            deliveryFeeMinor: 0,
            deliveryDistanceKm: 0,
            deliveryAddress: "123 Main St",
            latitude: 41.2995,
            longitude: 69.2401,
            qrToken: "abc123",
            paymentGateway: "GLOBAL_PAY",
            createdAt: "2026-04-12T10:00:00Z",
            updatedAt: "2026-04-12T10:00:00Z",
            items: [],
            estimatedArrivalAt: nil,
            etaDurationSec: nil,
            etaDistanceM: nil,
            routeId: nil,
            sequenceIndex: nil,
            isPartial: nil,
            splitGroupId: nil
        )
        #expect(order.displayTotal.contains("150"), "displayTotal should include the amount")
    }
}

// MARK: - Haversine Distance Tests

struct HaversineTests {

    @Test func samePoint_returnsZero() async throws {
        let a = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        let b = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        let dist = haversineDistance(from: a, to: b)
        #expect(dist < 1.0, "Same point should have ~0 distance, got \(dist)")
    }

    @Test func knownDistance_tashkentToSamarkand() async throws {
        let tashkent = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        let samarkand = CLLocationCoordinate2D(latitude: 39.6542, longitude: 66.9597)
        let dist = haversineDistance(from: tashkent, to: samarkand)
        let km = dist / 1000
        #expect(km > 250 && km < 280, "Tashkent→Samarkand ≈ 262km, got \(km)")
    }

    @Test func nearbyPoint_under100m() async throws {
        let base = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        let nearby = CLLocationCoordinate2D(latitude: 41.2999, longitude: 69.2401)
        let dist = haversineDistance(from: base, to: nearby)
        #expect(dist > 30 && dist < 100, "~44m offset should be 30-100m, got \(dist)")
    }

    @Test func geofenceThreshold_150m() async throws {
        let threshold = 150.0
        let base = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        let within = CLLocationCoordinate2D(latitude: 41.3005, longitude: 69.2401)
        let dist = haversineDistance(from: base, to: within)
        #expect(dist <= threshold, "Should be within 150m, got \(dist)m")
    }
}

// MARK: - Order State Immutability Tests

struct OrderImmutabilityTests {

    @Test func orderCopy_withNewState() async throws {
        let original = Order(
            id: "ORD-IMMUT",
            retailerId: "RET-001",
            retailerName: "Test Shop",
            state: .IN_TRANSIT,
            totalAmount: 50000,
            deliveryFeeMinor: 0,
            deliveryDistanceKm: 0,
            deliveryAddress: "456 Side St",
            latitude: 41.3000,
            longitude: 69.2500,
            qrToken: "tok999",
            paymentGateway: nil,
            createdAt: "2026-04-12T10:00:00Z",
            updatedAt: "2026-04-12T10:00:00Z",
            items: [],
            estimatedArrivalAt: nil,
            etaDurationSec: nil,
            etaDistanceM: nil,
            routeId: "ROUTE-01",
            sequenceIndex: 2,
            isPartial: nil,
            splitGroupId: nil
        )

        let arrived = Order(
            id: original.id,
            retailerId: original.retailerId,
            retailerName: original.retailerName,
            state: .ARRIVED,
            totalAmount: original.totalAmount,
            deliveryFeeMinor: original.deliveryFeeMinor,
            deliveryDistanceKm: original.deliveryDistanceKm,
            deliveryAddress: original.deliveryAddress,
            latitude: original.latitude,
            longitude: original.longitude,
            qrToken: original.qrToken,
            paymentGateway: original.paymentGateway,
            createdAt: original.createdAt,
            updatedAt: original.updatedAt,
            items: original.items,
            estimatedArrivalAt: original.estimatedArrivalAt,
            etaDurationSec: original.etaDurationSec,
            etaDistanceM: original.etaDistanceM,
            routeId: original.routeId,
            sequenceIndex: original.sequenceIndex,
            isPartial: original.isPartial,
            splitGroupId: original.splitGroupId
        )

        #expect(arrived.state == .ARRIVED)
        #expect(arrived.id == original.id)
        #expect(arrived.totalAmount == original.totalAmount)
        #expect(arrived.routeId == original.routeId)
        #expect(arrived.sequenceIndex == original.sequenceIndex)
        #expect(original.state == .IN_TRANSIT, "Original should still be IN_TRANSIT")
    }
}

// MARK: - Offline enqueue classifier (P0-4)

struct DriverOfflineEnqueueTests {

    @Test func networkError_isEnqueueable() {
        #expect(DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.networkError))
        #expect(DriverOfflineActionCatalog.isNetworkEnqueueable(URLError(.timedOut)))
        #expect(DriverOfflineActionCatalog.isNetworkEnqueueable(URLError(.notConnectedToInternet)))
    }

    @Test func retryableHTTP_isEnqueueable() {
        #expect(DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.httpError(503)))
        #expect(DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.httpError(429)))
        let problem = ProblemDetail(
            type: nil, title: nil, status: 503, detail: nil, traceId: nil,
            instance: nil, code: nil, messageKey: nil, retryable: true, action: nil
        )
        #expect(DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.problemDetail(problem)))
    }

    @Test func geofenceAndBusinessReject_neverEnqueue() {
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.forbidden))
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.unauthorized))
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.httpError(403)))
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.httpError(422)))
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.decodingError))
        let geofence = ProblemDetail(
            type: nil, title: "Forbidden", status: 403, detail: "too far",
            traceId: nil, instance: nil, code: "geofence_violation",
            messageKey: nil, retryable: false, action: nil
        )
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(APIError.problemDetail(geofence)))
        #expect(!DriverOfflineActionCatalog.isNetworkEnqueueable(
            FleetServiceError.deliveryRejected("denied")
        ))
    }
}

// MARK: - Offline HTTP Semantics Tests (S1-9)

struct OfflineHttpSemanticsTests {

    @Test func httpSuccessClassification() {
        #expect(OfflineHttpSemantics.isSuccessHTTP(200))
        #expect(OfflineHttpSemantics.isSuccessHTTP(201))
        #expect(OfflineHttpSemantics.isSuccessHTTP(204))
        #expect(!OfflineHttpSemantics.isSuccessHTTP(409), "409 is a conflict, not unconditional success")
        #expect(!OfflineHttpSemantics.isSuccessHTTP(422))
        #expect(!OfflineHttpSemantics.isSuccessHTTP(503))
    }

    @Test func httpConflictClassification() {
        #expect(OfflineHttpSemantics.isConflictHTTP(409))
        #expect(!OfflineHttpSemantics.isConflictHTTP(200))
        #expect(!OfflineHttpSemantics.isConflictHTTP(503))
    }

    @Test func idempotentReplay409Classification() {
        // Idempotent markers
        #expect(OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"error\":\"duplicate_mutation\"}"))
        #expect(OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"code\":\"already_exists\"}"))
        #expect(OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"detail\":\"idempotent replay acknowledged\"}"))
        #expect(OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"status\":\"already_processed\"}"))

        // Business conflicts (NOT idempotent replay)
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"error\":\"order_cancelled\"}"))
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"error\":\"invalid_state_transition\"}"))
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: "{\"error\":\"manifest_reassigned\"}"))
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: ""))
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: String?.none))

        // Non-409 with duplicate text must not match
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 200, responseBody: "duplicate"))
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 422, responseBody: "duplicate"))

        // Data overload
        let dataIdempotent = "{\"error\":\"duplicate_key\"}".data(using: .utf8)
        let dataConflict = "{\"error\":\"state_conflict\"}".data(using: .utf8)
        #expect(OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: dataIdempotent))
        #expect(!OfflineHttpSemantics.isIdempotentReplay409(statusCode: 409, responseBody: dataConflict))
    }

    @Test func outcomeMapping() {
        #expect(OfflineHttpSemantics.outcome(forHTTP: 200) == .ack)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 204) == .ack)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 503) == .retry)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 429) == .retry)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 408) == .retry)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 422) == .dead)

        // 409 outcomes: ack if idempotent, dead if business conflict
        #expect(OfflineHttpSemantics.outcome(forHTTP: 409, bodyString: "idempotent replay") == .ack)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 409, bodyString: "order cancelled") == .dead)
        #expect(OfflineHttpSemantics.outcome(forHTTP: 409, bodyString: nil) == .dead)
    }
}

