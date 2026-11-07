//
//  DeliveryActivityAttributes.swift
//  mobile-ios-design
//
//  Shared ActivityKit attributes contract for Pegasus delivery tracking and turn-by-turn navigation.
//

import ActivityKit
import Foundation

public struct DeliveryActivityAttributes: ActivityAttributes {
    public struct ContentState: Codable, Hashable {
        public var status: String                  // "IN_TRANSIT", "ARRIVING", "ARRIVED", "OFFLOADING", "COMPLETED"
        public var etaMinutes: Int                 // Estimated arrival time in minutes
        public var remainingDistanceMeters: Double // Distance to next waypoint or destination
        public var nextInstruction: String         // Next maneuver or delivery instruction
        public var nextManeuver: String?           // Maneuver symbol (e.g. "turn-right", "straight", "u-turn")
        public var progress: Double                // Overall route progress (0.0 to 1.0)
        public var currentStepIndex: Int           // Current turn/stop index
        public var totalSteps: Int                 // Total turns/stops along route
        public var deliveryToken: String?          // 4-digit verification PIN (e.g. "7392")
        public var lastUpdated: Date

        public init(
            status: String,
            etaMinutes: Int,
            remainingDistanceMeters: Double,
            nextInstruction: String,
            nextManeuver: String? = nil,
            progress: Double = 0.0,
            currentStepIndex: Int = 0,
            totalSteps: Int = 0,
            deliveryToken: String? = nil,
            lastUpdated: Date = Date()
        ) {
            self.status = status
            self.etaMinutes = etaMinutes
            self.remainingDistanceMeters = remainingDistanceMeters
            self.nextInstruction = nextInstruction
            self.nextManeuver = nextManeuver
            self.progress = progress
            self.currentStepIndex = currentStepIndex
            self.totalSteps = totalSteps
            self.deliveryToken = deliveryToken
            self.lastUpdated = lastUpdated
        }
    }

    // Static context for the delivery activity instance
    public var orderId: String
    public var routeId: String
    public var supplierName: String
    public var destinationName: String
    public var totalItemCount: Int
    public var totalAmountMinorUnits: Int64
    public var isDriverPerspective: Bool

    public init(
        orderId: String,
        routeId: String,
        supplierName: String,
        destinationName: String,
        totalItemCount: Int,
        totalAmountMinorUnits: Int64,
        isDriverPerspective: Bool = true
    ) {
        self.orderId = orderId
        self.routeId = routeId
        self.supplierName = supplierName
        self.destinationName = destinationName
        self.totalItemCount = totalItemCount
        self.totalAmountMinorUnits = totalAmountMinorUnits
        self.isDriverPerspective = isDriverPerspective
    }
}
