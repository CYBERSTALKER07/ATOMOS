//
//  LoginView.swift
//  payload-app-ios
//

import SwiftUI

struct LoginView: View {
    @State private var viewModel = LoginViewModel()
    @FocusState private var focus: Field?

    private enum Field { case phone, pin }

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
                    Text("Sign in with warehouse phone and 6-digit PIN.")
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
                            .disabled(viewModel.loading)
                    }

                    VStack(alignment: .leading, spacing: TermTheme.s8) {
                        Text("6-DIGIT PIN")
                            .font(.system(size: 10, weight: .bold, design: .monospaced))
                            .foregroundStyle(TermTheme.tertiary)
                        SecureField("••••••", text: Binding(
                            get: { viewModel.pin },
                            set: { viewModel.setPin($0) }
                        ))
                        .keyboardType(.numberPad)
                        .textContentType(.oneTimeCode)
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

                        PinDots(filled: viewModel.pin.count)
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

                Button {
                    Task { await viewModel.submit() }
                } label: {
                    HStack(spacing: TermTheme.s12) {
                        if viewModel.loading {
                            ProgressView().tint(TermTheme.card)
                        } else {
                            Image(systemName: "lock.shield.fill")
                            Text("AUTHENTICATE")
                                .font(.system(size: 16, weight: .black, design: .monospaced))
                        }
                    }
                    .frame(maxWidth: .infinity, minHeight: 52)
                    .background(viewModel.canSubmit ? TermTheme.accent : TermTheme.tertiary.opacity(0.4))
                    .foregroundStyle(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: TermTheme.radiusSM, style: .continuous))
                }
                .disabled(viewModel.loading || !viewModel.canSubmit)
                .buttonStyle(.tactical)
            }
            .padding(TermTheme.s32)
            .frame(maxWidth: 480)
        }
        .onAppear { focus = .phone }
    }
}

private struct PinDots: View {
    let filled: Int

    var body: some View {
        HStack(spacing: TermTheme.s12) {
            ForEach(0..<6, id: \.self) { index in
                Circle()
                    .fill(index < filled ? TermTheme.accent : TermTheme.tertiary.opacity(0.25))
                    .frame(width: 10, height: 10)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.top, TermTheme.s4)
        .accessibilityLabel("\(filled) of 6 PIN digits entered")
    }
}

private extension LoginViewModel {
    var canSubmit: Bool {
        !phone.trimmingCharacters(in: .whitespaces).isEmpty && pin.count == 6
    }
}

#Preview { LoginView() }
