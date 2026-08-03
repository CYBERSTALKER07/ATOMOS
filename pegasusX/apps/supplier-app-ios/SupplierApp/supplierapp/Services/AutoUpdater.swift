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
    }

    private(set) var lastManifest: Manifest?

    private init() {}

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
            )
        }

        let manifestURLString = (updateURL?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap {
            $0.isEmpty ? nil : $0
        } ?? EnterpriseUpdateConfig.defaultManifestURL.absoluteString

        let manifest = await fetchManifest(from: manifestURLString)
        lastManifest = manifest

        let currentCode = Int(Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "0") ?? 0
        let newer = manifest.map { $0.versionCode > currentCode } ?? false

        return UpdateState(
            available: newer || outdated,
            force: force,
            deferred: updateDeferred,
            message: message,
            manifest: manifest,
        )
    }

    /// Prompt user to install (enterprise OTA).
    @MainActor
    func promptInstall(manifest: Manifest? = nil, force: Bool = false) {
        if !EnterpriseUpdateConfig.enableCdnOta {
            openStoreListing(force: force)
            return
        }
        let target = manifest ?? lastManifest
        guard let target else { return }
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else {
            triggerOTAInstall(manifestURL: target.manifestURL)
            return
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
    func openStoreListing(force: Bool = false) {
        guard let storeURL = EnterpriseUpdateConfig.storeListingURL else {
            // No App Store ID configured yet — surface message only.
            return
        }
        let title = force ? "Update Required" : "Update Available"
        let message = "Please update from the App Store."
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else {
            UIApplication.shared.open(storeURL)
            return
        }
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        if !force {
            alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        }
        alert.addAction(UIAlertAction(title: "Open App Store", style: .default) { _ in
            UIApplication.shared.open(storeURL)
        })
        rootVC.present(alert, animated: true)
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
