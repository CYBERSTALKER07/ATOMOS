//
//  RetailerIdempotency.swift
//  reatilerapp
//
//  Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
//

import Foundation

enum RetailerIdempotency {
    static func orderCreate(items: [ProcurementOrderRequest.Item]) -> String {
        let fingerprint = items
            .map { "\($0.productId):\($0.quantity)" }
            .sorted()
            .joined(separator: "|")
        return "retailer-procurement:\(fingerprint)"
    }

    static func confirmCash(orderId: String) -> String {
        "retailer-confirm-cash:\(orderId)"
    }

    static func cancel(orderId: String) -> String {
        "retailer-cancel:\(orderId)"
    }

    static func confirmPreorder(orderId: String) -> String {
        "retailer-confirm-preorder:\(orderId)"
    }

    static func confirmAI(orderId: String) -> String {
        "retailer-confirm-ai:\(orderId)"
    }

    static func requestCancel(orderId: String) -> String {
        "retailer-request-cancel:\(orderId)"
    }

    static func shopClosedResponse(orderId: String, response: String) -> String {
        "shop-closed-response:\(orderId):\(response)"
    }

    static func acceptDeliveryProposal(orderId: String) -> String {
        "retailer-accept-delivery-proposal:\(orderId)"
    }

    static func rejectDeliveryProposal(orderId: String, reason: String = "") -> String {
        "retailer-reject-delivery-proposal:\(orderId):\(stableHash(reason))"
    }

    static func rejectPreorder(orderId: String, reason: String = "") -> String {
        "retailer-reject-preorder:\(orderId):\(stableHash(reason))"
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
