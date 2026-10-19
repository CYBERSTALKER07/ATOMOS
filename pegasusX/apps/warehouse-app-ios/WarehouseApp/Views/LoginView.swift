import SwiftUI

struct LoginView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.openURL) private var openURL
    @State private var phone = ""
    @State private var pin = ""
    @State private var otpCode = ""
    @State private var useOtp = true
    @State private var otpSent = false
    @State private var loading = false
    @State private var error: String?

    var body: some View {
        VStack(spacing: LabTheme.spacingXXL) {
            Spacer()

            VStack(spacing: LabTheme.spacingSM) {
                Text("mobile_warehouse.ui.pegasus_warehouse")
                    .font(.largeTitle.bold())
                Text(useOtp ? "Sign in with phone OTP" : "Dev fallback: phone + PIN")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            VStack(spacing: LabTheme.spacingLG) {
                TextField("common.field.phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                    .textFieldStyle(.roundedBorder)

                if useOtp {
                    if otpSent {
                        TextField("mobile_warehouse.ui.verification_code", text: $otpCode)
                            .textContentType(.oneTimeCode)
                            .keyboardType(.numberPad)
                            .textFieldStyle(.roundedBorder)
                            .onChange(of: otpCode) {
                                if otpCode.count > 6 { otpCode = String(otpCode.prefix(6)) }
                            }
                    }
                } else {
                    SecureField("PIN", text: $pin)
                        .textContentType(.oneTimeCode)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.roundedBorder)
                        .onChange(of: pin) {
                            if pin.count > 6 { pin = String(pin.prefix(6)) }
                        }
                }
            }
            .frame(maxWidth: 360)

            if let error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            Button {
                if useOtp {
                    if otpSent {
                        verifyOtp()
                    } else {
                        sendOtp()
                    }
                } else {
                    loginWithPin()
                }
            } label: {
                Group {
                    if loading {
                        ProgressView().tint(.white)
                    } else {
                        Text(useOtp ? (otpSent ? "Verify & Sign In" : "Send code") : "Sign In")
                    }
                }
                .frame(maxWidth: 360, minHeight: 44)
            }
            .buttonStyle(.borderedProminent)
            .tint(.primary)
            .disabled(loading || phone.isEmpty || (!useOtp && pin.isEmpty))

            Button {
                useOtp.toggle()
                otpSent = false
                otpCode = ""
                FirebaseAuthHelper.shared.resetFlow()
            } label: {
                Text(useOtp ? "Use PIN (dev)" : "Use phone OTP")
                    .font(.footnote)
            }

            Button {
                openURL(WarehousePortalLinks.url(for: .register))
            } label: {
                Text("mobile_warehouse.ui.new_warehouse_register_on_the_web_portal")
                    .font(.footnote)
            }
            .padding(.top, LabTheme.spacingSM)

            Spacer()
        }
        .padding()
        .task {
            FirebaseAuthHelper.shared.configure()
        }
    }

    private func sendOtp() {
        loading = true
        error = nil
        Task {
            do {
                try await FirebaseAuthHelper.shared.sendPhoneVerification(phone: phone)
                otpSent = true
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func verifyOtp() {
        loading = true
        error = nil
        Task {
            do {
                let idToken = try await FirebaseAuthHelper.shared.verifySmsCode(otpCode)
                let auth = try await WarehouseService.login(idToken: idToken)
                tokenStore.store(auth: auth)
                await registerPushTokenIfNeeded()
            } catch {
                if let apiError = error as? APIError, case .httpError(404) = apiError {
                    self.error = "No account found. Register your warehouse on the web portal."
                } else {
                    self.error = error.localizedDescription
                }
            }
            loading = false
        }
    }

    private func loginWithPin() {
        loading = true
        error = nil
        Task {
            do {
                let auth = try await WarehouseService.login(phone: phone, pin: pin)
                tokenStore.store(auth: auth)
                await registerPushTokenIfNeeded()
            } catch {
                if let apiError = error as? APIError, case .httpError(404) = apiError {
                    self.error = "No account found. Register your warehouse on the web portal."
                } else {
                    self.error = error.localizedDescription
                }
            }
            loading = false
        }
    }

    private func registerPushTokenIfNeeded() async {
        await PushNotificationManager.shared.uploadStoredTokenIfPossible()
    }
}
