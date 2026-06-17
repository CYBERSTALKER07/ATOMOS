import Foundation
import SwiftUI

// MARK: - Auth Manager

@Observable
final class AuthManager {
    static let shared = AuthManager()

    var isLoggedIn: Bool = false
    var needsSetup: Bool = false
    var currentUser: User?
    var isLoading = false
    var errorMessage: String?

    private let api = APIClient.shared

    private init() {
        if api.authToken != nil {
            isLoggedIn = true
            let userId = KeychainHelper.read(key: "user_id") ?? ""
            let userName = KeychainHelper.read(key: "user_name") ?? ""
            let company = KeychainHelper.read(key: "user_company") ?? ""
            currentUser = User(id: userId, name: userName, company: company, email: "", avatarURL: nil)
            let configured = KeychainHelper.read(key: "is_configured") == "true"
            needsSetup = !configured
        }
    }

    // MARK: - Phone Validation (UZ: +998 XX XXX XX XX)

    static func formatUzPhone(_ raw: String) -> String? {
        let digits = raw.filter(\.isNumber)
        let phone: String?
        if digits.hasPrefix("998"), digits.count == 12 {
            phone = "+\(digits)"
        } else if digits.count == 9 {
            phone = "+998\(digits)"
        } else if raw.hasPrefix("+998"), raw.count == 13 {
            phone = raw
        } else {
            phone = nil
        }
        guard let result = phone,
              result.range(of: #"^\+998\d{9}$"#, options: .regularExpression) != nil
        else { return nil }
        return result
    }

    // MARK: - Login

    func login(phone: String, password: String) async {
        guard let formatted = Self.formatUzPhone(phone) else {
            errorMessage = "Invalid number. Use +998 XX XXX XX XX."
            return
        }
        guard password.count >= 4 else {
            errorMessage = "Password too short."
            return
        }

        isLoading = true
        errorMessage = nil

        do {
            let body = LoginRequest(phoneNumber: formatted, password: password)
            let response: AuthResponse = try await api.post(path: "/v1/auth/retailer/login", body: body)
            applyAuthResponse(response)
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Sign-in access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry sign in.",
                fallback: "Sign in failed. Please try again.",
            )
        }

        isLoading = false
    }

    // MARK: - Register

    func register(
        phone: String,
        password: String,
        storeName: String,
        ownerName: String,
        addressText: String,
        latitude: Double,
        longitude: Double,
        taxId: String?,
        receivingWindowOpen: String? = nil,
        receivingWindowClose: String? = nil,
        accessType: String? = nil,
        storageCeilingHeightCM: Double? = nil
    ) async {
        guard let formatted = Self.formatUzPhone(phone) else {
            errorMessage = "Invalid number. Use +998 XX XXX XX XX."
            return
        }
        guard password.count >= 4 else {
            errorMessage = "Password too short."
            return
        }

        isLoading = true
        errorMessage = nil

        do {
            let body = RegisterRequest(
                phoneNumber: formatted, password: password,
                storeName: storeName, ownerName: ownerName,
                addressText: addressText, latitude: latitude, longitude: longitude,
                taxId: taxId?.isEmpty == true ? nil : taxId,
                receivingWindowOpen: receivingWindowOpen?.isEmpty == true ? nil : receivingWindowOpen,
                receivingWindowClose: receivingWindowClose?.isEmpty == true ? nil : receivingWindowClose,
                accessType: accessType?.isEmpty == true ? nil : accessType,
                storageCeilingHeightCM: storageCeilingHeightCM
            )
            let response: AuthResponse = try await api.post(path: "/v1/auth/retailer/register", body: body)
            applyAuthResponse(response)
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Registration access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry registration.",
                fallback: "Registration failed. Please try again.",
            )
        }

        isLoading = false
    }

    // MARK: - Logout

    func logout() {
        api.authToken = nil
        KeychainHelper.delete(key: "user_id")
        KeychainHelper.delete(key: "user_name")
        KeychainHelper.delete(key: "user_company")
        KeychainHelper.delete(key: "is_configured")
        FirebaseAuthHelper.shared.signOut()
        currentUser = nil
        isLoggedIn = false
        needsSetup = false
    }

    func markConfigured() {
        KeychainHelper.save(key: "is_configured", value: "true")
        needsSetup = false
    }

    private func applyAuthResponse(_ response: AuthResponse) {
        api.authToken = response.token
        KeychainHelper.save(key: "user_id", value: response.user.id)
        KeychainHelper.save(key: "user_name", value: response.user.name)
        KeychainHelper.save(key: "user_company", value: response.user.company)
        let configured = response.isConfigured ?? true
        KeychainHelper.save(key: "is_configured", value: configured ? "true" : "false")
        if let fbToken = response.firebaseToken, !fbToken.isEmpty {
            FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
        }
        currentUser = response.user
        isLoggedIn = true
        needsSetup = !configured
    }
}
