import SwiftUI

struct LocationSettingsView: View {
  @State private var factoryName = ""
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
        FactoryErrorView(message: error, retry: { load() })
      } else {
        Form {
          if !factoryName.isEmpty {
            Section { Text(factoryName) }
          }
          Section("Factory address") {
            Text("mobile_factory.ui.used_for_supply_routing_and_dispatch_coordinates_stay_hidden_fro")
              .font(.caption)
              .foregroundStyle(.secondary)
            AddressLocationField(value: $location, label: "Factory address")
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
    .navigationTitle("factory_portal.settings.location.text.factory_location")
    .toolbar {
      ToolbarItem(placement: .topBarTrailing) {
        Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
      }
    }
    .task { load() }
  }

  private func load() {
    loading = true
    error = nil
    Task {
      do {
        let resp = try await FactoryService.factoryLocation()
        factoryName = resp.name
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
        _ = try await FactoryService.patchFactoryLocation(
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
