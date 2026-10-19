import Foundation

/// Distribution config for native iOS.
///
/// **Website enterprise (default):** `channel=enterprise`, CDN itms-services OTA.
/// **App Store:** set Info.plist `PXDistributionChannel` = `production`
///   and `PXAppStoreURL` (or `PXAppStoreID`), or compile with `-DSTORE_BUILD`.
///
/// Do not ship App Store binaries with enterprise CDN OTA enabled.
enum EnterpriseUpdateConfig {
    /// Client-policy channel (`production` for App Store, `enterprise` for website).
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

    /// When false, open App Store listing instead of itms-services CDN install.
    static var enableCdnOta: Bool {
        if let v = Bundle.main.object(forInfoDictionaryKey: "PXEnableCdnOta") as? Bool {
            return v
        }
        if let s = Bundle.main.object(forInfoDictionaryKey: "PXEnableCdnOta") as? String {
            return (s as NSString).boolValue
        }
        #if STORE_BUILD
        return false
        #else
        return channel != "production" && channel != "store"
        #endif
    }

    static let policyRole = "RETAILER"

    static let defaultManifestURL = URL(
        string: "https://storage.googleapis.com/pegasusx-ssmr-app-updates/ios/retailer/updater.json"
    )!

    /// App Store listing URL for production builds.
    static var storeListingURL: URL? {
        if let s = Bundle.main.object(forInfoDictionaryKey: "PXAppStoreURL") as? String,
           let u = URL(string: s), !s.isEmpty {
            return u
        }
        if let id = Bundle.main.object(forInfoDictionaryKey: "PXAppStoreID") as? String {
            let trimmed = id.trimmingCharacters(in: .whitespacesAndNewlines)
                .replacingOccurrences(of: "id", with: "", options: [.anchored])
            if !trimmed.isEmpty {
                return URL(string: "https://apps.apple.com/app/id\(trimmed)")
            }
        }
        return nil
    }

    static var isStoreBuild: Bool { !enableCdnOta }
}
