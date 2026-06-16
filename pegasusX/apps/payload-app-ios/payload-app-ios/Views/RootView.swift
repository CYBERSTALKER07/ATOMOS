//
//  RootView.swift
//  payload-app-ios
//

import SwiftUI

/// Auth-gated root. Client-policy banner is global across login + home.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var clientPolicyMessage: String?

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(message: clientPolicyMessage)
            Group {
                if tokenStore.isAuthenticated {
                    HomeView()
                        .transition(.opacity)
                } else {
                    LoginView()
                        .transition(.opacity)
                }
            }
        }
        .animation(.snappy, value: tokenStore.isAuthenticated)
        .task(id: tokenStore.isAuthenticated) {
            await loadClientPolicy()
        }
    }

    @MainActor
    private func loadClientPolicy() async {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0.0"
        do {
            let policy = try await APIClient.shared.clientPolicy(platform: "ios", version: version)
            if policy.outdated || policy.forceUpdate {
                var message = policy.forceUpdate ? "Update required" : "Update available"
                if !policy.minimumVersion.isEmpty {
                    message += " — minimum version \(policy.minimumVersion)"
                }
                if let deferReason = policy.deferReason, !deferReason.isEmpty {
                    message += ". \(deferReason)"
                }
                clientPolicyMessage = message
                if policy.outdated || policy.forceUpdate {
                    AutoUpdater.checkForUpdates()
                }
            } else {
                clientPolicyMessage = nil
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
        }
    }
}
