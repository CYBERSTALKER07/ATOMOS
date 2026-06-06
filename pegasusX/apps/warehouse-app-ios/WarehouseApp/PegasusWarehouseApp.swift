import SwiftUI

@main
struct LabWarehouseApp: App {
    @State private var tokenStore = TokenStore.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
        }
    }
}
