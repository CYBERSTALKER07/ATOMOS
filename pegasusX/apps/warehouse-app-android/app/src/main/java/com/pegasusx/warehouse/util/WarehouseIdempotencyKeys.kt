package com.pegasusx.warehouse.util

import com.pegasusx.warehouse.data.remote.TokenHolder

/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */
object WarehouseIdempotencyKeys {
    private fun warehouseId(): String = TokenHolder.warehouseId?.takeIf { it.isNotBlank() } ?: "warehouse"

    private fun stableHash(input: String): String {
        var hash = 2166136261L
        for (ch in input) {
            hash = hash xor ch.code.toLong()
            hash = (hash * 16777619L) and 0xFFFFFFFFL
        }
        return hash.toString(36)
    }

    fun emergencyTransfer(volumeVu: Double, notes: String?): String =
        "warehouse-emergency-transfer:${warehouseId()}:$volumeVu:${stableHash(notes.orEmpty())}"

    fun forceReceive(volumeVu: Double, notes: String?, factoryId: String? = null): String =
        "warehouse-force-receive:${warehouseId()}:${factoryId.orEmpty()}:$volumeVu:${stableHash(notes.orEmpty())}"

    fun receiveTransfer(transferId: String): String = "warehouse-receive-transfer:$transferId"

    fun dispatchLockAcquire(entityType: String = "WAREHOUSE", entityId: String? = null): String {
        val wh = entityId?.takeIf { it.isNotBlank() } ?: warehouseId()
        return "warehouse-dispatch-lock-acquire:${warehouseId()}:$entityType:$wh"
    }

    fun dispatchLockRelease(lockId: String): String = "warehouse-dispatch-lock-release:$lockId"

    fun createDriver(phone: String): String =
        "warehouse-create-driver:${warehouseId()}:${stableHash(phone)}"

    fun createStaff(phone: String): String =
        "warehouse-create-staff:${warehouseId()}:${stableHash(phone)}"

    fun createVehicle(licensePlate: String): String =
        "warehouse-create-vehicle:${warehouseId()}:${stableHash(licensePlate)}"

    fun adjustInventory(productId: String, quantity: Int): String =
        "warehouse-adjust-inventory:${warehouseId()}:$productId:$quantity"

    fun assignDriverVehicle(driverId: String, vehicleId: String?): String =
        "warehouse-assign-driver-vehicle:${warehouseId()}:$driverId:${vehicleId?.takeIf { it.isNotBlank() } ?: "none"}"

    fun updateVehicle(vehicleId: String, isActive: Boolean, unavailableReason: String? = null): String {
        val reason = if (isActive) "active" else (unavailableReason?.trim()?.uppercase() ?: "MANUAL_HOLD")
        return "warehouse-update-vehicle:${warehouseId()}:$vehicleId:$isActive:$reason"
    }

    fun inventoryPolicy(productId: String, policy: String): String =
        "warehouse-inventory-policy:${warehouseId()}:$productId:${policy.trim().uppercase()}"

    fun replenishmentInsightAction(insightId: String, action: String): String =
        "warehouse-replenishment-action:$insightId:${action.trim().lowercase()}"

    fun dispatchSettings(autoDispatchEnabled: Boolean): String =
        "warehouse-dispatch-settings:${warehouseId()}:$autoDispatchEnabled"

    fun dispatch(actorId: String, routeFingerprint: String): String =
        "warehouse-dispatch:${warehouseId()}:$actorId:${stableHash(routeFingerprint)}"

    fun inboundScan(barcode: String, sessionId: String): String =
        "warehouse-inbound-scan:${warehouseId()}:${stableHash(barcode)}:$sessionId"

    fun inboundConfirm(returnIds: List<String>, disposition: String): String {
        val sorted = returnIds.sorted().joinToString(",")
        return "warehouse-inbound-confirm:${warehouseId()}:$disposition:${stableHash(sorted)}"
    }

    fun opsSettings(): String = "warehouse-ops-settings:${warehouseId()}"

    fun opsLocation(lat: Double, lng: Double, placeId: String? = null): String {
        val fingerprint = stableHash("${"%.6f".format(lat)}:${"%.6f".format(lng)}:${placeId.orEmpty()}")
        return "warehouse-ops-location:${warehouseId()}:$fingerprint"
    }

    fun createSupplyRequest(factoryId: String, mode: String, notes: String): String =
        "warehouse-create-supply-request:${warehouseId()}:$factoryId:$mode:${stableHash(notes)}"

    fun supplyRequestTransition(requestId: String, action: String): String =
        "warehouse-supply-transition:$requestId:${action.uppercase()}"

    fun orderDelay(orderId: String): String = "warehouse-order-delay:$orderId"

    fun orderReject(orderId: String, reason: String): String =
        "warehouse-order-reject:$orderId:${stableHash(reason)}"

    fun orderOverflow(orderId: String): String = "warehouse-order-overflow:$orderId"

    fun orderProposeDelivery(orderId: String, proposedDate: String, reason: String): String {
        val hash = stableHash("$proposedDate:$reason")
        return "warehouse-order-propose-delivery:$orderId:$hash"
    }

    fun recommendReassign(orderId: String): String = "warehouse-recommend-reassign:$orderId"

    fun reassignOrder(orderId: String, driverId: String): String =
        "warehouse-reassign-order:$orderId:$driverId"

    fun broadcast(role: String, title: String, body: String): String =
        "warehouse-broadcast:${warehouseId()}:${role.trim().uppercase()}:${stableHash("$title:$body")}"

    fun broadcastTemplateCreate(title: String, body: String): String =
        "warehouse-broadcast-template-create:${warehouseId()}:${stableHash("$title:$body")}"

    fun broadcastTemplateDelete(templateId: String): String =
        "warehouse-broadcast-template-delete:${warehouseId()}:$templateId"

    fun rescuePropose(rescueId: String, brokenDriverId: String, rescueDriverId: String): String =
        "warehouse-rescue-propose:${warehouseId()}:$rescueId:$brokenDriverId:$rescueDriverId"
}
