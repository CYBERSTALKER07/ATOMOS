//
//  LoginViewModel.swift
//  payload-app-ios
//

import Foundation
import Observation

@MainActor
@Observable
final class LoginViewModel {
    var phone: String = ""
    var pin: String = ""
    var loading: Bool = false
    var error: String?

    func setPin(_ value: String) {
        let digits = value.filter { $0.isNumber }
        pin = String(digits.prefix(6))
    }

    func submit() async {
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, pin.count == 6 else {
            error = "Phone and 6-digit PIN required"
            return
        }
        loading = true
        defer { loading = false }
        error = nil
        do {
            let resp = try await APIClient.shared.login(phone: trimmed, pin: pin)
            TokenStore.shared.saveSession(from: resp)
        } catch APIError.unauthorized {
            error = "Invalid credentials"
        } catch APIError.problemDetail(let p) {
            error = p.detail ?? p.title ?? "Login failed"
        } catch {
            self.error = "Login failed — check connection"
        }
    }
}
