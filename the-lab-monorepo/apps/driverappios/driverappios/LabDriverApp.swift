//
//  LabDriverApp.swift
//  driverappios
//

import SwiftData
import SwiftUI

@main
struct LabDriverApp: App {
    @State private var tokenStore = TokenStore.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
        }
        .modelContainer(for: OfflineDelivery.self)
    }
}

/// Auth-gated root: shows LoginView or MainTabView based on token state.
struct RootView: View {
    @Environment(TokenStore.self) private var tokenStore

    var body: some View {
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
        .animation(Anim.snappy, value: tokenStore.isAuthenticated)
    }
}
