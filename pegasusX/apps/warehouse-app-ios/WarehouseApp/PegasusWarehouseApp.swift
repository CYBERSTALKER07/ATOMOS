import SwiftUI

@main
struct LabWarehouseApp: App {
    @State private var tokenStore = TokenStore.shared
    @State private var realtimeHub = WarehouseRealtimeHub()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
                .environment(realtimeHub)
        }
    }
}
