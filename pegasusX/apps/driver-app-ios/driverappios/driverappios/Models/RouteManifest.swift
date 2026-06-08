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

// MARK: - Mock Data

extension RouteManifest {
    static let mock = RouteManifest(
        driver_id: "DRV-AMIR-001",
        date: "2026-03-18",
        expires_at: Date().timeIntervalSince1970 + 86400,
        hashes: [
            "ORD-TASH-0056": sha256Hex("secret-token-0056"),
            "ORD-TASH-0057": sha256Hex("secret-token-0057"),
            "ORD-TASH-0058": sha256Hex("secret-token-0058"),
        ]
    )
}
