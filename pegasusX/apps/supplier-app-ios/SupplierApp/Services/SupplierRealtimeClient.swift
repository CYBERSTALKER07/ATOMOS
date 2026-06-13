import Foundation

enum SupplierRealtimeStatus: Equatable {
    case idle
    case live
    case offline
}

struct SupplierLiveEvent: Decodable {
    let type: String
    let minimum_version: String?
}

/// Supplier realtime via short-lived `/v1/supplier/ws-session` tokens (portal parity).
final class SupplierRealtimeClient {
    private var task: URLSessionWebSocketTask?
    private var closed = true
    private var reconnectWorkItem: DispatchWorkItem?

    func connect(onEvent: @escaping (SupplierLiveEvent) -> Void) {
        closed = false
        Task { await openSocket(onEvent: onEvent) }
    }

    func disconnect() {
        closed = true
        reconnectWorkItem?.cancel()
        reconnectWorkItem = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
    }

    private func openSocket(onEvent: @escaping (SupplierLiveEvent) -> Void) async {
        guard !closed else { return }
        do {
            let session = try await SupplierOperationsService.wsSession()
            guard !session.token.isEmpty, let url = wsURL(sessionToken: session.token) else {
                scheduleReconnect(onEvent: onEvent)
                return
            }
            task?.cancel(with: .goingAway, reason: nil)
            task = URLSession.shared.webSocketTask(with: url)
            task?.resume()
            receiveLoop(onEvent: onEvent)
        } catch {
            scheduleReconnect(onEvent: onEvent)
        }
    }

    private func scheduleReconnect(onEvent: @escaping (SupplierLiveEvent) -> Void) {
        guard !closed else { return }
        reconnectWorkItem?.cancel()
        let work = DispatchWorkItem { [weak self] in
            guard let self, !self.closed else { return }
            Task { await self.openSocket(onEvent: onEvent) }
        }
        reconnectWorkItem = work
        DispatchQueue.main.asyncAfter(deadline: .now() + 3, execute: work)
    }

    private func wsURL(sessionToken: String) -> URL? {
        #if DEBUG
        let raw = (ProcessInfo.processInfo.environment["PEGASUS_DEV_HOST"] ?? "").trimmingCharacters(in: .whitespaces)
        let base: String
        if raw.isEmpty { base = "http://localhost:8180" }
        else if raw.hasPrefix("http") { base = raw.trimmingCharacters(in: CharacterSet(charactersIn: "/")) }
        else { base = "http://\(raw):8180" }
        #else
        let base = "https://api.pegasus.uz"
        #endif
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0"
        var components = URLComponents(string: base)!
        components.scheme = components.scheme == "https" ? "wss" : "ws"
        components.path = "/v1/ws"
        components.queryItems = [
            URLQueryItem(name: "token", value: sessionToken),
            URLQueryItem(name: "platform", value: "ios"),
            URLQueryItem(name: "version", value: version),
        ]
        return components.url
    }

    private func receiveLoop(onEvent: @escaping (SupplierLiveEvent) -> Void) {
        task?.receive { [weak self] result in
            guard let self, !self.closed else { return }
            switch result {
            case .success(let message):
                if case .string(let text) = message,
                   let data = text.data(using: .utf8),
                   let event = try? JSONDecoder().decode(SupplierLiveEvent.self, from: data) {
                    onEvent(event)
                }
                self.receiveLoop(onEvent: onEvent)
            case .failure:
                self.scheduleReconnect(onEvent: onEvent)
            }
        }
    }
}
