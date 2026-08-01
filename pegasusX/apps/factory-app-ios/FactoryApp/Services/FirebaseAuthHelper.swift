//
//  FirebaseAuthHelper.swift
//  FactoryAppIOS
//
//  Firebase phone OTP for factory staff auth.
//

import FirebaseAuth
import FirebaseCore
import Foundation

@MainActor
final class FirebaseAuthHelper {
    static let shared = FirebaseAuthHelper()

    private var verificationID: String?
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
                    googleAppID: "1:000000000000:ios:0000000000000001",
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

    func sendPhoneVerification(phone: String) async throws {
        configure()
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw NSError(
                domain: "FirebaseAuth",
                code: 1,
                userInfo: [NSLocalizedDescriptionKey: "Phone number required"]
            )
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
        return try await result.user.getIDToken()
    }

    func resetFlow() {
        verificationID = nil
    }

    func signOut() {
        resetFlow()
        try? Auth.auth().signOut()
    }
}
