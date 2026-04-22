//
//  Haversine.swift
//  driverappios
//

import CoreLocation

// MARK: - Equatable conformance for CLLocationCoordinate2D

extension CLLocationCoordinate2D: @retroactive Equatable {
    public static func == (lhs: CLLocationCoordinate2D, rhs: CLLocationCoordinate2D) -> Bool {
        lhs.latitude == rhs.latitude && lhs.longitude == rhs.longitude
    }
}

/// Returns the great-circle distance in **meters** between two coordinates.
func haversineDistance(from: CLLocationCoordinate2D, to: CLLocationCoordinate2D) -> CLLocationDistance {
    let R: Double = 6_371_000 // Earth radius in meters

    let dLat = (to.latitude  - from.latitude).degreesToRadians
    let dLon = (to.longitude - from.longitude).degreesToRadians

    let lat1 = from.latitude.degreesToRadians
    let lat2 = to.latitude.degreesToRadians

    let a = sin(dLat / 2) * sin(dLat / 2)
          + cos(lat1) * cos(lat2) * sin(dLon / 2) * sin(dLon / 2)
    let c = 2 * atan2(sqrt(a), sqrt(1 - a))

    return R * c
}

/// Formats a distance for display.
/// - < 1000 m → "245m"
/// - ≥ 1000 m → "12.50km"
func formattedDistance(_ meters: Double) -> String {
    if meters < 1000 {
        return "\(Int(meters))m"
    } else {
        return String(format: "%.2fkm", meters / 1000)
    }
}

private extension Double {
    var degreesToRadians: Double { self * .pi / 180 }
}
