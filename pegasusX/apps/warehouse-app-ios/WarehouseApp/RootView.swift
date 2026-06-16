import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var realtime = WarehouseRealtimeClient()
    @State private var clientPolicyMessage: String?

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(message: clientPolicyMessage)
            Group {
                if tokenStore.isAuthenticated {
                    WarehouseAdaptiveShell()
                } else {
                    LoginView()
                }
            }
        }
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
        .task(id: tokenStore.isAuthenticated) {
            await loadClientPolicy()
        }
        .task {
            await PushNotificationManager.shared.requestAuthorization()
        }
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

    @MainActor
    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            struct ClientPolicy: Decodable {
                let outdated: Bool
                let forceUpdate: Bool
                let minimumVersion: String
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case minimumVersion = "minimum_version"
                    case deferReason = "defer_reason"
                }
            }
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": "WAREHOUSE",
                    "platform": "ios",
                    "version": version,
                    "channel": "production",
                ],
            )
            if policy.outdated || policy.forceUpdate {
                var message = policy.forceUpdate ? "Update required" : "Update available"
                if !policy.minimumVersion.isEmpty {
                    message += " — minimum version \(policy.minimumVersion)"
                }
                if let deferReason = policy.deferReason, !deferReason.isEmpty {
                    message += ". \(deferReason)"
                }
                clientPolicyMessage = message
                AutoUpdater.shared.checkForUpdates()
            } else {
                clientPolicyMessage = nil
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
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
