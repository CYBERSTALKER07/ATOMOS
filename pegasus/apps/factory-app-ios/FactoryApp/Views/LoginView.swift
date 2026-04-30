import SwiftUI

struct LoginView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var phone = ""
    @State private var password = ""
    @State private var loading = false
    @State private var error: String?

    var body: some View {
        VStack(spacing: LabTheme.spacingXXL) {
            Spacer()

            VStack(spacing: LabTheme.spacingSM) {
                Text("Lab Factory")
                    .font(.largeTitle.bold())
                Text("Sign in to manage factory operations")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            VStack(spacing: LabTheme.spacingLG) {
                TextField("Phone", text: $phone)
                    .textContentType(.telephoneNumber)
                    .keyboardType(.phonePad)
                    .textFieldStyle(.roundedBorder)

                SecureField("Password", text: $password)
                    .textContentType(.password)
                    .textFieldStyle(.roundedBorder)
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
            .disabled(loading || phone.isEmpty || password.isEmpty)

            Spacer()
        }
        .padding()
    }

    private func login() {
        loading = true
        error = nil
        Task {
            do {
                let auth = try await FactoryService.login(phone: phone, password: password)
                tokenStore.store(auth: auth)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
