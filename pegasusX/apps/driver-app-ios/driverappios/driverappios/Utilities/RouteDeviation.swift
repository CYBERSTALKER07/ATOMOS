//
//  RouteDeviation.swift
//  driverappios
//

import CoreLocation
import Foundation

enum RouteDeviationAction {
    case none
    case reroute
}

enum RouteDeviationConfig {
    static let offRouteThresholdMeters: CLLocationDistance = 45
    static let sustainedOffRouteSeconds: TimeInterval = 4
    static let minRerouteIntervalSeconds: TimeInterval = 30
}

func distanceToPolylineMeters(
    coordinate: CLLocationCoordinate2D,
    polyline: [CLLocationCoordinate2D]
) -> CLLocationDistance {
    guard !polyline.isEmpty else { return .greatestFiniteMagnitude }
    if polyline.count == 1 {
        return haversineDistance(from: coordinate, to: polyline[0])
    }
    var minDistance = CLLocationDistance.greatestFiniteMagnitude
    for index in 0..<(polyline.count - 1) {
        let distance = distancePointToSegmentMeters(
            coordinate: coordinate,
            start: polyline[index],
            end: polyline[index + 1]
        )
        minDistance = min(minDistance, distance)
    }
    return minDistance
}

func distancePointToSegmentMeters(
    coordinate: CLLocationCoordinate2D,
    start: CLLocationCoordinate2D,
    end: CLLocationCoordinate2D
) -> CLLocationDistance {
    let deltaLng = end.longitude - start.longitude
    let deltaLat = end.latitude - start.latitude
    if deltaLng == 0, deltaLat == 0 {
        return haversineDistance(from: coordinate, to: start)
    }
    var t = ((coordinate.longitude - start.longitude) * deltaLng +
             (coordinate.latitude - start.latitude) * deltaLat) /
        (deltaLng * deltaLng + deltaLat * deltaLat)
    t = max(0, min(1, t))
    let closest = CLLocationCoordinate2D(
        latitude: start.latitude + t * deltaLat,
        longitude: start.longitude + t * deltaLng
    )
    return haversineDistance(from: coordinate, to: closest)
}

@MainActor
final class RouteDeviationTracker {
    private var offRouteSince: Date?
    private var lastRerouteAt: Date?

    func evaluate(
        now: Date,
        coordinate: CLLocationCoordinate2D,
        polyline: [CLLocationCoordinate2D]
    ) -> RouteDeviationAction {
        guard polyline.count >= 2 else {
            offRouteSince = nil
            return .none
        }
        let distance = distanceToPolylineMeters(coordinate: coordinate, polyline: polyline)
        if distance <= RouteDeviationConfig.offRouteThresholdMeters {
            offRouteSince = nil
            return .none
        }
        if offRouteSince == nil {
            offRouteSince = now
            return .none
        }
        guard let offRouteSince,
              now.timeIntervalSince(offRouteSince) >= RouteDeviationConfig.sustainedOffRouteSeconds else {
            return .none
        }
        if let lastRerouteAt,
           now.timeIntervalSince(lastRerouteAt) < RouteDeviationConfig.minRerouteIntervalSeconds {
            return .none
        }
        lastRerouteAt = now
        self.offRouteSince = nil
        return .reroute
    }

    func reset() {
        offRouteSince = nil
    }
}
