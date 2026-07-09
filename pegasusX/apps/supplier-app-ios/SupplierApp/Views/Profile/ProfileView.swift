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
                        Section("Business") {
                            LabeledContent("Legal name", value: profile.legalName)
                            LabeledContent("Contact", value: profile.contactName)
                            LabeledContent("Email", value: profile.email)
                            LabeledContent("Phone", value: profile.phone)
                            LabeledContent("Country", value: profile.country)
                            LabeledContent("Currency", value: profile.currency)
                        }
                        Section("Status") {
                            LabeledContent("Registered", value: profile.isRegistered ? "Yes" : "No")
                            LabeledContent("Configured", value: profile.isConfigured ? "Yes" : "No")
                            if !profile.selectedGateways.isEmpty {
                                LabeledContent("Gateways", value: profile.selectedGateways.joined(separator: ", "))
                            }
                        }
                        if !profile.categories.isEmpty {
                            Section("Categories") {
                                ForEach(profile.categories, id: \.self) { Text($0) }
                            }
                        }
                        Section {
                            if !profile.isConfigured {
                                Button("Complete billing setup") {
                                    tokenStore.showBillingGate()
                                }
                            }
                            Button("Sign out", role: .destructive) {
                                tokenStore.clear()
                            }
                        }
                    }
                    .supplierReadableWidth()
                }
            }
            .navigationTitle("Profile")
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
