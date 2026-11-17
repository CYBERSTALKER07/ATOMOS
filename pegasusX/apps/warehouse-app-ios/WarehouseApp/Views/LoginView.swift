import SwiftUI

struct LoginView: View {
    private enum Field: Hashable {
        case phone
        case otp
        case pin
    }

    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.openURL) private var openURL
    @State private var vm = LoginViewModel()
    @FocusState private var focusedField: Field?

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(spacing: LabTheme.spacingXXL) {
                Spacer(minLength: 40)

                VStack(spacing: LabTheme.spacingSM) {
                    Text("mobile_warehouse.ui.pegasus_warehouse")
                        .font(.largeTitle.bold())
                    Text(vm.useOtp ? "Sign in with phone OTP" : "Dev fallback: phone + PIN")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                VStack(spacing: LabTheme.spacingLG) {
                    TextField("common.field.phone", text: $vm.phone)
                        .textContentType(.telephoneNumber)
                        .keyboardType(.phonePad)
                        .textFieldStyle(.roundedBorder)
                        .focused($focusedField, equals: .phone)
                        .submitLabel(vm.useOtp ? .send : .next)

                    if vm.useOtp {
                        if vm.otpSent {
                            TextField("mobile_warehouse.ui.verification_code", text: $vm.otpCode)
                                .textContentType(.oneTimeCode)
                                .keyboardType(.numberPad)
                                .textFieldStyle(.roundedBorder)
                                .focused($focusedField, equals: .otp)
                                .submitLabel(.go)
                                .onChange(of: vm.otpCode) {
                                    vm.clampOtpCode()
                                }
                        }
                    } else {
                        SecureField("PIN", text: $vm.pin)
                            .textContentType(.oneTimeCode)
                            .keyboardType(.numberPad)
                            .textFieldStyle(.roundedBorder)
                            .focused($focusedField, equals: .pin)
                            .submitLabel(.go)
                            .onChange(of: vm.pin) {
                                vm.clampPin()
                            }
                    }
                }
                .frame(maxWidth: 360)

                if let error = vm.error {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(.red)
                }

                Button {
                    PegasusHaptics.light()
                    Task {
                        if vm.useOtp {
                            if vm.otpSent {
                                await vm.verifyOtp(tokenStore: tokenStore)
                            } else {
                                await vm.sendOtp()
                            }
                        } else {
                            await vm.loginWithPin(tokenStore: tokenStore)
                        }
                    }
                } label: {
                    Group {
                        if vm.loading {
                            ProgressView().tint(.white)
                        } else {
                            Text(vm.useOtp ? (vm.otpSent ? "Verify & Sign In" : "Send code") : "Sign In")
                        }
                    }
                    .frame(maxWidth: 360, minHeight: 44)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.borderedProminent)
                .tint(.primary)
                .disabled(vm.isSubmitDisabled)
                .accessibilityLabel(vm.useOtp ? (vm.otpSent ? "Verify code and sign in" : "Send verification code") : "Sign in with PIN")

                Button {
                    PegasusHaptics.selection()
                    vm.toggleAuthMode()
                    focusedField = .phone
                } label: {
                    Text(vm.useOtp ? "Use PIN (dev)" : "Use phone OTP")
                        .font(.footnote)
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel(vm.useOtp ? "Switch to PIN authentication" : "Switch to phone OTP authentication")

                Button {
                    PegasusHaptics.light()
                    openURL(WarehousePortalLinks.url(for: .register))
                } label: {
                    Text("mobile_warehouse.ui.new_warehouse_register_on_the_web_portal")
                        .font(.footnote)
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Register new warehouse on web portal")
                .padding(.top, LabTheme.spacingSM)

                Spacer(minLength: 40)
            }
            .padding()
            .frame(maxWidth: .infinity)
        }
        .scrollDismissesKeyboard(.interactively)
        .defaultFocus($focusedField, .phone)
        .onAppear {
            focusedField = .phone
        }
        .onChange(of: vm.otpSent) { _, sent in
            focusedField = sent ? .otp : .phone
        }
        .onSubmit {
            switch focusedField {
            case .phone:
                if vm.useOtp {
                    if !vm.phone.isEmpty {
                        Task { await vm.sendOtp() }
                    }
                } else {
                    focusedField = .pin
                }
            case .otp:
                if !vm.otpCode.isEmpty {
                    Task { await vm.verifyOtp(tokenStore: tokenStore) }
                }
            case .pin:
                if !vm.phone.isEmpty && !vm.pin.isEmpty {
                    Task { await vm.loginWithPin(tokenStore: tokenStore) }
                }
            case nil:
                break
            }
        }
        .task {
            vm.configure()
        }
    }
}
