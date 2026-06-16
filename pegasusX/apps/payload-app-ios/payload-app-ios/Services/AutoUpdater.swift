import Foundation
import UIKit

/// Enterprise OTA install prompt — mirrors payloadapp AutoUpdater, wired from RootView policy fetch.
enum AutoUpdater {
    private static let updateManifestURL = URL(string: "https://storage.googleapis.com/pegasusx-ssmr-app-updates/ios/payload/updater.json")!

    static func checkForUpdates() {
        let currentVersion = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "0"
        guard let currentVersionCode = Int(currentVersion) else { return }

        let task = URLSession.shared.dataTask(with: updateManifestURL) { data, _, error in
            guard let data, error == nil else { return }
            do {
                if let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                   let latestVersionCode = json["version_code"] as? Int,
                   let manifestUrlString = json["manifest_url"] as? String,
                   let manifestURL = URL(string: manifestUrlString),
                   latestVersionCode > currentVersionCode {
                    DispatchQueue.main.async { promptInstall(manifestURL: manifestURL) }
                }
            } catch {
                // Best-effort; policy banner still surfaces outdated builds.
            }
        }
        task.resume()
    }

    private static func promptInstall(manifestURL: URL) {
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else { return }
        let alert = UIAlertController(
            title: "Update Available",
            message: "A new enterprise update is available.",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        alert.addAction(UIAlertAction(title: "Install", style: .default) { _ in
            triggerOTAInstall(manifestURL: manifestURL)
        })
        rootVC.present(alert, animated: true)
    }

    private static func triggerOTAInstall(manifestURL: URL) {
        if let itmsURL = URL(string: "itms-services://?action=download-manifest&url=\(manifestURL.absoluteString)") {
            UIApplication.shared.open(itmsURL)
        }
    }
}
