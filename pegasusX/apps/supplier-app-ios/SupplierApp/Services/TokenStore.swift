import Foundation
import Security

@Observable
@MainActor
final class TokenStore {
    static let shared = TokenStore()

    private(set) var token: String?
    private(set) var refreshToken: String?
    private(set) var supplierId: String?
    private(set) var isConfigured: Bool = false
    private(set) var isRegistered: Bool = false
    /// Matches web portal "Skip for now" — shell access while JWT stays unconfigured.
    private(set) var billingGateDismissed: Bool = false

    var isAuthenticated: Bool { token != nil }

    var needsBillingGate: Bool {
        isAuthenticated && isRegistered && !isConfigured && !billingGateDismissed
    }

    var needsBusinessSetup: Bool {
        isAuthenticated && !isRegistered
    }

    private let service = "com.pegasusx.supplier"

    private init() {
        token = readKeychain(account: "pegasus_supplier_jwt")
        refreshToken = readKeychain(account: "refresh_token")
        supplierId = readKeychain(account: "supplier_id")
        if let configured = readKeychain(account: "is_configured") {
            isConfigured = configured == "1"
        }
        if let registered = readKeychain(account: "is_registered") {
            isRegistered = registered == "1"
        }
        billingGateDismissed = readKeychain(account: "billing_gate_dismissed") == "1"
    }

    func store(auth: LoginResponse) {
        guard let token = auth.token, !token.isEmpty else { return }
        self.token = token
        refreshToken = auth.refreshToken
        supplierId = auth.supplierId
        isConfigured = auth.isConfigured
        isRegistered = auth.isRegistered
        billingGateDismissed = false
        writeKeychain(account: "pegasus_supplier_jwt", value: token)
        if let refreshToken = auth.refreshToken, !refreshToken.isEmpty {
            writeKeychain(account: "refresh_token", value: refreshToken)
        }
        writeKeychain(account: "supplier_id", value: auth.supplierId)
        writeKeychain(account: "is_configured", value: auth.isConfigured ? "1" : "0")
        writeKeychain(account: "is_registered", value: auth.isRegistered ? "1" : "0")
        deleteKeychain(account: "billing_gate_dismissed")
    }

    func storeRegister(_ auth: RegisterResponse) {
        guard let token = auth.token, !token.isEmpty else { return }
        self.token = token
        supplierId = auth.supplierId
        isConfigured = auth.isConfigured
        isRegistered = auth.isRegistered
        billingGateDismissed = false
        writeKeychain(account: "pegasus_supplier_jwt", value: token)
        writeKeychain(account: "supplier_id", value: auth.supplierId)
        writeKeychain(account: "is_configured", value: auth.isConfigured ? "1" : "0")
        writeKeychain(account: "is_registered", value: auth.isRegistered ? "1" : "0")
        deleteKeychain(account: "billing_gate_dismissed")
    }

    func markRegistered(_ registered: Bool) {
        isRegistered = registered
        writeKeychain(account: "is_registered", value: registered ? "1" : "0")
    }

    func markConfigured(_ configured: Bool) {
        isConfigured = configured
        writeKeychain(account: "is_configured", value: configured ? "1" : "0")
    }

    func dismissBillingGate() {
        billingGateDismissed = true
        writeKeychain(account: "billing_gate_dismissed", value: "1")
    }

    func showBillingGate() {
        billingGateDismissed = false
        deleteKeychain(account: "billing_gate_dismissed")
    }

    func clear() {
        token = nil
        refreshToken = nil
        supplierId = nil
        isConfigured = false
        isRegistered = false
        billingGateDismissed = false
        deleteKeychain(account: "pegasus_supplier_jwt")
        deleteKeychain(account: "refresh_token")
        deleteKeychain(account: "supplier_id")
        deleteKeychain(account: "is_configured")
        deleteKeychain(account: "is_registered")
        deleteKeychain(account: "billing_gate_dismissed")
    }

    private func writeKeychain(account: String, value: String) {
        guard let data = value.data(using: .utf8) else { return }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
        var add = query
        add[kSecValueData as String] = data
        SecItemAdd(add as CFDictionary, nil)
    }

    private func readKeychain(account: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var ref: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &ref)
        guard status == errSecSuccess, let data = ref as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func deleteKeychain(account: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
    }
}
