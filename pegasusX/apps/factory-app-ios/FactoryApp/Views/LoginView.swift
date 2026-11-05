import SwiftUI

private enum LoginMode {
    case otp
    case passwordDev
}

struct LoginView: View {
    private enum Field: Hashable {
        case phone
        case otp
        case password
    }

    @Environment(TokenStore.self) private var tokenStore
    @State private var mode: LoginMode = .otp
    @State private var phone = "+998"
    @State private var otpCode = ""
    @State private var password = ""
    @State private var otpSent = false
    @State private var loading = false
    @State private var error: String?
    @FocusState private var focusedField: Field?

    var body: some View {
        VStack(spacing: LabTheme.spacingXXL) {
            Spacer()

            VStack(spacing: LabTheme.spacingSM) {
                Text("mobile_factory.ui.pegasus_factory")
                    .font(.largeTitle.bold())
                Text(mode == .otp
                     ? "Sign in with your registered phone number."
                     : "Dev login with phone and password when Firebase OTP is unavailable.")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
            }

            VStack(spacing: LabTheme.spacingLG) {
                TextField("common.field.phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                    .textFieldStyle(.roundedBorder)
                    .focused($focusedField, equals: .phone)
                    .submitLabel(mode == .passwordDev ? .next : .send)
                    .disabled(loading || (mode == .otp && otpSent))

                if mode == .otp, otpSent {
                    TextField("mobile_factory.ui.verification_code", text: $otpCode)
                        .textContentType(.oneTimeCode)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.roundedBorder)
                        .focused($focusedField, equals: .otp)
                        .submitLabel(.go)
                        .disabled(loading)
                        .onChange(of: otpCode) { _, newValue in
                            let filtered = newValue.filter(\.isNumber)
                            otpCode = String(filtered.prefix(6))
                        }
                }

                if mode == .passwordDev {
                    SecureField("Password", text: $password)
                        .textContentType(.password)
                        .textFieldStyle(.roundedBorder)
                        .focused($focusedField, equals: .password)
                        .submitLabel(.go)
                        .disabled(loading)
                }
            }
            .frame(maxWidth: 360)

            if let error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .multilineTextAlignment(.center)
            }

            VStack(spacing: LabTheme.spacingSM) {
                if mode == .otp {
                    if !otpSent {
                        Button {
                            sendOtp()
                        } label: {
                            Text("mobile_factory.ui.send_code")
                                .frame(maxWidth: 360, minHeight: 44)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.primary)
                        .disabled(loading || phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    } else {
                        Button {
                            verifyOtp()
                        } label: {
                            Text("common.action.sign_in")
                                .frame(maxWidth: 360, minHeight: 44)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.primary)
                        .disabled(loading || otpCode.count < 6)

                        Button("mobile_factory.ui.resend_code") {
                            otpSent = false
                            otpCode = ""
                            FirebaseAuthHelper.shared.resetFlow()
                            focusedField = .phone
                        }
                        .font(.caption)
                        .disabled(loading)
                    }
                } else {
                    Button {
                        passwordLogin()
                    } label: {
                        Text("common.action.sign_in")
                            .frame(maxWidth: 360, minHeight: 44)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.primary)
                    .disabled(loading || phone.isEmpty || password.isEmpty)
                }

                Button(mode == .otp ? "Use password (dev)" : "Use phone OTP") {
                    switchMode()
                }
                .font(.caption)
                .disabled(loading)
            }

            Spacer()
        }
        .padding()
        .defaultFocus($focusedField, .phone)
        .onAppear {
            FirebaseAuthHelper.shared.configure()
            focusedField = .phone
        }
        .onChange(of: otpSent) { _, sent in
            focusedField = sent ? .otp : .phone
        }
        .onSubmit {
            switch focusedField {
            case .phone:
                if mode == .passwordDev {
                    focusedField = .password
                } else if !phone.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    sendOtp()
                }
            case .otp:
                if otpCode.count == 6 {
                    verifyOtp()
                }
            case .password:
                if !phone.isEmpty && !password.isEmpty {
                    passwordLogin()
                }
            case nil:
                break
            }
        }
        .overlay {
            if loading {
                FactoryLoadingState(
                    title: otpSent ? "Signing in" : "Sending code",
                    message: "Connecting to Pegasus..."
                )
            }
        }
    }

    private func storeAuth(_ auth: AuthResponse) {
        tokenStore.store(auth: auth)
        Task { await PushNotificationManager.shared.uploadStoredTokenIfPossible() }
    }

    private func switchMode() {
        mode = mode == .otp ? .passwordDev : .otp
        error = nil
        otpSent = false
        otpCode = ""
        password = ""
        FirebaseAuthHelper.shared.resetFlow()
        focusedField = .phone
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
                let auth = try await FactoryService.login(idToken: idToken)
                storeAuth(auth)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func passwordLogin() {
        loading = true
        error = nil
        Task {
            do {
                let auth = try await FactoryService.login(phone: phone, password: password)
                storeAuth(auth)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
