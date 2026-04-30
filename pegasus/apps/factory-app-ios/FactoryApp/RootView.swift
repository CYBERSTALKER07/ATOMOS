import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore

    var body: some View {
        Group {
            if tokenStore.isAuthenticated {
                MainTabView()
            } else {
                LoginView()
            }
        }
        .animation(.smooth, value: tokenStore.isAuthenticated)
    }
}
