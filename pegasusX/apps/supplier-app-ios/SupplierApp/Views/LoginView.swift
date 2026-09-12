import SwiftUI

struct LoginView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var phone = ""
    @State private var password = ""
    @State private var loading = false
    @State private var error: String?

    private var formMaxWidth: CGFloat {
        horizontalSizeClass == .regular ? 420 : 360
    }

    var body: some View {
        NavigationStack {
        GeometryReader { proxy in
            ScrollView {
                VStack(spacing: SupplierTheme.spacingXXL) {
                    Spacer(minLength: proxy.size.height * 0.08)

                    VStack(spacing: SupplierTheme.spacingSM) {
                        Text("mobile_supplier.ui.pegasus_supplier")
                            .font(horizontalSizeClass == .regular ? .largeTitle.bold() : .title.bold())
                        Text("mobile_supplier.ui.fleet_orders_and_treasury_for_your_operation")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding(.horizontal)

                    VStack(spacing: SupplierTheme.spacingLG) {
                        TextField("common.field.phone", text: $phone)
                            .textContentType(.telephoneNumber)
                            .keyboardType(.phonePad)
                            .textFieldStyle(.roundedBorder)

                        SecureField("Password", text: $password)
                            .textContentType(.password)
                            .textFieldStyle(.roundedBorder)
                    }
                    .frame(maxWidth: formMaxWidth)

                    if let error {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.destructive)
                            .frame(maxWidth: formMaxWidth)
                    }

                    Button(action: login) {
                        Group {
                            if loading {
                                ProgressView().tint(.white)
                            } else {
                                Text("common.action.sign_in")
                            }
                        }
                        .frame(maxWidth: formMaxWidth, minHeight: 48)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(loading || phone.isEmpty || password.isEmpty)

                    NavigationLink {
                        RegisterView()
                    } label: {
                        Text("mobile_supplier.ui.create_supplier_account")
                            .frame(maxWidth: formMaxWidth, minHeight: 44)
                    }
                    .buttonStyle(.bordered)

                    Spacer(minLength: SupplierTheme.spacingXXL)
                }
                .frame(maxWidth: .infinity)
                .padding()
            }
        }
        .background(SupplierTheme.background)
        }
    }

    private func login() {
        loading = true
        error = nil
        Task {
            do {
                let auth = try await SupplierService.login(phone: phone, password: password)
                guard auth.token != nil else {
                    error = "Login succeeded but no session token was returned"
                    loading = false
                    return
                }
                tokenStore.store(auth: auth)
                Task { await PushNotificationManager.shared.uploadStoredTokenIfPossible() }
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}
