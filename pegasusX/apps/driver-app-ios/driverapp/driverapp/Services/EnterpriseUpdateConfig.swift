import Foundation

/// Legacy mirror — prefer driverappios tree for shipping builds.
enum EnterpriseUpdateConfig {
    static var channel: String {
        if let v = Bundle.main.object(forInfoDictionaryKey: "PXDistributionChannel") as? String,
           !v.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return v.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        }
        #if STORE_BUILD
        return "production"
        #else
        return "enterprise"
        #endif
    }

    static var enableCdnOta: Bool {
        if let v = Bundle.main.object(forInfoDictionaryKey: "PXEnableCdnOta") as? Bool { return v }
        #if STORE_BUILD
        return false
        #else
        return channel != "production" && channel != "store"
        #endif
    }

    static let policyRole = "DRIVER"

    static let defaultManifestURL = URL(
        string: "https://storage.googleapis.com/pegasusx-ssmr-app-updates/ios/driver/updater.json"
    )!

    static var storeListingURL: URL? {
        if let s = Bundle.main.object(forInfoDictionaryKey: "PXAppStoreURL") as? String,
           let u = URL(string: s), !s.isEmpty { return u }
        if let id = Bundle.main.object(forInfoDictionaryKey: "PXAppStoreID") as? String {
            let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
                .replacingOccurrences(of: "id", with: "", options: [.anchored])
            if !trimmed.isEmpty {
                return URL(string: "https://apps.apple.com/app/id\(trimmed)")
            }
        }
        return nil
    }
}
