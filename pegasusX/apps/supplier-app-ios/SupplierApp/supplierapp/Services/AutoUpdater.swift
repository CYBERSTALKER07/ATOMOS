import Foundation
import UIKit

/// Website-only enterprise OTA for supplier iOS (enterprise / ad-hoc distribution).
///
/// Flow:
/// 1. Shell loads client-policy with channel=enterprise
/// 2. [evaluate] fetches policy-driven manifest URL (or CDN default)
/// 3. If version_code > CFBundleVersion → soft banner or force prompt
/// 4. Install via itms-services + Apple enterprise plist
final class AutoUpdater {
    static let shared = AutoUpdater()

    struct Manifest: Sendable {
        let versionCode: Int
        let versionName: String
        let manifestURL: URL
        let notes: String
    }

    struct UpdateState: Sendable {
        let available: Bool
        let force: Bool
        let deferred: Bool
        let message: String?
        let manifest: Manifest?
        let manifestFailed: Bool
    }

    private(set) var lastManifest: Manifest?

    private init() {}

    /// A failed policy read is not "up to date" and must not offer an install.
    static func failedPolicy(message: String, hadBanner: Bool) -> (clearBanner: Bool, force: Bool, notice: String) {
        (!hadBanner, false, message)
    }

    /// Evaluate policy fields + optional CDN manifest.
    func evaluate(
        outdated: Bool,
        forceUpdate: Bool,
        updateDeferred: Bool,
        minimumVersion: String,
        recommendedVersion: String?,
        deferReason: String?,
        updateURL: String?,
    ) async -> UpdateState {
        let force = forceUpdate && !updateDeferred
        var message: String?
        if outdated || forceUpdate {
            var m = force ? "Update required" : "Update available"
            if !minimumVersion.isEmpty {
                m += " — minimum version \(minimumVersion)"
            }
            if let recommendedVersion, !recommendedVersion.isEmpty, recommendedVersion != "0.0.0" {
                m += " (recommended \(recommendedVersion))"
            }
            if let deferReason, !deferReason.isEmpty {
                m += ". \(deferReason)"
            }
            message = m
        }

        guard outdated || forceUpdate else {
            return UpdateState(
                available: false,
                force: false,
                deferred: updateDeferred,
                message: nil,
                manifest: nil,
                manifestFailed: false,
            )
        }

        // App Store builds: never fetch enterprise CDN manifests / itms-services.
        if !EnterpriseUpdateConfig.enableCdnOta {
            return UpdateState(
                available: true,
                force: force,
                deferred: updateDeferred,
                message: message,
                manifest: nil,
                manifestFailed: false,
            )
        }

        let manifestURLString = (updateURL?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap {
            $0.isEmpty ? nil : $0
        } ?? EnterpriseUpdateConfig.defaultManifestURL.absoluteString

        let manifest = await fetchManifest(from: manifestURLString)
        lastManifest = manifest

        let currentCode = AppBuildVersion.code(Bundle.main.infoDictionary?["CFBundleVersion"] as? String)
        let newer = manifest.map { AppBuildVersion.manifestIsNewer($0.versionCode, than: currentCode) } ?? false

        let manifestFailed = manifest == nil
        return UpdateState(
            available: !manifestFailed && (newer || outdated),
            force: manifestFailed ? false : force,
            deferred: updateDeferred,
            message: manifestFailed ? nil : message,
            manifest: manifest,
            manifestFailed: manifestFailed,
        )
    }

    /// Prompt user to install (enterprise OTA). Returns false when there is nothing to open.
    @MainActor
    @discardableResult
    func promptInstall(manifest: Manifest? = nil, force: Bool = false) -> Bool {
        if !EnterpriseUpdateConfig.enableCdnOta {
            return openStoreListing(force: force)
        }
        let target = manifest ?? lastManifest
        guard let target else { return false }
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else {
            triggerOTAInstall(manifestURL: target.manifestURL)
            return true
        }

        let title = force ? "Update Required" : "Update Available"
        let notes = target.notes.isEmpty
            ? "A new enterprise build (\(target.versionName.isEmpty ? "\(target.versionCode)" : target.versionName)) is available."
            : target.notes

        let alert = UIAlertController(title: title, message: notes, preferredStyle: .alert)
        if !force {
            alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        }
        alert.addAction(UIAlertAction(title: "Install", style: .default) { _ in
            self.triggerOTAInstall(manifestURL: target.manifestURL)
        })
        rootVC.present(alert, animated: true)
        return true
    }

    func triggerOTAInstall(manifestURL: URL) {
        if !EnterpriseUpdateConfig.enableCdnOta {
            Task { @MainActor in
                openStoreListing()
            }
            return
        }
        // itms-services requires the plist URL to be HTTPS and reachable without auth.
        var components = URLComponents(string: "itms-services://")
        components?.queryItems = [
            URLQueryItem(name: "action", value: "download-manifest"),
            URLQueryItem(name: "url", value: manifestURL.absoluteString)
        ]
        guard let itms = components?.url else {
            return
        }
        DispatchQueue.main.async {
            UIApplication.shared.open(itms, options: [:], completionHandler: nil)
        }
    }


    @MainActor
    @discardableResult
    func openStoreListing(force: Bool = false) -> Bool {
        guard let storeURL = EnterpriseUpdateConfig.storeListingURL else {
            return false
        }
        let title = force ? "Update Required" : "Update Available"
        let message = "Please update from the App Store."
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else {
            UIApplication.shared.open(storeURL)
            return true
        }
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        if !force {
            alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        }
        alert.addAction(UIAlertAction(title: "Open App Store", style: .default) { _ in
            UIApplication.shared.open(storeURL)
        })
        rootVC.present(alert, animated: true)
        return true
    }

    // MARK: - Manifest fetch

    private func fetchManifest(from urlString: String) async -> Manifest? {
        guard let url = URL(string: urlString) else { return nil }
        do {
            let (data, response) = try await URLSession.shared.data(from: url)
            guard let http = response as? HTTPURLResponse, (200 ... 299).contains(http.statusCode) else {
                return nil
            }
            guard let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                  let versionCode = json["version_code"] as? Int,
                  let manifestStr = json["manifest_url"] as? String,
                  let plistURL = URL(string: manifestStr)
            else {
                return nil
            }
            return Manifest(
                versionCode: versionCode,
                versionName: json["version_name"] as? String ?? "",
                manifestURL: plistURL,
                notes: json["notes"] as? String ?? "",
            )
        } catch {
            return nil
        }
    }
}

enum AppBuildVersion {
    /// A missing or non-numeric build is unknown. It is not build 0.
    static func code(_ raw: String?) -> Int? {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard let value = Int(trimmed), value >= 0 else { return nil }
        return value
    }

    static func manifestIsNewer(_ manifestCode: Int, than current: Int?) -> Bool {
        guard let current else { return false }
        return manifestCode > current
    }

    static func marketingVersion(_ raw: String?) -> String? {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }
}
