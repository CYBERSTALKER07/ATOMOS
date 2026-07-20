import Foundation
import UIKit

/// Legacy tree mirror of driverappios distribution-aware OTA.
enum AutoUpdater {
    private static let defaultManifestURL = URL(
        string: "https://storage.googleapis.com/pegasusx-ssmr-app-updates/ios/driver/updater.json"
    )!

    static func checkForUpdates(updateURL: String? = nil) {
        if !EnterpriseUpdateConfig.enableCdnOta {
            Task { @MainActor in
                openStoreListing()
            }
            return
        }
        Task {
            let urlString = (updateURL?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap {
                $0.isEmpty ? nil : $0
            } ?? defaultManifestURL.absoluteString
            guard let url = URL(string: urlString) else { return }
            do {
                let (data, response) = try await URLSession.shared.data(from: url)
                guard let http = response as? HTTPURLResponse, (200 ... 299).contains(http.statusCode),
                      let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                      let latestVersionCode = json["version_code"] as? Int,
                      let manifestUrlString = json["manifest_url"] as? String,
                      let manifestURL = URL(string: manifestUrlString)
                else { return }
                let currentCode = Int(Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "0") ?? 0
                if latestVersionCode > currentCode {
                    await MainActor.run { promptInstall(manifestURL: manifestURL) }
                }
            } catch {
                // Best-effort
            }
        }
    }

    @MainActor
    private static func openStoreListing() {
        guard let storeURL = EnterpriseUpdateConfig.storeListingURL else { return }
        UIApplication.shared.open(storeURL)
    }

    @MainActor
    private static func promptInstall(manifestURL: URL) {
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else { return }
        let alert = UIAlertController(
            title: "Update Available",
            message: "A new enterprise driver update is available.",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        alert.addAction(UIAlertAction(title: "Install", style: .default) { _ in
            triggerOTAInstall(manifestURL: manifestURL)
        })
        rootVC.present(alert, animated: true)
    }

    private static func triggerOTAInstall(manifestURL: URL) {
        if !EnterpriseUpdateConfig.enableCdnOta {
            Task { @MainActor in openStoreListing() }
            return
        }
        var components = URLComponents(string: "itms-services://")
        components?.queryItems = [
            URLQueryItem(name: "action", value: "download-manifest"),
            URLQueryItem(name: "url", value: manifestURL.absoluteString)
        ]
        if let itmsURL = components?.url {
            UIApplication.shared.open(itmsURL)
        }
    }
}
