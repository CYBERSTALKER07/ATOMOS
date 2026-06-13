//
//  MapCameraConfig.swift
//  driverappios
//

import Foundation

/// Shared driver-map camera constants — keep aligned with Android MapCameraConfig.
enum MapCameraConfig {
    static let idleRecenterMs: UInt64 = 8_000_000_000
    static let interpolationFrameNs: UInt64 = 33_000_000
    static let cameraAnimationSeconds = 0.5
    static let minZoomDistance: Double = 400
    static let maxZoomDistance: Double = 2_500
    static let navigationPitch: Double = 60
    static let speedZoomFactor = 20.0
    static let lookAheadSeconds = 2.0
    static let maxLookAheadMeters = 200.0
}
