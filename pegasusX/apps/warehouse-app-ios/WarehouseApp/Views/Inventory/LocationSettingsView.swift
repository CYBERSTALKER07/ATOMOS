import SwiftUI

struct LocationSettingsView: View {
    @State private var warehouseName = ""
    @State private var location = AddressLocationValue()
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var saveMessage: String?

    var body: some View {
        Group {
            if loading {
                ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView {
                    Label("Error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("Retry") { load() }
                }
            } else {
                Form {
                    if !warehouseName.isEmpty {
                        Section { Text(warehouseName) }
                    }
                    Section("Depot address") {
                        Text("Used for smart dispatch routing. Coordinates stay hidden from daily ops screens.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        AddressLocationField(value: $location, label: "Warehouse address")
                    }
                    if let saveMessage {
                        Section { Text(saveMessage).foregroundStyle(saveMessage.contains("saved") ? .green : .red) }
                    }
                    Section {
                        Button(saving ? "Saving…" : "Save location") { save() }
                            .disabled(saving || location.address.isEmpty)
                    }
                }
            }
        }
        .navigationTitle("Depot location")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await WarehouseService.warehouseLocation()
                warehouseName = resp.name
                location = AddressLocationValue(
                    address: resp.address,
                    lat: resp.lat,
                    lng: resp.lng,
                    placeId: resp.placeId
                )
                loading = false
            } catch {
                self.error = error.localizedDescription
                loading = false
            }
        }
    }

    private func save() {
        saving = true
        saveMessage = nil
        Task {
            defer { saving = false }
            guard let resolved = await GeocodeLocationSupport.resolveLocationValue(location) else {
                saveMessage = "Select an address from the suggestions or share your location."
                return
            }
            location = resolved
            do {
                _ = try await WarehouseService.patchWarehouseLocation(
                    address: resolved.address,
                    placeId: resolved.placeId,
                    lat: resolved.lat,
                    lng: resolved.lng
                )
                saveMessage = "Location saved"
                load()
            } catch let saveError {
                saveMessage = saveError.localizedDescription
            }
        }
    }
}
