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
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
    }
}
