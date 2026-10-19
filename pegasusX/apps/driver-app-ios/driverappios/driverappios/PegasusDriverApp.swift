//
//  LabDriverApp.swift
//  driverappios
//

import SwiftData
import SwiftUI

@main
struct LabDriverApp: App {
    @State private var tokenStore = TokenStore.shared
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
        }
        .modelContainer(for: [OfflineDelivery.self, QueuedDriverAction.self])
    }
}

/// Auth-gated root: shows LoginView or MainTabView based on token state.
/// Client-policy banner is global across login + authenticated shells.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var driverSocketState = DriverSocketState.shared
    @State private var clientPolicyMessage: String?
    @State private var clientPolicyForce = false
    @State private var pendingManifest: AutoUpdater.Manifest?

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: clientPolicyMessage,
                force: clientPolicyForce,
                onUpdate: clientPolicyMessage == nil ? nil : {
                    AutoUpdater.promptInstall(manifest: pendingManifest, force: clientPolicyForce)
                },
                onDismiss: clientPolicyForce ? nil : { clientPolicyMessage = nil },
            )
            Group {
                if tokenStore.isAuthenticated {
                    MainTabView()
                        .transition(.opacity)
                } else {
                    LoginView {
                        // onAuthenticated — token is already saved by LoginView
                    }
                    .transition(.opacity)
                }
            }
        }
        .buttonStyle(.pressable)
        .animation(Anim.snappy, value: tokenStore.isAuthenticated)
        .task(id: tokenStore.isAuthenticated) {
            await loadClientPolicy()
        }
        .task {
            await PushNotificationManager.shared.requestAuthorization()
        }
        .task(id: tokenStore.token) {
            if let token = tokenStore.token {
                driverSocketState.startMonitoring(
                    baseURL: APIClient.shared.apiBaseURL,
                    token: token
                )
            } else {
                driverSocketState.stopMonitoring()
            }
        }
        .overlay {
            if tokenStore.isAuthenticated, let notice = driverSocketState.outdatedNotice {
                DriverOutdatedGate(
                    message: notice.message,
                    onSignOut: {
                        driverSocketState.stopMonitoring()
                        tokenStore.logout()
                    }
                )
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
                let updateDeferred: Bool
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
            var components = URLComponents()
            components.queryItems = [
                URLQueryItem(name: "role", value: EnterpriseUpdateConfig.policyRole),
                URLQueryItem(name: "platform", value: "ios"),
                URLQueryItem(name: "version", value: version),
                URLQueryItem(name: "channel", value: EnterpriseUpdateConfig.channel),
            ]
            let query = components.percentEncodedQuery.map { "?\($0)" } ?? ""
            let policy: ClientPolicy = try await APIClient.shared.get(
                "/v1/platform/client-policy\(query)"
            )
            let state = await AutoUpdater.evaluate(
                outdated: policy.outdated,
                forceUpdate: policy.forceUpdate,
                updateDeferred: policy.updateDeferred,
                minimumVersion: policy.minimumVersion,
                recommendedVersion: policy.recommendedVersion,
                deferReason: policy.deferReason,
                updateURL: policy.updateURL,
            )
            clientPolicyMessage = state.message
            clientPolicyForce = state.force
            pendingManifest = state.manifest
            if state.force, state.available {
                AutoUpdater.promptInstall(manifest: state.manifest, force: true)
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
        }
    }
}

private struct DriverOutdatedGate: View {
    let message: String
    let onSignOut: () -> Void

    var body: some View {
        ZStack {
            LabTheme.bg.opacity(0.98)
                .ignoresSafeArea()

            VStack(spacing: LabTheme.s16) {
                Image(systemName: "arrow.triangle.2.circlepath.circle.fill")
                    .font(.system(size: 52, weight: .bold))
                    .foregroundStyle(LabTheme.warning)

                Text("mobile_driver.ui.app_update_required")
                    .font(.system(size: 24, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                Text(message)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)

                Button(action: onSignOut) {
                    Text("common.action.sign_out")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .padding(.horizontal, LabTheme.s24)
                .buttonStyle(.pressable)
            }
            .padding(.horizontal, LabTheme.s20)
        }
    }
}
