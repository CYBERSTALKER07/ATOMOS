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
}
