//
//  LocationInterpolator.swift
//  driverappios
//

import CoreLocation
import Foundation

@MainActor
final class LocationInterpolator {
    private struct Sample {
        let coordinate: CLLocationCoordinate2D
        let bearing: CLLocationDirection
        let speedMps: CLLocationSpeed
        let time: Date
    }

    private(set) var displayCoordinate: CLLocationCoordinate2D?
    private(set) var displayBearing: CLLocationDirection = 0
    private(set) var displaySpeedMps: CLLocationSpeed = 0

    private var from: Sample?
    private var to: Sample?

    func onGps(_ location: CLLocation) {
        let sample = Sample(
            coordinate: location.coordinate,
            bearing: location.course >= 0 ? location.course : displayBearing,
            speedMps: max(location.speed, 0),
            time: location.timestamp
        )
        if to == nil {
            to = sample
            publish(sample)
            return
        }
        from = to
        to = sample
    }

    func tick(now: Date = .now) {
        guard let end = to else { return }
        guard let start = from else {
            publish(end)
            return
        }
        let duration = max(end.time.timeIntervalSince(start.time), 0.001)
        let progress = min(max(now.timeIntervalSince(start.time) / duration, 0), 1)
        let lat = start.coordinate.latitude + (end.coordinate.latitude - start.coordinate.latitude) * progress
        let lng = start.coordinate.longitude + (end.coordinate.longitude - start.coordinate.longitude) * progress
        displayCoordinate = CLLocationCoordinate2D(latitude: lat, longitude: lng)
        displayBearing = start.bearing + (end.bearing - start.bearing) * progress
        displaySpeedMps = start.speedMps + (end.speedMps - start.speedMps) * progress
    }

    func clear() {
        from = nil
        to = nil
        displayCoordinate = nil
        displayBearing = 0
        displaySpeedMps = 0
    }

    private func publish(_ sample: Sample) {
        displayCoordinate = sample.coordinate
        displayBearing = sample.bearing
        displaySpeedMps = sample.speedMps
    }
}
