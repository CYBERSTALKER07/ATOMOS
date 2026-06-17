//
//  AuthService.swift
//  driverappios
//

import Foundation
import Security

// MARK: - Keychain Helper

private enum AuthNamespace {
    static let primaryService = "com.pegasusx.driver"
    static let primaryPrefix = "com.pegasusx.driver"
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
    var homeNodeType: String?
    var homeNodeId: String?
    var driverMode: String?
    var factoryId: String?
    var factoryName: String?
    var factoryLat: Double = 0
    var factoryLng: Double = 0

    var isAuthenticated: Bool { token != nil }

    var isFactoryScopedDriver: Bool {
        let node = homeNodeType?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        let mode = driverMode?.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        return node == "FACTORY" || mode == "FACTORY"
    }

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
    private let homeNodeTypeKey = "\(AuthNamespace.primaryPrefix).homeNodeType"
    private let homeNodeIdKey = "\(AuthNamespace.primaryPrefix).homeNodeId"
    private let driverModeKey = "\(AuthNamespace.primaryPrefix).driverMode"
    private let factoryIdKey = "\(AuthNamespace.primaryPrefix).factoryId"
    private let factoryNameKey = "\(AuthNamespace.primaryPrefix).factoryName"
    private let factoryLatKey = "\(AuthNamespace.primaryPrefix).factoryLat"
    private let factoryLngKey = "\(AuthNamespace.primaryPrefix).factoryLng"

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
        homeNodeType = loadString(key: homeNodeTypeKey)
        homeNodeId = loadString(key: homeNodeIdKey)
        driverMode = loadString(key: driverModeKey)
        factoryId = loadString(key: factoryIdKey)
        factoryName = loadString(key: factoryNameKey)
        factoryLat = loadDouble(key: factoryLatKey)
        factoryLng = loadDouble(key: factoryLngKey)
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
        homeNodeType = response.homeNodeType
        homeNodeId = response.homeNodeId
        driverMode = response.driverMode
        factoryId = response.factoryId
        factoryName = response.factoryName
        factoryLat = response.factoryLat ?? 0
        factoryLng = response.factoryLng ?? 0

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
        saveString(homeNodeType ?? "", key: homeNodeTypeKey)
        saveString(homeNodeId ?? "", key: homeNodeIdKey)
        saveString(driverMode ?? "", key: driverModeKey)
        saveString(factoryId ?? "", key: factoryIdKey)
        saveString(factoryName ?? "", key: factoryNameKey)
        saveDouble(factoryLat, key: factoryLatKey)
        saveDouble(factoryLng, key: factoryLngKey)
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
        homeNodeType = nil
        homeNodeId = nil
        driverMode = nil
        factoryId = nil
        factoryName = nil
        factoryLat = 0
        factoryLng = 0

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
        KeychainHelper.delete(forKey: homeNodeTypeKey)
        KeychainHelper.delete(forKey: homeNodeIdKey)
        KeychainHelper.delete(forKey: driverModeKey)
        KeychainHelper.delete(forKey: factoryIdKey)
        KeychainHelper.delete(forKey: factoryNameKey)
        KeychainHelper.delete(forKey: factoryLatKey)
        KeychainHelper.delete(forKey: factoryLngKey)
        DriverSocketState.shared.clearOutdatedNotice()
    }
}

@Observable
@MainActor
final class DriverSocketState {
    struct OutdatedNotice {
        let message: String
        let blockedEventType: String?
        let requiredSchemaVersion: Int?
        let clientSchemaVersion: Int?
    }

    struct DriverEvent {
        let type: String
        let orderId: String?
        let response: String?
        let bypassToken: String?
        let attemptId: String?
        let status: String?
        let state: String?
    }

    private struct DriverEnvelope: Decodable {
        let type: String
        let order_id: String?
        let response: String?
        let bypass_token: String?
        let attempt_id: String?
        let command_id: String?
        let trace_id: String?
        let required_schema_version: Int?
        let client_schema_version: Int?
        let blocked_event_type: String?
        let message: String?
        let status: String?
        let state: String?
    }

    static let shared = DriverSocketState()

    enum SocketConnectionState: Equatable {
        case connected
        case reconnecting
        case disconnected
    }

    var outdatedNotice: OutdatedNotice?
    var lastEvent: DriverEvent?
    var eventSequence = 0
    var reconnectEpoch = 0
    var connectionState: SocketConnectionState = .disconnected

    private let wsSchemaVersion = 2
    private var webSocketTask: URLSessionWebSocketTask?
    private var reconnectWorkItem: DispatchWorkItem?
    private var shouldReconnect = false
    private var reconnectAttempt = 0
    private var hasConnectedOnce = false
    private var currentBaseURL: String?
    private var currentToken: String?

    private init() {}

    func startMonitoring(baseURL: String, token: String) {
        if webSocketTask != nil, currentBaseURL == baseURL, currentToken == token {
            return
        }

        stopMonitoring()
        shouldReconnect = true
        currentBaseURL = baseURL
        currentToken = token
        reconnectAttempt = 0
        hasConnectedOnce = false
        establishConnection()
    }

    func stopMonitoring() {
        shouldReconnect = false
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        webSocketTask = nil
        connectionState = .disconnected
    }

    func publishOutdatedEvent(
        message: String?,
        blockedEventType: String?,
        requiredSchemaVersion: Int?,
        clientSchemaVersion: Int?
    ) {
        let blocked = blockedEventType ?? "this operation"
        let required = requiredSchemaVersion.map(String.init) ?? "latest"
        let fallback = "A critical update is required for \(blocked) (required schema: \(required))."

        outdatedNotice = OutdatedNotice(
            message: message ?? fallback,
            blockedEventType: blockedEventType,
            requiredSchemaVersion: requiredSchemaVersion,
            clientSchemaVersion: clientSchemaVersion
        )
    }

    func clearOutdatedNotice() {
        outdatedNotice = nil
    }

    private func establishConnection() {
        guard shouldReconnect,
              let baseURL = currentBaseURL,
              let token = currentToken
        else {
            return
        }

        let wsBase = baseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")

        guard let url = URL(string: "\(wsBase)/v1/ws?sv=\(wsSchemaVersion)") else {
            return
        }

        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let task = URLSession.shared.webSocketTask(with: request)
        webSocketTask = task
        task.resume()
        reconnectAttempt = 0
        connectionState = .connected
        if hasConnectedOnce {
            reconnectEpoch += 1
            Task {
                await FleetServiceLive.shared.flushOfflineQueue()
            }
        }
        hasConnectedOnce = true
        listenForMessages()
    }

    private func listenForMessages() {
        webSocketTask?.receive { [weak self] result in
            Task { @MainActor in
                guard let self else { return }

                switch result {
                case .success(let message):
                    self.handleIncoming(message)
                    self.listenForMessages()
                case .failure:
                    self.connectionState = .reconnecting
                    self.scheduleReconnect()
                }
            }
        }
    }

    private func handleIncoming(_ message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case .string(let text):
            guard let payloadData = text.data(using: .utf8) else { return }
            data = payloadData
        case .data(let payloadData):
            data = payloadData
        @unknown default:
            return
        }

        guard let envelope = try? JSONDecoder().decode(DriverEnvelope.self, from: data) else {
            return
        }

        if envelope.type == "SYSTEM_APP_OUTDATED" {
            publishOutdatedEvent(
                message: envelope.message,
                blockedEventType: envelope.blocked_event_type,
                requiredSchemaVersion: envelope.required_schema_version,
                clientSchemaVersion: envelope.client_schema_version
            )
            stopMonitoring()
            return
        }

        if let commandId = envelope.command_id {
            Task {
                try? await APIClient.shared.ackWebSocketCommand(
                    commandId: commandId,
                    traceId: envelope.trace_id,
                    eventType: envelope.type
                )
            }
        }

        publishDriverEvent(envelope)
    }

    private func publishDriverEvent(_ envelope: DriverEnvelope) {
        lastEvent = DriverEvent(
            type: envelope.type,
            orderId: envelope.order_id,
            response: envelope.response,
            bypassToken: envelope.bypass_token,
            attemptId: envelope.attempt_id,
            status: envelope.status,
            state: envelope.state
        )
        eventSequence += 1
    }

    private func scheduleReconnect() {
        guard shouldReconnect, outdatedNotice == nil else { return }
        connectionState = .reconnecting
        reconnectWorkItem?.cancel()
        reconnectAttempt += 1
        let delay = reconnectDelaySeconds(attempt: reconnectAttempt - 1, base: 3, maxDelay: 60)
        let work = DispatchWorkItem { [weak self] in
            Task { @MainActor in
                self?.establishConnection()
            }
        }
        reconnectWorkItem = work
        DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: work)
    }
}

private func reconnectDelaySeconds(attempt: Int, base: TimeInterval, maxDelay: TimeInterval, retryAfter: TimeInterval = 0) -> TimeInterval {
    let capped = min(max(attempt, 0), 10)
    let exp = min(base * pow(2.0, Double(capped)), maxDelay)
    let jittered = exp + Double.random(in: 0...(exp / 2))
    return min(Swift.max(jittered, retryAfter), maxDelay)
}

// MARK: - Auth Models

struct LoginRequest: Encodable {
    var phone: String = ""
    var pin: String = ""
    var idToken: String = ""

    enum CodingKeys: String, CodingKey {
        case phone, pin
        case idToken = "id_token"
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        if !phone.isEmpty { try container.encode(phone, forKey: .phone) }
        if !pin.isEmpty { try container.encode(pin, forKey: .pin) }
        if !idToken.isEmpty { try container.encode(idToken, forKey: .idToken) }
    }
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
    let homeNodeType: String?
    let homeNodeId: String?
    let driverMode: String?
    let factoryId: String?
    let factoryName: String?
    let factoryLat: Double?
    let factoryLng: Double?

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
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
        case driverMode = "driver_mode"
        case factoryId = "factory_id"
        case factoryName = "factory_name"
        case factoryLat = "factory_lat"
        case factoryLng = "factory_lng"
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
    let homeNodeType: String?
    let homeNodeId: String?
    let driverMode: String?
    let factoryId: String?
    let factoryName: String?
    let factoryLat: Double?
    let factoryLng: Double?

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
        case homeNodeType = "home_node_type"
        case homeNodeId = "home_node_id"
        case driverMode = "driver_mode"
        case factoryId = "factory_id"
        case factoryName = "factory_name"
        case factoryLat = "factory_lat"
        case factoryLng = "factory_lng"
    }
}
