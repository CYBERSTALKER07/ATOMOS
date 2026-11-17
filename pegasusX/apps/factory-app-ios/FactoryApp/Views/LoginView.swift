import SwiftUI

struct LoginView: View {
    private enum Field: Hashable {
        case phone
        case otp
        case password
    }

    @Environment(TokenStore.self) private var tokenStore
    @State private var vm = LoginViewModel()
    @FocusState private var focusedField: Field?

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            VStack(spacing: LabTheme.spacingXXL) {
                Spacer(minLength: LabTheme.spacingXL)

                VStack(spacing: LabTheme.spacingSM) {
                    Text("mobile_factory.ui.pegasus_factory")
                        .font(.largeTitle.bold())
                    Text(vm.mode == .otp
                         ? "Sign in with your registered phone number."
                         : "Dev login with phone and password when Firebase OTP is unavailable.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }
                .accessibilityElement(children: .combine)

                VStack(spacing: LabTheme.spacingLG) {
                    TextField("common.field.phone", text: $vm.phone)
                        .textContentType(.telephoneNumber)
                        .keyboardType(.phonePad)
                        .textFieldStyle(.roundedBorder)
                        .frame(minHeight: 44)
                        .focused($focusedField, equals: .phone)
                        .submitLabel(vm.mode == .passwordDev ? .next : .send)
                        .disabled(vm.loading || (vm.mode == .otp && vm.otpSent))
                        .accessibilityLabel("Phone number")

                    if vm.mode == .otp, vm.otpSent {
                        TextField("mobile_factory.ui.verification_code", text: $vm.otpCode)
                            .textContentType(.oneTimeCode)
                            .keyboardType(.numberPad)
                            .textFieldStyle(.roundedBorder)
                            .frame(minHeight: 44)
                            .focused($focusedField, equals: .otp)
                            .submitLabel(.go)
                            .disabled(vm.loading)
                            .onChange(of: vm.otpCode) { _, newValue in
                                vm.sanitizeOtp(newValue)
                            }
                            .accessibilityLabel("Six-digit verification code")
                    }

                    if vm.mode == .passwordDev {
                        SecureField("Password", text: $vm.password)
                            .textContentType(.password)
                            .textFieldStyle(.roundedBorder)
                            .frame(minHeight: 44)
                            .focused($focusedField, equals: .password)
                            .submitLabel(.go)
                            .disabled(vm.loading)
                            .accessibilityLabel("Password")
                    }
                }
                .frame(maxWidth: 360)

                if let error = vm.error {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(.red)
                        .multilineTextAlignment(.center)
                        .accessibilityLabel("Sign in error: \(error)")
                }

                VStack(spacing: LabTheme.spacingSM) {
                    if vm.mode == .otp {
                        if !vm.otpSent {
                            Button {
                                Task { await vm.sendOtp() }
                            } label: {
                                Text("mobile_factory.ui.send_code")
                                    .frame(maxWidth: 360, minHeight: 44)
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.primary)
                            .disabled(!vm.canSendOtp)
                            .accessibilityLabel("Send verification code")
                        } else {
                            Button {
                                Task { await vm.verifyOtp(onAuthenticated: storeAuth) }
                            } label: {
                                Text("common.action.sign_in")
                                    .frame(maxWidth: 360, minHeight: 44)
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.primary)
                            .disabled(!vm.canVerifyOtp)
                            .accessibilityLabel("Sign in with verification code")

                            Button("mobile_factory.ui.resend_code") {
                                vm.resetForResend()
                                focusedField = .phone
                            }
                            .font(.caption)
                            .frame(minWidth: 44, minHeight: 44)
                            .disabled(vm.loading)
                            .accessibilityLabel("Resend verification code")
                        }
                    } else {
                        Button {
                            Task { await vm.passwordLogin(onAuthenticated: storeAuth) }
                        } label: {
                            Text("common.action.sign_in")
                                .frame(maxWidth: 360, minHeight: 44)
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.primary)
                        .disabled(!vm.canPasswordLogin)
                        .accessibilityLabel("Sign in with password")
                    }

                    Button(vm.mode == .otp ? "Use password (dev)" : "Use phone OTP") {
                        vm.switchMode()
                        focusedField = .phone
                    }
                    .font(.caption)
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(vm.loading)
                }

                Spacer(minLength: LabTheme.spacingXL)
            }
            .frame(maxWidth: .infinity)
            .padding()
        }
        .scrollDismissesKeyboard(.interactively)
        .defaultFocus($focusedField, .phone)
        .onAppear {
            vm.configure()
            focusedField = .phone
        }
        .onChange(of: vm.otpSent) { _, sent in
            focusedField = sent ? .otp : .phone
        }
        .onSubmit {
            switch focusedField {
            case .phone:
                if vm.mode == .passwordDev {
                    focusedField = .password
                } else if vm.canSendOtp {
                    Task { await vm.sendOtp() }
                }
            case .otp:
                if vm.canVerifyOtp {
                    Task { await vm.verifyOtp(onAuthenticated: storeAuth) }
                }
            case .password:
                if vm.canPasswordLogin {
                    Task { await vm.passwordLogin(onAuthenticated: storeAuth) }
                }
            case nil:
                break
            }
        }
        .overlay {
            if vm.loading {
                FactoryLoadingState(
                    title: vm.otpSent ? "Signing in" : "Sending code",
                    message: "Connecting to Pegasus..."
                )
            }
        }
    }

    private func storeAuth(_ auth: AuthResponse) {
        tokenStore.store(auth: auth)
        Task { await PushNotificationManager.shared.uploadStoredTokenIfPossible() }
    }
}
