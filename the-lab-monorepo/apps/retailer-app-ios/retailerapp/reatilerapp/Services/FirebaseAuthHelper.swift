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

final class FirebaseAuthHelper {
    static let shared = FirebaseAuthHelper()
    
    /// The Firebase ID token (after custom token exchange). Preferred over legacy JWT.
    private(set) var idToken: String?
    
    private init() {}
    
    /// Initialize Firebase for auth. Call once from app launch.
    func initialize() {
        // TODO: Uncomment when FirebaseAuth SPM is added
    }
    
    /// Exchange a Firebase Custom Token from the backend for an ID token.
    func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
        guard !customToken.isEmpty else {
            completion(nil)
            return
        }
        // TODO: Uncomment when FirebaseAuth SPM is added:
        // Auth.auth().signIn(withCustomToken: customToken) { [weak self] result, error in
        //     guard error == nil, let user = result?.user else {
        //         completion(nil)
        //         return
        //     }
        //     user.getIDToken { token, _ in
        //         self?.idToken = token
        //         completion(token)
        //     }
        // }
        print("[FirebaseAuth] Stub — custom token received, awaiting SDK integration")
        completion(nil)
    }
    
    /// Sign out of Firebase Auth.
    func signOut() {
        idToken = nil
    }
}
