import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore

    var body: some View {
        Group {
            if !tokenStore.isAuthenticated {
                LoginView()
            } else if !tokenStore.isConfigured {
                LocationSetupView()
            } else {
                FactoryAdaptiveShell()
            }
        }
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
        .animation(.smooth, value: tokenStore.isConfigured)
    }
}
