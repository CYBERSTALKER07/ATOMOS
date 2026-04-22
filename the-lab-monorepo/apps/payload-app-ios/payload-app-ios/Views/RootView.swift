//
//  RootView.swift
//  payload-app-ios
//

import SwiftUI

/// Auth-gated root. Routes between LoginView and the home scaffold based on
/// TokenStore.isAuthenticated. Phases 3-9 hang their feature views off
/// HomeScaffold inside the authenticated branch.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore

    var body: some View {
        Group {
            if tokenStore.isAuthenticated {
                HomeView()
                    .transition(.opacity)
            } else {
                LoginView()
                    .transition(.opacity)
            }
        }
        .animation(.snappy, value: tokenStore.isAuthenticated)
    }
}
