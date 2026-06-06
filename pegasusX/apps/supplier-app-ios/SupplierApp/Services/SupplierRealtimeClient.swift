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

/// Mirrors supplier-portal `/v1/ws?token=` for native supplier row realtime.
final class SupplierRealtimeClient {
    private var task: URLSessionWebSocketTask?
    private var closed = true

    func connect(token: String, onEvent: @escaping (SupplierLiveEvent) -> Void) {
        closed = false
        guard let url = wsURL(token: token) else { return }
        task = URLSession.shared.webSocketTask(with: url)
        task?.resume()
        receiveLoop(onEvent: onEvent)
    }

    func disconnect() {
        closed = true
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
    }

    private func wsURL(token: String) -> URL? {
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
            URLQueryItem(name: "token", value: token),
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
                break
            }
        }
    }
}
