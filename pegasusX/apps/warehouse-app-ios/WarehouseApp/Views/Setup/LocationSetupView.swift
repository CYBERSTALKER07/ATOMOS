import SwiftUI

struct LocationSetupView: View {
    @Environment(TokenStore.self) private var tokenStore

    @State private var warehouseName = ""
    @State private var location = AddressLocationValue()
    @State private var loading = false
    @State private var submitting = false
    @State private var error: String?

    private var hasAssignedWarehouse: Bool { tokenStore.hasAssignedWarehouse }

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
                                hasAssignedWarehouse
                                    ? "Confirm or update your depot address. Changes sync with dispatch and delivery routing."
                                    : "Name your warehouse and set the depot address to start operations."
                            )
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        }
                        if !hasAssignedWarehouse {
                            Section("Warehouse name") {
                                TextField("Warehouse name", text: $warehouseName)
                            }
                        } else if !warehouseName.isEmpty {
                            Section { Text(warehouseName) }
                        }
                        Section("Depot address") {
                            AddressLocationField(value: $location, label: "Depot address")
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
            .navigationTitle("Warehouse location")
        }
        .task { await loadExistingIfNeeded() }
    }

    private func loadExistingIfNeeded() async {
        guard hasAssignedWarehouse else { return }
        loading = true
        defer { loading = false }
        do {
            let resp = try await WarehouseService.warehouseLocation()
            warehouseName = resp.name
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
                if hasAssignedWarehouse {
                    _ = try await WarehouseService.patchWarehouseLocation(
                        address: resolved.address,
                        placeId: resolved.placeId,
                        lat: resolved.lat,
                        lng: resolved.lng
                    )
                    if let refresh = tokenStore.refreshToken {
                        let auth = try await WarehouseOperationsService.refreshToken(refresh)
                        tokenStore.updateTokens(
                            token: auth.token,
                            refresh: auth.refreshToken,
                            warehouseId: auth.warehouseId
                        )
                    }
                } else {
                    let trimmed = warehouseName.trimmingCharacters(in: .whitespacesAndNewlines)
                    guard trimmed.count >= 3 else {
                        error = "Warehouse name is required."
                        return
                    }
                    let resp = try await WarehouseService.setup(
                        name: trimmed,
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
                        warehouseId: resp.warehouseId
                    )
                }
            } catch {
                self.error = error.localizedDescription
            }
        }
    }
}
