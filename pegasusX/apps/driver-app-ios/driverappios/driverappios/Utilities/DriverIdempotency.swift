//
//  DriverIdempotency.swift
//  driverappios
//
//  Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
//

import Foundation

enum DriverIdempotency {
    private static func driverId() -> String {
        let id = TokenStore.shared.userId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return id.isEmpty ? "driver" : id
    }

    static func deliver(orderId: String) -> String {
        "driver-deliver:\(driverId()):\(orderId)"
    }

    static func offload(orderId: String) -> String {
        "driver-offload:\(driverId()):\(orderId)"
    }

    static func complete(orderId: String) -> String {
        "driver-complete:\(driverId()):\(orderId)"
    }

    static func collectCash(orderId: String) -> String {
        "driver-collect-cash:\(driverId()):\(orderId)"
    }

    static func confirmPaymentBypass(orderId: String) -> String {
        "driver-confirm-payment-bypass:\(driverId()):\(orderId)"
    }

    static func bypassOffload(orderId: String) -> String {
        "driver-bypass-offload:\(driverId()):\(orderId)"
    }

    static func depart(truckId: String) -> String {
        "driver-depart:\(driverId()):\(truckId)"
    }

    static func returnComplete(truckId: String) -> String {
        "driver-return-complete:\(driverId()):\(truckId)"
    }
}
