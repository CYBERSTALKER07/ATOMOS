//
//  AuthService.swift
//  driverappios
//

import Foundation
import Security

// MARK: - Keychain Helper

private enum AuthNamespace {
    static let primaryService = "com.pegasus.driver"
    static let legacyService = "com.thelab.driver"
    static let primaryPrefix = "com.pegasus.driver"
    static let legacyPrefix = "com.thelab.driver"
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
        if let value = loadFromService(AuthNamespace.primaryService, forKey: key) {
            return value
        }
        return loadFromService(AuthNamespace.legacyService, forKey: key)
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
        deleteFromService(AuthNamespace.legacyService, forKey: key)
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

    private let legacyTokenKey = "\(AuthNamespace.legacyPrefix).token"
    private let legacyUserKey  = "\(AuthNamespace.legacyPrefix).userId"
    private let legacyNameKey  = "\(AuthNamespace.legacyPrefix).driverName"
    private let legacyVehicleKey = "\(AuthNamespace.legacyPrefix).vehicleType"
    private let legacyPlateKey = "\(AuthNamespace.legacyPrefix).licensePlate"
    private let legacyVehicleIdKey = "\(AuthNamespace.legacyPrefix).vehicleId"
    private let legacyVehicleClassKey = "\(AuthNamespace.legacyPrefix).vehicleClass"
    private let legacyMaxVolumeVUKey = "\(AuthNamespace.legacyPrefix).maxVolumeVU"
    private let legacyWarehouseIdKey = "\(AuthNamespace.legacyPrefix).warehouseId"
    private let legacyWarehouseNameKey = "\(AuthNamespace.legacyPrefix).warehouseName"
    private let legacyWarehouseLatKey = "\(AuthNamespace.legacyPrefix).warehouseLat"
    private let legacyWarehouseLngKey = "\(AuthNamespace.legacyPrefix).warehouseLng"

    private init() {
        token = loadString(primaryKey: tokenKey, legacyKey: legacyTokenKey)
            ?? migrateString(legacyTokenKey)
            ?? migrateString(tokenKey)
        userId = loadString(primaryKey: userKey, legacyKey: legacyUserKey)
            ?? migrateString(legacyUserKey)
            ?? migrateString(userKey)
        driverName = loadString(primaryKey: nameKey, legacyKey: legacyNameKey)
            ?? migrateString(legacyNameKey)
            ?? migrateString(nameKey)
        vehicleType = loadString(primaryKey: vehicleKey, legacyKey: legacyVehicleKey)
            ?? migrateString(legacyVehicleKey)
            ?? migrateString(vehicleKey)
        licensePlate = loadString(primaryKey: plateKey, legacyKey: legacyPlateKey)
            ?? migrateString(legacyPlateKey)
            ?? migrateString(plateKey)
        vehicleId = loadString(primaryKey: vehicleIdKey, legacyKey: legacyVehicleIdKey)
            ?? migrateString(legacyVehicleIdKey)
            ?? migrateString(vehicleIdKey)
        vehicleClass = loadString(primaryKey: vehicleClassKey, legacyKey: legacyVehicleClassKey)
            ?? migrateString(legacyVehicleClassKey)
            ?? migrateString(vehicleClassKey)
        maxVolumeVU = loadDouble(primaryKey: maxVolumeVUKey, legacyKey: legacyMaxVolumeVUKey)
        if maxVolumeVU == 0 {
            let legacyPrimary = UserDefaults.standard.double(forKey: maxVolumeVUKey)
            let legacyOld = UserDefaults.standard.double(forKey: legacyMaxVolumeVUKey)
            let persisted = legacyPrimary > 0 ? legacyPrimary : legacyOld
            if persisted > 0 {
                maxVolumeVU = persisted
                saveDouble(persisted, primaryKey: maxVolumeVUKey, legacyKey: legacyMaxVolumeVUKey)
                UserDefaults.standard.removeObject(forKey: maxVolumeVUKey)
                UserDefaults.standard.removeObject(forKey: legacyMaxVolumeVUKey)
            }
        }
        warehouseId = loadString(primaryKey: warehouseIdKey, legacyKey: legacyWarehouseIdKey)
        warehouseName = loadString(primaryKey: warehouseNameKey, legacyKey: legacyWarehouseNameKey)
        warehouseLat = loadDouble(primaryKey: warehouseLatKey, legacyKey: legacyWarehouseLatKey)
        warehouseLng = loadDouble(primaryKey: warehouseLngKey, legacyKey: legacyWarehouseLngKey)
    }

    /// One-shot migration from UserDefaults → Keychain
    private func migrateString(_ key: String) -> String? {
        guard let value = UserDefaults.standard.string(forKey: key) else { return nil }
        KeychainHelper.save(value, forKey: key)
        UserDefaults.standard.removeObject(forKey: key)
        return value
    }

    private func loadString(primaryKey: String, legacyKey: String) -> String? {
        if let value = KeychainHelper.load(forKey: primaryKey) {
            return value
        }
        guard let legacy = KeychainHelper.load(forKey: legacyKey) else { return nil }
        saveString(legacy, primaryKey: primaryKey, legacyKey: legacyKey)
        return legacy
    }

    private func loadDouble(primaryKey: String, legacyKey: String) -> Double {
        guard let value = loadString(primaryKey: primaryKey, legacyKey: legacyKey) else {
            return 0
        }
        return Double(value) ?? 0
    }

    private func saveString(_ value: String, primaryKey: String, legacyKey: String) {
        KeychainHelper.save(value, forKey: primaryKey)
        KeychainHelper.delete(forKey: legacyKey)
    }

    private func saveDouble(_ value: Double, primaryKey: String, legacyKey: String) {
        KeychainHelper.saveDouble(value, forKey: primaryKey)
        KeychainHelper.delete(forKey: legacyKey)
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

        saveString(response.token, primaryKey: tokenKey, legacyKey: legacyTokenKey)
        saveString(response.userId, primaryKey: userKey, legacyKey: legacyUserKey)
        saveString(response.name, primaryKey: nameKey, legacyKey: legacyNameKey)
        persistVehicleInfo()

        // Exchange Firebase custom token for ID token session (graceful degradation)
        if let fbToken = response.firebaseToken, !fbToken.isEmpty {
            FirebaseAuthHelper.shared.exchangeCustomToken(fbToken) { _ in }
        }
    }

    /// Persist current vehicle fields to Keychain. Called after profile polling updates.
    func persistVehicleInfo() {
        saveString(vehicleType ?? "", primaryKey: vehicleKey, legacyKey: legacyVehicleKey)
        saveString(licensePlate ?? "", primaryKey: plateKey, legacyKey: legacyPlateKey)
        saveString(vehicleId ?? "", primaryKey: vehicleIdKey, legacyKey: legacyVehicleIdKey)
        saveString(vehicleClass ?? "", primaryKey: vehicleClassKey, legacyKey: legacyVehicleClassKey)
        saveDouble(maxVolumeVU, primaryKey: maxVolumeVUKey, legacyKey: legacyMaxVolumeVUKey)
        saveString(warehouseId ?? "", primaryKey: warehouseIdKey, legacyKey: legacyWarehouseIdKey)
        saveString(warehouseName ?? "", primaryKey: warehouseNameKey, legacyKey: legacyWarehouseNameKey)
        saveDouble(warehouseLat, primaryKey: warehouseLatKey, legacyKey: legacyWarehouseLatKey)
        saveDouble(warehouseLng, primaryKey: warehouseLngKey, legacyKey: legacyWarehouseLngKey)
    }

    /// Update only the token (used after silent refresh).
    func updateToken(_ newToken: String) {
        token = newToken
        saveString(newToken, primaryKey: tokenKey, legacyKey: legacyTokenKey)
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
        KeychainHelper.delete(forKey: legacyTokenKey)
        KeychainHelper.delete(forKey: userKey)
        KeychainHelper.delete(forKey: legacyUserKey)
        KeychainHelper.delete(forKey: nameKey)
        KeychainHelper.delete(forKey: legacyNameKey)
        KeychainHelper.delete(forKey: vehicleKey)
        KeychainHelper.delete(forKey: legacyVehicleKey)
        KeychainHelper.delete(forKey: plateKey)
        KeychainHelper.delete(forKey: legacyPlateKey)
        KeychainHelper.delete(forKey: vehicleIdKey)
        KeychainHelper.delete(forKey: legacyVehicleIdKey)
        KeychainHelper.delete(forKey: vehicleClassKey)
        KeychainHelper.delete(forKey: legacyVehicleClassKey)
        KeychainHelper.delete(forKey: maxVolumeVUKey)
        KeychainHelper.delete(forKey: legacyMaxVolumeVUKey)
        KeychainHelper.delete(forKey: warehouseIdKey)
        KeychainHelper.delete(forKey: legacyWarehouseIdKey)
        KeychainHelper.delete(forKey: warehouseNameKey)
        KeychainHelper.delete(forKey: legacyWarehouseNameKey)
        KeychainHelper.delete(forKey: warehouseLatKey)
        KeychainHelper.delete(forKey: legacyWarehouseLatKey)
        KeychainHelper.delete(forKey: warehouseLngKey)
        KeychainHelper.delete(forKey: legacyWarehouseLngKey)
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
