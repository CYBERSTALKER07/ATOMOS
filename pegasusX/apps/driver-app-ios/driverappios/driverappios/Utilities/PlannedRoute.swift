//
//  PlannedRoute.swift
//  driverappios
//

import CoreLocation
import Foundation
import MapKit
import SwiftUI

func buildPlannedRouteCoordinates(orders: [Order], activeRouteId: String?) -> [CLLocationCoordinate2D] {
    guard let activeRouteId, !activeRouteId.isEmpty else { return [] }
    return orders
        .filter { order in
            order.routeId == activeRouteId && order.state.isActive
        }
        .sorted { ($0.sequenceIndex ?? 0) < ($1.sequenceIndex ?? 0) }
        .map { CLLocationCoordinate2D(latitude: $0.latitude, longitude: $0.longitude) }
}

func resolveActiveRouteId(orders: [Order]) -> String? {
    let executionStates: Set<OrderState> = [
        .IN_TRANSIT, .ARRIVING, .ARRIVED, .ARRIVED_SHOP_CLOSED,
        .AWAITING_PAYMENT, .PENDING_CASH_COLLECTION, .DISPATCHED, .LOADED,
    ]
    let active = orders.first { order in
        executionStates.contains(order.state)
    } ?? orders.first { order in
        order.state.isActive
    }
    return active?.routeId
}

enum MapCameraMath {
    static func lookAheadDistance(speedMps: CLLocationSpeed) -> Double {
        min(max(speedMps * MapCameraConfig.lookAheadSeconds, 0), MapCameraConfig.maxLookAheadMeters)
    }

    static func trackingCamera(
        coordinate: CLLocationCoordinate2D,
        bearing: CLLocationDirection,
        speedMps: CLLocationSpeed
    ) -> MapCamera {
        let aheadMeters = lookAheadDistance(speedMps: speedMps)
        let center = aheadMeters > 1
            ? offsetCoordinate(
                lat: coordinate.latitude,
                lon: coordinate.longitude,
                bearingDegrees: bearing,
                distanceMeters: aheadMeters
            )
            : coordinate
        let distance = min(
            max(speedMps * MapCameraConfig.speedZoomFactor, MapCameraConfig.minZoomDistance),
            MapCameraConfig.maxZoomDistance
        )
        return MapCamera(
            centerCoordinate: center,
            distance: distance,
            heading: bearing,
            pitch: MapCameraConfig.navigationPitch
        )
    }
}
