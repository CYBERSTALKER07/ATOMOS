//
//  reatilerappApp.swift
//  reatilerapp
//
//  Created by Shakhzod on 3/17/26.
//

import SwiftData
import SwiftUI

@main
struct reatilerappApp: App {
    @State private var cartManager = CartManager()
    @State private var authManager = AuthManager.shared

    var body: some Scene {
        WindowGroup {
            Group {
                if authManager.isLoggedIn {
                    ContentView()
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
