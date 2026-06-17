//
//  LoginView.swift
//  driverappios
//

import SwiftUI

struct LoginView: View {
    let onAuthenticated: () -> Void

    @State private var viewModel = LoginViewModel()
    @FocusState private var focusedField: Field?

    private enum Field { case phone, otp, pin }

    var body: some View {
        ZStack {
            LabTheme.bg.ignoresSafeArea()

            ScrollView {
                VStack(spacing: LabTheme.s24) {
                    Spacer().frame(height: 72)

                    VStack(spacing: LabTheme.s8) {
                        Image(systemName: "shippingbox.fill")
                            .font(.system(size: 48, weight: .medium))
                            .foregroundStyle(LabTheme.fg)
                        Text("PEGASUS DRIVER")
                            .font(.system(size: 12, weight: .black, design: .monospaced))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .tracking(2)
                        Text("TERMINAL ACCESS")
                            .font(.system(size: 28, weight: .black, design: .monospaced))
                            .foregroundStyle(LabTheme.fg)
                        Text(viewModel.mode == .otp
                             ? "Sign in with fleet phone OTP."
                             : "Dev login with phone and 6-digit PIN.")
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding(.bottom, LabTheme.s8)

                    VStack(alignment: .leading, spacing: LabTheme.s16) {
                        VStack(alignment: .leading, spacing: LabTheme.s8) {
                            Text("PHONE")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(LabTheme.fgTertiary)
                            HStack(spacing: LabTheme.s12) {
                                Image(systemName: "phone.fill")
                                    .font(.system(size: 14))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                TextField("+998 …", text: $viewModel.phone)
                                    .keyboardType(.phonePad)
                                    .textContentType(.telephoneNumber)
                                    .focused($focusedField, equals: .phone)
                                    .font(.system(size: 18, weight: .bold, design: .monospaced))
                                    .foregroundStyle(LabTheme.fg)
                            }
                            .padding(LabTheme.s16)
                            .background(LabTheme.card, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                                    .stroke(
                                        focusedField == .phone ? LabTheme.fg.opacity(0.3) : LabTheme.separator,
                                        lineWidth: 0.5
                                    )
                            }
                            .disabled(viewModel.loading || (viewModel.mode == .otp && viewModel.otpSent))
                        }

                        if viewModel.mode == .otp, viewModel.otpSent {
                            VStack(alignment: .leading, spacing: LabTheme.s8) {
                                Text("VERIFICATION CODE")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                TextField("000000", text: Binding(
                                    get: { viewModel.otpCode },
                                    set: { viewModel.setOtp($0) }
                                ))
                                .keyboardType(.numberPad)
                                .textContentType(.oneTimeCode)
                                .focused($focusedField, equals: .otp)
                                .font(.system(size: 24, weight: .black, design: .monospaced))
                                .foregroundStyle(LabTheme.fg)
                                .padding(LabTheme.s16)
                                .background(LabTheme.card, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
                                .overlay {
                                    RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                                        .stroke(
                                            focusedField == .otp ? LabTheme.fg.opacity(0.3) : LabTheme.separator,
                                            lineWidth: 0.5
                                        )
                                }
                                .disabled(viewModel.loading)
                            }
                        }

                        if viewModel.mode == .pinDev {
                            VStack(alignment: .leading, spacing: LabTheme.s8) {
                                Text("6-DIGIT PIN")
                                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                HStack(spacing: LabTheme.s12) {
                                    Image(systemName: "lock.fill")
                                        .font(.system(size: 14))
                                        .foregroundStyle(LabTheme.fgTertiary)
                                    SecureField("••••••", text: Binding(
                                        get: { viewModel.pin },
                                        set: { viewModel.setPin($0) }
                                    ))
                                    .keyboardType(.numberPad)
                                    .textContentType(.oneTimeCode)
                                    .focused($focusedField, equals: .pin)
                                    .font(.system(size: 24, weight: .black, design: .monospaced))
                                    .foregroundStyle(LabTheme.fg)
                                }
                                .padding(LabTheme.s16)
                                .background(LabTheme.card, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
                                .overlay {
                                    RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                                        .stroke(
                                            focusedField == .pin ? LabTheme.fg.opacity(0.3) : LabTheme.separator,
                                            lineWidth: 0.5
                                        )
                                }
                                .disabled(viewModel.loading)

                                PinDots(filled: viewModel.pin.count)
                            }
                        }
                    }
                    .padding(LabTheme.s20)
                    .labCard()

                    if let error = viewModel.error {
                        DriverStateCard(
                            icon: "exclamationmark.triangle.fill",
                            title: "LOGIN_FAILED",
                            message: error
                        )
                        .transition(.opacity.combined(with: .move(edge: .top)))
                    }

                    if viewModel.mode == .otp {
                        if !viewModel.otpSent {
                            Button {
                                Task { await viewModel.sendOtp() }
                            } label: {
                                authButtonLabel("SEND CODE")
                            }
                            .disabled(viewModel.loading || viewModel.phone.trimmingCharacters(in: .whitespaces).isEmpty)
                            .buttonStyle(.pressable)
                        } else {
                            Button {
                                Task { await completeAuth { await viewModel.verifyOtp() } }
                            } label: {
                                authButtonLabel("VERIFY & SIGN IN")
                            }
                            .disabled(viewModel.loading || viewModel.otpCode.count < 6)
                            .buttonStyle(.pressable)

                            Button {
                                Task { await viewModel.sendOtp() }
                            } label: {
                                Text("RESEND CODE")
                                    .font(.system(size: 14, weight: .bold, design: .monospaced))
                                    .frame(maxWidth: .infinity, minHeight: 48)
                            }
                            .disabled(viewModel.loading)
                            .buttonStyle(.pressable)
                        }
                    } else {
                        Button {
                            Task { await completeAuth { await viewModel.submitPin() } }
                        } label: {
                            authButtonLabel("AUTHENTICATE")
                        }
                        .disabled(viewModel.loading || viewModel.phone.trimmingCharacters(in: .whitespaces).count < 5 || viewModel.pin.count < 6)
                        .buttonStyle(.pressable)
                    }

                    Button {
                        viewModel.setMode(viewModel.mode == .otp ? .pinDev : .otp)
                    } label: {
                        Text(viewModel.mode == .otp ? "Use PIN (dev)" : "Use phone OTP")
                            .font(.system(size: 13, weight: .semibold, design: .monospaced))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .disabled(viewModel.loading)

                    Spacer()
                }
                .padding(.horizontal, LabTheme.s32)
                .labReadableWidth()
            }
        }
        .animation(Anim.snappy, value: viewModel.error)
        .onAppear {
            FirebaseAuthHelper.shared.configure()
            focusedField = .phone
        }
    }

    @ViewBuilder
    private func authButtonLabel(_ title: String) -> some View {
        HStack(spacing: LabTheme.s12) {
            if viewModel.loading {
                ProgressView()
                    .tint(LabTheme.buttonFg)
            } else {
                Image(systemName: "lock.shield.fill")
                Text(title)
                    .font(.system(size: 16, weight: .black, design: .monospaced))
            }
        }
        .frame(maxWidth: .infinity, minHeight: 52)
        .foregroundStyle(LabTheme.buttonFg)
        .background(LabTheme.fg, in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous))
    }

    private func completeAuth(_ action: () async -> Void) async {
        focusedField = nil
        await action()
        guard TokenStore.shared.isAuthenticated else { return }
        Haptics.success()
        onAuthenticated()
        if let pushToken = PushNotificationManager.shared.deviceToken
            ?? UserDefaults.standard.string(forKey: "pegasus_push_token"),
           !pushToken.isEmpty {
            try? await APIClient.shared.registerDeviceToken(token: pushToken)
        }
    }
}

private struct PinDots: View {
    let filled: Int

    var body: some View {
        HStack(spacing: LabTheme.s12) {
            ForEach(0..<6, id: \.self) { index in
                Circle()
                    .fill(index < filled ? LabTheme.fg : LabTheme.fgTertiary.opacity(0.25))
                    .frame(width: 10, height: 10)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.top, LabTheme.s4)
        .accessibilityLabel("\(filled) of 6 PIN digits entered")
    }
}

#Preview {
    LoginView(onAuthenticated: {})
}
