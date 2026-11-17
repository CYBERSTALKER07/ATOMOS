import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = RootViewModel()

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: vm.clientPolicyMessage,
                force: vm.clientPolicyForce,
                onUpdate: vm.clientPolicyMessage == nil ? nil : {
                    vm.promptInstall()
                },
                onDismiss: vm.clientPolicyForce ? nil : {
                    vm.dismissBanner()
                }
            )
            Group {
                if !tokenStore.isAuthenticated {
                    LoginView()
                } else if !tokenStore.isConfigured {
                    NavigationStack {
                        LocationSetupView()
                    }
                } else {
                    WarehouseAdaptiveShell()
                }
            }
        }
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
        .animation(.smooth, value: tokenStore.isConfigured)
        .task(id: tokenStore.isAuthenticated) {
            await vm.loadClientPolicy()
        }
        .task(id: tokenStore.isConfigured) {
            if tokenStore.isAuthenticated && tokenStore.isConfigured {
                vm.connectRealtime(realtimeHub: realtimeHub)
            }
        }
        .task {
            await PushNotificationManager.shared.requestAuthorization()
        }
        .onChange(of: tokenStore.isAuthenticated) { _, authenticated in
            if authenticated && tokenStore.isConfigured {
                vm.connectRealtime(realtimeHub: realtimeHub)
            } else {
                vm.disconnectRealtime()
            }
        }
        .onAppear {
            if tokenStore.isAuthenticated && tokenStore.isConfigured {
                vm.connectRealtime(realtimeHub: realtimeHub)
            }
        }
    }
}
