//
//  DriverLiveActivityManager.swift
//  driverappios
//
//  Manages Driver-perspective ActivityKit Live Activities and Dynamic Island widgets for active route navigation.
//

import ActivityKit
import Foundation

@MainActor
final class DriverLiveActivityManager {
    static let shared = DriverLiveActivityManager()

    private var currentActivity: Activity<DeliveryActivityAttributes>?
    private var activeOrderId: String?

    private init() {}

    /// Start a Live Activity for the active delivery route navigation.
    func startNavigationActivity(
        orderId: String,
        routeId: String,
        destinationName: String,
        totalItems: Int,
        totalAmountMinor: Int64,
        initialCue: NavigationCue?,
        totalSteps: Int
    ) {
        // ActivityKit availability check
        guard ActivityAuthorizationInfo().areActivitiesEnabled else {
            return
        }

        // Avoid re-requesting if already active for same order
        if currentActivity != nil && activeOrderId == orderId {
            return
        }

        endActivity()

        let attributes = DeliveryActivityAttributes(
            orderId: orderId,
            routeId: routeId,
            supplierName: "Pegasus Logistics",
            destinationName: destinationName,
            totalItemCount: totalItems,
            totalAmountMinorUnits: totalAmountMinor,
            isDriverPerspective: true
        )

        let initialContentState = DeliveryActivityAttributes.ContentState(
            status: "IN_TRANSIT",
            etaMinutes: max(1, Int((initialCue?.distanceM ?? 500) / 400)),
            remainingDistanceMeters: initialCue?.distanceM ?? 500,
            nextInstruction: initialCue?.instruction ?? "Proceed along planned route",
            nextManeuver: initialCue?.maneuver,
            progress: 0.0,
            currentStepIndex: 1,
            totalSteps: max(totalSteps, 1)
        )

        do {
            currentActivity = try Activity.request(
                attributes: attributes,
                content: .init(state: initialContentState, staleDate: nil),
                pushType: nil
            )
            activeOrderId = orderId
        } catch {
            print("[DriverLiveActivityManager] Failed to start Live Activity: \(error.localizedDescription)")
        }
    }

    /// Updates the Live Activity with latest turn-by-turn instruction, distance countdown, and progress.
    func updateNavigationState(
        cue: NavigationCue?,
        etaMinutes: Int,
        progress: Double,
        currentStepIndex: Int,
        totalSteps: Int
    ) {
        guard let activity = currentActivity else { return }

        let updatedState = DeliveryActivityAttributes.ContentState(
            status: "IN_TRANSIT",
            etaMinutes: etaMinutes,
            remainingDistanceMeters: cue?.distanceM ?? 0,
            nextInstruction: cue?.instruction ?? "Continue along route",
            nextManeuver: cue?.maneuver,
            progress: min(max(progress, 0.0), 1.0),
            currentStepIndex: currentStepIndex,
            totalSteps: max(totalSteps, 1),
            lastUpdated: Date()
        )

        Task {
            await activity.update(.init(state: updatedState, staleDate: nil))
        }
    }

    /// Ends the Live Activity when a delivery stop completes or shift ends.
    func endActivity(status: String = "COMPLETED") {
        guard let activity = currentActivity else { return }

        Task {
            let finalState = DeliveryActivityAttributes.ContentState(
                status: status,
                etaMinutes: 0,
                remainingDistanceMeters: 0,
                nextInstruction: status == "COMPLETED" ? "Arrived at destination" : "Navigation ended",
                progress: 1.0,
                lastUpdated: Date()
            )
            await activity.end(.init(state: finalState, staleDate: nil), dismissalPolicy: .default)
        }

        currentActivity = nil
        activeOrderId = nil
    }
}
