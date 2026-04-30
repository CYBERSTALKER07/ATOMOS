//
//  AuthService.swift
//  driverappios
//

import Foundation
import Security

// MARK: - Keychain Helper

private enum AuthNamespace {
    static let primaryService = "com.pegasus.driver"
    static let primaryPrefix = "com.pegasus.driver"
}

private enum KeychainHelper {
    static func save(_ value: String, forKey key: String) {
        guard let data = value.data(using: .utf8) else { return }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: AuthNamespace.primaryService,
        ]
        SecItemDelete(query as CFDictionary)
        var add = query
        add[kSecValueData as String] = data
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        SecItemAdd(add as CFDictionary, nil)
    }

    static func load(forKey key: String) -> String? {
        loadFromService(AuthNamespace.primaryService, forKey: key)
    }

    static func saveDouble(_ value: Double, forKey key: String) {
        save(String(value), forKey: key)
    }

    static func loadDouble(forKey key: String) -> Double {
        guard let str = load(forKey: key) else { return 0 }
        return Double(str) ?? 0
    }

    static func delete(forKey key: String) {
        deleteFromService(AuthNamespace.primaryService, forKey: key)
    }

    private static func loadFromService(_ service: String, forKey key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: service,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private static func deleteFromService(_ service: String, forKey key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: service,
        ]
        SecItemDelete(query as CFDictionary)
    }
}

// MARK: - Token Store (Keychain-backed singleton)

@Observable
@MainActor
final class TokenStore {
    static let shared = TokenStore()

    var token: String?
    var userId: String?
    var driverName: String?
    var vehicleType: String?
    var licensePlate: String?
    var supplierId: String?
    var vehicleId: String?
    var vehicleClass: String?
    var maxVolumeVU: Double = 0
    var warehouseId: String?
    var warehouseName: String?
    var warehouseLat: Double = 0
    var warehouseLng: Double = 0

    var isAuthenticated: Bool { token != nil }

    private let tokenKey = "\(AuthNamespace.primaryPrefix).token"
    private let userKey  = "\(AuthNamespace.primaryPrefix).userId"
    private let nameKey  = "\(AuthNamespace.primaryPrefix).driverName"
    private let vehicleKey = "\(AuthNamespace.primaryPrefix).vehicleType"
    private let plateKey = "\(AuthNamespace.primaryPrefix).licensePlate"
    private let vehicleIdKey = "\(AuthNamespace.primaryPrefix).vehicleId"
    private let vehicleClassKey = "\(AuthNamespace.primaryPrefix).vehicleClass"
    private let maxVolumeVUKey = "\(AuthNamespace.primaryPrefix).maxVolumeVU"
    private let warehouseIdKey = "\(AuthNamespace.primaryPrefix).warehouseId"
    private let warehouseNameKey = "\(AuthNamespace.primaryPrefix).warehouseName"
    private let warehouseLatKey = "\(AuthNamespace.primaryPrefix).warehouseLat"
    private let warehouseLngKey = "\(AuthNamespace.primaryPrefix).warehouseLng"

    private init() {
        token = loadString(key: tokenKey)
            ?? migrateString(tokenKey)
        userId = loadString(key: userKey)
            ?? migrateString(userKey)
        driverName = loadString(key: nameKey)
            ?? migrateString(nameKey)
        vehicleType = loadString(key: vehicleKey)
            ?? migrateString(vehicleKey)
        licensePlate = loadString(key: plateKey)
            ?? migrateString(plateKey)
        vehicleId = loadString(key: vehicleIdKey)
            ?? migrateString(vehicleIdKey)
        vehicleClass = loadString(key: vehicleClassKey)
            ?? migrateString(vehicleClassKey)
        maxVolumeVU = loadDouble(key: maxVolumeVUKey)
        if maxVolumeVU == 0 {
            let persisted = UserDefaults.standard.double(forKey: maxVolumeVUKey)
            if persisted > 0 {
                maxVolumeVU = persisted
                saveDouble(persisted, key: maxVolumeVUKey)
                UserDefaults.standard.removeObject(forKey: maxVolumeVUKey)
            }
        }
        warehouseId = loadString(key: warehouseIdKey)
        warehouseName = loadString(key: warehouseNameKey)
        warehouseLat = loadDouble(key: warehouseLatKey)
        warehouseLng = loadDouble(key: warehouseLngKey)
    }

    /// One-shot migration from UserDefaults → Keychain
    private func migrateString(_ key: String) -> String? {
        guard let value = UserDefaults.standard.string(forKey: key) else { return nil }
        KeychainHelper.save(value, forKey: key)
        UserDefaults.standard.removeObject(forKey: key)
        return value
    }

    private func loadString(key: String) -> String? {
        KeychainHelper.load(forKey: key)
    }

    private func loadDouble(key: String) -> Double {
        guard let value = loadString(key: key) else {
            return 0
        }
        return Double(value) ?? 0
    }

    private func saveString(_ value: String, key: String) {
        KeychainHelper.save(value, forKey: key)
    }

    private func saveDouble(_ value: Double, key: String) {
        KeychainHelper.saveDouble(value, forKey: key)
    }

    func save(response: AuthResponse) {
        token = response.token
        userId = response.userId
        driverName = response.name
        vehicleType = response.vehicleType
        licensePlate = response.licensePlate
        supplierId = response.supplierId
        vehicleId = response.vehicleId
        vehicleClass = response.vehicleClass
        maxVolumeVU = response.maxVolumeVU
        warehouseId = response.warehouseId
        warehouseName = response.warehouseName
        warehouseLat = response.warehouseLat ?? 0
        warehouseLng = response.warehouseLng ?? 0

        saveString(response.token, key: tokenKey)
        saveString(response.userId, key: userKey)
        saveString(response.name, key: nameKey)
        persistVehicleInfo()

        // Exchange Firebase custom token for ID token session (graceful degradation)
        if let fbToken = response.firebaseToken, !fbToken.isEmpty {
            FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
        }
    }

    /// Persist current vehicle fields to Keychain. Called after profile polling updates.
    func persistVehicleInfo() {
        saveString(vehicleType ?? "", key: vehicleKey)
        saveString(licensePlate ?? "", key: plateKey)
        saveString(vehicleId ?? "", key: vehicleIdKey)
        saveString(vehicleClass ?? "", key: vehicleClassKey)
        saveDouble(maxVolumeVU, key: maxVolumeVUKey)
        saveString(warehouseId ?? "", key: warehouseIdKey)
        saveString(warehouseName ?? "", key: warehouseNameKey)
        saveDouble(warehouseLat, key: warehouseLatKey)
        saveDouble(warehouseLng, key: warehouseLngKey)
    }

    /// Update only the token (used after silent refresh).
    func updateToken(_ newToken: String) {
        token = newToken
        saveString(newToken, key: tokenKey)
    }

    func logout() {
        token = nil
        userId = nil
        driverName = nil
        vehicleType = nil
        licensePlate = nil
        supplierId = nil
        vehicleId = nil
        vehicleClass = nil
        maxVolumeVU = 0
        warehouseId = nil
        warehouseName = nil
        warehouseLat = 0
        warehouseLng = 0

        KeychainHelper.delete(forKey: tokenKey)
        KeychainHelper.delete(forKey: userKey)
        KeychainHelper.delete(forKey: nameKey)
        KeychainHelper.delete(forKey: vehicleKey)
        KeychainHelper.delete(forKey: plateKey)
        KeychainHelper.delete(forKey: vehicleIdKey)
        KeychainHelper.delete(forKey: vehicleClassKey)
        KeychainHelper.delete(forKey: maxVolumeVUKey)
        KeychainHelper.delete(forKey: warehouseIdKey)
        KeychainHelper.delete(forKey: warehouseNameKey)
        KeychainHelper.delete(forKey: warehouseLatKey)
        KeychainHelper.delete(forKey: warehouseLngKey)
    }
}

// MARK: - Auth Models

struct LoginRequest: Codable {
    let phone: String
    let pin: String
}

struct AuthResponse: Codable {
    let token: String
    let userId: String
    let driverId: String
    let name: String
    let vehicleType: String
    let licensePlate: String
    let supplierId: String
    let vehicleId: String
    let vehicleClass: String
    let maxVolumeVU: Double
    let firebaseToken: String?
    let warehouseId: String?
    let warehouseName: String?
    let warehouseLat: Double?
    let warehouseLng: Double?

    enum CodingKeys: String, CodingKey {
        case token
        case userId = "user_id"
        case driverId = "driver_id"
        case name
        case vehicleType = "vehicle_type"
        case licensePlate = "license_plate"
        case supplierId = "supplier_id"
        case vehicleId = "vehicle_id"
        case vehicleClass = "vehicle_class"
        case maxVolumeVU = "max_volume_vu"
        case firebaseToken = "firebase_token"
        case warehouseId = "warehouse_id"
        case warehouseName = "warehouse_name"
        case warehouseLat = "warehouse_lat"
        case warehouseLng = "warehouse_lng"
    }
}

// MARK: - Driver Profile (polling response)

struct DriverProfileResponse: Codable {
    let driverId: String
    let name: String
    let phone: String
    let driverType: String
    let vehicleType: String
    let licensePlate: String
    let isActive: Bool
    let supplierId: String
    let vehicleId: String
    let vehicleClass: String
    let maxVolumeVU: Double
    let warehouseId: String?
    let warehouseName: String?
    let warehouseLat: Double?
    let warehouseLng: Double?

    enum CodingKeys: String, CodingKey {
        case driverId = "driver_id"
        case name, phone
        case driverType = "driver_type"
        case vehicleType = "vehicle_type"
        case licensePlate = "license_plate"
        case isActive = "is_active"
        case supplierId = "supplier_id"
        case vehicleId = "vehicle_id"
        case vehicleClass = "vehicle_class"
        case maxVolumeVU = "max_volume_vu"
        case warehouseId = "warehouse_id"
        case warehouseName = "warehouse_name"
        case warehouseLat = "warehouse_lat"
        case warehouseLng = "warehouse_lng"
    }
}
