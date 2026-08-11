//
//  FirebaseAuthHelper.swift
//  retailerapp
//
//  Firebase custom-token exchange for dual-mode auth with backend JWT.
//

import FirebaseAuth
import FirebaseCore
import Foundation

@MainActor
final class FirebaseAuthHelper {
    static let shared = FirebaseAuthHelper()

    private(set) var idToken: String?
    private var initialized = false

    private init() {}

    func configure() {
        guard !initialized else { return }
        if FirebaseApp.app() == nil {
            if let path = Bundle.main.path(forResource: "GoogleService-Info", ofType: "plist"),
               let options = FirebaseOptions(contentsOfFile: path) {
                FirebaseApp.configure(options: options)
            } else {
                let options = FirebaseOptions(
                    googleAppID: "1:000000000000:ios:0000000000000000",
                    gcmSenderID: "000000000000"
                )
                options.projectID = "demo-pegasus"
                options.apiKey = "demo-key"
                FirebaseApp.configure(options: options)
            }
        }
        #if DEBUG
        if let host = ProcessInfo.processInfo.environment["FIREBASE_AUTH_EMULATOR_HOST"], !host.isEmpty {
            let parts = host.split(separator: ":", maxSplits: 1).map(String.init)
            let emulatorHost = parts.first ?? "localhost"
            let emulatorPort = Int(parts.dropFirst().first ?? "9099") ?? 9099
            Auth.auth().useEmulator(withHost: emulatorHost, port: emulatorPort)
        }
        #endif
        initialized = true
    }

    func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
        guard !customToken.isEmpty else {
            completion(nil)
            return
        }
        configure()
        Auth.auth().signIn(withCustomToken: customToken) { [weak self] result, error in
            if let error {
                print("[FirebaseAuth] Custom token exchange failed: \(error.localizedDescription)")
                completion(nil)
                return
            }
            guard let user = result?.user else {
                completion(nil)
                return
            }
            user.getIDToken { token, _ in
                self?.idToken = token
                completion(token)
            }
        }
    }

    func signOut() {
        idToken = nil
        if initialized {
            try? Auth.auth().signOut()
        }
    }
}
