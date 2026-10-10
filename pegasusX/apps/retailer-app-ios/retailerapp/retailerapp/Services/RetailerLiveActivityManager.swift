//
//  RetailerLiveActivityManager.swift
//  retailerapp
//
//  Manages Retailer-perspective ActivityKit Live Activities and Dynamic Island widgets for inbound delivery arrival and PIN countdown.
//

import ActivityKit
import Foundation

@MainActor
final class RetailerLiveActivityManager {
    static let shared = RetailerLiveActivityManager()

    private var currentActivity: Activity<DeliveryActivityAttributes>?
    private var activeOrderId: String?

    private init() {}

    /// Start a Live Activity tracking an approaching inbound delivery.
    func startInboundDeliveryActivity(
        orderId: String,
        routeId: String = "inbound",
        supplierName: String,
        destinationName: String,
        totalItems: Int,
        totalAmountMinor: Int64,
        etaMinutes: Int,
        deliveryToken: String?
    ) {
        guard ActivityAuthorizationInfo().areActivitiesEnabled else {
            return
        }

        if currentActivity != nil && activeOrderId == orderId {
            return
        }

        endActivity()

        let attributes = DeliveryActivityAttributes(
            orderId: orderId,
            routeId: routeId,
            supplierName: supplierName,
            destinationName: destinationName,
            totalItemCount: totalItems,
            totalAmountMinorUnits: totalAmountMinor,
            isDriverPerspective: false
        )

        let initialContentState = DeliveryActivityAttributes.ContentState(
            status: "IN_TRANSIT",
            etaMinutes: etaMinutes,
            remainingDistanceMeters: Double(etaMinutes) * 400.0,
            nextInstruction: "Inbound delivery from \(supplierName)",
            nextManeuver: "truck.box.fill",
            progress: 0.1,
            currentStepIndex: 1,
            totalSteps: 1,
            deliveryToken: deliveryToken,
            lastUpdated: Date()
        )

        do {
            currentActivity = try Activity.request(
                attributes: attributes,
                content: .init(state: initialContentState, staleDate: nil),
                pushType: nil
            )
            activeOrderId = orderId
        } catch {
            print("[RetailerLiveActivityManager] Failed to start Live Activity: \(error.localizedDescription)")
        }
    }

    /// Update arrival status, ETA countdown, and verification PIN.
    func updateInboundDeliveryState(
        status: String,
        etaMinutes: Int,
        remainingDistanceMeters: Double,
        progress: Double,
        deliveryToken: String?
    ) {
        guard let activity = currentActivity else { return }

        let updatedState = DeliveryActivityAttributes.ContentState(
            status: status,
            etaMinutes: etaMinutes,
            remainingDistanceMeters: remainingDistanceMeters,
            nextInstruction: status == "ARRIVING" ? "Driver approaching store" : "Inbound delivery in transit",
            nextManeuver: "truck.box.fill",
            progress: min(max(progress, 0.0), 1.0),
            currentStepIndex: 1,
            totalSteps: 1,
            deliveryToken: deliveryToken,
            lastUpdated: Date()
        )

        Task {
            await activity.update(.init(state: updatedState, staleDate: nil))
        }
    }

    /// Ends the inbound Live Activity when the order is received, offloaded, or completed.
    func endActivity(status: String = "COMPLETED") {
        guard let activity = currentActivity else { return }

        Task {
            let finalState = DeliveryActivityAttributes.ContentState(
                status: status,
                etaMinutes: 0,
                remainingDistanceMeters: 0,
                nextInstruction: status == "COMPLETED" ? "Delivery received & verified" : "Delivery completed",
                progress: 1.0,
                lastUpdated: Date()
            )
            await activity.end(.init(state: finalState, staleDate: nil), dismissalPolicy: .default)
        }

        currentActivity = nil
        activeOrderId = nil
    }
}
