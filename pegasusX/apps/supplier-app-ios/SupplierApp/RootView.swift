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
                NavigationStack {
                    BusinessSetupView()
                }
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
            syncRealtime(authenticated: authenticated)
        }
        .task {
            syncRealtime(authenticated: tokenStore.isAuthenticated)
            await PushNotificationManager.shared.requestAuthorization()
        }
    }

    private func syncRealtime(authenticated: Bool) {
        if authenticated {
            realtime.connect(
                onEvent: { event in
                    guard !event.type.hasPrefix("SYSTEM") else { return }
                    Task { @MainActor in
                        realtimeHub.bump(event: event)
                    }
                },
                onReconnect: {
                    realtimeHub.bumpReconnect()
                }
            )
        } else {
            realtime.disconnect()
        }
    }
}
