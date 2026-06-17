//
//  WarehouseIdempotency.swift
//  WarehouseApp
//
//  Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
//

import Foundation

@MainActor
enum WarehouseIdempotency {
    private static func warehouseId() -> String {
        let id = TokenStore.shared.warehouseId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return id.isEmpty ? "warehouse" : id
    }

    private static func stableHash(_ input: String) -> String {
        var hash: UInt32 = 2166136261
        for scalar in input.unicodeScalars {
            hash ^= scalar.value
            hash = hash &* 16777619
        }
        return String(hash, radix: 36)
    }

    static func emergencyTransfer(volumeVu: Double, notes: String?) -> String {
        "warehouse-emergency-transfer:\(warehouseId()):\(volumeVu):\(stableHash(notes ?? ""))"
    }

    static func forceReceive(volumeVu: Double, notes: String?, factoryId: String? = nil) -> String {
        "warehouse-force-receive:\(warehouseId()):\(factoryId ?? ""):\(volumeVu):\(stableHash(notes ?? ""))"
    }

    static func receiveTransfer(transferId: String) -> String {
        "warehouse-receive-transfer:\(transferId)"
    }

    static func createDriver(phone: String) -> String {
        "warehouse-create-driver:\(warehouseId()):\(stableHash(phone))"
    }

    static func createStaff(phone: String) -> String {
        "warehouse-create-staff:\(warehouseId()):\(stableHash(phone))"
    }

    static func createVehicle(licensePlate: String) -> String {
        "warehouse-create-vehicle:\(warehouseId()):\(stableHash(licensePlate))"
    }

    static func adjustInventory(productId: String, quantity: Int) -> String {
        "warehouse-adjust-inventory:\(warehouseId()):\(productId):\(quantity)"
    }

    static func assignDriverVehicle(driverId: String, vehicleId: String?) -> String {
        let vid = vehicleId?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false ? vehicleId! : "none"
        return "warehouse-assign-driver-vehicle:\(warehouseId()):\(driverId):\(vid)"
    }

    static func updateVehicle(vehicleId: String, isActive: Bool, unavailableReason: String? = nil) -> String {
        let reason = isActive ? "active" : (unavailableReason?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() ?? "MANUAL_HOLD")
        return "warehouse-update-vehicle:\(warehouseId()):\(vehicleId):\(isActive):\(reason)"
    }

    static func inventoryPolicy(productId: String, policy: String) -> String {
        "warehouse-inventory-policy:\(warehouseId()):\(productId):\(policy.trimmingCharacters(in: .whitespacesAndNewlines).uppercased())"
    }

    static func replenishmentInsightAction(insightId: String, action: String) -> String {
        "warehouse-replenishment-action:\(insightId):\(action.trimmingCharacters(in: .whitespacesAndNewlines).lowercased())"
    }

    static func dispatchSettings(autoDispatchEnabled: Bool) -> String {
        "warehouse-dispatch-settings:\(warehouseId()):\(autoDispatchEnabled)"
    }

    static func dispatchLockAcquire(entityType: String = "WAREHOUSE", entityId: String? = nil) -> String {
        let wh: String
        if let entityId, !entityId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            wh = entityId.trimmingCharacters(in: .whitespacesAndNewlines)
        } else {
            wh = warehouseId()
        }
        return "warehouse-dispatch-lock-acquire:\(warehouseId()):\(entityType):\(wh)"
    }

    static func dispatchLockRelease(lockId: String) -> String {
        "warehouse-dispatch-lock-release:\(lockId)"
    }

    static func dispatch(actorId: String, routeFingerprint: String) -> String {
        "warehouse-dispatch:\(warehouseId()):\(actorId):\(stableHash(routeFingerprint))"
    }

    static func inboundScan(barcode: String, sessionId: String) -> String {
        "warehouse-inbound-scan:\(warehouseId()):\(stableHash(barcode)):\(sessionId)"
    }

    static func inboundConfirm(returnIds: [String], disposition: String) -> String {
        let sorted = returnIds.sorted().joined(separator: ",")
        return "warehouse-inbound-confirm:\(warehouseId()):\(disposition):\(stableHash(sorted))"
    }

    static func createSupplyRequest(factoryId: String, mode: String, notes: String) -> String {
        "warehouse-create-supply-request:\(warehouseId()):\(factoryId):\(mode):\(stableHash(notes))"
    }

    static func opsSettings() -> String {
        "warehouse-ops-settings:\(warehouseId())"
    }

    static func supplyRequestTransition(requestId: String, action: String) -> String {
        "warehouse-supply-transition:\(requestId):\(action.uppercased())"
    }

    static func orderDelay(orderId: String) -> String {
        "warehouse-order-delay:\(orderId)"
    }

    static func orderReject(orderId: String, reason: String) -> String {
        "warehouse-order-reject:\(orderId):\(stableHash(reason))"
    }

    static func orderOverflow(orderId: String) -> String {
        "warehouse-order-overflow:\(orderId)"
    }
}
