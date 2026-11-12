//
//  LoginView.swift
//  driverappios
//

import SwiftUI

struct LoginView: View {
    let onAuthenticated: () -> Void

    @State private var viewModel = LoginViewModel()
    @FocusState private var focusedField: Field?

    private enum Field: Hashable { case phone, otp, pin }

    var body: some View {
        @Bindable var viewModel = viewModel
        ZStack {
            LabTheme.bg.ignoresSafeArea()

            ScrollView {
                VStack(spacing: LabTheme.s24) {
                    Spacer().frame(height: 72)

                    VStack(spacing: LabTheme.s8) {
                        Image(systemName: "shippingbox.fill")
                            .font(.system(.largeTitle, weight: .medium))
                            .foregroundStyle(LabTheme.fg)
                            .accessibilityHidden(true)
                        Text("mobile_driver.ui.pegasus_driver_2")
                            .font(.system(.caption, design: .monospaced, weight: .black))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .tracking(2)
                        Text("mobile_driver.ui.terminal_access")
                            .font(.system(.title, design: .monospaced, weight: .black))
                            .foregroundStyle(LabTheme.fg)
                        Text(viewModel.mode == .otp
                             ? "Sign in with fleet phone OTP."
                             : "Dev login with phone and 6-digit PIN.")
                            .font(.system(.footnote, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding(.bottom, LabTheme.s8)

                    VStack(alignment: .leading, spacing: LabTheme.s16) {
                        VStack(alignment: .leading, spacing: LabTheme.s8) {
                            Text("PHONE")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .foregroundStyle(LabTheme.fgTertiary)
                            HStack(spacing: LabTheme.s12) {
                                Image(systemName: "phone.fill")
                                    .font(.system(.subheadline))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                TextField("mobile_driver.ui.998", text: $viewModel.phone)
                                    .keyboardType(.phonePad)
                                    .textContentType(.telephoneNumber)
                                    .focused($focusedField, equals: .phone)
                                    .font(.system(.headline, design: .monospaced, weight: .bold))
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
                                Text("mobile_driver.ui.verification_code")
                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                TextField("000000", text: $viewModel.otpCode)
                                    .onChange(of: viewModel.otpCode) { _, newValue in
                                        viewModel.setOtp(newValue)
                                    }
                                    .keyboardType(.numberPad)
                                    .textContentType(.oneTimeCode)
                                    .focused($focusedField, equals: .otp)
                                    .font(.system(.title2, design: .monospaced, weight: .black))
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
                                Text("mobile_driver.ui.6_digit_pin")
                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                HStack(spacing: LabTheme.s12) {
                                    Image(systemName: "lock.fill")
                                        .font(.system(.subheadline))
                                        .foregroundStyle(LabTheme.fgTertiary)
                                    SecureField("••••••", text: $viewModel.pin)
                                        .onChange(of: viewModel.pin) { _, newValue in
                                            viewModel.setPin(newValue)
                                        }
                                        .keyboardType(.numberPad)
                                        .textContentType(.oneTimeCode)
                                        .focused($focusedField, equals: .pin)
                                        .font(.system(.title2, design: .monospaced, weight: .black))
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
                                Haptics.light()
                                Task { await viewModel.sendOtp() }
                            } label: {
                                authButtonLabel("SEND CODE")
                            }
                            .disabled(viewModel.loading || viewModel.phone.trimmingCharacters(in: .whitespaces).isEmpty)
                            .buttonStyle(.pressable)
                        } else {
                            Button {
                                Haptics.light()
                                Task { await completeAuth { await viewModel.verifyOtp() } }
                            } label: {
                                authButtonLabel("VERIFY & SIGN IN")
                            }
                            .disabled(viewModel.loading || viewModel.otpCode.count < 6)
                            .buttonStyle(.pressable)

                            Button {
                                Haptics.light()
                                Task { await viewModel.sendOtp() }
                            } label: {
                                Text("mobile_driver.ui.resend_code")
                                    .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                    .frame(maxWidth: .infinity, minHeight: 48)
                            }
                            .disabled(viewModel.loading)
                            .buttonStyle(.pressable)
                        }
                    } else {
                        Button {
                            Haptics.light()
                            Task { await completeAuth { await viewModel.submitPin() } }
                        } label: {
                            authButtonLabel("AUTHENTICATE")
                        }
                        .disabled(viewModel.loading || viewModel.phone.trimmingCharacters(in: .whitespaces).count < 5 || viewModel.pin.count < 6)
                        .buttonStyle(.pressable)
                    }

                    Button {
                        Haptics.selection()
                        viewModel.setMode(viewModel.mode == .otp ? .pinDev : .otp)
                        focusedField = .phone
                    } label: {
                        Text(viewModel.mode == .otp ? "Use PIN (dev)" : "Use phone OTP")
                            .font(.system(.footnote, design: .monospaced, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .disabled(viewModel.loading)

                    Spacer()
                }
                .padding(.horizontal, LabTheme.s32)
                .labReadableWidth()
            }
            .scrollDismissesKeyboard(.interactively)
        }
        .defaultFocus($focusedField, .phone)
        .animation(Anim.snappy, value: viewModel.error)
        .onAppear {
            FirebaseAuthHelper.shared.configure()
            focusedField = .phone
        }
        .onChange(of: viewModel.otpSent) { _, sent in
            focusedField = sent ? .otp : .phone
        }
        .onSubmit {
            switch focusedField {
            case .phone:
                if viewModel.mode == .pinDev {
                    focusedField = .pin
                } else if !viewModel.phone.trimmingCharacters(in: .whitespaces).isEmpty {
                    Task { await viewModel.sendOtp() }
                }
            case .otp:
                if viewModel.otpCode.count == 6 {
                    Task { await completeAuth { await viewModel.verifyOtp() } }
                }
            case .pin:
                if viewModel.pin.count == 6 {
                    Task { await completeAuth { await viewModel.submitPin() } }
                }
            case nil:
                break
            }
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
                    .font(.system(.callout, design: .monospaced, weight: .black))
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
        await PushNotificationManager.shared.uploadStoredTokenIfPossible()
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
