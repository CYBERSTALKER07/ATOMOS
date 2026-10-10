import FirebaseCore
import FirebaseMessaging
import UIKit
import UserNotifications

/// Bridges APNs + Firebase Messaging into [PushNotificationManager].
final class AppDelegate: NSObject, UIApplicationDelegate, MessagingDelegate {
    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        if FirebaseApp.app() == nil {
            if let path = Bundle.main.path(forResource: "GoogleService-Info", ofType: "plist"),
               let options = FirebaseOptions(contentsOfFile: path) {
                FirebaseApp.configure(options: options)
            } else {
                #if DEBUG
                let options = FirebaseOptions(
                    googleAppID: "1:000000000000:ios:0000000000000001",
                    gcmSenderID: "000000000000"
                )
                options.projectID = "demo-pegasus"
                options.apiKey = "demo-key"
                FirebaseApp.configure(options: options)
                #else
                fatalError("GoogleService-Info.plist missing in release build")
                #endif
            }
        }
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
