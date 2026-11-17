import Foundation
import Observation

@Observable
@MainActor
public final class WebSocketStreamClient {
    public static var defaultStreamURL: URL {
        var components = URLComponents()
        components.scheme = "ws"
        components.host = "127.0.0.1"
        components.port = 8180
        components.path = "/ws/telemetry"
        return components.url ?? URL(fileURLWithPath: "/")
    }

    public private(set) var isConnected: Bool = false
    public private(set) var events: [TelemetryLogEvent] = []
    public var isPaused: Bool = false

    public static func isAllowedStreamURL(_ url: URL) -> Bool {
        guard let scheme = url.scheme?.lowercased(), let host = url.host?.lowercased(), !host.isEmpty else {
            return false
        }
        if scheme == "wss" {
            return true
        }
        return scheme == "ws" && (host == "127.0.0.1" || host == "localhost" || host == "::1")
    }

    public let url: URL
    private var webSocketTask: URLSessionWebSocketTask?
    private var urlSession: URLSession?
    private var isManuallyDisconnected: Bool = false
    private let maxBufferSize: Int = 50
    private var retryAttempt: Int = 0

    public init(url: URL? = nil) {
        self.url = url ?? Self.defaultStreamURL
        seedInitialEvents()
    }

    public func connect() {
        guard Self.isAllowedStreamURL(url) else {
            isConnected = false
            return
        }
        guard webSocketTask == nil else { return }
        isManuallyDisconnected = false
        urlSession?.invalidateAndCancel()
        let session = URLSession(configuration: .ephemeral)
        self.urlSession = session
        let task = session.webSocketTask(with: url)
        self.webSocketTask = task
        task.resume()
        self.isConnected = true
        self.retryAttempt = 0
        receiveNextMessage()
    }

    public func disconnect() {
        isManuallyDisconnected = true
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        urlSession?.invalidateAndCancel()
        urlSession = nil
        self.isConnected = false
    }

    public func clearSession() {
        disconnect()
        events.removeAll()
        retryAttempt = 0
    }

    public func togglePause() {
        ControlTowerHaptics.levelChange()
        isPaused.toggle()
    }

    public func pushEvent(_ event: TelemetryLogEvent) {
        guard !isPaused else { return }
        events.insert(event, at: 0)
        if events.count > maxBufferSize {
            events.removeLast(events.count - maxBufferSize)
        }
    }

    public func ingestRawPayload(_ data: Data) {
        let decoder = JSONDecoder()
        if let event = try? decoder.decode(TelemetryLogEvent.self, from: data) {
            pushEvent(event)
        }
    }

    private func receiveNextMessage() {
        guard let task = webSocketTask else { return }
        task.receive { [weak self] result in
            Task { @MainActor [weak self] in
                guard let self = self else { return }
                switch result {
                case .success(let message):
                    switch message {
                    case .string(let text):
                        self.handleIncomingString(text)
                    case .data(let data):
                        self.ingestRawPayload(data)
                    @unknown default:
                        break
                    }
                    self.receiveNextMessage()
                case .failure:
                    let shouldReconnect = !self.isManuallyDisconnected
                    self.webSocketTask = nil
                    self.urlSession?.invalidateAndCancel()
                    self.urlSession = nil
                    self.isConnected = false
                    if shouldReconnect {
                        self.scheduleReconnect()
                    }
                }
            }
        }
    }

    private func handleIncomingString(_ text: String) {
        guard let data = text.data(using: .utf8) else { return }
        ingestRawPayload(data)
    }

    private func scheduleReconnect() {
        guard !isManuallyDisconnected else { return }
        let backoff = min(pow(2.0, Double(retryAttempt)), 30.0)
        let jitter = Double.random(in: 0.1...0.8)
        retryAttempt += 1
        DispatchQueue.main.asyncAfter(deadline: .now() + backoff + jitter) { [weak self] in
            Task { @MainActor [weak self] in
                self?.connect()
            }
        }
    }

    private func seedInitialEvents() {
        self.events = [
            TelemetryLogEvent(
                eventId: "EVT-8921",
                traceId: "TRC-SPANNER-20261031-891",
                timestampIso: "2026-10-31T11:15:02.120Z",
                severity: .info,
                role: "Supplier",
                entityId: "SPL-SAM-901",
                service: "catalog-sync-worker",
                message: "Sync catalog GTIN bundle 48 items validated against Soliq MXIK standard."
            ),
            TelemetryLogEvent(
                eventId: "EVT-8922",
                traceId: "TRC-KAFKA-ORD-092",
                timestampIso: "2026-10-31T11:15:14.450Z",
                severity: .info,
                role: "Retailer",
                entityId: "TIN-309182746",
                service: "order-lifecycle-svc",
                message: "Order placed PO-2026-9012. Cloud Spanner mutation committed in 18ms."
            ),
            TelemetryLogEvent(
                eventId: "EVT-8923",
                traceId: "TRC-REDIS-DEP-301",
                timestampIso: "2026-10-31T11:15:28.890Z",
                severity: .warn,
                role: "Warehouse",
                entityId: "DEPOT-TAS-CENTRAL",
                service: "wave-execution-engine",
                message: "Wave 81 permit queue wait reached 0.8ms (within safety ceiling 5.0ms)."
            ),
            TelemetryLogEvent(
                eventId: "EVT-8924",
                traceId: "TRC-DVIR-FLT-114",
                timestampIso: "2026-10-31T11:15:42.010Z",
                severity: .info,
                role: "Fleet",
                entityId: "FLT-EAST-CARRIER",
                service: "fleet-telematics-hub",
                message: "DVIR pre-trip walkaround passed for vehicle UZ-01-778-AAA. Reefer temp -18.2°C nominal."
            ),
            TelemetryLogEvent(
                eventId: "EVT-8925",
                traceId: "TRC-FISCAL-SOLIQ-550",
                timestampIso: "2026-10-31T11:15:58.300Z",
                severity: .error,
                role: "Supplier",
                entityId: "SPL-FER-402",
                service: "fiscal-gateway-worker",
                message: "Soliq endpoint retry backoff triggered (HTTP 429). Routed to Kafka DLQ partition 2."
            )
        ]
    }
}
