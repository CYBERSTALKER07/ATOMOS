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
        #expect(count == 15, "OrderState should have 15 cases, got \(count)")
    }

    @Test func orderStateEnum_labels() async throws {
        #expect(OrderState.IN_TRANSIT.label == "In Transit")
        #expect(OrderState.ARRIVED.label == "Arrived")
        #expect(OrderState.PENDING.label == "Pending")
    }

    @Test func orderState_activeStates() async throws {
        #expect(OrderState.IN_TRANSIT.isActive == true)
        #expect(OrderState.ARRIVED.isActive == true)
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
            sequenceIndex: nil
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

    @Test func geofenceThreshold_100m() async throws {
        let base = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
        let edge = CLLocationCoordinate2D(latitude: 41.3004, longitude: 69.2401)
        let dist = haversineDistance(from: base, to: edge)
        #expect(dist > 90 && dist < 120, "Should be near 100m boundary, got \(dist)")
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
            sequenceIndex: 2
        )

        let arrived = Order(
            id: original.id,
            retailerId: original.retailerId,
            retailerName: original.retailerName,
            state: .ARRIVED,
            totalAmount: original.totalAmount,
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
            sequenceIndex: original.sequenceIndex
        )

        #expect(arrived.state == .ARRIVED)
        #expect(arrived.id == original.id)
        #expect(arrived.totalAmount == original.totalAmount)
        #expect(arrived.routeId == original.routeId)
        #expect(arrived.sequenceIndex == original.sequenceIndex)
        #expect(original.state == .IN_TRANSIT, "Original should still be IN_TRANSIT")
    }
}
