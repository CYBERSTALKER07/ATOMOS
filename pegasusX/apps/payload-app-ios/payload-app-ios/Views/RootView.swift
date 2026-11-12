//
//  RootView.swift
//  payload-app-ios
//

import SwiftUI

/// Auth-gated root. Client-policy banner is global across login + home.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var viewModel = RootViewModel()

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: viewModel.clientPolicyMessage,
                force: viewModel.clientPolicyForce,
                onUpdate: viewModel.clientPolicyMessage == nil ? nil : {
                    viewModel.triggerInstall()
                },
                onDismiss: viewModel.clientPolicyForce ? nil : {
                    viewModel.dismissBanner()
                }
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
            await viewModel.loadClientPolicy()
        }
    }
}
