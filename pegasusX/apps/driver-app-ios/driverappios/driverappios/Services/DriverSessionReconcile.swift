//
//  DriverSessionReconcile.swift
//  driverappios
//
//  Refetch server-authoritative driver snapshots after transport reconnect.
//

import Foundation

enum DriverSessionReconcile {
    static func run() async {
        _ = try? await APIClient.shared.getAssignedOrders()
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        let today = formatter.string(from: Date())
        _ = try? await APIClient.shared.getManifest(date: today)
    }
}
