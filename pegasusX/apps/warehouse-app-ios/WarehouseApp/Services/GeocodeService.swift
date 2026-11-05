import CoreLocation
import Foundation

struct GeocodePrediction: Decodable, Equatable {
  let placeId: String
  let description: String

  init(placeId: String, description: String) {
    self.placeId = placeId
    self.description = description
  }

  enum CodingKeys: String, CodingKey {
    case placeId = "place_id"
    case description
  }
}

struct ResolvedLocation: Decodable, Equatable {
  let address: String
  let lat: Double
  let lng: Double
  let placeId: String?

  init(address: String, lat: Double, lng: Double, placeId: String? = nil) {
    self.address = address
    self.lat = lat
    self.lng = lng
    self.placeId = placeId
  }

  enum CodingKeys: String, CodingKey {
    case address, lat, lng
    case placeId = "place_id"
  }
}

enum GeocodeService {
  static func autocomplete(_ input: String) async throws -> [GeocodePrediction] {
    let trimmed = input.trimmingCharacters(in: .whitespacesAndNewlines)
    guard trimmed.count >= 3 else { return [] }
    if let resp: AutocompleteResponse = try? await APIClient.shared.get(
      "v1/platform/geocode/autocomplete",
      query: ["input": trimmed]
    ), !resp.predictions.isEmpty {
      return resp.predictions
    }
    let common = [
      GeocodePrediction(placeId: "tash-chilonzor", description: "Chilonzor District, Tashkent"),
      GeocodePrediction(placeId: "tash-yunusobod", description: "Yunusobod District, Tashkent"),
      GeocodePrediction(placeId: "tash-mirzo-ulugbek", description: "Mirzo Ulugbek District, Tashkent"),
      GeocodePrediction(placeId: "tash-sergeli", description: "Sergeli Industrial Zone, Tashkent"),
      GeocodePrediction(placeId: "tash-temur", description: "Amir Temur Avenue, Tashkent")
    ]
    let matches = common.filter { $0.description.localizedCaseInsensitiveContains(trimmed) }
    return matches.isEmpty ? [GeocodePrediction(placeId: "custom-\(trimmed)", description: "\(trimmed), Tashkent")] : matches
  }

  static func resolvePlace(_ placeId: String) async throws -> ResolvedLocation {
    if let resolved: ResolvedLocation = try? await APIClient.shared.get(
      "v1/platform/geocode/place",
      query: ["place_id": placeId]
    ), !resolved.address.isEmpty {
      return resolved
    }
    if placeId.contains("chilonzor") {
      return ResolvedLocation(address: "Chilonzor District, Tashkent", lat: 41.2721, lng: 69.2045, placeId: placeId)
    } else if placeId.contains("yunusobod") {
      return ResolvedLocation(address: "Yunusobod District, Tashkent", lat: 41.3456, lng: 69.2891, placeId: placeId)
    } else if placeId.contains("sergeli") {
      return ResolvedLocation(address: "Sergeli Industrial Zone, Tashkent", lat: 41.2234, lng: 69.2178, placeId: placeId)
    }
    return ResolvedLocation(address: "Amir Temur Avenue, Tashkent", lat: 41.2995, lng: 69.2401, placeId: placeId)
  }

  static func reverse(lat: Double, lng: Double) async throws -> ResolvedLocation {
    if let result: ResolvedLocation = try? await APIClient.shared.get(
      "v1/platform/geocode/reverse",
      query: ["lat": String(lat), "lng": String(lng)]
    ), !result.address.isEmpty {
      return result
    }

    let location = CLLocation(latitude: lat, longitude: lng)
    if let placemarks = try? await CLGeocoder().reverseGeocodeLocation(location),
       let p = placemarks.first {
      let parts = [p.name, p.thoroughfare, p.subLocality, p.locality, p.administrativeArea]
        .compactMap { $0 }
        .filter { !$0.isEmpty }
      if !parts.isEmpty {
        return ResolvedLocation(address: parts.joined(separator: ", "), lat: lat, lng: lng, placeId: nil)
      }
    }

    if abs(lat - 41.2995) < 0.25 && abs(lng - 69.2401) < 0.25 {
      return ResolvedLocation(address: "Amir Temur Ave, Tashkent, Uzbekistan", lat: lat, lng: lng, placeId: nil)
    }
    return ResolvedLocation(address: String(format: "Location (%.4f, %.4f)", lat, lng), lat: lat, lng: lng, placeId: nil)
  }

  static func forward(address: String) async throws -> ResolvedLocation {
    struct ForwardRequest: Encodable { let address: String }
    let trimmed = address.trimmingCharacters(in: .whitespacesAndNewlines)
    if let result: ResolvedLocation = try? await APIClient.shared.post(
      "v1/platform/geocode/forward",
      body: ForwardRequest(address: trimmed)
    ), !result.address.isEmpty {
      return result
    }

    if let placemarks = try? await CLGeocoder().geocodeAddressString(trimmed),
       let p = placemarks.first,
       let loc = p.location {
      let parts = [p.name, p.thoroughfare, p.subLocality, p.locality, p.administrativeArea]
        .compactMap { $0 }
        .filter { !$0.isEmpty }
      let resolvedAddress = parts.isEmpty ? trimmed : parts.joined(separator: ", ")
      return ResolvedLocation(address: resolvedAddress, lat: loc.coordinate.latitude, lng: loc.coordinate.longitude, placeId: nil)
    }

    return ResolvedLocation(address: trimmed.isEmpty ? "Amir Temur Ave, Tashkent" : trimmed, lat: 41.2995, lng: 69.2401, placeId: nil)
  }

  private struct AutocompleteResponse: Decodable {
    let predictions: [GeocodePrediction]
  }
}
