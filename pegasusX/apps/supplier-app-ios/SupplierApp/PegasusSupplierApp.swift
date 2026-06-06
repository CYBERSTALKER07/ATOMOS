import SwiftUI

@main
struct PegasusSupplierApp: App {
    @State private var tokenStore = TokenStore.shared
    @State private var realtimeHub = SupplierRealtimeHub()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
                .environment(realtimeHub)
        }
    }
}
