import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var realtime = WarehouseRealtimeClient()

    var body: some View {
        Group {
            if tokenStore.isAuthenticated {
                WarehouseAdaptiveShell()
            } else {
                LoginView()
            }
        }
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
        .onChange(of: tokenStore.isAuthenticated) { _, authenticated in
            if authenticated {
                connectRealtime()
            } else {
                realtime.disconnect()
            }
        }
        .onAppear {
            if tokenStore.isAuthenticated {
                connectRealtime()
            }
        }
    }

    private func connectRealtime() {
        realtime.connect(
            onStateChange: { _ in },
            onEvent: { event in
                guard !event.type.hasPrefix("SYSTEM") else { return }
                realtimeHub.bump()
            },
            onReconnect: {
                realtimeHub.bumpReconnect()
            }
        )
    }
}
