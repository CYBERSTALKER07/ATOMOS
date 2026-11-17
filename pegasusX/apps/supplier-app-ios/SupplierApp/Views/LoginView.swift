import SwiftUI

struct LoginView: View {
    private enum Field: Hashable {
        case phone
        case password
    }

    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var vm = SupplierLoginViewModel()
    @FocusState private var focusedField: Field?

    private var formMaxWidth: CGFloat {
        horizontalSizeClass == .regular ? 420 : 360
    }

    var body: some View {
        @Bindable var vm = vm
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
                            TextField("common.field.phone", text: $vm.phone)
                                .textContentType(.telephoneNumber)
                                .keyboardType(.phonePad)
                                .textFieldStyle(.roundedBorder)
                                .focused($focusedField, equals: .phone)
                                .submitLabel(.next)

                            SecureField("mobile_supplier.chrome.password", text: $vm.password)
                                .textContentType(.password)
                                .textFieldStyle(.roundedBorder)
                                .focused($focusedField, equals: .password)
                                .submitLabel(.go)
                        }
                        .frame(maxWidth: formMaxWidth)

                        if let error = vm.error {
                            Text(error)
                                .font(.caption)
                                .foregroundStyle(SupplierTheme.destructive)
                                .frame(maxWidth: formMaxWidth)
                        }

                        Button(action: submitLogin) {
                            Group {
                                if vm.loading {
                                    ProgressView().tint(.white)
                                } else {
                                    Text("common.action.sign_in")
                                }
                            }
                            .frame(maxWidth: formMaxWidth, minHeight: 48)
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(!vm.canLogin)

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
                .scrollDismissesKeyboard(.interactively)
            }
            .background(SupplierTheme.background)
            .defaultFocus($focusedField, .phone)
            .onAppear {
                focusedField = .phone
            }
            .onSubmit {
                switch focusedField {
                case .phone:
                    focusedField = .password
                case .password:
                    if vm.canLogin {
                        submitLogin()
                    }
                case nil:
                    break
                }
            }
        }
    }

    private func submitLogin() {
        focusedField = nil
        PegasusHaptics.light()
        Task { await vm.login(tokenStore: tokenStore) }
    }
}
