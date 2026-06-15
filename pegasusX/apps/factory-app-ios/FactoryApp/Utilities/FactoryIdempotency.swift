//
//  FactoryIdempotency.swift
//  FactoryApp
//
//  Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts
//

import Foundation

enum FactoryIdempotency {
    private static func factoryId() -> String {
        let id = TokenStore.shared.factoryId?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return id.isEmpty ? "factory" : id
    }

    static func startLoading(manifestId: String) -> String {
        "factory-start-loading:\(manifestId)"
    }

    static func seal(manifestId: String) -> String {
        "factory-manifest-seal:\(factoryId()):\(manifestId)"
    }

    static func dispatch(manifestId: String) -> String {
        "factory-manifest-dispatch:\(factoryId()):\(manifestId)"
    }

    static func complete(manifestId: String) -> String {
        "factory-manifest-complete:\(factoryId()):\(manifestId)"
    }

    static func batchDispatch(transferIds: [String]) -> String {
        let sorted = transferIds
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .sorted()
        return "factory-dispatch:\(factoryId()):\(stableHash(sorted.joined(separator: ",")))"
    }

    static func rebalance(
        manifestId: String,
        transferId: String,
        toDriverId: String = "",
        toVehicle: String = "",
        targetManifestId: String = ""
    ) -> String {
        let fingerprint = [toDriverId, toVehicle, targetManifestId].joined(separator: ":")
        return "factory-manifest-rebalance:\(manifestId):\(transferId):\(stableHash(fingerprint))"
    }

    static func cancelTransfer(manifestId: String, transferId: String) -> String {
        "factory-manifest-cancel-transfer:\(manifestId):\(transferId)"
    }

    static func cancelManifest(manifestId: String, reason: String = "") -> String {
        "factory-manifest-cancel:\(manifestId):\(stableHash(reason))"
    }

    static func transferCreate(
        orderId: String,
        totalVu: Int64,
        driverId: String = "",
        vehicleId: String = ""
    ) -> String {
        let fingerprint = "\(orderId.trimmingCharacters(in: .whitespacesAndNewlines)):\(totalVu):\(driverId.trimmingCharacters(in: .whitespacesAndNewlines)):\(vehicleId.trimmingCharacters(in: .whitespacesAndNewlines))"
        return "factory-transfer-create:\(factoryId()):\(stableHash(fingerprint))"
    }

    static func forLifecycleAction(_ action: String, manifestId: String) -> String {
        switch action {
        case ManifestLifecycleAction.startLoading.rawValue:
            return startLoading(manifestId: manifestId)
        case ManifestLifecycleAction.seal.rawValue:
            return seal(manifestId: manifestId)
        case ManifestLifecycleAction.dispatch.rawValue:
            return dispatch(manifestId: manifestId)
        case ManifestLifecycleAction.complete.rawValue:
            return complete(manifestId: manifestId)
        default:
            return "factory-manifest-transition:\(factoryId()):\(manifestId):\(action)"
        }
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
