//
//  LabPayloadApp.swift
//  payload-app-ios
//
//  Single-target SwiftUI iPad app for the PAYLOAD role. Mirrors the
//  driverappios composition pattern: TokenStore is the manual DI root,
//  passed via .environment(_:) so child views can drive auth state.
//

import SwiftUI

@main
struct LabPayloadApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @State private var tokenStore = TokenStore.shared

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(tokenStore)
        }
    }
}
