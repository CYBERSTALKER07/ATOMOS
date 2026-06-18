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

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      TextField(label, text: $query)
        .textInputAutocapitalization(.words)
        .onChange(of: query) { _, text in
          Task { await search(text) }
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
      .disabled(locating)
      if let error {
        Text(error).foregroundStyle(.red).font(.caption)
      }
    }
    .onAppear { query = value.address }
  }

  @MainActor
  private func search(_ text: String) async {
    suggestions = (try? await GeocodeService.autocomplete(text)) ?? []
  }

  @MainActor
  private func pick(placeId: String, fallback: String) async {
    do {
      let loc = try await GeocodeService.resolvePlace(placeId)
      apply(loc, fallback: fallback)
    } catch let pickError {
      error = pickError.localizedDescription
    }
  }

  @MainActor
  private func useMyLocation() async {
    locating = true
    defer { locating = false }
    let manager = CLLocationManager()
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
  }
}
