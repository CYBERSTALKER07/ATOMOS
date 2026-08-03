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
    let todayMinor: Int64
    let weekMinor: Int64
    let monthMinor: Int64

    enum CodingKeys: String, CodingKey {
        case totalDeliveries = "total_deliveries"
        case totalVolume = "total_volume"
        case totalRoutes = "total_routes"
        case last30Days = "last_30_days"
        case todayMinor = "today_minor"
        case weekMinor = "week_minor"
        case monthMinor = "month_minor"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        totalDeliveries = (try? c.decode(Int64.self, forKey: .totalDeliveries)) ?? 0
        totalVolume = (try? c.decode(Int64.self, forKey: .totalVolume)) ?? 0
        totalRoutes = (try? c.decode(Int64.self, forKey: .totalRoutes)) ?? 0
        last30Days = (try? c.decode([DailyEarning].self, forKey: .last30Days)) ?? []
        todayMinor = (try? c.decode(Int64.self, forKey: .todayMinor)) ?? 0
        weekMinor = (try? c.decode(Int64.self, forKey: .weekMinor)) ?? 0
        monthMinor = (try? c.decode(Int64.self, forKey: .monthMinor)) ?? 0
    }

    init(
        totalDeliveries: Int64,
        totalVolume: Int64,
        totalRoutes: Int64,
        last30Days: [DailyEarning],
        todayMinor: Int64 = 0,
        weekMinor: Int64 = 0,
        monthMinor: Int64 = 0
    ) {
        self.totalDeliveries = totalDeliveries
        self.totalVolume = totalVolume
        self.totalRoutes = totalRoutes
        self.last30Days = last30Days
        self.todayMinor = todayMinor
        self.weekMinor = weekMinor
        self.monthMinor = monthMinor
    }
}

struct DriverHistoryRow: Codable, Hashable, Identifiable {
    let orderId: String
    let status: String
    let totalMinor: Int64
    let currency: String
    let completedAt: String

    var id: String { orderId }

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case status
        case totalMinor = "total_minor"
        case currency
        case completedAt = "completed_at"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        orderId = (try? c.decode(String.self, forKey: .orderId)) ?? ""
        status = (try? c.decode(String.self, forKey: .status)) ?? ""
        totalMinor = (try? c.decode(Int64.self, forKey: .totalMinor)) ?? 0
        currency = (try? c.decode(String.self, forKey: .currency)) ?? ""
        completedAt = (try? c.decode(String.self, forKey: .completedAt)) ?? ""
    }
}

struct DriverHistoryResponse: Codable {
    let rows: [DriverHistoryRow]

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        rows = (try? c.decode([DriverHistoryRow].self, forKey: .rows)) ?? []
    }

    enum CodingKeys: String, CodingKey { case rows }
}
