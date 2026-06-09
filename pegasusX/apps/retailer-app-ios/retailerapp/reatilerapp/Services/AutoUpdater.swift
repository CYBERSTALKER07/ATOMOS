import Foundation
import UIKit

/// Enterprise AutoUpdater for iOS
/// 
/// Handles checking for updates via `updater.json` and triggering the
/// B2B Enterprise Over-The-Air (OTA) install prompt via `itms-services`.
/// iOS handles network drop resilience and app replacement natively.
class AutoUpdater {
    static val shared = AutoUpdater()
    
    private let updateManifestURL = URL(string: "https://storage.googleapis.com/pegasusx-ssmr-app-updates/ios/retailer/updater.json")!
    
    func checkForUpdates() {
        let currentVersion = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "0"
        guard let currentVersionCode = Int(currentVersion) else { return }
        
        let task = URLSession.shared.dataTask(with: updateManifestURL) { data, response, error in
            guard let data = data, error == nil else {
                print("Failed to check for updates: \(error?.localizedDescription ?? "Unknown error")")
                return
            }
            
            do {
                if let json = try JSONSerialization.jsonObject(with: data, options: []) as? [String: Any],
                   let latestVersionCode = json["version_code"] as? Int,
                   let manifestUrlString = json["manifest_url"] as? String,
                   let manifestURL = URL(string: manifestUrlString) {
                    
                    if latestVersionCode > currentVersionCode {
                        print("Update found! Version \(latestVersionCode). Prompting user.")
                        DispatchQueue.main.async {
                            self.promptInstall(manifestURL: manifestURL)
                        }
                    } else {
                        print("App is up-to-date.")
                    }
                }
            } catch {
                print("Error parsing update manifest: \(error.localizedDescription)")
            }
        }
        task.resume()
    }
    
    private func promptInstall(manifestURL: URL) {
        // Find top view controller to present alert
        guard let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
              let rootVC = windowScene.windows.first?.rootViewController else { return }
        
        let alert = UIAlertController(
            title: "Update Available",
            message: "A new enterprise update is available. Do you want to install it now? The app will close during installation.",
            preferredStyle: .alert
        )
        
        alert.addAction(UIAlertAction(title: "Later", style: .cancel))
        alert.addAction(UIAlertAction(title: "Install", style: .default) { _ in
            self.triggerOTAInstall(manifestURL: manifestURL)
        })
        
        rootVC.present(alert, animated: true)
    }
    
    private func triggerOTAInstall(manifestURL: URL) {
        // Trigger the native iOS enterprise app install process
        let itmsURLString = "itms-services://?action=download-manifest&url=\(manifestURL.absoluteString)"
        if let itmsURL = URL(string: itmsURLString) {
            UIApplication.shared.open(itmsURL, options: [:], completionHandler: nil)
        }
    }
}
