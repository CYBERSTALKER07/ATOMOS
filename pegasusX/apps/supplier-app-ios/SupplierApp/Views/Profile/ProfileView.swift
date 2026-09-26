import SwiftUI

struct ProfileView: View {
    @Environment(TokenStore.self) private var tokenStore
    @State private var profile: SupplierProfile?
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    SupplierLoadingView(title: "Loading profile…")
                } else if let error {
                    SupplierErrorView(message: error) { Task { await load() } }
                } else if let profile {
                    ResponsiveGridContentWrapper {
                        SupplierIdentityCard(profile: profile)
                        Section {
                            if !profile.isConfigured {
                                Button("mobile_supplier.ui.complete_billing_setup") {
                                    tokenStore.showBillingGate()
                                }
                            }
                            Button("mobile_supplier.ui.sign_out", role: .destructive) {
                                tokenStore.clear()
                            }
                        }
                    }
                    .supplierReadableWidth()
                }
            }
            .navigationTitle("portal.nav.profile")
            .task { await load() }
            .refreshable { await load(silent: true) }
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            profile = try await SupplierService.profile()
            if let configured = profile?.isConfigured {
                tokenStore.markConfigured(configured)
            }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }
}
