//
//  RouteManifest.swift
//  driverappios
//

import Foundation

struct RouteManifest: Codable {
    let driver_id: String
    let date: String             // "2025-01-15"
    let expires_at: TimeInterval // Unix epoch seconds
    let hashes: [String: String] // OrderId → SHA256(DeliveryToken)

    var isValid: Bool { Date().timeIntervalSince1970 < expires_at }
}

