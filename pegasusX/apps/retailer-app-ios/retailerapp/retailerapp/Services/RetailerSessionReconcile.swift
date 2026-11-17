//
//  RetailerSessionReconcile.swift
//  retailerapp
//
//  Refetch server-authoritative retailer snapshots after transport reconnect.
//

import Foundation

enum RetailerSessionReconcile {
    @MainActor
    static func run(api: APIClient? = nil) async {
        let api = api ?? .shared
        _ = try? await api.getActiveFulfillments()
        _ = try? await api.getPendingPayments()
        _ = try? await api.getTrackingOrders()
    }
}
