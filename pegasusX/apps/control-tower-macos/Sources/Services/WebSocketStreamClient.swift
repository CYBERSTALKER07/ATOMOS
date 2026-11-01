import Foundation
import Combine

@MainActor
public final class WebSocketStreamClient: ObservableObject {
    @Published public private(set) var isConnected: Bool = false
    @Published public private(set) var events: [TelemetryLogEvent] = []
    @Published public var isPaused: Bool = false

    private var webSocketTask: URLSessionWebSocketTask?
    private let url: URL
    private let maxBufferSize: Int = 50
    private var retryAttempt: Int = 0

    public init(url: URL = URL(string: "ws://127.0.0.1:8180/ws/telemetry")!) {
        self.url = url
        seedInitialEvents()
    }

    public func connect() {
        guard webSocketTask == nil else { return }
        let session = URLSession(configuration: .default)
        let task = session.webSocketTask(with: url)
        self.webSocketTask = task
        task.resume()
        self.isConnected = true
        self.retryAttempt = 0
        receiveNextMessage()
    }

    public func disconnect() {
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        self.isConnected = false
    }

    public func togglePause() {
        isPaused.toggle()
    }

    public func pushEvent(_ event: TelemetryLogEvent) {
        guard !isPaused else { return }
        events.insert(event, at: 0)
        if events.count > maxBufferSize {
            events.removeLast(events.count - maxBufferSize)
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
                        self.handleIncomingData(data)
                    @unknown default:
                        break
                    }
                    self.receiveNextMessage()
                case .failure:
                    self.disconnect()
                    self.scheduleReconnect()
                }
            }
        }
    }

    private func handleIncomingString(_ text: String) {
        guard let data = text.data(using: .utf8) else { return }
        handleIncomingData(data)
    }

    private func handleIncomingData(_ data: Data) {
        let decoder = JSONDecoder()
        if let event = try? decoder.decode(TelemetryLogEvent.self, from: data) {
            pushEvent(event)
        }
    }

    private func scheduleReconnect() {
        let backoff = min(pow(2.0, Double(retryAttempt)), 30.0)
        let jitter = Double.random(in: 0.1...0.8)
        retryAttempt += 1
        DispatchQueue.main.asyncAfter(deadline: .now() + backoff + jitter) { [weak self] in
            self?.connect()
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
