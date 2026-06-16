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
        .modelContainer(for: OfflineDelivery.self)
    }
}

/// Auth-gated root: shows LoginView or MainTabView based on token state.
/// Client-policy banner is global across login + authenticated shells.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var driverSocketState = DriverSocketState.shared
    @State private var clientPolicyMessage: String?

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(message: clientPolicyMessage)
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
                let minimumVersion: String
                let deferReason: String?

                enum CodingKeys: String, CodingKey {
                    case outdated
                    case forceUpdate = "force_update"
                    case minimumVersion = "minimum_version"
                    case deferReason = "defer_reason"
                }
            }
            var components = URLComponents()
            components.queryItems = [
                URLQueryItem(name: "role", value: "DRIVER"),
                URLQueryItem(name: "platform", value: "ios"),
                URLQueryItem(name: "version", value: version),
                URLQueryItem(name: "channel", value: "production"),
            ]
            let query = components.percentEncodedQuery.map { "?\($0)" } ?? ""
            let policy: ClientPolicy = try await APIClient.shared.get(
                "/v1/platform/client-policy\(query)"
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
                AutoUpdater.checkForUpdates()
            } else {
                clientPolicyMessage = nil
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

                Text("App Update Required")
                    .font(.system(size: 24, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                Text(message)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)

                Button(action: onSignOut) {
                    Text("Sign Out")
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
