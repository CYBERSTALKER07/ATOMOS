//
//  DriverEarnings.swift
//  driverappios
//
//  Mirror of backend-go/fleet/driver_api.go::DriverEarningsResponse + DailyEarning.
//  Returned by GET /v1/driver/earnings.
//

import Foundation

struct DailyEarning: Codable, Hashable, Identifiable {
    let date: String
    let deliveryCount: Int64
    let volume: Int64

    var id: String { date }

    enum CodingKeys: String, CodingKey {
        case date
        case deliveryCount = "delivery_count"
        case volume
    }
}

struct DriverEarningsResponse: Codable {
    let totalDeliveries: Int64
    let totalVolume: Int64
    let totalRoutes: Int64
    let last30Days: [DailyEarning]

    enum CodingKeys: String, CodingKey {
        case totalDeliveries = "total_deliveries"
        case totalVolume = "total_volume"
        case totalRoutes = "total_routes"
        case last30Days = "last_30_days"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        totalDeliveries = (try? c.decode(Int64.self, forKey: .totalDeliveries)) ?? 0
        totalVolume = (try? c.decode(Int64.self, forKey: .totalVolume)) ?? 0
        totalRoutes = (try? c.decode(Int64.self, forKey: .totalRoutes)) ?? 0
        last30Days = (try? c.decode([DailyEarning].self, forKey: .last30Days)) ?? []
    }

    init(totalDeliveries: Int64, totalVolume: Int64, totalRoutes: Int64, last30Days: [DailyEarning]) {
        self.totalDeliveries = totalDeliveries
        self.totalVolume = totalVolume
        self.totalRoutes = totalRoutes
        self.last30Days = last30Days
    }
}
