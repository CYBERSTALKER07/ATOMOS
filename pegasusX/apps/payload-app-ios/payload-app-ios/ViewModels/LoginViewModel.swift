//
//  LoginViewModel.swift
//  payload-app-ios
//

import Foundation
import Observation

enum LoginMode {
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

    func setMode(_ next: LoginMode) {
        mode = next
        error = nil
        otpSent = false
        otpCode = ""
        pin = ""
    }

    func setPin(_ value: String) {
        let digits = value.filter { $0.isNumber }
        pin = String(digits.prefix(8))
    }

    func setOtp(_ value: String) {
        let digits = value.filter { $0.isNumber }
        otpCode = String(digits.prefix(6))
    }

    func sendOtp() async {
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            error = "Phone number required"
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            try await FirebaseAuthHelper.shared.sendPhoneVerification(phone: trimmed)
            otpSent = true
        } catch {
            self.error = error.localizedDescription
        }
    }

    func verifyOtp() async {
        guard otpCode.count >= 6 else {
            error = "Enter the 6-digit verification code"
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            let idToken = try await FirebaseAuthHelper.shared.verifySmsCode(otpCode)
            let resp = try await APIClient.shared.login(idToken: idToken)
            TokenStore.shared.saveSession(from: resp)
            if let fbToken = resp.firebaseToken {
                FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
            }
        } catch APIError.unauthorized {
            error = "Invalid credentials"
        } catch APIError.problemDetail(let p) {
            error = p.detail ?? p.title ?? "Login failed"
        } catch {
            self.error = error.localizedDescription
        }
    }

    func submitPin() async {
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, pin.count >= 6 else {
            error = "Phone and PIN required"
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            let resp = try await APIClient.shared.login(phone: trimmed, pin: pin)
            TokenStore.shared.saveSession(from: resp)
            if let fbToken = resp.firebaseToken {
                FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
            }
        } catch APIError.unauthorized {
            error = "Invalid credentials"
        } catch APIError.problemDetail(let p) {
            error = p.detail ?? p.title ?? "Login failed"
        } catch {
            self.error = "Login failed — check connection"
        }
    }
}
