import Foundation
import UserNotifications
import UIKit

// MARK: - Push Notification Manager

/// FCM registration token lifecycle. APNs hex is never POSTed — Admin Messaging
/// send only accepts Firebase registration tokens (`notifications/fcm.go`).
@Observable
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
    static let shared = PushNotificationManager()

    var isAuthorized = false
    var deviceToken: String?
    var errorMessage: String?

    private let api = APIClient.shared
    private let storedTokenKey = "pegasus_push_token"

    private override init() {
        super.init()
    }

    private var isRunningTests: Bool {
        ProcessInfo.processInfo.environment["XCTestConfigurationFilePath"] != nil
    }

    // MARK: - Foreground Push Display

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        [.banner, .sound, .badge]
    }

    // MARK: - Request Authorization

    func requestAuthorization() async {
        if isRunningTests { return }
        do {
            let center = UNUserNotificationCenter.current()
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            isAuthorized = granted
            await MainActor.run {
                UIApplication.shared.registerForRemoteNotifications()
            }
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Notification permission is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry notification setup.",
                fallback: "Notification permission request failed. Please try again.",
            )
        }
    }

    func didFailToRegisterForRemoteNotifications(error: Error) {
        errorMessage = RetailerErrorSupport.message(
            for: error,
            restricted: "Push registration is restricted for this account.",
            offline: "Offline mode active. Reconnect and retry push registration.",
            fallback: "Push registration failed. Please try again.",
        )
    }

    func didReceiveFCMToken(_ token: String?) async {
        let trimmed = token?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else { return }
        deviceToken = trimmed
        UserDefaults.standard.set(trimmed, forKey: storedTokenKey)
        await uploadStoredTokenIfPossible()
    }

    func uploadStoredTokenIfPossible() async {
        if isRunningTests { return }
        let token = (deviceToken ?? UserDefaults.standard.string(forKey: storedTokenKey) ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard !token.isEmpty, api.authToken != nil else { return }
        await sendTokenToServer(token: token)
    }

    private func sendTokenToServer(token: String) async {
        let payload = DeviceTokenPayload(token: token, platform: "ios")
        do {
            let _: APIResponse<String> = try await api.post(path: "/v1/user/device-token", body: payload)
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Device registration is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry device registration.",
                fallback: "Device registration failed. Please try again.",
            )
        }
    }
}

// MARK: - QR Code Generator

enum QRCodeGenerator {
    static func generate(from string: String, size: CGSize = CGSize(width: 200, height: 200)) -> UIImage? {
        guard let data = string.data(using: .ascii),
              let filter = CIFilter(name: "CIQRCodeGenerator") else { return nil }

        filter.setValue(data, forKey: "inputMessage")
        filter.setValue("M", forKey: "inputCorrectionLevel")

        guard let ciImage = filter.outputImage else { return nil }

        let scaleX = size.width / ciImage.extent.width
        let scaleY = size.height / ciImage.extent.height
        let transformedImage = ciImage.transformed(by: CGAffineTransform(scaleX: scaleX, y: scaleY))

        let context = CIContext()
        guard let cgImage = context.createCGImage(transformedImage, from: transformedImage.extent) else { return nil }

        return UIImage(cgImage: cgImage)
    }
}

// MARK: - QR Code SwiftUI View

import SwiftUI

struct QRCodeView: View {
    let data: String
    var size: Double = 160

    var body: some View {
        if let image = QRCodeGenerator.generate(from: data, size: CGSize(width: size, height: size)) {
            Image(uiImage: image)
                .interpolation(.none)
                .resizable()
                .frame(width: size, height: size)
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
        } else {
            RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                .fill(AppTheme.accentSoft)
                .frame(width: size, height: size)
                .overlay {
                    Image(systemName: "qrcode")
                        .font(.largeTitle)
                        .foregroundStyle(AppTheme.textTertiary)
                }
        }
    }
}

#Preview {
    QRCodeView(data: "ORD-001-DELIVERY")
}
