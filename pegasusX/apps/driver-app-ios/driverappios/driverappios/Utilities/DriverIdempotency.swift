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

    static func reportShopClosed(orderId: String) -> String {
        "driver-report-shop-closed:\(driverId()):\(orderId)"
    }

    static func depart(truckId: String) -> String {
        "driver-depart:\(driverId()):\(truckId)"
    }

    static func returnComplete(truckId: String) -> String {
        "driver-return-complete:\(driverId()):\(truckId)"
    }

    static func syncBatch(orderSignatures: [String]) -> String {
        let sorted = orderSignatures
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .sorted()
        return "driver-sync-batch:\(driverId()):\(stableHash(sorted.joined(separator: ",")))"
    }

    static func markArrived(orderId: String) -> String {
        "driver-mark-arrived-\(orderId)"
    }

    static func splitPayment(orderId: String, cashMinor: Int64, cardMinor: Int64) -> String {
        "driver-split-payment:\(driverId()):\(orderId):\(cashMinor):\(cardMinor)"
    }

    static func creditDelivery(orderId: String) -> String {
        "driver-credit-delivery:\(driverId()):\(orderId)"
    }

    static func missingItems(orderId: String) -> String {
        "driver-missing-items:\(driverId()):\(orderId)"
    }

    static func reportDamage(orderId: String) -> String {
        "driver-report-damage:\(driverId()):\(orderId)"
    }

    static func negotiate(orderId: String) -> String {
        "driver-negotiate:\(driverId()):\(orderId)"
    }

    private static func stableHash(_ input: String) -> String {
        var hash: UInt32 = 2166136261
        for scalar in input.unicodeScalars {
            hash ^= scalar.value
            hash = hash &* 16777619
        }
        return String(hash, radix: 36)
    }
}
