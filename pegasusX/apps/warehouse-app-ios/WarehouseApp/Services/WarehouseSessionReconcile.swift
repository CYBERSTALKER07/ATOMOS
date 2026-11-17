//
//  WarehouseSessionReconcile.swift
//  WarehouseApp
//
//  Refetch server-authoritative warehouse snapshots after transport reconnect.
//

import Foundation

enum WarehouseSessionReconcile {
    private static let utcDateFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        return formatter
    }()

    static func run() async {
        let tomorrow = Calendar.current.date(byAdding: .day, value: 1, to: Date()) ?? Date()
        let tomorrowStr = utcDateFormatter.string(from: tomorrow)
        async let previewTask = WarehouseService.dispatchPreview()
        async let locksTask = WarehouseService.dispatchLocks()
        async let forecastTask = WarehouseService.demandForecast(days: 7)
        async let replenishmentTask = WarehouseService.replenishmentInsights()
        async let opsBoardTask = WarehouseService.opsBoard(date: tomorrowStr)
        _ = try? await previewTask
        _ = try? await locksTask
        _ = try? await forecastTask
        _ = try? await replenishmentTask
        _ = try? await opsBoardTask
    }
}
