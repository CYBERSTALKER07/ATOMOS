//
//  RouteNavigation.swift
//  driverappios
//

import CoreLocation
import Foundation

struct NavigationCue: Equatable {
    let instruction: String
    let distanceM: Double
    let maneuver: String?
}

enum RouteNavigation {
    static let stepPassedMeters: CLLocationDistance = 35

    static func advanceStepIndex(
        currentIndex: Int,
        steps: [RouteStep],
        coordinate: CLLocationCoordinate2D
    ) -> Int {
        guard !steps.isEmpty else { return 0 }
        var lastPassedIndex = -1
        for index in steps.indices {
            let step = steps[index]
            let stepCoordinate = CLLocationCoordinate2D(latitude: step.lat, longitude: step.lng)
            let distance = haversineDistance(from: coordinate, to: stepCoordinate)
            if distance <= stepPassedMeters {
                lastPassedIndex = index
            }
        }
        if lastPassedIndex < 0 {
            return min(max(currentIndex, 0), steps.count - 1)
        }
        return min(lastPassedIndex + 1, steps.count - 1)
    }

    static func resolveCue(
        steps: [RouteStep],
        stepIndex: Int,
        coordinate: CLLocationCoordinate2D
    ) -> NavigationCue? {
        guard steps.indices.contains(stepIndex) else { return nil }
        let step = steps[stepIndex]
        let stepCoordinate = CLLocationCoordinate2D(latitude: step.lat, longitude: step.lng)
        let distance = haversineDistance(from: coordinate, to: stepCoordinate)
        return NavigationCue(
            instruction: step.instruction,
            distanceM: distance,
            maneuver: step.maneuver
        )
    }

    static func formatDistance(_ meters: Double) -> String {
        if meters < 1000 {
            return "\(Int(meters.rounded())) m"
        }
        return String(format: "%.1f km", meters / 1000.0)
    }
}
