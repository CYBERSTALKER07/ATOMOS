//
//  QRPayload.swift
//  driverappios
//

import Foundation

/// Scanned from retailer QR code.
struct QRPayload: Codable {
    let order_id: String
    let token: String
}
