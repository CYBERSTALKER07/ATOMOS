//
//  RetailerSessionReconcile.swift
//  retailerapp
//
//  Refetch server-authoritative retailer snapshots after transport reconnect.
//

import Foundation

enum RetailerSessionReconcile {
    static func run(api: APIClient = .shared) async {
        _ = try? await api.getActiveFulfillments()
        _ = try? await api.getPendingPayments()
        _ = try? await api.getTrackingOrders()
    }
}
