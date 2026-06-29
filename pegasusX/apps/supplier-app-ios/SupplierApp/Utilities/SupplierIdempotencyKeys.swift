import Foundation

/// Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
enum SupplierIdempotencyKeys {
    static func supplierScopeId() -> String {
        let id = TokenStore.shared.supplierId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return id.isEmpty ? "supplier" : id
    }

    static func importCreate(scopeId: String, fileName: String, fileSizeBytes: Int) -> String {
        "supplier-import-create:\(scopeId):\(stableHash("\(fileName):\(fileSizeBytes)"))"
    }

    static func importIngest(sessionId: String, csvBody: String) -> String {
        "supplier-import-ingest:\(sessionId):\(stableHash(csvBody))"
    }

    static func importApprove(sessionId: String) -> String {
        "supplier-import-approve:\(sessionId)"
    }

    static func importApply(sessionId: String) -> String {
        "supplier-import-apply:\(sessionId)"
    }

    static func orgMemberCreate(scopeId: String, phone: String) -> String {
        "supplier-org-member-create:\(scopeId):\(stableHash(phone))"
    }

    static func orgMemberUpdate(scopeId: String, userId: String, revision: String) -> String {
        "supplier-org-member-update:\(scopeId):\(userId):\(stableHash(revision))"
    }

    static func orgMemberDeactivate(scopeId: String, userId: String) -> String {
        "supplier-org-member-deactivate:\(scopeId):\(userId)"
    }

    static func fleetDriverCreate(scopeId: String, phone: String) -> String {
        "supplier-fleet-driver-create:\(scopeId):\(stableHash(phone))"
    }

    static func fleetVehicleCreate(scopeId: String, licensePlate: String) -> String {
        "supplier-fleet-vehicle-create:\(scopeId):\(stableHash(licensePlate))"
    }

    static func chargeback(orderId: String, reason: String) -> String {
        "supplier-chargeback:\(orderId):\(stableHash(reason))"
    }

    static func chargebackReversal(chargebackId: String, reason: String) -> String {
        "supplier-chargeback-reversal:\(chargebackId):\(stableHash(reason))"
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
