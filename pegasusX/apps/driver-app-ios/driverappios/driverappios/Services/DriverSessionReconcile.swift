//
//  DriverSessionReconcile.swift
//  driverappios
//
//  Refetch server-authoritative driver snapshots after transport reconnect.
//

import Foundation

enum DriverSessionReconcile {
    private static let dateFormatter: DateFormatter = {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        return formatter
    }()

    static func run() async {
        let today = dateFormatter.string(from: Date())
        async let ordersTask = APIClient.shared.getAssignedOrders()
        async let manifestTask = APIClient.shared.getManifest(date: today)
        _ = try? await ordersTask
        _ = try? await manifestTask
    }
}
