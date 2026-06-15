//
//  PayloadIdempotency.swift
//  payload-app-ios
//
//  Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
//

import Foundation

enum PayloadIdempotency {
    static func key(action: String, entityId: String) -> String {
        "payload-\(action)-\(entityId)"
    }

    static func recommendReassign(orderId: String) -> String {
        key(action: "recommend-reassign", entityId: orderId)
    }

    static func fleetReassign(orderIds: [String]) -> String {
        let sorted = orderIds
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .sorted()
            .joined(separator: ",")
        return key(action: "fleet-reassign", entityId: sorted)
    }

    static func applyReassign(orderId: String, toDriverId: String) -> String {
        key(action: "reassign-order", entityId: "\(orderId)-\(toDriverId)")
    }
}
