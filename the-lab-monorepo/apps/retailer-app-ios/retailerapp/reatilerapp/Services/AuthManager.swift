import Foundation
import SwiftUI

// MARK: - Auth Manager

@Observable
final class AuthManager {
    static let shared = AuthManager()

    var isLoggedIn: Bool = false
    var currentUser: User?
    var isLoading = false
    var errorMessage: String?

    private let api = APIClient.shared

    private init() {
        if let token = api.authToken {
            isLoggedIn = true
            // Restore user from Keychain
            let userId = KeychainHelper.read(key: "user_id") ?? ""
            let userName = KeychainHelper.read(key: "user_name") ?? ""
            let company = KeychainHelper.read(key: "user_company") ?? ""
            currentUser = User(id: userId, name: userName, company: company, email: "", avatarURL: nil)
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
            api.authToken = response.token
            KeychainHelper.save(key: "user_id", value: response.user.id)
            KeychainHelper.save(key: "user_name", value: response.user.name)
            KeychainHelper.save(key: "user_company", value: response.user.company)
            // Exchange Firebase custom token (graceful degradation)
            if let fbToken = response.firebaseToken, !fbToken.isEmpty {
                FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
            }
            currentUser = response.user
            isLoggedIn = true
        } catch {
            errorMessage = error.localizedDescription
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
            api.authToken = response.token
            KeychainHelper.save(key: "user_id", value: response.user.id)
            KeychainHelper.save(key: "user_name", value: response.user.name)
            KeychainHelper.save(key: "user_company", value: response.user.company)
            // Exchange Firebase custom token (graceful degradation)
            if let fbToken = response.firebaseToken, !fbToken.isEmpty {
                FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
            }
            currentUser = response.user
            isLoggedIn = true
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    // MARK: - Logout

    func logout() {
        api.authToken = nil
        KeychainHelper.delete(key: "user_id")
        KeychainHelper.delete(key: "user_name")
        KeychainHelper.delete(key: "user_company")
        FirebaseAuthHelper.shared.signOut()
        currentUser = nil
        isLoggedIn = false
    }
}
