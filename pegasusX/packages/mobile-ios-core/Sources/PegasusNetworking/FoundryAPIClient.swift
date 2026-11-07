import Foundation

private actor FoundryInFlightCoalescer {
    private var inFlightTasks: [String: Task<Data, Error>] = [:]
    private var responseCache: [String: (data: Data, expiresAt: Date)] = [:]
    private let cacheTTL: TimeInterval = 2.0

    func fetchOrCoalesce(key: String, operation: @Sendable @escaping () async throws -> Data) async throws -> Data {
        let now = Date()
        if let cached = responseCache[key], cached.expiresAt > now {
            return cached.data
        }
        if let existing = inFlightTasks[key] {
            return try await existing.value
        }
        let task = Task<Data, Error> {
            try await operation()
        }
        inFlightTasks[key] = task
        do {
            let data = try await task.value
            inFlightTasks.removeValue(forKey: key)
            responseCache[key] = (data, Date().addingTimeInterval(cacheTTL))
            return data
        } catch {
            inFlightTasks.removeValue(forKey: key)
            throw error
        }
    }

    func invalidateAll() {
        responseCache.removeAll()
    }
}

public final class FoundryAPIClient: @unchecked Sendable {
    public static let shared = FoundryAPIClient()

    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    private let coalescer = FoundryInFlightCoalescer()

    public var baseURL: String = "https://api.pegasusx.app"

    public init(session: URLSession = .shared) {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 15
        config.timeoutIntervalForResource = 30
        self.session = URLSession(configuration: config)
        self.decoder = JSONDecoder()
        self.encoder = JSONEncoder()
    }

    private func buildRequest(path: String, method: String, idempotencyKey: String? = nil) throws -> URLRequest {
        let cleanBase = baseURL.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        let cleanPath = path.hasPrefix("/") ? String(path.dropFirst()) : path
        guard let url = URL(string: "\(cleanBase)/\(cleanPath)") else {
            throw URLError(.badURL)
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(UUID().uuidString, forHTTPHeaderField: "X-Trace-Id")
        if method != "GET" {
            request.setValue(idempotencyKey ?? UUID().uuidString, forHTTPHeaderField: "Idempotency-Key")
        }
        return request
    }

    private func executeRaw(_ request: URLRequest) async throws -> Data {
        let isIdempotent = request.httpMethod == "GET" || request.value(forHTTPHeaderField: "Idempotency-Key") != nil
        let maxAttempts = isIdempotent ? 3 : 1
        var lastError: Error = URLError(.unknown)

        for attempt in 0..<maxAttempts {
            try Task.checkCancellation()
            do {
                let (data, response) = try await session.data(for: request)
                guard let http = response as? HTTPURLResponse else {
                    throw URLError(.badServerResponse)
                }
                if (200...299).contains(http.statusCode) {
                    return data
                }
                let isRetryableStatus = http.statusCode == 429 || (502...504).contains(http.statusCode)
                if isIdempotent && isRetryableStatus && attempt + 1 < maxAttempts {
                    let delayNs = UInt64(300_000_000 * (1 << attempt)) + UInt64.random(in: 0...150_000_000)
                    try await Task.sleep(nanoseconds: delayNs)
                    continue
                }
                throw URLError(.init(rawValue: http.statusCode))
            } catch is CancellationError {
                throw CancellationError()
            } catch let urlErr as URLError where urlErr.code == .cancelled {
                throw CancellationError()
            } catch {
                lastError = error
                if isIdempotent && attempt + 1 < maxAttempts {
                    let delayNs = UInt64(300_000_000 * (1 << attempt)) + UInt64.random(in: 0...150_000_000)
                    try await Task.sleep(nanoseconds: delayNs)
                    continue
                }
                throw error
            }
        }
        throw lastError
    }

    private func execute<T: Decodable>(_ request: URLRequest) async throws -> T {
        let method = request.httpMethod ?? "GET"
        let data: Data
        if method == "GET", let urlKey = request.url?.absoluteString {
            data = try await coalescer.fetchOrCoalesce(key: urlKey) { [self] in
                try await self.executeRaw(request)
            }
        } else {
            await coalescer.invalidateAll()
            data = try await executeRaw(request)
        }
        return try decoder.decode(T.self, from: data)
    }

    // MARK: - Ontology & Scenario Endpoints

    public func triggerLotRecall(_ req: LotRecallRequest) async throws -> RecallBlastRadiusReport {
        var request = try buildRequest(path: "/v1/foundry/ontology/lot-recall", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func executeAction(_ req: ActionExecutionRequest) async throws -> ActionExecutionRecord {
        var request = try buildRequest(path: "/v1/foundry/kinetic/actions/execute", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func applyScenarioPatch(scenarioID: String, req: ScenarioPatchRequest) async throws -> [String: String] {
        var request = try buildRequest(path: "/v1/foundry/scenarios/\(scenarioID)/patches", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func resolveScenarioObject(scenarioID: String, cellID: String, objectType: String, objectID: String) async throws -> OntologyObjectDto {
        let request = try buildRequest(path: "/v1/foundry/scenarios/\(scenarioID)/resolve/\(cellID)/\(objectType)/\(objectID)", method: "GET")
        return try await execute(request)
    }

    public func commitScenario(scenarioID: String) async throws -> ScenarioCommitResult {
        let request = try buildRequest(path: "/v1/foundry/scenarios/\(scenarioID)/commit", method: "POST")
        return try await execute(request)
    }

    public func searchAround(_ req: SearchAroundRequest) async throws -> SearchAroundResponse {
        var request = try buildRequest(path: "/v1/foundry/ontology/search-around", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    // MARK: - Role 7: Cashier Endpoints

    public func startCashierSession(_ req: StartCashierSessionRequest) async throws -> CashierSessionView {
        var request = try buildRequest(path: "/v1/foundry/cashier/sessions", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func submitBlindCount(sessionID: String, req: BlindCountRequest) async throws -> CashierSessionView {
        var request = try buildRequest(path: "/v1/foundry/cashier/sessions/\(sessionID)/blind-count", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func reclassifyTender(_ req: RRNTenderReclassificationRequest) async throws -> CashierSessionView {
        var request = try buildRequest(path: "/v1/foundry/cashier/reclassify-tender", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func executeCITHandover(_ req: CITHandoverRequest) async throws -> DepotVaultLedgerEntry {
        var request = try buildRequest(path: "/v1/foundry/cashier/vault/cit-handover", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func checkAML(_ req: AMLCheckRequest) async throws -> AMLCheckResponse {
        var request = try buildRequest(path: "/v1/foundry/guards/aml-check", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    // MARK: - Role 9: Fleet Endpoints

    public func submitDVIR(_ req: FleetDVIRInspection) async throws -> FleetDVIRInspection {
        var request = try buildRequest(path: "/v1/foundry/fleet/dvir", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func recordGateEvent(_ req: YardGateCheckEvent) async throws -> YardGateCheckEvent {
        var request = try buildRequest(path: "/v1/foundry/fleet/gate-events", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func executeMidRouteRescue(_ req: MidRouteRescueRequest) async throws -> MidRouteRescueResult {
        var request = try buildRequest(path: "/v1/foundry/fleet/mid-route-rescue", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func checkPredictiveReefer(_ req: PredictiveReeferRequest) async throws -> PredictiveReeferResponse {
        var request = try buildRequest(path: "/v1/foundry/fleet/predictive-reefer", method: "POST")
        request.httpBody = try encoder.encode(req)
        return try await execute(request)
    }

    public func checkVehicleDispatch(vehicleID: String) async throws -> DispatchCheckResponse {
        let request = try buildRequest(path: "/v1/foundry/fleet/vehicles/\(vehicleID)/dispatch-check", method: "GET")
        return try await execute(request)
    }
}
