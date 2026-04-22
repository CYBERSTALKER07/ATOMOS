//
//  AuthService.swift
//  driverappios
//

import Foundation
import Security

// MARK: - Keychain Helper

private enum KeychainHelper {
    static func save(_ value: String, forKey key: String) {
        guard let data = value.data(using: .utf8) else { return }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: "com.thelab.driver",
        ]
        SecItemDelete(query as CFDictionary)
        var add = query
        add[kSecValueData as String] = data
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        SecItemAdd(add as CFDictionary, nil)
    }

    static func load(forKey key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: "com.thelab.driver",
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess, let data = result as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func saveDouble(_ value: Double, forKey key: String) {
        save(String(value), forKey: key)
    }

    static func loadDouble(forKey key: String) -> Double {
        guard let str = load(forKey: key) else { return 0 }
        return Double(str) ?? 0
    }

    static func delete(forKey key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecAttrService as String: "com.thelab.driver",
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

    private let tokenKey = "com.thelab.driver.token"
    private let userKey  = "com.thelab.driver.userId"
    private let nameKey  = "com.thelab.driver.driverName"
    private let vehicleKey = "com.thelab.driver.vehicleType"
    private let plateKey = "com.thelab.driver.licensePlate"
    private let vehicleIdKey = "com.thelab.driver.vehicleId"
    private let vehicleClassKey = "com.thelab.driver.vehicleClass"
    private let maxVolumeVUKey = "com.thelab.driver.maxVolumeVU"
    private let warehouseIdKey = "com.thelab.driver.warehouseId"
    private let warehouseNameKey = "com.thelab.driver.warehouseName"
    private let warehouseLatKey = "com.thelab.driver.warehouseLat"
    private let warehouseLngKey = "com.thelab.driver.warehouseLng"

    private init() {
        // Restore from Keychain (migrate from UserDefaults on first run)
        token = KeychainHelper.load(forKey: tokenKey) ?? migrateString(tokenKey)
        userId = KeychainHelper.load(forKey: userKey) ?? migrateString(userKey)
        driverName = KeychainHelper.load(forKey: nameKey) ?? migrateString(nameKey)
        vehicleType = KeychainHelper.load(forKey: vehicleKey) ?? migrateString(vehicleKey)
        licensePlate = KeychainHelper.load(forKey: plateKey) ?? migrateString(plateKey)
        vehicleId = KeychainHelper.load(forKey: vehicleIdKey) ?? migrateString(vehicleIdKey)
        vehicleClass = KeychainHelper.load(forKey: vehicleClassKey) ?? migrateString(vehicleClassKey)
        maxVolumeVU = KeychainHelper.loadDouble(forKey: maxVolumeVUKey)
        if maxVolumeVU == 0 {
            let legacy = UserDefaults.standard.double(forKey: maxVolumeVUKey)
            if legacy > 0 {
                maxVolumeVU = legacy
                KeychainHelper.saveDouble(legacy, forKey: maxVolumeVUKey)
                UserDefaults.standard.removeObject(forKey: maxVolumeVUKey)
            }
        }
        warehouseId = KeychainHelper.load(forKey: warehouseIdKey)
        warehouseName = KeychainHelper.load(forKey: warehouseNameKey)
        warehouseLat = KeychainHelper.loadDouble(forKey: warehouseLatKey)
        warehouseLng = KeychainHelper.loadDouble(forKey: warehouseLngKey)
    }

    /// One-shot migration from UserDefaults → Keychain
    private func migrateString(_ key: String) -> String? {
        guard let value = UserDefaults.standard.string(forKey: key) else { return nil }
        KeychainHelper.save(value, forKey: key)
        UserDefaults.standard.removeObject(forKey: key)
        return value
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

        KeychainHelper.save(response.token, forKey: tokenKey)
        KeychainHelper.save(response.userId, forKey: userKey)
        KeychainHelper.save(response.name, forKey: nameKey)
        persistVehicleInfo()

        // Exchange Firebase custom token for ID token session (graceful degradation)
        if let fbToken = response.firebaseToken, !fbToken.isEmpty {
            FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
        }
    }

    /// Persist current vehicle fields to Keychain. Called after profile polling updates.
    func persistVehicleInfo() {
        KeychainHelper.save(vehicleType ?? "", forKey: vehicleKey)
        KeychainHelper.save(licensePlate ?? "", forKey: plateKey)
        KeychainHelper.save(vehicleId ?? "", forKey: vehicleIdKey)
        KeychainHelper.save(vehicleClass ?? "", forKey: vehicleClassKey)
        KeychainHelper.saveDouble(maxVolumeVU, forKey: maxVolumeVUKey)
        KeychainHelper.save(warehouseId ?? "", forKey: warehouseIdKey)
        KeychainHelper.save(warehouseName ?? "", forKey: warehouseNameKey)
        KeychainHelper.saveDouble(warehouseLat, forKey: warehouseLatKey)
        KeychainHelper.saveDouble(warehouseLng, forKey: warehouseLngKey)
    }

    /// Update only the token (used after silent refresh).
    func updateToken(_ newToken: String) {
        token = newToken
        KeychainHelper.save(newToken, forKey: tokenKey)
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
