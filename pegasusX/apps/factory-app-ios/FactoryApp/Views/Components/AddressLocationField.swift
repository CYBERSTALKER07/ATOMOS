import CoreLocation
import SwiftUI

struct AddressLocationValue: Equatable {
  var address: String = ""
  var lat: Double = 0
  var lng: Double = 0
  var placeId: String?
}

struct AddressLocationField: View {
  @Binding var value: AddressLocationValue
  var label: String = "Address"

  @State private var query = ""
  @State private var suggestions: [GeocodePrediction] = []
  @State private var error: String?
  @State private var locating = false
  @State private var resolving = false
  @FocusState private var focused: Bool

  private var pinned: Bool {
    GeocodeLocationSupport.hasValidCoordinates(lat: value.lat, lng: value.lng)
  }

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      TextField(label, text: $query)
        .textInputAutocapitalization(.words)
        .focused($focused)
        .onChange(of: query) { _, text in
          value.address = text
          Task { await search(text) }
        }
        .onChange(of: focused) { _, isFocused in
          if !isFocused {
            Task { await resolveOnBlur() }
          }
        }
      ForEach(suggestions.prefix(5), id: \.description) { item in
        Button(item.description) {
          Task { await pick(placeId: item.placeId, fallback: item.description) }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
      }
      Button(locating ? "Locating…" : "Share my location") {
        Task { await useMyLocation() }
      }
      .disabled(locating || resolving)
      if pinned {
        Text("factory_portal.location_picker.text.pinned_for_supply_routing")
          .font(.caption)
          .foregroundStyle(.secondary)
      } else if resolving {
        Text("factory_portal.location_picker.text.resolving_address")
          .font(.caption)
          .foregroundStyle(.secondary)
      }
      if let error {
        Text(error).foregroundStyle(.red).font(.caption)
      }
    }
    .onAppear { query = value.address }
    .onChange(of: value.address) { _, next in
      if query != next { query = next }
    }
  }

  @MainActor
  private func search(_ text: String) async {
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.count >= 3 else {
      suggestions = []
      return
    }
    suggestions = (try? await GeocodeService.autocomplete(trimmed)) ?? []
  }

  @MainActor
  private func pick(placeId: String, fallback: String) async {
    let id = placeId.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !id.isEmpty else {
      await resolveText(fallback)
      return
    }
    do {
      let loc = try await GeocodeService.resolvePlace(id)
      apply(loc, fallback: fallback)
    } catch {
      await resolveText(fallback)
    }
  }

  @MainActor
  private func resolveOnBlur() async {
    let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else { return }
    if trimmed == value.address.trimmingCharacters(in: .whitespacesAndNewlines),
       GeocodeLocationSupport.hasValidCoordinates(lat: value.lat, lng: value.lng) {
      return
    }
    await resolveText(trimmed)
  }

  @MainActor
  private func resolveText(_ text: String) async {
    resolving = true
    defer { resolving = false }
    if let resolved = await GeocodeLocationSupport.resolveLocationValue(
      AddressLocationValue(address: text, lat: value.lat, lng: value.lng, placeId: value.placeId)
    ) {
      value = resolved
      query = resolved.address
      suggestions = []
      error = nil
    }
  }

  @MainActor
  private func useMyLocation() async {
    locating = true
    defer { locating = false }
    let manager = CLLocationManager()
    manager.requestWhenInUseAuthorization()
    guard let coord = manager.location?.coordinate else {
      error = "Current location unavailable."
      return
    }
    do {
      let loc = try await GeocodeService.reverse(lat: coord.latitude, lng: coord.longitude)
      apply(loc, fallback: query)
    } catch let locError {
      error = locError.localizedDescription
    }
  }

  @MainActor
  private func apply(_ loc: ResolvedLocation, fallback: String) {
    value = AddressLocationValue(
      address: loc.address.isEmpty ? fallback : loc.address,
      lat: loc.lat,
      lng: loc.lng,
      placeId: loc.placeId
    )
    query = value.address
    suggestions = []
    error = nil
  }
}
