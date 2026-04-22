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
                VStack(spacing: 0) {
                    Spacer().frame(height: 100)

                    // Logo
                    Image(systemName: "shippingbox.fill")
                        .font(.system(size: 56, weight: .medium))
                        .foregroundStyle(LabTheme.fg)
                        .padding(.bottom, 16)

                    Text("The Lab")
                        .font(.system(size: 28, weight: .bold))
                        .foregroundStyle(LabTheme.fg)

                    Text("Driver Terminal")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .padding(.bottom, 48)

                    // Fields
                    VStack(spacing: 16) {
                        // Phone
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Phone Number")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(LabTheme.fgSecondary)

                            HStack(spacing: 12) {
                                Image(systemName: "phone.fill")
                                    .font(.system(size: 14))
                                    .foregroundStyle(LabTheme.fgTertiary)

                                TextField("+998 XX XXX XX XX", text: $phone)
                                    .keyboardType(.phonePad)
                                    .textContentType(.telephoneNumber)
                                    .focused($focusedField, equals: .phone)
                                    .font(.system(size: 16, weight: .medium, design: .monospaced))
                                    .foregroundStyle(LabTheme.fg)
                            }
                            .padding(.horizontal, 16)
                            .padding(.vertical, 14)
                            .background(
                                RoundedRectangle(cornerRadius: 14, style: .continuous)
                                    .fill(LabTheme.card)
                            )
                            .overlay {
                                RoundedRectangle(cornerRadius: 14, style: .continuous)
                                    .stroke(
                                        focusedField == .phone
                                            ? LabTheme.fg.opacity(0.3)
                                            : LabTheme.separator,
                                        lineWidth: 0.5
                                    )
                            }
                        }

                        // PIN
                        VStack(alignment: .leading, spacing: 6) {
                            Text("PIN")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(LabTheme.fgSecondary)

                            HStack(spacing: 12) {
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
                                .focused($focusedField, equals: .pin)
                                .font(.system(size: 16, weight: .medium, design: .monospaced))
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
                            .padding(.horizontal, 16)
                            .padding(.vertical, 14)
                            .background(
                                RoundedRectangle(cornerRadius: 14, style: .continuous)
                                    .fill(LabTheme.card)
                            )
                            .overlay {
                                RoundedRectangle(cornerRadius: 14, style: .continuous)
                                    .stroke(
                                        focusedField == .pin
                                            ? LabTheme.fg.opacity(0.3)
                                            : LabTheme.separator,
                                        lineWidth: 0.5
                                    )
                            }
                        }
                    }
                    .padding(.horizontal, 32)

                    // Error
                    if let error {
                        Text(error)
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(LabTheme.destructive)
                            .multilineTextAlignment(.center)
                            .padding(.top, 12)
                            .padding(.horizontal, 32)
                            .transition(.opacity.combined(with: .move(edge: .top)))
                    }

                    Spacer().frame(height: 32)

                    // Login button
                    Button {
                        doLogin()
                    } label: {
                        HStack(spacing: 8) {
                            if isLoading {
                                ProgressView()
                                    .tint(LabTheme.buttonFg)
                                    .scaleEffect(0.8)
                            } else {
                                Text("Sign In")
                                    .font(.system(size: 16, weight: .bold))
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                        .foregroundStyle(LabTheme.buttonFg)
                        .background(
                            RoundedRectangle(cornerRadius: 14, style: .continuous)
                                .fill(isFormValid
                                    ? LabTheme.fg
                                    : LabTheme.fg.opacity(0.3))
                        )
                    }
                    .disabled(isLoading || !isFormValid)
                    .padding(.horizontal, 32)
                    .buttonStyle(.pressable)

                    Spacer()
                }
            }
        }
        .animation(Anim.snappy, value: error)
        .onSubmit { doLogin() }
    }

    // MARK: - Validation

    private var isFormValid: Bool {
        phone.count >= 5 && !pin.isEmpty
    }

    // MARK: - Login

    private func doLogin() {
        guard isFormValid else {
            error = "Phone and PIN are required"
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

#Preview {
    LoginView(onAuthenticated: {})
}
