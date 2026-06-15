//
//  FirebaseAuthHelper.swift
//  payload-app-ios
//
//  Firebase Auth stub — exchanges backend custom tokens when FirebaseAuth SPM is added.
//  PIN + legacy JWT remain the production auth path until SDK integration completes.
//

import Foundation

final class FirebaseAuthHelper {
    static let shared = FirebaseAuthHelper()

    private(set) var idToken: String?

    private init() {}

    func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
        guard !customToken.isEmpty else {
            completion(nil)
            return
        }
        // TODO: Uncomment when FirebaseAuth SPM is added to the Xcode project.
        print("[FirebaseAuth] Stub — custom token stored, awaiting SDK integration")
        completion(nil)
    }

    func signOut() {
        idToken = nil
    }
}
