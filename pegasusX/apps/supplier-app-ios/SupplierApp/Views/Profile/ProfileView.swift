import SwiftUI

struct ProfileView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var viewModel = ProfileViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.profile == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.profile")
            } else if let error = viewModel.error, viewModel.profile == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load(tokenStore: tokenStore) } }
            } else if let profile = viewModel.profile {
                ResponsiveGridContentWrapper {
                    SupplierIdentityCard(profile: profile)
                    Section {
                        if !profile.isConfigured {
                            Button("mobile_supplier.ui.complete_billing_setup") {
                                PegasusHaptics.light()
                                tokenStore.showBillingGate()
                            }
                            .frame(minHeight: 44)
                        }
                        Button("mobile_supplier.ui.sign_out", role: .destructive) {
                            PegasusHaptics.selection()
                            tokenStore.clear()
                        }
                        .frame(minHeight: 44)
                    }
                }
                .supplierReadableWidth()
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.profile")
        .task { await viewModel.load(tokenStore: tokenStore) }
        .refreshable { await viewModel.load(silent: true, tokenStore: tokenStore) }
    }
}
