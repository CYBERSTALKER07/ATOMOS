import Foundation

/// Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
enum SupplierIdempotencyKeys {
    @MainActor
    static func supplierScopeId() -> String {
        let id = TokenStore.shared.supplierId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return id.isEmpty ? "supplier" : id
    }

    static func importDirect(scopeId: String, csvBody: String) -> String {
        "supplier-import-direct:\(scopeId):\(stableHash(csvBody))"
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

    static func billingSetup(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-billing-setup:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func supplierConfigure(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-configure:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func topologyPut(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-topology:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func payoutPolicyPatch(scopeId: String, payloadFingerprint: String) -> String {
        "supplier-payout-policy:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    /// One resolve of this exception. The note is not part of the key: a retry
    /// with a different note must not issue the credit note or cash acceptance again.
    static func exceptionResolve(scopeId: String, kind: String, id: String, creditNoteId: String) -> String {
        let kindKey = kind.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        let noteKey = creditNoteId.trimmingCharacters(in: .whitespacesAndNewlines)
        return "supplier-exception-resolve:\(scopeId):\(kindKey):\(id):\(noteKey)"
    }

    /// The body is always mark-all. The unread set identifies which tap this is,
    /// so a later inbox with new notifications is a new operation.
    static func notificationsMarkAll(scopeId: String, payloadFingerprint: String) -> String {
        "notifications-mark-all:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    static func notificationPreferencesPatch(scopeId: String, payloadFingerprint: String) -> String {
        "notification-preferences:\(scopeId):\(stableHash(payloadFingerprint))"
    }

    /// The support ticket is the action. The reason stays in the body so a
    /// reworded reason cannot start a second permanent disable.
    static func creditAdminDisable(mode: String, supplierId: String, retailerId: String, ticketId: String) -> String {
        let ticket = ticketId.trimmingCharacters(in: .whitespacesAndNewlines)
        return "credit-admin-disable:\(mode):\(supplierId):\(retailerId):\(stableHash(ticket))"
    }

    /// Exact encoded body. The same value produces the same fingerprint, so a retry
    /// reuses the key and a changed form does not. Matches the IETF rule that one
    /// idempotency key must not be reused with a different payload.
    static func encodedPayload<T: Encodable>(_ value: T) -> String {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        guard let data = try? encoder.encode(value), !data.isEmpty else { return "" }
        return String(decoding: data, as: UTF8.self)
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

    static func catalogCreate(scopeId: String, name: String, priceMinor: Int64, barcode: String?) -> String {
        "supplier-catalog-create:\(scopeId):\(stableHash("\(name):\(priceMinor):\(barcode ?? "")"))"
    }

    static func catalogUpdate(productId: String, priceMinor: Int64, unitVolumeVu: Double, barcode: String?) -> String {
        "supplier-catalog-update:\(productId):\(priceMinor):\(unitVolumeVu):\(barcode ?? "")"
    }

    static func claimApprove(claimId: String, settlementMode: String) -> String {
        "supplier-claim-approve:\(claimId):\(settlementMode)"
    }

    static func claimReject(claimId: String) -> String {
        "supplier-claim-reject:\(claimId)"
    }

    static func deviceToken(platform: String, token: String) -> String {
        "device-token:\(platform):\(stableHash(token))"
    }

    static func payoutExport(batchId: String) -> String {
        "supplier-payout-export:\(batchId)"
    }

    static func payoutMarkPaid(batchId: String) -> String {
        "supplier-payout-mark-paid:\(batchId)"
    }

    static func payoutDispatch(batchId: String, live: Bool) -> String {
        "supplier-payout-dispatch:\(batchId):\(live ? "live" : "dry")"
    }

    static func replenishmentTrigger(scopeId: String, day: String) -> String {
        "supplier-replenishment-trigger:\(scopeId):\(day)"
    }

    static func utcDay(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone(identifier: "UTC")
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
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

    static func migrationProjectCreate(scopeId: String, name: String) -> String {
        "mig-proj-create:\(scopeId):\(stableHash(name.trimmingCharacters(in: .whitespacesAndNewlines)))"
    }

    static func migrationBundleCreate(projectId: String, fingerprint: String) -> String {
        "mig-bundle-create:\(projectId):\(stableHash(fingerprint))"
    }

    static func migrationBundleUploaded(bundleId: String, fingerprint: String) -> String {
        "mig-bundle-up:\(bundleId):\(stableHash(fingerprint))"
    }

    static func migrationBundleSort(bundleId: String, fingerprint: String) -> String {
        "mig-bundle-sort:\(bundleId):\(stableHash(fingerprint))"
    }

    static func migrationMappingConfirm(sessionId: String, fingerprint: String) -> String {
        "mig-map:\(sessionId):\(stableHash(fingerprint))"
    }

    static func migrationDryRun(scopeId: String, applyMode: String) -> String {
        "mig-dryrun:\(scopeId):\(applyMode.uppercased())"
    }

    static func migrationSessionApprove(sessionId: String, reportId: String?) -> String {
        "mig-approve:\(sessionId):\(reportId ?? "latest")"
    }

    static func migrationSessionApply(sessionId: String, fingerprint: String?) -> String {
        "mig-apply:\(sessionId):\(fingerprint ?? "latest")"
    }

    static func migrationSessionRollback(sessionId: String) -> String {
        "mig-rollback:\(sessionId)"
    }

    static func migrationOpenOrdersRelease(scopeId: String, orderIds: [String], targetStatus: String) -> String {
        let joined = orderIds.sorted().joined(separator: ",")
        return "mig-release:\(scopeId):\(targetStatus.uppercased()):\(stableHash(joined))"
    }

    static func migrationCutover(projectId: String) -> String {
        "mig-cutover:\(projectId)"
    }

    static func joinRequestDecide(requestId: String, decision: String) -> String {
        "sup-join-req:\(requestId):\(decision.lowercased())"
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
