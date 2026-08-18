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
    /// C1.3: intermediate multi-org picker state (nil when not pending).
    var pendingMemberships: [RetailerMembershipDTO]?
    var needsOrgSelect: Bool = false

    private let api = APIClient.shared

    private init() {
        if api.authToken != nil {
            isLoggedIn = true
            let userId = KeychainHelper.read(key: "user_id") ?? ""
            let userName = KeychainHelper.read(key: "user_name") ?? ""
            let company = KeychainHelper.read(key: "user_company") ?? ""
            currentUser = User(id: userId, name: userName, company: company, email: nil, avatarURL: nil)
            let configured = KeychainHelper.read(key: "is_configured") == "true"
            needsSetup = !configured
        }
    }

    // MARK: - C1.3 clear-on-switch contract

    /// Clears cart, POS session, offline drafts, assist context before org change.
    func clearOrgScopedState() {
        let defaults = UserDefaults.standard
        let keys = [
            "retailer_cart",
            "retailer_pos_parked_cart_v1",
            "retailer_pending_pos_sales_v1",
            "retailer_stock_count_draft_v1",
            "retailer_assist_context_v1",
            "retailer_pos_session_v1",
        ]
        for key in keys {
            defaults.removeObject(forKey: key)
        }
        NotificationCenter.default.post(name: .pegasusOrgSwitched, object: nil)
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

    /// C1.3: complete multi-org login after picker selection.
    func selectOrg(retailerId: String) async {
        isLoading = true
        errorMessage = nil
        do {
            let body = SelectOrgRequest(retailerId: retailerId)
            let response: AuthResponse = try await api.post(path: "/v1/auth/retailer/select-org", body: body)
            applyAuthResponse(response, fromOrgChange: true)
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Organization access denied.",
                offline: "Offline. Reconnect to select organization.",
                fallback: "Could not select organization.",
            )
        }
        isLoading = false
    }

    /// C1.3: switch active org (full JWT required).
    func switchOrg(retailerId: String) async {
        isLoading = true
        errorMessage = nil
        do {
            let body = SelectOrgRequest(retailerId: retailerId)
            let response: AuthResponse = try await api.post(path: "/v1/auth/retailer/switch-org", body: body)
            applyAuthResponse(response, fromOrgChange: true)
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Organization switch denied.",
                offline: "Offline. Reconnect to switch organization.",
                fallback: "Could not switch organization.",
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
                addressText: addressText,
                deliveryAddress: addressText.isEmpty ? nil : addressText,
                placeId: nil,
                latitude: latitude, longitude: longitude,
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
        KeychainHelper.delete(key: "pending_org_token")
        FirebaseAuthHelper.shared.signOut()
        currentUser = nil
        isLoggedIn = false
        needsSetup = false
        needsOrgSelect = false
        pendingMemberships = nil
    }

    func markConfigured() {
        KeychainHelper.save(key: "is_configured", value: "true")
        needsSetup = false
    }

    private func applyAuthResponse(_ response: AuthResponse, fromOrgChange: Bool = false) {
        if response.isPendingOrgSelect {
            // Intermediate: do not mark fully logged in for business routes.
            api.authToken = response.token
            KeychainHelper.save(key: "pending_org_token", value: response.token)
            pendingMemberships = (response.memberships ?? []).filter(\.isActive)
            needsOrgSelect = true
            isLoggedIn = false
            needsSetup = false
            return
        }
        if fromOrgChange {
            clearOrgScopedState()
        }
        api.authToken = response.token
        KeychainHelper.delete(key: "pending_org_token")
        let user = response.user ?? User(
            id: response.retailerId ?? "unknown",
            name: "Retailer",
            company: "Workspace",
            email: nil,
            avatarURL: nil
        )
        KeychainHelper.save(key: "user_id", value: user.id)
        KeychainHelper.save(key: "user_name", value: user.name)
        KeychainHelper.save(key: "user_company", value: user.company)
        let configured = response.isConfigured ?? true
        KeychainHelper.save(key: "is_configured", value: configured ? "true" : "false")
        if let fbToken = response.firebaseToken, !fbToken.isEmpty {
            FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
        }
        currentUser = user
        isLoggedIn = true
        needsSetup = !configured
        needsOrgSelect = false
        pendingMemberships = nil
        Task { await PushNotificationManager.shared.uploadStoredTokenIfPossible() }
    }
}

extension Notification.Name {
    static let pegasusOrgSwitched = Notification.Name("pegasusx.orgSwitched")
}
