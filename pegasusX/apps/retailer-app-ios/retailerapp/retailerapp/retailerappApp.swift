//
//  retailerappApp.swift
//  retailerapp
//
//  Created by Shakhzod on 3/17/26.
//

import SwiftData
import SwiftUI

@main
struct retailerappApp: App {
    @State private var cartManager = CartManager()
    @State private var authManager = AuthManager.shared

    init() {
        FirebaseAuthHelper.shared.configure()
    }

    var body: some Scene {
        WindowGroup {
            Group {
                if authManager.needsOrgSelect {
                    SelectOrgView(auth: authManager)
                } else if authManager.isLoggedIn {
                    if authManager.needsSetup {
                        SetupView()
                    } else {
                        ContentView()
                    }
                } else {
                    LoginView()
                }
            }
            .environment(cartManager)
            .environment(authManager)
        }
        .modelContainer(for: PendingOrder.self)
    }
}
