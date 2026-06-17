import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var realtime = SupplierRealtimeClient()

    var body: some View {
        Group {
            if !tokenStore.isAuthenticated {
                LoginView()
            } else if tokenStore.needsBusinessSetup {
                BusinessSetupView()
            } else if tokenStore.needsBillingGate {
                BillingGateView()
            } else {
                SupplierAdaptiveShell()
            }
        }
        .animation(SupplierAnim.smooth, value: tokenStore.isAuthenticated)
        .animation(SupplierAnim.smooth, value: tokenStore.needsBusinessSetup)
        .animation(SupplierAnim.smooth, value: tokenStore.needsBillingGate)
        .onChange(of: tokenStore.isAuthenticated) { _, authenticated in
            if authenticated {
                realtime.connect(
                    onEvent: { event in
                        guard !event.type.hasPrefix("SYSTEM") else { return }
                        realtimeHub.bump()
                    },
                    onReconnect: {
                        realtimeHub.bumpReconnect()
                    }
                )
            } else {
                realtime.disconnect()
            }
        }
        .onAppear {
            if tokenStore.isAuthenticated {
                realtime.connect(
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
    }
}
