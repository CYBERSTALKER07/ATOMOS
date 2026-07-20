//
//  RootView.swift
//  payload-app-ios
//

import SwiftUI

/// Auth-gated root. Client-policy banner is global across login + home.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
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
            let state = await AutoUpdater.evaluate(
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
                AutoUpdater.promptInstall(manifest: state.manifest, force: true)
            }
        } catch {
            // Policy fetch is optional on local/dev stacks.
        }
    }
}
