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
}
