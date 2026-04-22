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
    let gateway: String         // "CASH" | "PAYME" | "CLICK" | "UZCARD"
    let estimated_arrival_at: String?
    let route_id: String?
    let sequence_index: Int?

    var id: String { order_id }
}

// MARK: - Mock Data

extension Mission {
    static let mockMissions: [Mission] = [
        Mission(
            order_id: "ORD-TASH-0056",
            state: "EN_ROUTE",
            target_lat: 41.3111,
            target_lng: 69.2797,
            amount: 1_247_000,
            gateway: "PAYME",
            estimated_arrival_at: nil,
            route_id: nil,
            sequence_index: nil
        ),
        Mission(
            order_id: "ORD-TASH-0057",
            state: "EN_ROUTE",
            target_lat: 41.2887,
            target_lng: 69.2044,
            amount: 856_400,
            gateway: "CASH",
            estimated_arrival_at: nil,
            route_id: nil,
            sequence_index: nil
        ),
        Mission(
            order_id: "ORD-TASH-0058",
            state: "EN_ROUTE",
            target_lat: 41.3275,
            target_lng: 69.3341,
            amount: 2_100_000,
            gateway: "UZCARD",
            estimated_arrival_at: nil,
            route_id: nil,
            sequence_index: nil
        ),
    ]
}
