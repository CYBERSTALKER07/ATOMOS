//
//  OfflineVerifierViewModel.swift
//  driverappios
//

import SwiftUI
import SwiftData

// MARK: - Verification State Machine

enum VerificationState: Equatable {
    case idle
    case syncing
    case ready(RouteManifest)
    case scanning
    case verified(String)        // order ID
    case fraud(String)           // reason
    case error(String)           // reason

    // Equatable conformance for RouteManifest
    static func == (lhs: VerificationState, rhs: VerificationState) -> Bool {
        switch (lhs, rhs) {
        case (.idle, .idle), (.syncing, .syncing), (.scanning, .scanning):
            return true
        case (.verified(let a), .verified(let b)):
            return a == b
        case (.fraud(let a), .fraud(let b)):
            return a == b
        case (.error(let a), .error(let b)):
            return a == b
        case (.ready(let a), .ready(let b)):
            return a.driver_id == b.driver_id && a.date == b.date
        default:
            return false
        }
    }
}

@Observable
@MainActor
final class OfflineVerifierViewModel {

    // MARK: - State

    var state: VerificationState = .idle
    private var manifest: RouteManifest?
    private var scanLocked = false

    private let manifestService: ManifestServiceProtocol
    private let syncService: SyncServiceProtocol
    var store: OfflineDeliveryStore?

    // MARK: - Init

    convenience init() {
        self.init(manifestService: ManifestServiceLive.shared, syncService: SyncServiceLive.shared)
    }

    init(manifestService: ManifestServiceProtocol, syncService: SyncServiceProtocol = SyncServiceStub.shared) {
        self.manifestService = manifestService
        self.syncService = syncService
    }

    // MARK: - Status label for display

    var statusLabel: String {
        switch state {
        case .idle:              return "IDLE"
        case .syncing:           return "SYNCING"
        case .ready:             return "READY"
        case .scanning:          return "SCANNING"
        case .verified:          return "VERIFIED"
        case .fraud:             return "FRAUD"
        case .error:             return "ERROR"
        }
    }

    var statusColor: Color {
        switch state {
        case .idle:              return .secondary
        case .syncing:           return .orange
        case .ready:             return .blue
        case .scanning:          return .purple
        case .verified:          return .green
        case .fraud:             return .red
        case .error:             return .red
        }
    }

    // MARK: - Sync Manifest

    func syncManifest() async {
        state = .syncing
        do {
            let token = TokenStore.shared.token ?? ""
            let m = try await manifestService.downloadManifest(bearerToken: token)
            manifest = m
            state = .ready(m)

            // Also attempt to sync any pending offline deliveries
            await syncPendingDeliveries()
        } catch {
            state = .error("Failed to download manifest: \(error.localizedDescription)")
        }
    }

    func activateScanner() {
        state = .scanning
    }

    func cancelScanner() {
        if let m = manifest {
            state = .ready(m)
        } else {
            state = .idle
        }
    }

    func resetTerminal() {
        manifest = nil
        state = .idle
    }

    func nextDelivery() {
        if let m = manifest {
            state = .ready(m)
        } else {
            state = .idle
        }
    }

    // MARK: - Handle Barcode Scan

    func handleBarcodeScan(_ stringValue: String) {
        guard !scanLocked else { return }
        scanLocked = true

        // Unlock after 2 seconds
        Task {
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            scanLocked = false
        }

        guard let manifest else {
            state = .error("No manifest loaded.")
            return
        }

        guard manifest.isValid else {
            state = .error("Manifest expired. Re-sync required.")
            return
        }

        // Parse QR
        guard let data = stringValue.data(using: .utf8),
              let payload = try? JSONDecoder().decode(QRPayload.self, from: data) else {
            state = .error("Invalid QR code format.")
            Haptics.error()
            return
        }

        // Lookup order in manifest
        guard let expectedHash = manifest.hashes[payload.order_id] else {
            state = .fraud("ORDER NOT FOUND IN ROUTE MANIFEST")
            Haptics.error()
            return
        }

        // SHA-256 the raw token and compare
        let computedHash = sha256Hex(payload.token)
        if computedHash == expectedHash {
            state = .verified(payload.order_id)
            Haptics.success()

            // Buffer to SwiftData offline queue
            store?.enqueue(
                orderId: payload.order_id,
                signature: computedHash,
                status: "DELIVERED"
            )
        } else {
            state = .fraud("CRYPTOGRAPHIC MISMATCH")
            Haptics.error()
        }
    }

    // MARK: - Sync Offline Deliveries to Backend

    func syncPendingDeliveries() async {
        guard let store else { return }
        let pending = store.fetchPending()
        guard !pending.isEmpty else { return }

        let dtos = pending.map {
            SyncDeliveryDTO(
                orderId: $0.orderId,
                signature: $0.signature,
                timestamp: $0.timestamp,
                status: $0.status
            )
        }

        let driverId = TokenStore.shared.userId ?? "UNKNOWN"
        let token = TokenStore.shared.token ?? ""

        do {
            let result = try await syncService.uploadBatch(
                driverId: driverId,
                deliveries: dtos,
                bearerToken: token
            )
            // Remove synced deliveries from local store
            store.deleteSynced(orderIds: result.processed)
        } catch {
            // Sync failed — deliveries remain in queue for next attempt
        }
    }
}
