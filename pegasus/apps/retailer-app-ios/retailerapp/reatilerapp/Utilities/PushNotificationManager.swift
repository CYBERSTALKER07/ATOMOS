import Foundation
import UserNotifications
import UIKit

// MARK: - Push Notification Manager

@Observable
final class PushNotificationManager: NSObject, UNUserNotificationCenterDelegate {
    static let shared = PushNotificationManager()

    var isAuthorized = false
    var deviceToken: String?
    var errorMessage: String?

    private let api = APIClient.shared

    private override init() {
        super.init()
        UNUserNotificationCenter.current().delegate = self
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
        do {
            let center = UNUserNotificationCenter.current()
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            isAuthorized = granted

            if granted {
                await MainActor.run {
                    UIApplication.shared.registerForRemoteNotifications()
                }
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    // MARK: - Handle Device Token

    func didRegisterForRemoteNotifications(deviceToken: Data) {
        let token = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        self.deviceToken = token
        Task {
            await sendTokenToServer(token: token)
        }
    }

    func didFailToRegisterForRemoteNotifications(error: Error) {
        errorMessage = error.localizedDescription
    }

    // MARK: - Send Token to Server

    private func sendTokenToServer(token: String) async {
        let payload = DeviceTokenPayload(
            token: token,
            platform: "ios",
            retailerId: AuthManager.shared.currentUser?.id ?? ""
        )
        do {
            let _: APIResponse<String> = try await api.post(path: "/v1/user/device-token", body: payload)
        } catch {
            errorMessage = error.localizedDescription
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
