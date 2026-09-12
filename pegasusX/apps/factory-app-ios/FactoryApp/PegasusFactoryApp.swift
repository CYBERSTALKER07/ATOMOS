import SwiftUI

@main
struct LabFactoryApp: App {
    @State private var tokenStore = TokenStore.shared
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
        }
    }
}
