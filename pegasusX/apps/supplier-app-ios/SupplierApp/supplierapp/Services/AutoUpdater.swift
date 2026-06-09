import Foundation
import UIKit

class AutoUpdater {
    static let shared = AutoUpdater()
    
    private let updateManifestURL = URL(string: "https://storage.googleapis.com/pegasusx-ssmr-app-updates/ios/supplier/updater.json")!
    
    func checkForUpdates() {
        let currentVersion = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "0"
        guard let currentVersionCode = Int(currentVersion) else { return }
        
        let task = URLSession.shared.dataTask(with: updateManifestURL) { data, _, error in
            guard let data = data, error == nil else { return }
            do {
                if let json = try JSONSerialization.jsonObject(with: data, options: []) as? [String: Any],
                   let latestVersionCode = json["version_code"] as? Int,
                   let manifestUrlString = json["manifest_url"] as? String,
                   let manifestURL = URL(string: manifestUrlString) {
                    if latestVersionCode > currentVersionCode {
                        DispatchQueue.main.async { self.promptInstall(manifestURL: manifestURL) }
                    }
                }
            } catch { print(error.localizedDescription) }
        }
        task.resume()
    }
    
    private func promptInstall(manifestURL: URL) {
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else { return }
        let alert = UIAlertController(title: "Update Available", message: "A new enterprise update is available.", preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        alert.addAction(UIAlertAction(title: "Install", style: .default) { _ in self.triggerOTAInstall(manifestURL: manifestURL) })
        rootVC.present(alert, animated: true)
    }
    
    private func triggerOTAInstall(manifestURL: URL) {
        if let itmsURL = URL(string: "itms-services://?action=download-manifest&url=\(manifestURL.absoluteString)") {
            UIApplication.shared.open(itmsURL, options: [:], completionHandler: nil)
        }
    }
}
