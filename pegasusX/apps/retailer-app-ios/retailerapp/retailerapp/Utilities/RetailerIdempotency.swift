//
//  RetailerIdempotency.swift
//  retailerapp
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

    static func rejectAI(orderId: String, reason: String = "") -> String {
        "retailer-reject-ai:\(orderId):\(stableHash(reason))"
    }

    static func editPreorder(orderId: String) -> String {
        "retailer-edit-preorder:\(orderId)"
    }

    static func setup(retailerId: String) -> String {
        "retailer-setup:\(retailerId)"
    }

    static func supplierAdd(supplierId: String) -> String {
        "retailer-supplier-add:\(supplierId)"
    }

    static func supplierRemove(supplierId: String) -> String {
        "retailer-supplier-remove:\(supplierId)"
    }

    static func profileUpdate(retailerId: String, payloadFingerprint: String) -> String {
        "retailer-profile-update:\(retailerId):\(stableHash(payloadFingerprint))"
    }

    /// Stable file-claim key — same body retries return the same claim_id (G11/G25).
    static func claimFile(orderId: String, bodyFingerprint: String) -> String {
        "claim-file:\(orderId):\(stableHash(bodyFingerprint))"
    }

    /// RFC 9562 UUIDv7 generator (48-bit Unix epoch ms + ver 7 + variant 0b10 + 74 random bits)
    /// for offline outbox & POS mutation idempotency (Invariant I1).
    static func uuidV7(timestampMs: Int64? = nil) -> String {
        let ms = UInt64(max(0, timestampMs ?? Int64(Date().timeIntervalSince1970 * 1000))) & 0x0000_FFFF_FFFF_FFFF
        let randA = UInt16.random(in: 0...0x0FFF)
        let randB = UInt64.random(in: 0...0x3FFF_FFFF_FFFF_FFFF)

        let p1 = UInt32((ms >> 16) & 0xFFFF_FFFF)
        let p2 = UInt16(ms & 0xFFFF)
        let p3 = UInt16(0x7000) | randA
        let p4 = UInt16(0x8000) | UInt16((randB >> 48) & 0x3FFF)
        let p5 = randB & 0x0000_FFFF_FFFF_FFFF

        return String(format: "%08x-%04x-%04x-%04x-%012llx", p1, p2, p3, p4, p5)
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
