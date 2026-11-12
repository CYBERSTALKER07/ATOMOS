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
    @State private var rootVM = RootViewModel()

    var body: some View {
        VStack(spacing: 0) {
            ClientPolicyBanner(
                message: rootVM.clientPolicyMessage,
                force: rootVM.clientPolicyForce,
                onUpdate: rootVM.clientPolicyMessage == nil ? nil : {
                    AutoUpdater.promptInstall(
                        manifest: rootVM.pendingManifest,
                        force: rootVM.clientPolicyForce
                    )
                },
                onDismiss: rootVM.clientPolicyForce ? nil : {
                    rootVM.dismissBanner()
                }
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
            await rootVM.loadClientPolicy()
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
                    .font(.largeTitle.weight(.bold))
                    .foregroundStyle(LabTheme.warning)
                    .accessibilityHidden(true)

                Text("mobile_driver.ui.app_update_required")
                    .font(.system(.title2, weight: .bold))
                    .foregroundStyle(LabTheme.fg)

                Text(message)
                    .font(.system(.subheadline, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)

                Button(action: onSignOut) {
                    Text("common.action.sign_out")
                        .font(.system(.subheadline, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 6)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .padding(.horizontal, LabTheme.s24)
                .buttonStyle(.pressable)
            }
            .padding(.horizontal, LabTheme.s20)
        }
    }
}
