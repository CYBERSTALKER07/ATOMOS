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
  @State private var searchTask: Task<Void, Never>?

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      TextField(label, text: $query)
        .textInputAutocapitalization(.words)
        .onChange(of: query) { _, text in
          searchTask?.cancel()
          searchTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: 280_000_000) // 280ms keystroke debounce
            guard !Task.isCancelled else { return }
            await search(text)
          }
        }
        .onDisappear {
          searchTask?.cancel()
          searchTask = nil
        }
      if !suggestions.isEmpty {
        ForEach(suggestions.prefix(5), id: \.placeId) { item in
          Button(item.description) {
            Task { await pick(placeId: item.placeId, fallback: item.description) }
          }
          .frame(maxWidth: .infinity, alignment: .leading)
        }
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
    .onChange(of: value.address) { _, next in
      if query != next { query = next }
    }
  }

  @MainActor
  private func search(_ text: String) async {
    error = nil
    let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.count >= 3 else {
      suggestions = []
      return
    }
    do {
      suggestions = try await GeocodeService.autocomplete(trimmed)
    } catch {
      suggestions = []
    }
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
    manager.requestWhenInUseAuthorization()
    var coord = manager.location?.coordinate
    if coord == nil || (coord?.latitude == 0 && coord?.longitude == 0) {
      coord = CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401)
    }
    guard let target = coord else {
      error = "Current location unavailable."
      return
    }
    do {
      let loc = try await GeocodeService.reverse(lat: target.latitude, lng: target.longitude)
      apply(loc, fallback: query.isEmpty ? "Amir Temur Ave, Tashkent" : query)
    } catch {
      apply(
        ResolvedLocation(address: "Amir Temur Ave, Tashkent, Uzbekistan", lat: target.latitude, lng: target.longitude, placeId: nil),
        fallback: "Amir Temur Ave, Tashkent, Uzbekistan"
      )
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
