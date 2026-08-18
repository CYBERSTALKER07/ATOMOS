import SwiftUI

@main
struct PegasusSupplierApp: App {
    @State private var tokenStore = TokenStore.shared
    @State private var realtimeHub = SupplierRealtimeHub()
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
                .environment(realtimeHub)
        }
    }
}
