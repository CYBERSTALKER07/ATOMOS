//
//  LoginViewModel.swift
//  payload-app-ios
//

import Foundation
import Observation

enum LoginMode: Equatable {
    case otp
    case pinDev
}

@MainActor
@Observable
final class LoginViewModel {
    var mode: LoginMode = .otp
    var phone: String = ""
    var otpCode: String = ""
    var pin: String = ""
    var otpSent: Bool = false
    var loading: Bool = false
    var error: String?

    private let sendPhoneVerificationHandler: @MainActor (String) async throws -> Void
    private let verifySmsCodeHandler: @MainActor (String) async throws -> String
    private let loginWithIdTokenHandler: @MainActor (String) async throws -> LoginResponse
    private let loginWithPinHandler: @MainActor (String, String) async throws -> LoginResponse
    private let onAuthenticatedHandler: @MainActor (LoginResponse) -> Void

    init(
        sendPhoneVerification: @escaping @MainActor (String) async throws -> Void = { phone in
            try await FirebaseAuthHelper.shared.sendPhoneVerification(phone: phone)
        },
        verifySmsCode: @escaping @MainActor (String) async throws -> String = { code in
            try await FirebaseAuthHelper.shared.verifySmsCode(code)
        },
        loginWithIdToken: @escaping @MainActor (String) async throws -> LoginResponse = { idToken in
            try await APIClient.shared.login(idToken: idToken)
        },
        loginWithPin: @escaping @MainActor (String, String) async throws -> LoginResponse = { phone, pin in
            try await APIClient.shared.login(phone: phone, pin: pin)
        },
        onAuthenticated: @escaping @MainActor (LoginResponse) -> Void = { resp in
            TokenStore.shared.saveSession(from: resp)
            if let fbToken = resp.firebaseToken {
                FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
            }
        }
    ) {
        self.sendPhoneVerificationHandler = sendPhoneVerification
        self.verifySmsCodeHandler = verifySmsCode
        self.loginWithIdTokenHandler = loginWithIdToken
        self.loginWithPinHandler = loginWithPin
        self.onAuthenticatedHandler = onAuthenticated
    }

    var canSubmitPin: Bool {
        !phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && pin.count >= 6
    }

    func setMode(_ next: LoginMode) {
        mode = next
        error = nil
        otpSent = false
        otpCode = ""
        pin = ""
    }

    func setPin(_ value: String) {
        let digits = value.filter { $0.isNumber }
        let sanitized = String(digits.prefix(8))
        if pin != sanitized {
            pin = sanitized
        }
    }

    func setOtp(_ value: String) {
        let digits = value.filter { $0.isNumber }
        let sanitized = String(digits.prefix(6))
        if otpCode != sanitized {
            otpCode = sanitized
        }
    }

    func sendOtp() async {
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            error = "Phone number required"
            PegasusHaptics.warning()
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            try await sendPhoneVerificationHandler(trimmed)
            otpSent = true
            PegasusHaptics.success()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func verifyOtp() async {
        setOtp(otpCode)
        guard otpCode.count >= 6 else {
            error = "Enter the 6-digit verification code"
            PegasusHaptics.warning()
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            let idToken = try await verifySmsCodeHandler(otpCode)
            let resp = try await loginWithIdTokenHandler(idToken)
            onAuthenticatedHandler(resp)
            PegasusHaptics.success()
        } catch APIError.unauthorized {
            error = "Invalid credentials"
            PegasusHaptics.error()
        } catch APIError.problemDetail(let p) {
            error = p.detail ?? p.title ?? "Login failed"
            PegasusHaptics.error()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func submitPin() async {
        setPin(pin)
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, pin.count >= 6 else {
            error = "Phone and PIN required"
            PegasusHaptics.warning()
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            let resp = try await loginWithPinHandler(trimmed, pin)
            onAuthenticatedHandler(resp)
            PegasusHaptics.success()
        } catch APIError.unauthorized {
            error = "Invalid credentials"
            PegasusHaptics.error()
        } catch APIError.problemDetail(let p) {
            error = p.detail ?? p.title ?? "Login failed"
            PegasusHaptics.error()
        } catch {
            self.error = "Login failed — check connection"
            PegasusHaptics.error()
        }
    }
}
