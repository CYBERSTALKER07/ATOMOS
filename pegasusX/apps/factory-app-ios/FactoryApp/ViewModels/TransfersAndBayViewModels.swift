import Foundation
import Observation

@Observable
@MainActor
final class LoadingBayViewModel {
    var transfers: [Transfer] = []
    var loading: Bool = true
    var error: String?
    var dispatching: Bool = false
    var handoffEvents: [FactoryPulseEvent] = []
    var handoffLoading: Bool = true
    var handoffError: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchBayTransfers: @MainActor () async throws -> [Transfer]
    private let fetchPulse: @MainActor () async throws -> [FactoryPulseEvent]
    private let dispatchTransfers: @MainActor ([String]) async throws -> DispatchResponse

    init(
        fetchBayTransfers: @escaping @MainActor () async throws -> [Transfer] = {
            try await FactoryService.loadingBayTransfers().transfers
        },
        fetchPulse: @escaping @MainActor () async throws -> [FactoryPulseEvent] = {
            try await FactoryService.pulse().events
        },
        dispatchTransfers: @escaping @MainActor ([String]) async throws -> DispatchResponse = {
            try await FactoryService.dispatch(transferIds: $0)
        }
    ) {
        self.fetchBayTransfers = fetchBayTransfers
        self.fetchPulse = fetchPulse
        self.dispatchTransfers = dispatchTransfers
    }

    var approved: [Transfer] { transfers.filter { $0.state == "APPROVED" } }
    var loadingState: [Transfer] { transfers.filter { $0.state == "LOADING" } }
    var dispatched: [Transfer] { transfers.filter { $0.state == "DISPATCHED" } }

    func load() async {
        loading = true
        error = nil

        do {
            transfers = try await fetchBayTransfers()
        } catch {
            self.error = error.localizedDescription
        }

        handoffLoading = true
        handoffError = nil
        do {
            let pulseEvents = try await fetchPulse()
            let result = PulseHonesty.apply(
                ok: true,
                incoming: FactoryHandoffPulseSupport.filter(pulseEvents),
                previous: handoffEvents
            )
            handoffEvents = result.events
            handoffError = result.error
        } catch {
            let result = PulseHonesty.apply(ok: false, incoming: nil, previous: handoffEvents)
            handoffEvents = result.events
            handoffError = result.error
        }
        handoffLoading = false
        loading = false
    }

    func batchDispatch() async {
        guard !loadingState.isEmpty else { return }
        dispatching = true
        do {
            let ids = loadingState.map(\.id)
            let result = try await dispatchTransfers(ids)
            if result.createdManifestCount == 0 {
                self.error = "No transfers to dispatch"
                PegasusHaptics.warning()
            } else {
                PegasusHaptics.confirm()
            }
            await load()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        dispatching = false
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                Task { await self.load() }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class TransferListViewModel {
    static let filters = [
        "ALL", "DRAFT", "APPROVED", "LOADING",
        "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED",
    ]

    var transfers: [Transfer] = []
    var loading: Bool = true
    var error: String?
    var selectedFilter: String = "ALL"
    var selectedTransferID: String?
    var showCreateTransfer: Bool = false

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchTransfers: @MainActor (String?) async throws -> [Transfer]

    init(
        fetchTransfers: @escaping @MainActor (String?) async throws -> [Transfer] = { state in
            try await FactoryService.transfers(state: state).transfers
        }
    ) {
        self.fetchTransfers = fetchTransfers
    }

    var selectedTransfer: Transfer? {
        transfers.first { $0.id == selectedTransferID }
    }

    func applyInitialFilter(_ initialFilter: String?) {
        if let initialFilter, !initialFilter.isEmpty {
            selectedFilter = initialFilter
        }
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil

        do {
            let state = selectedFilter == "ALL" ? nil : selectedFilter
            transfers = try await fetchTransfers(state)

            if let selectedTransferID, transfers.contains(where: { $0.id == selectedTransferID }) {
                self.selectedTransferID = selectedTransferID
            } else {
                selectedTransferID = transfers.first?.id
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

@Observable
@MainActor
final class TransferDetailViewModel {
    let transferId: String
    var transfer: Transfer?
    var loading: Bool = true
    var error: String?
    var transitioning: Bool = false

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchTransfer: @MainActor (String) async throws -> Transfer
    private let transitionAction: @MainActor (String, String) async throws -> Transfer

    init(
        transferId: String,
        fetchTransfer: @escaping @MainActor (String) async throws -> Transfer = { try await FactoryService.transfer(id: $0) },
        transitionAction: @escaping @MainActor (String, String) async throws -> Transfer = {
            try await FactoryService.transitionTransfer(id: $0, target: $1)
        }
    ) {
        self.transferId = transferId
        self.fetchTransfer = fetchTransfer
        self.transitionAction = transitionAction
    }

    func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil

        do {
            transfer = try await fetchTransfer(transferId)
        } catch {
            self.error = error.localizedDescription
        }

        if !silent {
            loading = false
        }
    }

    func transition(to target: String) async {
        transitioning = true
        do {
            transfer = try await transitionAction(transferId, target)
            PegasusHaptics.confirm()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        transitioning = false
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                if !self.transitioning {
                    Task { await self.load(silent: true) }
                }
            },
            onReconnect: { [weak self] in
                guard let self else { return }
                if self.transitioning {
                    self.transitioning = false
                    self.error = "Connection restored — transfer state refreshed from server."
                }
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
final class CreateTransferViewModel {
    var loadingFleet: Bool = true
    var submitting: Bool = false
    var error: String?
    var drivers: [FactoryFleetDriverRow] = []
    var vehicles: [FactoryFleetVehicleRow] = []
    var orderId: String = ""
    var totalVu: String = "25"
    var driverId: String = ""
    var vehicleId: String = ""

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchDrivers: @MainActor () async throws -> [FactoryFleetDriverRow]
    private let fetchVehicles: @MainActor () async throws -> [FactoryFleetVehicleRow]
    private let createAction: @MainActor (FactoryCreateTransferRequest) async throws -> FactoryCreateTransferResponse

    init(
        fetchDrivers: @escaping @MainActor () async throws -> [FactoryFleetDriverRow] = { try await FactoryService.fleetDrivers() },
        fetchVehicles: @escaping @MainActor () async throws -> [FactoryFleetVehicleRow] = { try await FactoryService.fleetVehicles() },
        createAction: @escaping @MainActor (FactoryCreateTransferRequest) async throws -> FactoryCreateTransferResponse = {
            try await FactoryService.createTransfer($0)
        }
    ) {
        self.fetchDrivers = fetchDrivers
        self.fetchVehicles = fetchVehicles
        self.createAction = createAction
    }

    func loadFleet() async {
        loadingFleet = true
        error = nil
        do {
            async let driverRows = fetchDrivers()
            async let vehicleRows = fetchVehicles()
            drivers = try await driverRows
            vehicles = try await vehicleRows
        } catch {
            self.error = error.localizedDescription
        }
        loadingFleet = false
    }

    func submit() async -> String? {
        let trimmedVu = totalVu.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let parsedVu = Int(trimmedVu), parsedVu > 0 else {
            error = "Total VU must be a positive number."
            PegasusHaptics.warning()
            return nil
        }
        submitting = true
        error = nil
        defer { submitting = false }

        do {
            let trimmedOrder = orderId.trimmingCharacters(in: .whitespacesAndNewlines)
            let response = try await createAction(
                FactoryCreateTransferRequest(
                    orderId: trimmedOrder.isEmpty ? nil : trimmedOrder,
                    totalVu: parsedVu,
                    driverId: driverId.isEmpty ? nil : driverId,
                    vehicleId: vehicleId.isEmpty ? nil : vehicleId
                )
            )
            PegasusHaptics.confirm()
            return response.transferId
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
            return nil
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { _ in },
            onReconnect: { [weak self] in
                guard let self else { return }
                if self.submitting {
                    self.submitting = false
                    self.error = "Connection restored — verify transfer was created before retrying."
                }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}
