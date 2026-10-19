import SwiftUI

struct LocationSetupView: View {
    @Environment(TokenStore.self) private var tokenStore

    @State private var factoryName = ""
    @State private var location = AddressLocationValue()
    @State private var loading = false
    @State private var submitting = false
    @State private var error: String?

    private var hasAssignedFactory: Bool { tokenStore.hasAssignedFactory }

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView("Loading…")
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    Form {
                        Section {
                            Text(
                                hasAssignedFactory
                                    ? "Confirm or update your facility address. Changes sync with supply routing and loading bay operations."
                                    : "Name your factory and set the facility address to start operations."
                            )
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        }
                        if !hasAssignedFactory {
                            Section("factory_portal.residual.text.factory_name") {
                                TextField("factory_portal.residual.text.factory_name", text: $factoryName)
                            }
                        } else if !factoryName.isEmpty {
                            Section { Text(factoryName) }
                        }
                        Section("Factory address") {
                            AddressLocationField(value: $location, label: "Factory address")
                        }
                        if let error {
                            Section {
                                Text(error).foregroundStyle(.red)
                            }
                        }
                        Section {
                            Button(submitting ? "Saving…" : "Complete setup") {
                                submit()
                            }
                            .disabled(submitting)
                        }
                    }
                }
            }
            .navigationTitle("factory_portal.settings.location.text.factory_location")
        }
        .task { await loadExistingIfNeeded() }
    }

    private func loadExistingIfNeeded() async {
        guard hasAssignedFactory else { return }
        loading = true
        defer { loading = false }
        do {
            let resp = try await FactoryService.factoryLocation()
            factoryName = resp.name
            location = AddressLocationValue(
                address: resp.address,
                lat: resp.lat,
                lng: resp.lng,
                placeId: resp.placeId
            )
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func submit() {
        submitting = true
        error = nil
        Task {
            defer { submitting = false }
            guard let resolved = await GeocodeLocationSupport.resolveLocationValue(location) else {
                error = "Select an address from the suggestions or share your location."
                return
            }
            location = resolved
            do {
                if hasAssignedFactory {
                    _ = try await FactoryService.patchFactoryLocation(
                        address: resolved.address,
                        placeId: resolved.placeId,
                        lat: resolved.lat,
                        lng: resolved.lng
                    )
                    let auth = try await FactoryService.refresh()
                    tokenStore.updateTokens(
                        token: auth.token,
                        refresh: auth.refreshToken,
                        factoryId: auth.factoryId
                    )
                } else {
                    let trimmed = factoryName.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard trimmed.count >= 3 else {
                        error = "Factory name is required."
                        return
                    }
                    let resp = try await FactoryService.setup(
                        factoryName: trimmed,
                        address: resolved.address,
                        placeId: resolved.placeId,
                        lat: resolved.lat,
                        lng: resolved.lng
                    )
                    guard let token = resp.token, !token.isEmpty else {
                        error = "Setup failed."
                        return
                    }
                    tokenStore.updateTokens(
                        token: token,
                        refresh: resp.refreshToken ?? tokenStore.refreshToken ?? "",
                        factoryId: resp.factoryId
                    )
                }
            } catch {
                self.error = error.localizedDescription
            }
        }
    }
}
