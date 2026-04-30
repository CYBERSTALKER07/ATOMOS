import Foundation

// MARK: - Firebase Auth Helper (Stub)
//
// Firebase Auth integration for dual-mode authentication.
// This file is a stub that stores the Firebase custom token for future
// exchange once the Firebase/Auth SPM package is added to the Xcode project.
//
// TODO: Add Firebase/Auth SPM dependency in Xcode:
//   1. File → Add Package Dependencies → https://github.com/firebase/firebase-ios-sdk
//   2. Select "FirebaseAuth" product
//   3. Uncomment the Firebase calls below
//
// Once FirebaseAuth is available, the flow is:
//   1. Backend returns firebase_token (custom token)
//   2. Call Auth.auth().signIn(withCustomToken:) to exchange it
//   3. Use auth.currentUser?.getIDToken() for API calls

final class FirebaseAuthHelper {
    static let shared = FirebaseAuthHelper()
    
    /// The Firebase ID token (after custom token exchange). Preferred over legacy JWT.
    private(set) var idToken: String?
    
    private init() {}
    
    /// Initialize Firebase for auth. Call once from app launch.
    /// In debug, connects to the emulator on localhost:9099.
    func initialize() {
        // TODO: Uncomment when FirebaseAuth SPM is added:
        // import FirebaseCore
        // import FirebaseAuth
        //
        // if FirebaseApp.app() == nil {
        //     let options = FirebaseOptions(googleAppID: "1:000000000000:ios:0000000000000000", gcmSenderID: "000000000000")
        //     options.projectID = "demo-thelab"
        //     options.apiKey = "demo-key"
        //     FirebaseApp.configure(options: options)
        // }
        // #if DEBUG
        // Auth.auth().useEmulator(withHost: "localhost", port: 9099)
        // #endif
    }
    
    /// Exchange a Firebase Custom Token from the backend for an ID token.
    /// Returns the ID token string via completion. Degrades gracefully.
    func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
        guard !customToken.isEmpty else {
            completion(nil)
            return
        }
        // TODO: Uncomment when FirebaseAuth SPM is added:
        // Auth.auth().signIn(withCustomToken: customToken) { [weak self] result, error in
        //     guard error == nil, let user = result?.user else {
        //         print("[FirebaseAuth] Custom token exchange failed: \(error?.localizedDescription ?? "unknown")")
        //         completion(nil)
        //         return
        //     }
        //     user.getIDToken { token, _ in
        //         self?.idToken = token
        //         completion(token)
        //     }
        // }
        
        // Stub: store the custom token as-is for now (will be replaced with ID token after SDK integration)
        print("[FirebaseAuth] Stub — custom token received, awaiting SDK integration")
        completion(nil)
    }
    
    /// Get a fresh Firebase ID token. Returns nil if no session exists.
    func getIdToken(completion: @escaping (String?) -> Void) {
        // TODO: Uncomment when FirebaseAuth SPM is added:
        // guard let user = Auth.auth().currentUser else {
        //     completion(nil)
        //     return
        // }
        // user.getIDToken { token, _ in completion(token) }
        completion(idToken)
    }
    
    /// Sign out of Firebase Auth.
    func signOut() {
        idToken = nil
        // TODO: Uncomment when FirebaseAuth SPM is added:
        // try? Auth.auth().signOut()
    }
}
