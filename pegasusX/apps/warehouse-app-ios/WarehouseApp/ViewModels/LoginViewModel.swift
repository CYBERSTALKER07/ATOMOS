import Foundation
import Observation

@Observable
@MainActor
final class LoginViewModel {
    var phone = ""
    var pin = ""
    var otpCode = ""
    var useOtp = true
    var otpSent = false
    var loading = false
    var error: String?

    private let configureAuthHandler: @MainActor () -> Void
    private let resetFlowHandler: @MainActor () -> Void
    private let sendOtpHandler: @MainActor (String) async throws -> Void
    private let verifySmsHandler: @MainActor (String) async throws -> String
    private let loginIdTokenHandler: @Sendable (String) async throws -> AuthResponse
    private let loginPinHandler: @Sendable (String, String) async throws -> AuthResponse
    private let uploadPushTokenHandler: @MainActor () async -> Void

    init(
        configureAuthHandler: @escaping @MainActor () -> Void = {
            FirebaseAuthHelper.shared.configure()
        },
        resetFlowHandler: @escaping @MainActor () -> Void = {
            FirebaseAuthHelper.shared.resetFlow()
        },
        sendOtpHandler: @escaping @MainActor (String) async throws -> Void = { phone in
            try await FirebaseAuthHelper.shared.sendPhoneVerification(phone: phone)
        },
        verifySmsHandler: @escaping @MainActor (String) async throws -> String = { code in
            try await FirebaseAuthHelper.shared.verifySmsCode(code)
        },
        loginIdTokenHandler: @escaping @Sendable (String) async throws -> AuthResponse = { idToken in
            try await WarehouseService.login(idToken: idToken)
        },
        loginPinHandler: @escaping @Sendable (String, String) async throws -> AuthResponse = { phone, pin in
            try await WarehouseService.login(phone: phone, pin: pin)
        },
        uploadPushTokenHandler: @escaping @MainActor () async -> Void = {
            await PushNotificationManager.shared.uploadStoredTokenIfPossible()
        }
    ) {
        self.configureAuthHandler = configureAuthHandler
        self.resetFlowHandler = resetFlowHandler
        self.sendOtpHandler = sendOtpHandler
        self.verifySmsHandler = verifySmsHandler
        self.loginIdTokenHandler = loginIdTokenHandler
        self.loginPinHandler = loginPinHandler
        self.uploadPushTokenHandler = uploadPushTokenHandler
    }

    var isSubmitDisabled: Bool {
        loading || phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || (!useOtp && pin.isEmpty)
    }

    func configure() {
        configureAuthHandler()
    }

    func toggleAuthMode() {
        useOtp.toggle()
        otpSent = false
        otpCode = ""
        error = nil
        resetFlowHandler()
    }

    func clampOtpCode() {
        if otpCode.count > 6 {
            otpCode = String(otpCode.prefix(6))
        }
    }

    func clampPin() {
        if pin.count > 6 {
            pin = String(pin.prefix(6))
        }
    }

    func sendOtp() async {
        let trimmed = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            error = "Phone number is required."
            return
        }
        loading = true
        error = nil
        defer { loading = false }
        do {
            try await sendOtpHandler(trimmed)
            otpSent = true
            PegasusHaptics.success()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }

    func verifyOtp(tokenStore: TokenStore) async {
        guard !otpCode.isEmpty else {
            error = "Verification code is required."
            return
        }
        loading = true
        error = nil
        defer { loading = false }
        do {
            let idToken = try await verifySmsHandler(otpCode)
            let auth = try await loginIdTokenHandler(idToken)
            tokenStore.store(auth: auth)
            await uploadPushTokenHandler()
            PegasusHaptics.success()
        } catch {
            if let apiError = error as? APIError, case .httpError(404) = apiError {
                self.error = "No account found. Register your warehouse on the web portal."
            } else {
                self.error = error.localizedDescription
            }
            PegasusHaptics.error()
        }
    }

    func loginWithPin(tokenStore: TokenStore) async {
        let trimmedPhone = phone.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedPhone.isEmpty, !pin.isEmpty else {
            error = "Phone and PIN are required."
            return
        }
        loading = true
        error = nil
        defer { loading = false }
        do {
            let auth = try await loginPinHandler(trimmedPhone, pin)
            tokenStore.store(auth: auth)
            await uploadPushTokenHandler()
            PegasusHaptics.success()
        } catch {
            if let apiError = error as? APIError, case .httpError(404) = apiError {
                self.error = "No account found. Register your warehouse on the web portal."
            } else {
                self.error = error.localizedDescription
            }
            PegasusHaptics.error()
        }
    }
}
