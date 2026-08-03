import BackgroundTasks
import UIKit

/// Bridges APNs callbacks into [PushNotificationManager] and registers offline sync BGTask.
final class AppDelegate: NSObject, UIApplicationDelegate {
    static let syncTaskId = "com.pegasus.driver.sync"

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        FirebaseAuthHelper.shared.configure()
        BGTaskScheduler.shared.register(
            forTaskWithIdentifier: Self.syncTaskId,
            using: nil
        ) { task in
            guard let refresh = task as? BGAppRefreshTask else {
                task.setTaskCompleted(success: false)
                return
            }
            Self.handleSyncTask(refresh)
        }
        Self.scheduleSyncTask()
        return true
    }

    func applicationDidEnterBackground(_ application: UIApplication) {
        Self.scheduleSyncTask()
    }

    func application(
        _ application: UIApplication,
        didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data
    ) {
        Task { @MainActor in
            PushNotificationManager.shared.didRegisterForRemoteNotifications(deviceToken: deviceToken)
        }
    }

    func application(
        _ application: UIApplication,
        didFailToRegisterForRemoteNotificationsWithError error: Error
    ) {
        Task { @MainActor in
            PushNotificationManager.shared.didFailToRegisterForRemoteNotifications(error: error)
        }
    }

    private static func scheduleSyncTask() {
        let request = BGAppRefreshTaskRequest(identifier: syncTaskId)
        request.earliestBeginDate = Date(timeIntervalSinceNow: 15 * 60)
        try? BGTaskScheduler.shared.submit(request)
    }

    private static func handleSyncTask(_ task: BGAppRefreshTask) {
        scheduleSyncTask()
        let work = Task {
            await FleetServiceLive.shared.flushOfflineQueue()
        }
        task.expirationHandler = { work.cancel() }
        Task {
            _ = await work.result
            task.setTaskCompleted(success: true)
        }
    }
}
