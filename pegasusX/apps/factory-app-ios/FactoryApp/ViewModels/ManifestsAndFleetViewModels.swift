import Foundation
import Observation

@Observable
@MainActor
final class PayloadOverrideViewModel {
    var manifests: [Manifest] = []
    var loading: Bool = true
    var error: String?

    var moveCandidate: MoveCandidate?
    var selectedTargetManifestId: String?

    var cancelTransferCandidate: CancelTransferCandidate?
    var showReleaseAlert: Bool = false

    var cancelManifestCandidate: Manifest?
    var showCancelManifestAlert: Bool = false

    var isProcessing: Bool = false
    var processingError: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchLoadingManifests: @MainActor () async throws -> [Manifest]
    private let rebalanceAction: @MainActor (String, String, String) async throws -> ManifestRebalanceResponse
    private let cancelTransferAction: @MainActor (String, String) async throws -> ManifestCancelTransferResponse
    private let cancelManifestAction: @MainActor (String) async throws -> ManifestCancelResponse

    init(
        fetchLoadingManifests: @escaping @MainActor () async throws -> [Manifest] = {
            try await FactoryService.loadingManifests().manifests
        },
        rebalanceAction: @escaping @MainActor (String, String, String) async throws -> ManifestRebalanceResponse = {
            try await FactoryService.rebalanceManifestTransfer(sourceManifestId: $0, targetManifestId: $1, transferId: $2)
        },
        cancelTransferAction: @escaping @MainActor (String, String) async throws -> ManifestCancelTransferResponse = {
            try await FactoryService.cancelManifestTransfer(manifestId: $0, transferId: $1)
        },
        cancelManifestAction: @escaping @MainActor (String) async throws -> ManifestCancelResponse = {
            try await FactoryService.cancelManifest(manifestId: $0)
        }
    ) {
        self.fetchLoadingManifests = fetchLoadingManifests
        self.rebalanceAction = rebalanceAction
        self.cancelTransferAction = cancelTransferAction
        self.cancelManifestAction = cancelManifestAction
    }

    func promptMove(manifestId: String, transfer: ManifestTransfer) {
        selectedTargetManifestId = nil
        moveCandidate = MoveCandidate(sourceManifestId: manifestId, transfer: transfer)
    }

    func dismissMove() {
        moveCandidate = nil
        selectedTargetManifestId = nil
    }

    func promptRelease(manifestId: String, transfer: ManifestTransfer) {
        cancelTransferCandidate = CancelTransferCandidate(manifestId: manifestId, transfer: transfer)
        showReleaseAlert = true
    }

    func clearReleaseAlert() {
        showReleaseAlert = false
        cancelTransferCandidate = nil
    }

    func promptCancelManifest(_ manifest: Manifest) {
        cancelManifestCandidate = manifest
        showCancelManifestAlert = true
    }

    func clearCancelManifestAlert() {
        showCancelManifestAlert = false
        cancelManifestCandidate = nil
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            manifests = try await fetchLoadingManifests()
        } catch {
            self.error = error.localizedDescription
        }
        if !silent { loading = false }
    }

    func moveTransfer(_ candidate: MoveCandidate, targetManifestId: String) async {
        isProcessing = true
        processingError = nil
        do {
            _ = try await rebalanceAction(candidate.sourceManifestId, targetManifestId, candidate.transfer.id)
            PegasusHaptics.confirm()
            dismissMove()
            await load(silent: true)
        } catch {
            processingError = error.localizedDescription
            PegasusHaptics.error()
        }
        isProcessing = false
    }

    func confirmReleaseTransfer() async {
        guard let candidate = cancelTransferCandidate else { return }
        isProcessing = true
        processingError = nil
        do {
            _ = try await cancelTransferAction(candidate.manifestId, candidate.transfer.id)
            PegasusHaptics.confirm()
            clearReleaseAlert()
            await load(silent: true)
        } catch {
            processingError = error.localizedDescription
            PegasusHaptics.error()
        }
        isProcessing = false
    }

    func confirmCancelManifest() async {
        guard let manifest = cancelManifestCandidate else { return }
        isProcessing = true
        processingError = nil
        do {
            _ = try await cancelManifestAction(manifest.id)
            PegasusHaptics.confirm()
            clearCancelManifestAlert()
            await load(silent: true)
        } catch {
            processingError = error.localizedDescription
            PegasusHaptics.error()
        }
        isProcessing = false
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                Task { await self.load(silent: true) }
            },
            onReconnect: { [weak self] in
                guard let self else { return }
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class ManifestsViewModel {
    var manifests: [Manifest] = []
    var loading: Bool = true
    var error: String?
    var selectedManifestID: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchManifests: @MainActor () async throws -> [Manifest]

    init(
        fetchManifests: @escaping @MainActor () async throws -> [Manifest] = {
            try await FactoryService.manifests().manifests
        }
    ) {
        self.fetchManifests = fetchManifests
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil
        do {
            manifests = try await fetchManifests()
            if selectedManifestID == nil || !manifests.contains(where: { $0.id == selectedManifestID }) {
                selectedManifestID = manifests.first?.id
            }
        } catch {
            self.error = error.localizedDescription
        }
        if !silent {
            loading = false
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                Task { await self.load(silent: true) }
            },
            onReconnect: { [weak self] in
                guard let self else { return }
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class PayloadLoadViewModel {
    var manifests: [Manifest] = []
    var loading: Bool = true
    var error: String?
    var actingId: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchManifests: @MainActor () async throws -> [Manifest]
    private let transitionAction: @MainActor (String, String) async throws -> FactoryManifestTransitionResponse

    init(
        fetchManifests: @escaping @MainActor () async throws -> [Manifest] = {
            try await FactoryService.manifests().manifests
        },
        transitionAction: @escaping @MainActor (String, String) async throws -> FactoryManifestTransitionResponse = {
            try await FactoryService.transitionManifest(id: $0, action: $1)
        }
    ) {
        self.fetchManifests = fetchManifests
        self.transitionAction = transitionAction
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            let all = try await fetchManifests()
            manifests = all.filter { $0.state == "DRAFT" || $0.state == "LOADING" }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        if !silent { loading = false }
    }

    func run(_ manifest: Manifest, _ action: ManifestLifecycleAction) async {
        actingId = manifest.id
        do {
            _ = try await transitionAction(manifest.id, action.rawValue)
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        actingId = nil
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class ManifestDetailViewModel {
    let manifestId: String
    var detail: ManifestDetailSnapshot?
    var loading: Bool = true
    var acting: Bool = false
    var error: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchDetail: @MainActor (String) async throws -> ManifestDetailSnapshot
    private let transitionAction: @MainActor (String, String) async throws -> FactoryManifestTransitionResponse

    init(
        manifestId: String,
        fetchDetail: @escaping @MainActor (String) async throws -> ManifestDetailSnapshot = {
            try await FactoryService.manifestDetail(id: $0)
        },
        transitionAction: @escaping @MainActor (String, String) async throws -> FactoryManifestTransitionResponse = {
            try await FactoryService.transitionManifest(id: $0, action: $1)
        }
    ) {
        self.manifestId = manifestId
        self.fetchDetail = fetchDetail
        self.transitionAction = transitionAction
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil
        do {
            detail = try await fetchDetail(manifestId)
        } catch {
            self.error = error.localizedDescription
        }
        if !silent {
            loading = false
        }
    }

    func runLifecycle(_ action: ManifestLifecycleAction) async {
        acting = true
        do {
            _ = try await transitionAction(manifestId, action.rawValue)
            PegasusHaptics.confirm()
            await load()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        acting = false
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("MANIFEST_") else { return }
                Task { await self.load(silent: true) }
            },
            onReconnect: { [weak self] in
                guard let self else { return }
                self.acting = false
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class ManifestExceptionsViewModel {
    var exceptions: [ManifestException] = []
    var loading: Bool = true
    var error: String?
    var escalatedOnly: Bool = false
    var resolvingId: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchExceptions: @MainActor (Bool) async throws -> [ManifestException]
    private let resolveAction: @MainActor (String) async throws -> ResolveManifestExceptionResponse

    init(
        fetchExceptions: @escaping @MainActor (Bool) async throws -> [ManifestException] = {
            try await FactoryService.manifestExceptions(escalatedOnly: $0).exceptions
        },
        resolveAction: @escaping @MainActor (String) async throws -> ResolveManifestExceptionResponse = {
            try await FactoryService.resolveManifestException(exceptionId: $0)
        }
    ) {
        self.fetchExceptions = fetchExceptions
        self.resolveAction = resolveAction
    }

    func toggleEscalated() {
        PegasusHaptics.selection()
        escalatedOnly.toggle()
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil
        do {
            exceptions = try await fetchExceptions(escalatedOnly)
        } catch {
            self.error = error.localizedDescription
        }
        if !silent {
            loading = false
        }
    }

    func resolve(_ exceptionId: String) async {
        resolvingId = exceptionId
        do {
            _ = try await resolveAction(exceptionId)
            PegasusHaptics.confirm()
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        resolvingId = nil
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class FleetViewModel {
    var vehicles: [Vehicle] = []
    var liveRoutes: [FactoryFleetLiveRoute] = []
    var loading: Bool = true
    var error: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchVehicles: @MainActor () async throws -> [Vehicle]
    private let fetchLiveRoutes: @MainActor () async throws -> [FactoryFleetLiveRoute]

    init(
        fetchVehicles: @escaping @MainActor () async throws -> [Vehicle] = {
            try await FactoryService.fleet().vehicles
        },
        fetchLiveRoutes: @escaping @MainActor () async throws -> [FactoryFleetLiveRoute] = {
            try await FactoryService.fleetLiveMap().routes
        }
    ) {
        self.fetchVehicles = fetchVehicles
        self.fetchLiveRoutes = fetchLiveRoutes
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil
        do {
            vehicles = try await fetchVehicles()
            if let routes = try? await fetchLiveRoutes() {
                liveRoutes = routes
            }
        } catch {
            self.error = error.localizedDescription
        }
        if !silent {
            loading = false
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                Task { await self.load(silent: true) }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}
