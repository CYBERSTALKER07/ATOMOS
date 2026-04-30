//
//  ShopClosedWaitingView.swift
//  driverappios
//
//  Shown when driver reports shop closed.
//  Connects to /v1/ws/driver and waits for retailer response or bypass token.
//

import SwiftUI

struct ShopClosedWaitingView: View {

    let orderId: String
    let driverId: String
    let onResolved: () -> Void
    let onCancel: () -> Void

    @State private var isReporting = true
    @State private var reportError: String?
    @State private var retailerResponse: String?
    @State private var bypassToken: String?
    @State private var bypassInput = ""
    @State private var isSubmittingBypass = false
    @State private var bypassError: String?
    @State private var isEscalated = false
    @State private var countdown: Int = 180
    @State private var webSocketTask: URLSessionWebSocketTask?
    @State private var timer: Timer?

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // MARK: - Status Icon
            Image(systemName: statusIcon)
                .font(.system(size: 64))
                .foregroundStyle(statusColor)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Title
            Text(statusTitle)
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            // MARK: - Order ID
            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            // MARK: - Countdown / Response
            if isReporting {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.bottom, LabTheme.s8)
                Text("Reporting shop closed...")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else if let response = retailerResponse {
                retailerResponseView(response)
            } else if isEscalated {
                Text("Escalated to supplier. Awaiting resolution.")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(LabTheme.warning)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)
            } else {
                // Countdown timer
                Text(countdownFormatted)
                    .font(.system(size: 48, weight: .bold, design: .monospaced))
                    .foregroundStyle(countdown <= 30 ? LabTheme.destructive : LabTheme.fg)
                    .padding(.bottom, LabTheme.s8)

                Text("Waiting for retailer response...")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            }

            // MARK: - Bypass Token
            if let token = bypassToken {
                VStack(spacing: 12) {
                    Divider().padding(.horizontal, LabTheme.s24)

                    Text("Bypass Token Issued")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(LabTheme.success)

                    Text(token)
                        .font(.system(size: 32, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                        .tracking(8)

                    TextField("Enter token", text: $bypassInput)
                        .font(.system(size: 18, weight: .semibold, design: .monospaced))
                        .multilineTextAlignment(.center)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.roundedBorder)
                        .frame(maxWidth: 200)

                    if let err = bypassError {
                        Text(err)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(LabTheme.destructive)
                    }

                    Button {
                        submitBypass()
                    } label: {
                        HStack(spacing: 8) {
                            if isSubmittingBypass {
                                ProgressView().tint(LabTheme.buttonFg)
                            }
                            Text("Confirm Bypass")
                                .font(.system(size: 15, weight: .bold))
                        }
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .disabled(bypassInput.count != 6 || isSubmittingBypass)
                    .padding(.horizontal, LabTheme.s24)
                }
                .padding(.top, LabTheme.s16)
            }

            Spacer()

            // MARK: - Report Error
            if let error = reportError {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            // MARK: - Back Button
            Button {
                onCancel()
            } label: {
                Text("Back")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LabTheme.fg.opacity(0.08), in: .rect(cornerRadius: LabTheme.buttonRadius))
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(LabTheme.bg)
        .task {
            await reportShopClosed()
            connectWebSocket()
            startCountdown()
        }
        .onDisappear {
            webSocketTask?.cancel(with: .goingAway, reason: nil)
            timer?.invalidate()
        }
    }

    // MARK: - Computed

    private var statusIcon: String {
        if isReporting { return "clock.fill" }
        if retailerResponse != nil { return "bubble.left.fill" }
        if isEscalated { return "exclamationmark.triangle.fill" }
        if bypassToken != nil { return "key.fill" }
        return "door.left.hand.closed"
    }

    private var statusColor: Color {
        if retailerResponse != nil { return LabTheme.success }
        if isEscalated { return LabTheme.warning }
        if countdown <= 30 { return LabTheme.destructive }
        return Color.orange
    }

    private var statusTitle: String {
        if isReporting { return "Reporting..." }
        if retailerResponse != nil { return "Retailer Responded" }
        if isEscalated { return "Escalated" }
        return "Shop Closed"
    }

    private var countdownFormatted: String {
        let m = countdown / 60
        let s = countdown % 60
        return String(format: "%d:%02d", m, s)
    }

    // MARK: - Retailer Response View

    @ViewBuilder
    private func retailerResponseView(_ response: String) -> some View {
        let (label, icon): (String, String) = switch response {
        case "OPEN_NOW": ("Retailer says they are open now", "door.left.hand.open")
        case "5_MIN": ("Retailer will be ready in 5 minutes", "clock.badge.checkmark")
        case "CALL_ME": ("Retailer requests a phone call", "phone.fill")
        case "CLOSED_TODAY": ("Retailer confirmed closed today", "xmark.circle.fill")
        default: ("Response: \(response)", "questionmark.circle")
        }

        VStack(spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 28))
                .foregroundStyle(LabTheme.fg)
            Text(label)
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(LabTheme.fg)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
        .padding(.vertical, LabTheme.s16)
    }

    // MARK: - API Calls

    private func reportShopClosed() async {
        do {
            _ = try await APIClient.shared.reportShopClosed(orderId: orderId)
            isReporting = false
        } catch {
            isReporting = false
            reportError = "Failed to report: \(error.localizedDescription)"
        }
    }

    private func submitBypass() {
        isSubmittingBypass = true
        bypassError = nil
        Task {
            do {
                _ = try await APIClient.shared.bypassOffload(orderId: orderId, token: bypassInput)
                Haptics.success()
                onResolved()
            } catch {
                isSubmittingBypass = false
                bypassError = error.localizedDescription
            }
        }
    }

    // MARK: - Countdown

    private func startCountdown() {
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { _ in
            if countdown > 0 {
                countdown -= 1
            } else {
                timer?.invalidate()
            }
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
            let response: String?
            let bypass_token: String?
            let attempt_id: String?
        }

        guard let payload = try? JSONDecoder().decode(WSPayload.self, from: data),
              payload.order_id == orderId else { return }

        DispatchQueue.main.async {
            switch payload.type {
            case "SHOP_CLOSED_RESPONSE":
                retailerResponse = payload.response
                Haptics.medium()
            case "BYPASS_TOKEN_ISSUED":
                bypassToken = payload.bypass_token
                Haptics.medium()
            case "SHOP_CLOSED_ESCALATED":
                isEscalated = true
                Haptics.warning()
            default:
                break
            }
        }
    }
}
