import SwiftUI

struct LoginView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var phone = ""
    @State private var pin = ""
    @State private var loading = false
    @State private var error: String?

    var body: some View {
        VStack(spacing: LabTheme.spacingXXL) {
            Spacer()

            VStack(spacing: LabTheme.spacingSM) {
                Text("Pegasus Warehouse")
                    .font(.largeTitle.bold())
                Text("Sign in with your phone and PIN")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            VStack(spacing: LabTheme.spacingLG) {
                TextField("Phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                    .textFieldStyle(.roundedBorder)

                SecureField("PIN", text: $pin)
                    .textContentType(.oneTimeCode)
                    .keyboardType(.numberPad)
                    .textFieldStyle(.roundedBorder)
                    .onChange(of: pin) {
                        if pin.count > 6 { pin = String(pin.prefix(6)) }
                    }
            }
            .frame(maxWidth: 360)

            if let error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }

            Button {
                login()
            } label: {
                Group {
                    if loading {
                        ProgressView()
                            .tint(.white)
                    } else {
                        Text("Sign In")
                    }
                }
                .frame(maxWidth: 360, minHeight: 44)
            }
            .buttonStyle(.borderedProminent)
            .tint(.primary)
            .disabled(loading || phone.isEmpty || pin.isEmpty)

            Spacer()
        }
        .padding()
    }

    private func login() {
        loading = true
        error = nil
        Task {
            do {
                let auth = try await WarehouseService.login(phone: phone, pin: pin)
                tokenStore.store(auth: auth)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
