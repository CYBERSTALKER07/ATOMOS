//
//  LoginView.swift
//  payload-app-ios
//

import SwiftUI

struct LoginView: View {
    @State private var viewModel = LoginViewModel()
    @FocusState private var focus: Field?

    private enum Field: Hashable { case phone, otp, pin }

    var body: some View {
        @Bindable var vm = viewModel
        ZStack {
            TermTheme.bg.ignoresSafeArea()
            ScrollView {
                VStack(spacing: TermTheme.s24) {
                    Image(systemName: "box.truck.fill")
                        .font(.system(.largeTitle))
                        .imageScale(.large)
                        .foregroundStyle(TermTheme.accent)
                        .padding(.top, TermTheme.s32)
                        .accessibilityHidden(true)

                    VStack(spacing: TermTheme.s8) {
                        Text(vm.mode == .otp ? "PAYLOAD_AUTH" : "DEV_AUTH")
                            .font(.system(.title2, design: .monospaced, weight: .black))
                            .foregroundStyle(TermTheme.accent)
                        Text(vm.mode == .otp
                             ? "Sign in with warehouse phone OTP."
                             : "Dev login with phone and PIN.")
                            .font(.system(.footnote, design: .monospaced, weight: .medium))
                            .foregroundStyle(TermTheme.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .accessibilityElement(children: .combine)
                    .padding(.bottom, TermTheme.s8)

                    VStack(alignment: .leading, spacing: TermTheme.s16) {
                        VStack(alignment: .leading, spacing: TermTheme.s8) {
                            Text("PHONE")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .foregroundStyle(TermTheme.tertiary)
                            TextField("mobile_payload.ui.998", text: $vm.phone)
                                .keyboardType(.phonePad)
                                .textContentType(.telephoneNumber)
                                .focused($focus, equals: .phone)
                                .font(.system(.headline, design: .monospaced, weight: .bold))
                                .padding(TermTheme.s16)
                                .frame(minHeight: 44)
                                .background(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                                .overlay {
                                    RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous)
                                        .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                                }
                                .disabled(vm.loading || (vm.mode == .otp && vm.otpSent))
                                .accessibilityLabel("Phone number")
                        }

                        if vm.mode == .otp, vm.otpSent {
                            VStack(alignment: .leading, spacing: TermTheme.s8) {
                                Text("mobile_payload.ui.verification_code")
                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                    .foregroundStyle(TermTheme.tertiary)
                                TextField("000000", text: $vm.otpCode)
                                    .onChange(of: vm.otpCode) { _, next in
                                        vm.setOtp(next)
                                    }
                                    .keyboardType(.numberPad)
                                    .textContentType(.oneTimeCode)
                                    .focused($focus, equals: .otp)
                                    .font(.system(.title2, design: .monospaced, weight: .black))
                                    .padding(TermTheme.s16)
                                    .frame(minHeight: 44)
                                    .background(TermTheme.card)
                                    .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                                    .overlay {
                                        RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous)
                                            .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                                    }
                                    .disabled(vm.loading)
                                    .accessibilityLabel("Six digit verification code")
                            }
                        }

                        if vm.mode == .pinDev {
                            VStack(alignment: .leading, spacing: TermTheme.s8) {
                                Text("PIN")
                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                    .foregroundStyle(TermTheme.tertiary)
                                SecureField("••••••", text: $vm.pin)
                                    .onChange(of: vm.pin) { _, next in
                                        vm.setPin(next)
                                    }
                                    .keyboardType(.numberPad)
                                    .textContentType(.password)
                                    .focused($focus, equals: .pin)
                                    .font(.system(.title2, design: .monospaced, weight: .black))
                                    .padding(TermTheme.s16)
                                    .frame(minHeight: 44)
                                    .background(TermTheme.card)
                                    .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                                    .overlay {
                                        RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous)
                                            .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                                    }
                                    .disabled(vm.loading)
                                    .accessibilityLabel("Security PIN")
                            }
                        }
                    }
                    .padding(TermTheme.s20)
                    .tacticalCard()

                    if let error = vm.error {
                        PayloadStateView(
                            variant: .warning,
                            title: "LOGIN_FAILED",
                            message: error,
                            compact: true,
                            tone: .warning
                        )
                    }

                    if vm.mode == .otp {
                        if !vm.otpSent {
                            Button {
                                PegasusHaptics.medium()
                                Task { await vm.sendOtp() }
                            } label: {
                                primaryButtonLabel("SEND CODE")
                            }
                            .disabled(vm.loading || vm.phone.trimmingCharacters(in: .whitespaces).isEmpty)
                            .buttonStyle(.tactical)
                        } else {
                            Button {
                                PegasusHaptics.medium()
                                Task { await vm.verifyOtp() }
                            } label: {
                                primaryButtonLabel("VERIFY & SIGN IN")
                            }
                            .disabled(vm.loading || vm.otpCode.count < 6)
                            .buttonStyle(.tactical)

                            Button {
                                PegasusHaptics.light()
                                Task { await vm.sendOtp() }
                            } label: {
                                Text("mobile_payload.ui.resend_code")
                                    .font(.system(.footnote, design: .monospaced, weight: .bold))
                                    .frame(maxWidth: .infinity, minHeight: 44)
                            }
                            .disabled(vm.loading)
                            .buttonStyle(.tactical)
                        }
                    } else {
                        Button {
                            PegasusHaptics.medium()
                            Task { await vm.submitPin() }
                        } label: {
                            primaryButtonLabel("SIGN IN WITH PIN")
                        }
                        .disabled(vm.loading || !vm.canSubmitPin)
                        .buttonStyle(.tactical)
                    }

                    Button {
                        PegasusHaptics.selection()
                        vm.setMode(vm.mode == .otp ? .pinDev : .otp)
                        focus = .phone
                    } label: {
                        Text(vm.mode == .otp ? "USE PIN (DEV)" : "USE PHONE OTP")
                            .font(.system(.caption, design: .monospaced, weight: .bold))
                            .foregroundStyle(TermTheme.secondary)
                            .frame(minWidth: 44, minHeight: 44)
                    }
                    .disabled(vm.loading)
                }
                .padding(TermTheme.s32)
                .frame(maxWidth: 480)
                .frame(maxWidth: .infinity, alignment: .center)
            }
            .scrollIndicators(.hidden)
            .scrollDismissesKeyboard(.interactively)
        }
        .defaultFocus($focus, .phone)
        .overlay {
            if vm.loading {
                Color.black.opacity(0.4).ignoresSafeArea()
                PayloadLoadingView(title: "AUTHENTICATING", message: "Verifying credentials with dispatch.")
                    .background(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusMD, style: .continuous))
                    .padding(32)
            }
        }
        .onAppear {
            FirebaseAuthHelper.shared.configure()
            focus = .phone
        }
        .onChange(of: vm.otpSent) { _, sent in
            focus = sent ? .otp : .phone
        }
        .onSubmit {
            switch focus {
            case .phone:
                if vm.mode == .pinDev {
                    focus = .pin
                } else if !vm.phone.trimmingCharacters(in: .whitespaces).isEmpty {
                    Task { await vm.sendOtp() }
                }
            case .otp:
                if vm.otpCode.count == 6 {
                    Task { await vm.verifyOtp() }
                }
            case .pin:
                if vm.canSubmitPin {
                    Task { await vm.submitPin() }
                }
            case nil:
                break
            }
        }
    }

    @ViewBuilder
    private func primaryButtonLabel(_ title: String) -> some View {
        HStack(spacing: TermTheme.s12) {
            Image(systemName: "lock.shield.fill")
            Text(title)
                .font(.system(.callout, design: .monospaced, weight: .black))
        }
        .frame(maxWidth: .infinity, minHeight: 52)
        .background(TermTheme.accent)
        .foregroundStyle(TermTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
    }
}

#Preview { LoginView() }
