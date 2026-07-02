//
//  WarehouseSessionReconcile.swift
//  WarehouseApp
//
//  Refetch server-authoritative warehouse snapshots after transport reconnect.
//

import Foundation

enum WarehouseSessionReconcile {
    static func run() async {
        _ = try? await WarehouseService.dispatchPreview()
        _ = try? await WarehouseService.dispatchLocks()
        _ = try? await WarehouseService.demandForecast(days: 7)
        _ = try? await WarehouseService.replenishmentInsights()
        let tomorrow = Calendar.current.date(byAdding: .day, value: 1, to: Date()) ?? Date()
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.timeZone = TimeZone(secondsFromGMT: 0)
        _ = try? await WarehouseService.opsBoard(date: formatter.string(from: tomorrow))
    }
}
