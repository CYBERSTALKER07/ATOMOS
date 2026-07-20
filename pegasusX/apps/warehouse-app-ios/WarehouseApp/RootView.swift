import SwiftUI

struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var realtime = WarehouseRealtimeClient()
    @State private var clientPolicyMessage: String?
    @State private var clientPolicyForce = false
    @State private var pendingManifest: AutoUpdater.Manifest?

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: clientPolicyMessage,
                force: clientPolicyForce,
                onUpdate: clientPolicyMessage == nil ? nil : {
                    AutoUpdater.shared.promptInstall(manifest: pendingManifest, force: clientPolicyForce)
                },
                onDismiss: clientPolicyForce ? nil : { clientPolicyMessage = nil },
            )
            Group {
                if !tokenStore.isAuthenticated {
                    LoginView()
                } else if !tokenStore.isConfigured {
                    LocationSetupView()
                } else {
                    WarehouseAdaptiveShell()
                }
            }
        }
        .buttonStyle(PressableButtonStyle())
        .animation(.smooth, value: tokenStore.isAuthenticated)
        .animation(.smooth, value: tokenStore.isConfigured)
        .task(id: tokenStore.isAuthenticated) {
            await loadClientPolicy()
        }
        .task(id: tokenStore.isConfigured) {
            if tokenStore.isAuthenticated && tokenStore.isConfigured {
                connectRealtime()
            }
        }
        .task {
            await PushNotificationManager.shared.requestAuthorization()
        }
        .onChange(of: tokenStore.isAuthenticated) { _, authenticated in
            if authenticated && tokenStore.isConfigured {
                connectRealtime()
            } else {
                realtime.disconnect()
            }
        }
        .onAppear {
            if tokenStore.isAuthenticated && tokenStore.isConfigured {
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
                let updateDeferred: Bool?
                let minimumVersion: String
                let recommendedVersion: String?
                let updateURL: String?
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case updateDeferred = "update_deferred"
                    case minimumVersion = "minimum_version"
                    case recommendedVersion = "recommended_version"
                    case updateURL = "update_url"
                    case deferReason = "defer_reason"
                }
            }
            let policy: ClientPolicy = try await APIClient.shared.get(
                "v1/platform/client-policy",
                query: [
                    "role": EnterpriseUpdateConfig.policyRole,
                    "platform": "ios",
                    "version": version,
                    "channel": EnterpriseUpdateConfig.channel,
                ],
            )
            let state = await AutoUpdater.shared.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred ?? false,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL,
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                AutoUpdater.shared.promptInstall(manifest: state.manifest, force: true)
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
