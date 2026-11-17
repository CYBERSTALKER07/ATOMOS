import Foundation
import Observation

@Observable
@MainActor
final class StaffViewModel {
    var staff: [StaffMember] = []
    var loading: Bool = true
    var error: String?
    var showCreate: Bool = false

    var newStaffName: String = ""
    var newStaffRole: String = "FACTORY_OPERATOR"
    var submittingCreate: Bool = false
    var createError: String?

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchStaff: @MainActor () async throws -> [StaffMember]
    private let createStaffAction: @MainActor (String, String) async throws -> StaffMember

    init(
        fetchStaff: @escaping @MainActor () async throws -> [StaffMember] = {
            try await FactoryService.staff().staff
        },
        createStaffAction: @escaping @MainActor (String, String) async throws -> StaffMember = {
            try await FactoryService.createStaff(name: $0, role: $1)
        }
    ) {
        self.fetchStaff = fetchStaff
        self.createStaffAction = createStaffAction
    }

    var canSubmitCreate: Bool {
        !submittingCreate && !newStaffName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func presentCreate() {
        newStaffName = ""
        newStaffRole = "FACTORY_OPERATOR"
        createError = nil
        showCreate = true
    }

    func load() async {
        loading = true
        error = nil
        do {
            staff = try await fetchStaff()
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }

    func createStaff() async -> Bool {
        let trimmedName = newStaffName.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty else {
            createError = "Staff name is required."
            PegasusHaptics.warning()
            return false
        }
        let trimmedRole = newStaffRole.trimmingCharacters(in: .whitespacesAndNewlines)
        submittingCreate = true
        createError = nil
        defer { submittingCreate = false }

        do {
            _ = try await createStaffAction(trimmedName, trimmedRole.isEmpty ? "FACTORY_OPERATOR" : trimmedRole)
            PegasusHaptics.confirm()
            showCreate = false
            await load()
            return true
        } catch {
            createError = error.localizedDescription
            PegasusHaptics.error()
            return false
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { _ in },
            onEvent: { [weak self] event in
                guard let self else { return }
                if event.type.hasPrefix("TRANSFER_")
                    || event.type.hasPrefix("MANIFEST_")
                    || event.type.hasPrefix("WAREHOUSE_TRANSFER_")
                    || event.type.hasPrefix("FACTORY_SUPPLY_") {
                    Task { await self.load() }
                }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }
}

@Observable
@MainActor
final class StaffDetailViewModel {
    let staffId: String
    var staff: StaffMember?
    var loading: Bool = true
    var error: String?
    var pin: String = ""
    var setMsg: String?
    var setting: Bool = false

    private let fetchDetail: @MainActor (String) async throws -> StaffMember
    private let setPasswordAction: @MainActor (String, String) async throws -> Void

    init(
        staffId: String,
        fetchDetail: @escaping @MainActor (String) async throws -> StaffMember = {
            try await FactoryService.staffDetail(id: $0)
        },
        setPasswordAction: @escaping @MainActor (String, String) async throws -> Void = {
            try await FactoryService.setStaffPassword(id: $0, pin: $1)
        }
    ) {
        self.staffId = staffId
        self.fetchDetail = fetchDetail
        self.setPasswordAction = setPasswordAction
    }

    var canSetPassword: Bool {
        !setting && pin.trimmingCharacters(in: .whitespacesAndNewlines).count >= 4
    }

    func load() async {
        loading = true
        error = nil
        do {
            staff = try await fetchDetail(staffId)
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }

    func setPassword() async {
        let trimmed = pin.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 4 else {
            setMsg = "PIN must be at least 4 characters."
            PegasusHaptics.warning()
            return
        }
        setting = true
        setMsg = nil
        do {
            try await setPasswordAction(staffId, trimmed)
            setMsg = "Password set"
            pin = ""
            PegasusHaptics.confirm()
        } catch {
            setMsg = error.localizedDescription
            PegasusHaptics.error()
        }
        setting = false
    }
}

struct SupplyRequestActionSpec: Equatable {
    let title: String
    let action: String
    let emphasized: Bool
}

let supplyRequestFilters = [
    "ALL", "SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY", "FULFILLED", "CANCELLED",
]

func supplyRequestActions(for state: String) -> [SupplyRequestActionSpec] {
    switch state {
    case "SUBMITTED":
        return [
            SupplyRequestActionSpec(title: "Acknowledge", action: "ACKNOWLEDGE", emphasized: true),
            SupplyRequestActionSpec(title: "Cancel", action: "CANCEL", emphasized: false),
        ]
    case "ACKNOWLEDGED":
        return [
            SupplyRequestActionSpec(title: "Start Production", action: "START_PRODUCTION", emphasized: true),
            SupplyRequestActionSpec(title: "Cancel", action: "CANCEL", emphasized: false),
        ]
    case "IN_PRODUCTION":
        return [
            SupplyRequestActionSpec(title: "Mark Ready", action: "MARK_READY", emphasized: true),
        ]
    case "READY":
        return [
            SupplyRequestActionSpec(title: "Fulfill", action: "FULFILL", emphasized: true),
        ]
    default:
        return []
    }
}

struct SupplyFulfillModalItem: Identifiable {
    let request: SupplyRequest
    let options: SupplyFulfillOptions
    var id: String { request.id }
}

@Observable
@MainActor
final class SupplyRequestsViewModel {
    var requests: [SupplyRequest] = []
    var loading: Bool = true
    var error: String?
    var selectedFilter: String = "ALL"
    var viewMode: String = "TABLE"
    var fulfillModalItem: SupplyFulfillModalItem?
    var transitioningID: String?
    var qcById: [String: String] = [:]
    var refreshing: Bool = false
    var staleMessage: String?
    var lastSyncedAt: Date?
    var realtimeStatus: FactoryRealtimeStatus = .idle

    private let realtimeClient = FactoryRealtimeClient()
    private let fetchRequests: @MainActor () async throws -> [SupplyRequest]
    private let fetchQC: @MainActor (String) async throws -> SupplyRequestQCResponse
    private let postQC: @MainActor (String, String) async throws -> SupplyRequestQCResponse
    private let fetchFulfillOptions: @MainActor (String) async throws -> SupplyFulfillOptions
    private let transitionAction: @MainActor (String, String) async throws -> SupplyRequestTransitionResponse

    init(
        fetchRequests: @escaping @MainActor () async throws -> [SupplyRequest] = {
            try await FactoryService.supplyRequests()
        },
        fetchQC: @escaping @MainActor (String) async throws -> SupplyRequestQCResponse = {
            try await FactoryService.supplyRequestQC(id: $0)
        },
        postQC: @escaping @MainActor (String, String) async throws -> SupplyRequestQCResponse = {
            try await FactoryService.postSupplyRequestQC(id: $0, result: $1)
        },
        fetchFulfillOptions: @escaping @MainActor (String) async throws -> SupplyFulfillOptions = {
            try await FactoryService.supplyFulfillOptions(id: $0)
        },
        transitionAction: @escaping @MainActor (String, String) async throws -> SupplyRequestTransitionResponse = {
            try await FactoryService.transitionSupplyRequest(id: $0, action: $1)
        }
    ) {
        self.fetchRequests = fetchRequests
        self.fetchQC = fetchQC
        self.postQC = postQC
        self.fetchFulfillOptions = fetchFulfillOptions
        self.transitionAction = transitionAction
    }

    var filteredRequests: [SupplyRequest] {
        selectedFilter == "ALL" ? requests : requests.filter { $0.state == selectedFilter }
    }

    var runtimeStatus: String {
        if let staleMessage {
            return staleMessage
        }

        switch realtimeStatus {
        case .offline:
            return "Offline — showing last sync \(supplySyncText(lastSyncedAt))"
        case .reconnecting:
            return "Reconnecting live queue — last sync \(supplySyncText(lastSyncedAt))"
        case .connecting:
            return "Connecting to the live supply queue…"
        case .idle, .live:
            break
        }

        if refreshing {
            return "Refreshing live queue — last sync \(supplySyncText(lastSyncedAt))"
        }

        if lastSyncedAt != nil {
            return "Live sync active — last sync \(supplySyncText(lastSyncedAt))"
        }

        return "Waiting for first sync"
    }

    var runtimeTone: FactoryRuntimeTone {
        if staleMessage != nil && realtimeStatus == .offline {
            return .offline
        }
        if staleMessage != nil {
            return .warning
        }
        switch realtimeStatus {
        case .offline:
            return .offline
        case .reconnecting, .connecting:
            return .refreshing
        case .idle, .live:
            break
        }
        return refreshing ? .refreshing : .live
    }

    func handleAction(request: SupplyRequest, action: String) async {
        if action == "FULFILL" {
            do {
                let options = try await fetchFulfillOptions(request.id)
                fulfillModalItem = SupplyFulfillModalItem(request: request, options: options)
                PegasusHaptics.selection()
            } catch {
                self.error = error.localizedDescription
                PegasusHaptics.error()
            }
            return
        }
        await runTransition(request: request, action: action)
    }

    func confirmFulfill(_ item: SupplyFulfillModalItem) async {
        fulfillModalItem = nil
        await runTransition(request: item.request, action: "FULFILL")
    }

    func handleQC(request: SupplyRequest, result: String) async {
        transitioningID = request.id
        defer { transitioningID = nil }
        do {
            _ = try await postQC(request.id, result)
            qcById[request.id] = result
            PegasusHaptics.confirm()
        } catch {
            staleMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func load(background: Bool = false) async {
        if background {
            refreshing = true
        } else if requests.isEmpty {
            loading = true
            error = nil
        }

        do {
            requests = try await fetchRequests()
            staleMessage = nil
            error = nil
            lastSyncedAt = Date()
            var next: [String: String] = [:]
            for request in requests {
                if let qc = try? await fetchQC(request.id) {
                    next[request.id] = qc.result
                }
            }
            qcById = next
        } catch {
            let message = error.localizedDescription
            if requests.isEmpty {
                self.error = message
            } else {
                staleMessage = "Showing last synced queue. \(message)"
            }
        }

        loading = false
        refreshing = false
    }

    func runTransition(request: SupplyRequest, action: String) async {
        transitioningID = request.id
        do {
            _ = try await transitionAction(request.id, action)
            PegasusHaptics.confirm()
            await load(background: true)
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
        transitioningID = nil
    }

    func runPollingLoop() async {
        startRealtime()
        await load()
        while !Task.isCancelled {
            try? await Task.sleep(for: .seconds(30))
            guard !Task.isCancelled else { break }
            if transitioningID == nil {
                await load(background: true)
            }
        }
    }

    func startRealtime() {
        realtimeClient.connect(
            onStateChange: { [weak self] status in
                self?.realtimeStatus = status
            },
            onEvent: { [weak self] event in
                guard let self else { return }
                guard event.type.hasPrefix("FACTORY_SUPPLY_") else { return }
                if self.transitioningID == nil {
                    Task { await self.load(background: !self.requests.isEmpty) }
                }
            }
        )
    }

    func stopRealtime() {
        realtimeClient.disconnect()
    }

    private func supplySyncText(_ value: Date?) -> String {
        guard let value else { return "waiting" }
        return value.formatted(date: .omitted, time: .shortened)
    }
}

@Observable
@MainActor
final class SupplyRequestDetailViewModel {
    let requestId: String
    var request: SupplyRequest?
    var loading: Bool = true
    var error: String?
    var actionError: String?
    var isProcessing: Bool = false

    private let fetchRequests: @MainActor () async throws -> [SupplyRequest]
    private let transitionAction: @MainActor (String, String) async throws -> SupplyRequestTransitionResponse

    init(
        requestId: String,
        fetchRequests: @escaping @MainActor () async throws -> [SupplyRequest] = {
            try await FactoryService.supplyRequests()
        },
        transitionAction: @escaping @MainActor (String, String) async throws -> SupplyRequestTransitionResponse = {
            try await FactoryService.transitionSupplyRequest(id: $0, action: $1)
        }
    ) {
        self.requestId = requestId
        self.fetchRequests = fetchRequests
        self.transitionAction = transitionAction
    }

    func availableActions(for state: String) -> [String] {
        switch state.uppercased() {
        case "OPEN", "SUBMITTED": return ["ACKNOWLEDGE"]
        case "ACKNOWLEDGED": return ["IN_PRODUCTION"]
        case "PRODUCTION", "IN_PRODUCTION": return ["READY"]
        case "READY": return ["FULFILL"]
        default: return []
        }
    }

    func load() async {
        loading = true
        error = nil
        do {
            let requests = try await fetchRequests()
            self.request = requests.first { $0.id == requestId }
            if self.request == nil {
                self.error = "Request not found."
            }
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }

    func performAction(_ action: String) async {
        isProcessing = true
        actionError = nil
        do {
            _ = try await transitionAction(requestId, action)
            PegasusHaptics.confirm()
            await load()
        } catch {
            self.actionError = error.localizedDescription
            PegasusHaptics.error()
        }
        isProcessing = false
    }
}

@Observable
@MainActor
final class SupplyRequestsHubViewModel {
    var requests: [SupplyRequest] = []
    var loading: Bool = true
    var error: String?
    var selectedRequestID: String?

    private let fetchRequests: @MainActor () async throws -> [SupplyRequest]

    init(
        fetchRequests: @escaping @MainActor () async throws -> [SupplyRequest] = {
            try await FactoryService.supplyRequests()
        }
    ) {
        self.fetchRequests = fetchRequests
    }

    func load() async {
        loading = true
        error = nil
        do {
            requests = try await fetchRequests()
            if selectedRequestID == nil || !requests.contains(where: { $0.id == selectedRequestID }) {
                selectedRequestID = requests.first?.id
            }
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
    }
}
