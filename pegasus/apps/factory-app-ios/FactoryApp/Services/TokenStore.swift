import Foundation
import Security

@Observable
@MainActor
final class TokenStore {
    static let shared = TokenStore()

    private(set) var token: String?
    private(set) var refreshToken: String?
    private(set) var factoryId: String?

    var isAuthenticated: Bool { token != nil }

    private let service = "com.pegasus.factory"

    private init() {
        token = readKeychain(account: "factory_jwt")
        refreshToken = readKeychain(account: "factory_refresh_token")
        factoryId = readKeychain(account: "factory_id")
    }

    func store(auth: AuthResponse) {
        token = auth.token
        refreshToken = auth.refreshToken
        factoryId = auth.factoryId
        writeKeychain(account: "factory_jwt", value: auth.token)
        writeKeychain(account: "factory_refresh_token", value: auth.refreshToken)
        writeKeychain(account: "factory_id", value: auth.factoryId)
    }

    func updateTokens(token: String, refresh: String) {
        self.token = token
        self.refreshToken = refresh
        writeKeychain(account: "factory_jwt", value: token)
        writeKeychain(account: "factory_refresh_token", value: refresh)
    }

    func clear() {
        token = nil
        refreshToken = nil
        factoryId = nil
        deleteKeychain(account: "factory_jwt")
        deleteKeychain(account: "factory_refresh_token")
        deleteKeychain(account: "factory_id")
    }

    // MARK: - Keychain Helpers

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
