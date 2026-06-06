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
    @State private var driverSocketState = DriverSocketState.shared

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
        .buttonStyle(.pressable)
        .animation(Anim.snappy, value: tokenStore.isAuthenticated)
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
