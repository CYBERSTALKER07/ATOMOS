//
//  TelemetryPayload.swift
//  driverappios
//

import Foundation

struct TelemetryPayload: Codable {
    let driver_id: String
    let latitude: Double
    let longitude: Double
    let accuracy: Double?
    let timestamp: Double
}
