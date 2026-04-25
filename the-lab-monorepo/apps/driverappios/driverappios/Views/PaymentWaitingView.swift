//
//  PaymentWaitingView.swift
//  driverappios
//
//  Shown after driver confirms offload for Payme orders.
//  Connects to /v1/ws/driver and waits for PAYMENT_SETTLED push.
//  Once settled, driver can tap "Complete" to finalize delivery.
//

import SwiftUI

struct PaymentWaitingView: View {

    let orderId: String
    let amount: Int
    let driverId: String
    let onCompleted: () -> Void

    @State private var isSettled = false
    @State private var isCompleting = false
    @State private var errorMessage: String?
    @State private var webSocketTask: URLSessionWebSocketTask?

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // MARK: - Status Icon
            Image(systemName: isSettled ? "checkmark.seal.fill" : "clock.fill")
                .font(.system(size: 64))
                .foregroundStyle(isSettled ? LabTheme.success : LabTheme.warning)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Title
            Text(isSettled ? "Payment Received" : "Awaiting Payment")
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            // MARK: - Order ID
            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Amount
            Text(amount.formattedAmount)
                .font(.system(size: 36, weight: .bold, design: .monospaced))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s24)

            if !isSettled {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.bottom, LabTheme.s8)
                Text("Retailer is completing payment via Payme...")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .multilineTextAlignment(.center)
            }

            Spacer()

            // MARK: - Error
            if let error = errorMessage {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Complete Button
            Button {
                completeDelivery()
            } label: {
                Text("Complete Delivery")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(isSettled ? LabTheme.buttonFg : LabTheme.fgTertiary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(
                        isSettled ? LabTheme.fg : LabTheme.fg.opacity(0.08),
                        in: .rect(cornerRadius: LabTheme.buttonRadius)
                    )
            }
            .disabled(!isSettled || isCompleting)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(LabTheme.bg)
        .task {
            connectWebSocket()
        }
        .onDisappear {
            webSocketTask?.cancel(with: .goingAway, reason: nil)
        }
    }

    // MARK: - WebSocket

    private func connectWebSocket() {
        let baseURL = APIClient.shared.apiBaseURL
        let wsURL = baseURL
            .replacingOccurrences(of: "https://", with: "wss://")
            .replacingOccurrences(of: "http://", with: "ws://")
        guard let url = URL(string: "\(wsURL)/v1/ws/driver"),
              let token = TokenStore.shared.token else { return }

        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let task = URLSession.shared.webSocketTask(with: request)
        webSocketTask = task
        task.resume()
        listenForMessages()
    }

    private func listenForMessages() {
        webSocketTask?.receive { [self] result in
            switch result {
            case .success(let message):
                if case .string(let text) = message,
                   let data = text.data(using: .utf8) {
                    handleWSMessage(data)
                }
                listenForMessages()
            case .failure:
                // Reconnect after delay
                DispatchQueue.main.asyncAfter(deadline: .now() + 3) {
                    connectWebSocket()
                }
            }
        }
    }

    private func handleWSMessage(_ data: Data) {
        struct WSPayload: Decodable {
            let type: String
            let order_id: String?
        }
        guard let payload = try? JSONDecoder().decode(WSPayload.self, from: data),
              payload.type == "PAYMENT_SETTLED",
              payload.order_id == orderId else { return }

        DispatchQueue.main.async {
            isSettled = true
            Haptics.success()
        }
    }

    // MARK: - Complete

    private func completeDelivery() {
        isCompleting = true
        errorMessage = nil
        Task {
            do {
                try await FleetServiceLive.shared.completeOrder(orderId: orderId)
                Haptics.success()
                onCompleted()
            } catch {
                isCompleting = false
                errorMessage = error.localizedDescription
            }
        }
    }
}
