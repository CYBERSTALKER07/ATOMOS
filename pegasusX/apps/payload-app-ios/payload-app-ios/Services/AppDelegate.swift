import FirebaseMessaging
import UIKit
import UserNotifications

/// AppDelegate — UIKit adaptor bridging APNs + Firebase Messaging into
/// [PushNotificationManager]. Registered on [PegasusPayloadApp] via
/// `@UIApplicationDelegateAdaptor` so SwiftUI remains the app root.
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        FirebaseAuthHelper.shared.configure()
        Messaging.messaging().delegate = self
        return true
    }

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        Messaging.messaging().apnsToken = deviceToken
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        Task { @MainActor in
            PushNotificationManager.shared.didFailToRegisterForRemoteNotifications(error: error)
        }
    }

    nonisolated func messaging(_ messaging: Messaging, didReceiveRegistrationToken fcmToken: String?) {
        Task { @MainActor in
            await PushNotificationManager.shared.didReceiveFCMToken(fcmToken)
        }
    }
}
