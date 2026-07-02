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

    static func requestEarlyComplete(reason: String) -> String {
        "driver-request-early-complete:\(driverId()):\(stableHash(reason))"
    }

    static func routeReorder(routeId: String, orderSequence: [String]) -> String {
        let seq = orderSequence.map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }
        return "driver-route-reorder:\(driverId()):\(routeId):\(stableHash(seq.joined(separator: ",")))"
    }

    static func supplyTransferArrive(transferId: String) -> String {
        "driver-supply-arrive:\(driverId()):\(transferId)"
    }

    static func amendOrder(orderId: String, items: [AmendItemPayload]) -> String {
        let fingerprint = items
            .sorted { $0.productId < $1.productId }
            .map { "\($0.productId):\($0.acceptedQty):\($0.rejectedQty):\($0.reason)" }
            .joined(separator: "|")
        return "driver-amend:\(driverId()):\(orderId):\(stableHash(fingerprint))"
    }

    static func transitionState(orderId: String, newState: String) -> String {
        let state = newState.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        return "driver-transition-state:\(driverId()):\(orderId):\(state)"
    }

    static func availability(onShift: Bool, reason: String = "", note: String? = nil) -> String {
        let fingerprint = stableHash("\(onShift):\(reason.trimmingCharacters(in: .whitespacesAndNewlines)):\((note ?? "").trimmingCharacters(in: .whitespacesAndNewlines))")
        return "driver-availability:\(driverId()):\(fingerprint)"
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
