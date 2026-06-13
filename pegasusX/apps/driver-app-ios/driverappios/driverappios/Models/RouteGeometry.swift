//
//  RouteGeometry.swift
//  driverappios
//
//  mirror of backend-go/routing.RouteGeometry (keep JSON tags aligned)
//

import Foundation

struct RouteCoordinate: Codable {
    let lat: Double
    let lng: Double
}

struct RouteStep: Codable {
    let instruction: String
    let distanceM: Double
    let durationS: Double
    let maneuver: String?
    let lat: Double
    let lng: Double

    enum CodingKeys: String, CodingKey {
        case instruction
        case distanceM = "distance_m"
        case durationS = "duration_s"
        case maneuver
        case lat
        case lng
    }
}

struct RouteGeometryResponse: Codable {
    let routeId: String
    let encodedPolyline: String?
    let coordinates: [RouteCoordinate]
    let source: String
    let stopCount: Int
    let steps: [RouteStep]

    enum CodingKeys: String, CodingKey {
        case routeId = "route_id"
        case encodedPolyline = "encoded_polyline"
        case coordinates
        case source
        case stopCount = "stop_count"
        case steps
    }
}
