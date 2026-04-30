//
//  TokenStore.swift
//  payload-app-ios
//
//  Keychain-backed session store. Mirrors the SecureStore pattern used by
//  the Android sibling and the TokenStore pattern used by driverappios.
//

import Foundation
import Observation
import Security

@Observable
final class TokenStore {
    static let shared = TokenStore()

    private let service = "com.pegasus.payload"
    private let legacyService = "com.thelab.payload"

    private(set) var token: String?
    private(set) var name: String?
    private(set) var supplierId: String?
    private(set) var warehouseId: String?
    private(set) var warehouseName: String?
    private(set) var firebaseToken: String?

    var isAuthenticated: Bool { token != nil }

    private init() {
        token = read(.token)
        name = read(.name)
        supplierId = read(.supplierId)
        warehouseId = read(.warehouseId)
        warehouseName = read(.warehouseName)
        firebaseToken = read(.firebaseToken)
    }

    func saveSession(from resp: LoginResponse) {
        write(.token, value: resp.token)
        write(.name, value: resp.name)
        write(.supplierId, value: resp.supplierId)
        write(.warehouseId, value: resp.warehouseId)
        write(.warehouseName, value: resp.warehouseName)
        if let fb = resp.firebaseToken { write(.firebaseToken, value: fb) }

        token = resp.token
        name = resp.name
        supplierId = resp.supplierId
        warehouseId = resp.warehouseId
        warehouseName = resp.warehouseName
        firebaseToken = resp.firebaseToken
    }

    @MainActor
    func logout() {
        for k in Key.allCases { delete(k) }
        token = nil
        name = nil
        supplierId = nil
        warehouseId = nil
        warehouseName = nil
        firebaseToken = nil
    }

    // MARK: - Keychain plumbing

    private enum Key: String, CaseIterable {
        case token         = "payloader_token"
        case name          = "payloader_name"
        case supplierId    = "payloader_supplier_id"
        case warehouseId   = "payloader_warehouse_id"
        case warehouseName = "payloader_warehouse_name"
        case firebaseToken = "payloader_firebase_token"
    }

    private func read(_ key: Key) -> String? {
        if let value = readFromService(service, account: key.rawValue) {
            return value
        }

        guard let legacy = readFromService(legacyService, account: key.rawValue) else { return nil }
        writeToService(service, account: key.rawValue, value: legacy)
        deleteFromService(legacyService, account: key.rawValue)
        return legacy
    }

    private func readFromService(_ serviceName: String, account: String) -> String? {
        let q: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccount: account,
            kSecAttrService: serviceName,
            kSecReturnData: true,
            kSecMatchLimit: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        guard SecItemCopyMatching(q as CFDictionary, &item) == errSecSuccess,
              let data = item as? Data,
              let str = String(data: data, encoding: .utf8) else { return nil }
        return str
    }

    private func write(_ key: Key, value: String) {
        writeToService(service, account: key.rawValue, value: value)
        deleteFromService(legacyService, account: key.rawValue)
    }

    private func writeToService(_ serviceName: String, account: String, value: String) {
        deleteFromService(serviceName, account: account)
        let q: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccount: account,
            kSecAttrService: serviceName,
            kSecValueData: Data(value.utf8),
            kSecAttrAccessible: kSecAttrAccessibleAfterFirstUnlock,
        ]
        SecItemAdd(q as CFDictionary, nil)
    }

    private func delete(_ key: Key) {
        deleteFromService(service, account: key.rawValue)
        deleteFromService(legacyService, account: key.rawValue)
    }

    private func deleteFromService(_ serviceName: String, account: String) {
        let q: [CFString: Any] = [
            kSecClass: kSecClassGenericPassword,
            kSecAttrAccount: account,
            kSecAttrService: serviceName,
        ]
        SecItemDelete(q as CFDictionary)
    }
}
