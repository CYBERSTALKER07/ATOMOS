import Foundation

final class WarehouseRealtimeClient {
    private let session: URLSession
    private let decoder = JSONDecoder()
    private var task: URLSessionWebSocketTask?
    private var closed = false

    init(session: URLSession = .shared) {
        self.session = session
    }

    func connect(onEvent: @escaping @MainActor (WarehouseLiveEvent) -> Void) {
        guard let token = TokenStore.shared.token else { return }
        disconnect()
        closed = false
        let socketURL = APIClient.shared.warehouseWebSocketURL(token: token)
        let socketTask = session.webSocketTask(with: socketURL)
        task = socketTask
        socketTask.resume()
        receive(onEvent: onEvent)
    }

    func disconnect() {
        closed = true
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
    }

    private func receive(onEvent: @escaping @MainActor (WarehouseLiveEvent) -> Void) {
        task?.receive { [weak self] result in
            guard let self else { return }
            if self.closed {
                return
            }

            switch result {
            case .success(let message):
                let payload: Data?
                switch message {
                case .string(let text):
                    payload = text.data(using: .utf8)
                case .data(let data):
                    payload = data
                @unknown default:
                    payload = nil
                }

                if let payload,
                   let event = try? self.decoder.decode(WarehouseLiveEvent.self, from: payload) {
                    Task { @MainActor in
                        onEvent(event)
                    }
                }

                self.receive(onEvent: onEvent)
            case .failure:
                self.disconnect()
            }
        }
    }
}