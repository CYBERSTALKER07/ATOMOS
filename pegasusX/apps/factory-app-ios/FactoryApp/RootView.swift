import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore

    var body: some View {
        Group {
            if tokenStore.isAuthenticated {
                FactoryAdaptiveShell()
            } else {
                LoginView()
            }
        }
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
    }
}
