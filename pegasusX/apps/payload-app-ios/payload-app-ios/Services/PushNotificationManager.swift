import Foundation
import UIKit
import UserNotifications

/// FCM registration token lifecycle. APNs hex is never POSTed — Admin Messaging
/// send only accepts Firebase registration tokens.
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
    private let storedTokenKey = "pegasus_push_token"

    private override init() {
        super.init()
        UNUserNotificationCenter.current().delegate = self
    }

    private var isRunningTests: Bool {
        ProcessInfo.processInfo.environment["XCTestConfigurationFilePath"] != nil
    }

    func requestAuthorization() async {
        if isRunningTests { return }
        do {
            let center = UNUserNotificationCenter.current()
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            isAuthorized = granted
            UIApplication.shared.registerForRemoteNotifications()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func didFailToRegisterForRemoteNotifications(error: Error) {
        errorMessage = error.localizedDescription
    }

    func didReceiveFCMToken(_ token: String?) async {
        let trimmed = token?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else { return }
        deviceToken = trimmed
        UserDefaults.standard.set(trimmed, forKey: storedTokenKey)
        await uploadStoredTokenIfPossible()
    }

    func uploadStoredTokenIfPossible() async {
        if isRunningTests { return }
        let token = (deviceToken ?? UserDefaults.standard.string(forKey: storedTokenKey) ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard !token.isEmpty, TokenStore.shared.isAuthenticated else { return }
        _ = try? await api.registerDeviceToken(token)
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        [.banner, .sound, .badge]
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        onOpenPanel?()
    }
}
