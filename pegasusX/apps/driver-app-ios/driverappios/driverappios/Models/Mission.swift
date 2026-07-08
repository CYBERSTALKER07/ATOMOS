//
//  Mission.swift
//  driverappios
//

import Foundation

struct Mission: Codable, Identifiable, Hashable {
    let order_id: String
    let state: String           // "EN_ROUTE", "IN_TRANSIT", "ARRIVED", etc.
    let target_lat: Double
    let target_lng: Double
    let amount: Int
    let gateway: String         // "CASH" | "GLOBAL_PAY"
    let estimated_arrival_at: String?
    let route_id: String?
    let sequence_index: Int?

    var id: String { order_id }
}

