//
//  FirebaseAuthHelper.swift
//  payload-app-ios
//
//  Firebase phone OTP + custom-token exchange for payload terminal auth.
//

import FirebaseAuth
import FirebaseCore
import Foundation

@MainActor
final class FirebaseAuthHelper {
    static let shared = FirebaseAuthHelper()

    private(set) var idToken: String?
    private var verificationID: String?
    private var initialized = false

    private init() {}

    func configure() {
        guard !initialized else { return }
        if FirebaseApp.app() == nil {
            let options = FirebaseOptions(
                googleAppID: "1:000000000000:ios:0000000000000001",
                gcmSenderID: "000000000000"
            )
            options.projectID = "demo-pegasus"
            options.apiKey = "demo-key"
            FirebaseApp.configure(options: options)
        }
        #if DEBUG
        Auth.auth().useEmulator(withHost: "localhost", port: 9099)
        #endif
        initialized = true
    }

    func sendPhoneVerification(phone: String) async throws {
        configure()
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw NSError(domain: "FirebaseAuth", code: 1, userInfo: [NSLocalizedDescriptionKey: "Phone number required"])
        }
        let verificationID = try await withCheckedThrowingContinuation { (cont: CheckedContinuation<String, Error>) in
            PhoneAuthProvider.provider().verifyPhoneNumber(trimmed, uiDelegate: nil) { verificationID, error in
                if let error {
                    cont.resume(throwing: error)
                    return
                }
                guard let verificationID else {
                    cont.resume(throwing: NSError(
                        domain: "FirebaseAuth",
                        code: 2,
                        userInfo: [NSLocalizedDescriptionKey: "Missing verification ID"]
                    ))
                    return
                }
                cont.resume(returning: verificationID)
            }
        }
        self.verificationID = verificationID
    }

    func verifySmsCode(_ code: String) async throws -> String {
        configure()
        guard let verificationID else {
            throw NSError(
                domain: "FirebaseAuth",
                code: 3,
                userInfo: [NSLocalizedDescriptionKey: "No verification in progress; request a code first"]
            )
        }
        let credential = PhoneAuthProvider.provider().credential(
            withVerificationID: verificationID,
            verificationCode: code.trimmingCharacters(in: .whitespacesAndNewlines)
        )
        let result = try await Auth.auth().signIn(with: credential)
        self.verificationID = nil
        let token = try await result.user.getIDToken()
        idToken = token
        return token
    }

    func exchangeCustomToken(_ customToken: String, completion: @escaping (String?) -> Void) {
        configure()
        guard !customToken.isEmpty else {
            completion(nil)
            return
        }
        Auth.auth().signIn(withCustomToken: customToken) { result, error in
            if let error {
                print("[FirebaseAuth] Custom token exchange failed: \(error.localizedDescription)")
                completion(nil)
                return
            }
            result?.user.getIDToken { token, _ in
                self.idToken = token
                completion(token)
            }
        }
    }

    func signOut() {
        verificationID = nil
        idToken = nil
        try? Auth.auth().signOut()
    }
}
