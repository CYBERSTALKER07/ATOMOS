//
//  WarehouseIdempotency.swift
//  WarehouseApp
//
//  Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
//

import Foundation

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
}
