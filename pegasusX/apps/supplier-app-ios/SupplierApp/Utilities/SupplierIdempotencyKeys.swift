import Foundation

/// Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
enum SupplierIdempotencyKeys {
    @MainActor
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

    static func profileUpdate(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-profile-update:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func businessSetup(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-business-setup:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func pricingRulePatch(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-pricing-rule-patch:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func inventoryAdjust(scopeId: String, skuId: String, quantityDelta: Int64, version: Int64) -> String {
        "supplier-inventory-adjust:\(scopeId):\(skuId):\(quantityDelta):\(version)"
    }

    static func retailerPriceOverrideCreate(scopeId: String, retailerId: String, productId: String, priceMinor: Int64) -> String {
        "supplier-retailer-price-create:\(scopeId):\(retailerId):\(productId):\(priceMinor)"
    }

    static func retailerPriceOverrideDelete(scopeId: String, overrideId: String) -> String {
        "supplier-retailer-price-delete:\(scopeId):\(overrideId)"
    }

    static func promotionCreate(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-promotion-create:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func promotionUpdate(scopeId: String, promotionId: String, payloadFingerprint: String) -> String {
        "supplier-promotion-update:\(scopeId):\(promotionId):\(stableHash(payloadFingerprint))"
    }

    static func promotionDeactivate(scopeId: String, promotionId: String) -> String {
        "supplier-promotion-deactivate:\(scopeId):\(promotionId)"
    }

    static func planningScenario(scopeId: String, factoryDowntimeHours: Int, demandDeltaPct: Double) -> String {
        "supplier-planning-scenario:\(scopeId):\(factoryDowntimeHours):\(demandDeltaPct)"
    }

    static func seasonalOverrideCreate(scopeId: String, startDate: String, endDate: String) -> String {
        "supplier-seasonal-override:\(scopeId):\(stableHash("\(startDate):\(endDate)"))"
    }

    static func networkModePut(scopeId: String, mode: String) -> String {
        "supplier-network-mode:\(scopeId):\(mode.trimmingCharacters(in: .whitespacesAndNewlines).uppercased())"
    }

    static func planningPullMatrix(scopeId: String) -> String {
        "supplier-planning-pull-matrix:\(scopeId)"
    }

    static func planningPredictivePush(scopeId: String) -> String {
        "supplier-planning-predictive-push:\(scopeId)"
    }

    static func loyaltyProgramPatch(scopeId: String, reason: String) -> String {
        "supplier-loyalty-program:\(scopeId):\(stableHash(reason.trimmingCharacters(in: .whitespacesAndNewlines)))"
    }

    static func planningKillSwitch(scopeId: String, reason: String) -> String {
        "supplier-planning-kill-switch:\(scopeId):\(stableHash(reason.trimmingCharacters(in: .whitespacesAndNewlines)))"
    }

    static func payoutGenerate(scopeId: String, periodStart: String, periodEnd: String) -> String {
        "supplier-payout-generate:\(scopeId):\(periodStart):\(periodEnd)"
    }

    static func returnPolicyPut(scopeId: String, hours: Int64) -> String {
        "supplier-return-policy:\(scopeId):\(hours)"
    }

    static func controlTowerZoneOverride(scopeId: String, action: String, polygonFingerprint: String) -> String {
        "supplier-control-tower-override:\(scopeId):\(stableHash("\(action):\(polygonFingerprint)"))"
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
