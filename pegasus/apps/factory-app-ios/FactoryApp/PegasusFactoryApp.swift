import SwiftUI

@main
struct LabFactoryApp: App {
    @State private var tokenStore = TokenStore.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
        }
    }
}
