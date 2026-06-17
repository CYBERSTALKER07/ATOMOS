//
//  LoginView.swift
//  payload-app-ios
//

import SwiftUI

struct LoginView: View {
    @State private var viewModel = LoginViewModel()
    @FocusState private var focus: Field?

    private enum Field { case phone, otp, pin }

    var body: some View {
        ZStack {
            TermTheme.bg.ignoresSafeArea()
            VStack(spacing: TermTheme.s24) {
                VStack(spacing: TermTheme.s8) {
                    Text("PEGASUS PAYLOAD")
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                        .tracking(2)
                    Text("TERMINAL ACCESS")
                        .font(.system(size: 32, weight: .black, design: .monospaced))
                        .foregroundStyle(TermTheme.accent)
                    Text(viewModel.mode == .otp
                         ? "Sign in with warehouse phone OTP."
                         : "Dev login with phone and PIN.")
                        .font(.system(size: 13, weight: .medium, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.bottom, TermTheme.s8)

                VStack(alignment: .leading, spacing: TermTheme.s16) {
                    VStack(alignment: .leading, spacing: TermTheme.s8) {
                        Text("PHONE")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.tertiary)
                        TextField("+998 …", text: $viewModel.phone)
                            .keyboardType(.phonePad)
                            .textContentType(.telephoneNumber)
                            .focused($focus, equals: .phone)
                            .font(.system(size: 18, weight: .bold, design: .monospaced))
                            .padding(TermTheme.s16)
                            .background(TermTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous)
                                    .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                            }
                            .disabled(viewModel.loading || (viewModel.mode == .otp && viewModel.otpSent))
                    }

                    if viewModel.mode == .otp, viewModel.otpSent {
                        VStack(alignment: .leading, spacing: TermTheme.s8) {
                            Text("VERIFICATION CODE")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(TermTheme.tertiary)
                            TextField("000000", text: Binding(
                                get: { viewModel.otpCode },
                                set: { viewModel.setOtp($0) }
                            ))
                            .keyboardType(.numberPad)
                            .textContentType(.oneTimeCode)
                            .focused($focus, equals: .otp)
                            .font(.system(size: 24, weight: .black, design: .monospaced))
                            .padding(TermTheme.s16)
                            .background(TermTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous)
                                    .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                            }
                            .disabled(viewModel.loading)
                        }
                    }

                    if viewModel.mode == .pinDev {
                        VStack(alignment: .leading, spacing: TermTheme.s8) {
                            Text("PIN")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(TermTheme.tertiary)
                            SecureField("••••••", text: Binding(
                                get: { viewModel.pin },
                                set: { viewModel.setPin($0) }
                            ))
                            .keyboardType(.numberPad)
                            .textContentType(.password)
                            .focused($focus, equals: .pin)
                            .font(.system(size: 24, weight: .black, design: .monospaced))
                            .padding(TermTheme.s16)
                            .background(TermTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous)
                                    .stroke(TermTheme.separator.opacity(0.12), lineWidth: 1)
                            }
                            .disabled(viewModel.loading)
                        }
                    }
                }
                .padding(TermTheme.s20)
                .tacticalCard()

                if let error = viewModel.error {
                    PayloadStateView(
                        variant: .warning,
                        title: "LOGIN_FAILED",
                        message: error,
                        compact: true,
                        tone: .warning
                    )
                }

                if viewModel.mode == .otp {
                    if !viewModel.otpSent {
                        Button {
                            Task { await viewModel.sendOtp() }
                        } label: {
                            primaryButtonLabel("SEND CODE")
                        }
                        .disabled(viewModel.loading || viewModel.phone.trimmingCharacters(in: .whitespaces).isEmpty)
                        .buttonStyle(.tactical)
                    } else {
                        Button {
                            Task { await viewModel.verifyOtp() }
                        } label: {
                            primaryButtonLabel("VERIFY & SIGN IN")
                        }
                        .disabled(viewModel.loading || viewModel.otpCode.count < 6)
                        .buttonStyle(.tactical)

                        Button {
                            Task { await viewModel.sendOtp() }
                        } label: {
                            Text("RESEND CODE")
                                .font(.system(size: 13, weight: .bold, design: .monospaced))
                                .frame(maxWidth: .infinity, minHeight: 44)
                        }
                        .disabled(viewModel.loading)
                        .buttonStyle(.tactical)
                    }
                } else {
                    Button {
                        Task { await viewModel.submitPin() }
                    } label: {
                        primaryButtonLabel("SIGN IN WITH PIN")
                    }
                    .disabled(viewModel.loading || !viewModel.canSubmitPin)
                    .buttonStyle(.tactical)
                }

                Button {
                    viewModel.setMode(viewModel.mode == .otp ? .pinDev : .otp)
                } label: {
                    Text(viewModel.mode == .otp ? "USE PIN (DEV)" : "USE PHONE OTP")
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(TermTheme.secondary)
                }
                .disabled(viewModel.loading)
            }
            .padding(TermTheme.s32)
            .frame(maxWidth: 480)
        }
        .onAppear {
            FirebaseAuthHelper.shared.configure()
            focus = .phone
        }
    }

    @ViewBuilder
    private func primaryButtonLabel(_ title: String) -> some View {
        HStack(spacing: TermTheme.s12) {
            if viewModel.loading {
                ProgressView().tint(TermTheme.card)
            } else {
                Image(systemName: "lock.shield.fill")
                Text(title)
                    .font(.system(size: 16, weight: .black, design: .monospaced))
            }
        }
        .frame(maxWidth: .infinity, minHeight: 52)
        .background(TermTheme.accent)
        .foregroundStyle(TermTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
    }
}

private extension LoginViewModel {
    var canSubmitPin: Bool {
        !phone.trimmingCharacters(in: .whitespaces).isEmpty && pin.count >= 6
    }
}

#Preview { LoginView() }
