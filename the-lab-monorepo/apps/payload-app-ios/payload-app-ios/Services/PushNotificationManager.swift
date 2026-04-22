import Foundation
import UIKit
import UserNotifications

/// PushNotificationManager — APNs permission + token lifecycle for the PAYLOAD iPad app.
///
/// Mirrors the pattern in retailer-app-ios. Uses pure `UNUserNotificationCenter`
/// (no Firebase SDK); the backend `/v1/user/device-token` accepts any opaque
/// string + `platform: "IOS"` and routes delivery server-side.
///
/// Tap-action deep-links call `onOpenPanel` so the HomeView can reveal the
/// in-app notifications sheet.
@MainActor
@Observable
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
    static let shared = PushNotificationManager()

    var isAuthorized = false
    var deviceToken: String?
    var errorMessage: String?

    /// Fires when the user taps a remote notification. HomeView observes
    /// and opens the notifications sheet.
    var onOpenPanel: (() -> Void)?

    private let api = APIClient.shared

    private override init() {
        super.init()
        UNUserNotificationCenter.current().delegate = self
    }

    // MARK: - Permission

    func requestAuthorization() async {
        do {
            let center = UNUserNotificationCenter.current()
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            isAuthorized = granted
            if granted {
                UIApplication.shared.registerForRemoteNotifications()
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    // MARK: - Token lifecycle (called from AppDelegate)

    func didRegisterForRemoteNotifications(deviceToken data: Data) {
        let token = data.map { String(format: "%02.2hhx", $0) }.joined()
        self.deviceToken = token
        Task {
            _ = try? await api.registerDeviceToken(token)
        }
    }

    func didFailToRegisterForRemoteNotifications(error: Error) {
        errorMessage = error.localizedDescription
    }

    // MARK: - UNUserNotificationCenterDelegate

    /// Present banners while the app is foregrounded.
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        [.banner, .sound, .badge]
    }

    /// Handle notification tap — deep-link into the notifications panel.
    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        onOpenPanel?()
    }
}
