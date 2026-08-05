package com.pegasusx.supplier.util

import com.pegasusx.supplier.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object SupplierIdempotencyKeys {
    fun shopClosedResolve(attemptId: String, action: String): String =
        "shop-closed-resolve:$attemptId:$action"

    fun approveEarlyComplete(driverId: String): String =
        "supplier-approve-early-complete:$driverId"

    fun negotiationResolve(proposalId: String, action: String): String =
        "supplier-negotiate-resolve:$proposalId:$action"

    fun resolveReturn(returnId: String, resolution: String): String =
        "supplier-resolve-return:$returnId:${resolution.uppercase()}"

    fun dispatch(
        supplierId: String,
        warehouseId: String,
        mode: String,
        routeFingerprint: String,
    ): String =
        "supplier-dispatch:$supplierId:$warehouseId:$mode:${stableHash(routeFingerprint)}"

    fun broadcast(scopeId: String, role: String, title: String, body: String): String =
        "supplier-broadcast:$scopeId:${stableHash("$role:$title:$body")}"

    fun paymentBypass(orderId: String, reason: String): String =
        "supplier-payment-bypass:$orderId:${stableHash(reason)}"

    fun vetOrder(orderId: String, decision: String): String =
        "supplier-vet-order:$orderId:${decision.uppercase()}"

    fun warehouseOrderPropose(orderId: String, proposedDate: String, reason: String): String =
        "warehouse-order-propose-delivery:$orderId:${stableHash("$proposedDate:$reason")}"

    fun warehouseOrderDelay(orderId: String): String = "warehouse-order-delay:$orderId"

    fun warehouseOrderReject(orderId: String, reason: String): String =
        "warehouse-order-reject:$orderId:${stableHash(reason)}"

    fun importCreate(scopeId: String, fileName: String, fileSizeBytes: Int): String =
        "supplier-import-create:$scopeId:${stableHash("$fileName:$fileSizeBytes")}"

    fun importIngest(sessionId: String, csvBody: String): String =
        "supplier-import-ingest:$sessionId:${stableHash(csvBody)}"

    fun importApprove(sessionId: String): String = "supplier-import-approve:$sessionId"

    fun importApply(sessionId: String): String = "supplier-import-apply:$sessionId"

    fun orgMemberCreate(scopeId: String, phone: String): String =
        "supplier-org-member-create:$scopeId:${stableHash(phone)}"

    fun orgMemberUpdate(scopeId: String, userId: String, revision: String): String =
        "supplier-org-member-update:$scopeId:$userId:${stableHash(revision)}"

    fun orgMemberDeactivate(scopeId: String, userId: String): String =
        "supplier-org-member-deactivate:$scopeId:$userId"

    fun fleetDriverCreate(scopeId: String, phone: String): String =
        "supplier-fleet-driver-create:$scopeId:${stableHash(phone)}"

    fun fleetVehicleCreate(scopeId: String, licensePlate: String): String =
        "supplier-fleet-vehicle-create:$scopeId:${stableHash(licensePlate)}"

    fun chargeback(orderId: String, reason: String): String =
        "supplier-chargeback:$orderId:${stableHash(reason)}"

    fun chargebackReversal(chargebackId: String, reason: String): String =
        "supplier-chargeback-reversal:$chargebackId:${stableHash(reason)}"

    fun supplierScopeId(): String = TokenHolder.supplierId?.takeIf { it.isNotBlank() } ?: "supplier"

    fun profileUpdate(scopeId: String, payloadFingerprint: String): String =
        "supplier-profile-update:$scopeId:${stableHash(payloadFingerprint)}"

    fun businessSetup(scopeId: String, payloadFingerprint: String): String =
        "supplier-business-setup:$scopeId:${stableHash(payloadFingerprint)}"

    fun pricingRulePatch(scopeId: String, payloadFingerprint: String): String =
        "supplier-pricing-rule-patch:$scopeId:${stableHash(payloadFingerprint)}"

    fun inventoryAdjust(scopeId: String, skuId: String, quantityDelta: Long, version: Long): String =
        "supplier-inventory-adjust:$scopeId:$skuId:$quantityDelta:$version"

    fun retailerPriceOverrideCreate(scopeId: String, retailerId: String, productId: String, priceMinor: Long): String =
        "supplier-retailer-price-create:$scopeId:$retailerId:$productId:$priceMinor"

    fun retailerPriceOverrideDelete(scopeId: String, overrideId: String): String =
        "supplier-retailer-price-delete:$scopeId:$overrideId"

    fun promotionCreate(scopeId: String, payloadFingerprint: String): String =
        "supplier-promotion-create:$scopeId:${stableHash(payloadFingerprint)}"

    fun promotionUpdate(scopeId: String, promotionId: String, payloadFingerprint: String): String =
        "supplier-promotion-update:$scopeId:$promotionId:${stableHash(payloadFingerprint)}"

    fun promotionDeactivate(scopeId: String, promotionId: String): String =
        "supplier-promotion-deactivate:$scopeId:$promotionId"

    fun planningScenario(scopeId: String, factoryDowntimeHours: Int, demandDeltaPct: Double): String =
        "supplier-planning-scenario:$scopeId:$factoryDowntimeHours:$demandDeltaPct"

    fun seasonalOverrideCreate(scopeId: String, startDate: String, endDate: String): String =
        "supplier-seasonal-override:$scopeId:${stableHash("$startDate:$endDate")}"

    fun returnPolicyPut(scopeId: String, hours: Long): String =
        "supplier-return-policy:$scopeId:$hours"

    fun controlTowerZoneOverride(scopeId: String, action: String, polygonFingerprint: String): String =
        "supplier-control-tower-override:$scopeId:${stableHash("$action:$polygonFingerprint")}"

    /** FNV-1a 32-bit — matches api-client `stableHash`. */
    private fun stableHash(input: String): String {
        var hash = 2166136261L
        for (c in input) {
            hash = hash xor c.code.toLong()
            hash = (hash * 16777619L) and 0xFFFFFFFFL
        }
        return hash.toULong().toString(36)
    }
}

const val SUPPLIER_RECONNECT_RECOVERY_HINT =
    "Connection restored — verify status before retrying."
