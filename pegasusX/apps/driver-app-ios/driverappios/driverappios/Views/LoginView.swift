//
//  LoginView.swift
//  driverappios
//

import SwiftUI

struct LoginView: View {
    let onAuthenticated: () -> Void

    @State private var phone = "+998"
    @State private var pin = ""
    @State private var pinVisible = false
    @State private var isLoading = false
    @State private var error: String?
    @FocusState private var focusedField: Field?

    private enum Field { case phone, pin }

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
                        Text("Sign in with fleet phone and 6-digit PIN.")
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
                                TextField("+998 …", text: $phone)
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
                            .disabled(isLoading)
                        }

                        VStack(alignment: .leading, spacing: LabTheme.s8) {
                            Text("6-DIGIT PIN")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(LabTheme.fgTertiary)
                            HStack(spacing: LabTheme.s12) {
                                Image(systemName: "lock.fill")
                                    .font(.system(size: 14))
                                    .foregroundStyle(LabTheme.fgTertiary)
                                Group {
                                    if pinVisible {
                                        TextField("••••••", text: $pin)
                                    } else {
                                        SecureField("••••••", text: $pin)
                                    }
                                }
                                .keyboardType(.numberPad)
                                .textContentType(.oneTimeCode)
                                .focused($focusedField, equals: .pin)
                                .font(.system(size: 24, weight: .black, design: .monospaced))
                                .foregroundStyle(LabTheme.fg)
                                .onChange(of: pin) { _, newValue in
                                    if newValue.count > 6 { pin = String(newValue.prefix(6)) }
                                }
                                Button {
                                    pinVisible.toggle()
                                } label: {
                                    Image(systemName: pinVisible ? "eye.fill" : "eye.slash.fill")
                                        .font(.system(size: 14))
                                        .foregroundStyle(LabTheme.fgTertiary)
                                }
                                .accessibilityLabel(pinVisible ? "Hide PIN" : "Show PIN")
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
                            .disabled(isLoading)

                            PinDots(filled: pin.count)
                        }
                    }
                    .padding(LabTheme.s20)
                    .labCard()

                    if let error {
                        DriverStateCard(
                            icon: "exclamationmark.triangle.fill",
                            title: "LOGIN_FAILED",
                            message: error
                        )
                        .transition(.opacity.combined(with: .move(edge: .top)))
                    }

                    Button {
                        doLogin()
                    } label: {
                        HStack(spacing: LabTheme.s12) {
                            if isLoading {
                                ProgressView()
                                    .tint(LabTheme.buttonFg)
                            } else {
                                Image(systemName: "lock.shield.fill")
                                Text("AUTHENTICATE")
                                    .font(.system(size: 16, weight: .black, design: .monospaced))
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 52)
                        .foregroundStyle(LabTheme.buttonFg)
                        .background(
                            isFormValid ? LabTheme.fg : LabTheme.fg.opacity(0.3),
                            in: RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                        )
                    }
                    .disabled(isLoading || !isFormValid)
                    .buttonStyle(.pressable)

                    Spacer()
                }
                .padding(.horizontal, LabTheme.s32)
                .labReadableWidth()
            }
        }
        .animation(Anim.snappy, value: error)
        .onSubmit { doLogin() }
        .onAppear { focusedField = .phone }
    }

    private var isFormValid: Bool {
        phone.trimmingCharacters(in: .whitespaces).count >= 5 && pin.count == 6
    }

    private func doLogin() {
        guard isFormValid else {
            error = "Phone and 6-digit PIN are required"
            return
        }
        focusedField = nil
        isLoading = true
        error = nil

        Task {
            do {
                let response = try await APIClient.shared.login(
                    phone: phone.trimmingCharacters(in: .whitespaces),
                    pin: pin.trimmingCharacters(in: .whitespaces)
                )
                await MainActor.run {
                    TokenStore.shared.save(response: response)
                    Haptics.success()
                    onAuthenticated()
                }
                if let pushToken = PushNotificationManager.shared.deviceToken
                    ?? UserDefaults.standard.string(forKey: "pegasus_push_token"),
                   !pushToken.isEmpty {
                    try? await APIClient.shared.registerDeviceToken(token: pushToken)
                }
            } catch let apiError as APIError {
                await MainActor.run {
                    Haptics.error()
                    switch apiError {
                    case .unauthorized:
                        error = "Invalid phone or PIN"
                    case .forbidden:
                        error = "Account deactivated"
                    case .httpError(let code):
                        error = "Login failed (\(code))"
                    case .networkError:
                        error = "Network error. Check connection."
                    default:
                        error = "Something went wrong."
                    }
                }
            } catch {
                await MainActor.run {
                    Haptics.error()
                    self.error = "Network error. Check connection."
                }
            }
            await MainActor.run { isLoading = false }
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
