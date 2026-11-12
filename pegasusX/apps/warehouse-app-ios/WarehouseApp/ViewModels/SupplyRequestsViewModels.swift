import Foundation
import Observation

struct ManualSupplyLine: Identifiable, Sendable {
    let id: UUID
    var productId: String
    var quantity: String

    init(id: UUID = UUID(), productId: String = "", quantity: String = "") {
        self.id = id
        self.productId = productId
        self.quantity = quantity
    }
}

@Observable
@MainActor
final class CreateSupplyRequestViewModel {
    typealias FetchFactory = @MainActor () async throws -> WarehouseSupplyFactoryResponse
    typealias FetchForecast = @MainActor (Int) async throws -> DemandForecastResponse

    private static let isoFormatter = ISO8601DateFormatter()

    var factoryId = ""
    var factoryLocked = false
    var priority = "NORMAL"
    var notes = ""
    var useForecast = true
    var deliveryDate = Date()
    var includeDeliveryDate = false
    var forecast: [DemandForecastProduct] = []
    var forecastLoading = false
    var manualItems: [ManualSupplyLine] = [ManualSupplyLine()]

    var canSubmit: Bool {
        guard !factoryId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else { return false }
        if useForecast { return true }
        return manualItems.contains { line in
            !line.productId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                && (Int(line.quantity) ?? 0) > 0
        }
    }

    private let fetchFactory: FetchFactory
    private let fetchForecast: FetchForecast

    init(
        fetchFactory: FetchFactory? = nil,
        fetchForecast: FetchForecast? = nil
    ) {
        self.fetchFactory = fetchFactory ?? {
            try await WarehouseService.opsSupplyFactory()
        }
        self.fetchForecast = fetchForecast ?? { days in
            try await WarehouseService.demandForecast(days: days)
        }
    }

    func loadEngineFactory() async {
        do {
            let supply = try await fetchFactory()
            if !supply.factoryId.isEmpty {
                factoryId = supply.factoryId
                factoryLocked = true
            }
        } catch {
            factoryLocked = false
        }
    }

    func loadForecast() async {
        forecastLoading = true
        defer { forecastLoading = false }
        do {
            let response = try await fetchForecast(7)
            forecast = response.products
        } catch {
            forecast = []
        }
    }

    func addManualLine() {
        manualItems.append(ManualSupplyLine())
    }

    func buildFormData() -> SupplyRequestFormData {
        let deliveryISO: String? = includeDeliveryDate
            ? Self.isoFormatter.string(from: deliveryDate)
            : nil
        let items: [CreateWarehouseSupplyRequestItem]
        if useForecast {
            items = []
        } else {
            items = manualItems.compactMap { line in
                let pid = line.productId.trimmingCharacters(in: .whitespacesAndNewlines)
                guard let qty = Int(line.quantity), !pid.isEmpty, qty > 0 else { return nil }
                return CreateWarehouseSupplyRequestItem(
                    productId: pid,
                    requestedQuantity: qty,
                    recommendedQty: qty,
                    unitVolumeVu: 0
                )
            }
        }
        return SupplyRequestFormData(
            factoryId: factoryId.trimmingCharacters(in: .whitespacesAndNewlines),
            priority: priority,
            notes: notes.trimmingCharacters(in: .whitespacesAndNewlines),
            useDemandForecast: useForecast,
            requestedDeliveryDate: deliveryISO,
            items: items
        )
    }
}

@Observable
@MainActor
final class SupplyRequestDetailViewModel {
    typealias FetchRequest = @MainActor (String) async throws -> WarehouseSupplyRequest
    typealias FetchQC = @MainActor (String) async throws -> SupplyRequestQCResponse
    typealias CancelRequest = @MainActor (String) async throws -> WarehouseSupplyRequestTransitionResponse

    var request: WarehouseSupplyRequest?
    var qcResult = ""
    var loading = true
    var error: String?
    var busy = false
    var statusMessage: String?

    private let fetchRequest: FetchRequest
    private let fetchQC: FetchQC
    private let cancelRequestFn: CancelRequest

    init(
        fetchRequest: FetchRequest? = nil,
        fetchQC: FetchQC? = nil,
        cancelRequest: CancelRequest? = nil
    ) {
        self.fetchRequest = fetchRequest ?? { requestId in
            try await WarehouseService.supplyRequest(id: requestId)
        }
        self.fetchQC = fetchQC ?? { requestId in
            try await WarehouseService.supplyRequestQC(id: requestId)
        }
        self.cancelRequestFn = cancelRequest ?? { requestId in
            try await WarehouseService.cancelSupplyRequest(id: requestId)
        }
    }

    func load(requestId: String, silent: Bool = false) async {
        if !silent && request == nil { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            request = try await fetchRequest(requestId)
            if let qc = try? await fetchQC(requestId) {
                qcResult = qc.result
            } else {
                qcResult = ""
            }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func cancelRequest(requestId: String) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await cancelRequestFn(requestId)
            statusMessage = "Request cancelled"
            PegasusHaptics.success()
            await load(requestId: requestId)
        } catch {
            statusMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func handleReconnect() {
        if busy {
            busy = false
            statusMessage = "Connection restored — verify status before retrying."
        }
    }
}

@Observable
@MainActor
final class SupplyRequestsHubViewModel {
    typealias FetchRequests = @MainActor (String?) async throws -> [WarehouseSupplyRequest]
    typealias CreateRequest = @MainActor (SupplyRequestFormData) async throws -> CreateWarehouseSupplyRequestResponse

    let filters = ["ALL", "OPEN", "CANCELLED"]

    var requests: [WarehouseSupplyRequest] = []
    var loading = true
    var error: String?
    var stateFilter = "ALL"
    var showCreate = false

    var filtered: [WarehouseSupplyRequest] {
        guard stateFilter != "ALL" else { return requests }
        return requests.filter { $0.state.uppercased() == stateFilter }
    }

    private let fetchRequests: FetchRequests
    private let createRequestFn: CreateRequest

    init(
        fetchRequests: FetchRequests? = nil,
        createRequest: CreateRequest? = nil
    ) {
        self.fetchRequests = fetchRequests ?? { state in
            try await WarehouseService.supplyRequests(state: state)
        }
        self.createRequestFn = createRequest ?? { form in
            try await WarehouseService.createSupplyRequest(form: form)
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let state = stateFilter == "ALL" ? nil : stateFilter
            requests = try await fetchRequests(state)
        } catch {
            self.error = error.localizedDescription
        }
    }

    func submitCreate(form: SupplyRequestFormData) async {
        do {
            _ = try await createRequestFn(form)
            showCreate = false
            PegasusHaptics.success()
            await load()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}
